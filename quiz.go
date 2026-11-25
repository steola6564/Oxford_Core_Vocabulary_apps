package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type QuestionSpec struct {
	QuestionID ID     `json:"question_id"`
	QuizType   string `json:"quiz_type"`
	Variant    string `json:"variant"`
	Parameters string `json:"parameters"`
}

func (s *Store) parseSpec(q QuestionSpec) ([]int, error) {
	if err := validQuiz(q.QuizType, q.Variant); err != nil {
		return nil, err
	}
	p := strings.Split(q.Parameters, ",")
	values := make([]int, len(p))
	for i, v := range p {
		if v == "" {
			return nil, fmt.Errorf("empty parameter")
		}
		for _, c := range v {
			if c < '0' || c > '9' {
				return nil, fmt.Errorf("parameters must be unsigned decimal integers")
			}
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
		values[i] = n
	}
	if len(values) < 1 || s.Corpus.ByID[values[0]] == nil {
		return nil, fmt.Errorf("unknown target Corpus ID")
	}
	switch q.QuizType {
	case "word_recall":
		if len(values) != 4 {
			return nil, fmt.Errorf("recall requires four Corpus IDs")
		}
		seen := map[int]bool{}
		labels := map[string]bool{}
		for _, v := range values {
			r := s.Corpus.ByID[v]
			if r == nil || seen[v] {
				return nil, fmt.Errorf("invalid or duplicate recall Corpus ID")
			}
			seen[v] = true
			label := normalize(r.JP)
			if q.Variant == "jp_to_en" {
				label = normalize(r.Word)
			}
			if labels[label] {
				return nil, fmt.Errorf("ambiguous duplicate choice label")
			}
			labels[label] = true
		}
	case "cloze":
		if len(values) != 3 {
			return nil, fmt.Errorf("cloze requires ID,start,length")
		}
		ex := []rune(s.Corpus.ByID[values[0]].Example)
		start, length := values[1], values[2]
		if start < 0 || length < 1 || start > len(ex) || length > len(ex)-start || strings.TrimSpace(string(ex[start:start+length])) == "" {
			return nil, fmt.Errorf("cloze span outside example or empty")
		}
	case "reading_review":
		if len(values) != 1 {
			return nil, fmt.Errorf("reading requires one Corpus ID")
		}
	}
	return values, nil
}

func (s *Store) getSpec(q querier, id ID) (QuestionSpec, error) {
	var spec QuestionSpec
	err := q.QueryRow("SELECT question_id,quiz_type,COALESCE(variant,''),parameters FROM question_specs WHERE question_id=?", id).Scan(&spec.QuestionID, &spec.QuizType, &spec.Variant, &spec.Parameters)
	if err != nil {
		return spec, asNotFound(err, "Question")
	}
	_, err = s.parseSpec(spec)
	return spec, err
}

func (s *Store) makeSpec(row *Record, quiz, variant string) (QuestionSpec, error) {
	q := QuestionSpec{QuizType: quiz, Variant: variant}
	values := []int{row.ID}
	switch quiz {
	case "word_recall":
		label := func(r *Record) string {
			if variant == "en_to_jp" {
				return normalize(r.JP)
			}
			return normalize(r.Word)
		}
		seen := map[string]bool{label(row): true}
		used := map[int]bool{row.ID: true}
		// Prefer candidates with the same part of speech, then broaden if needed.
		for _, samePOS := range []bool{true, false} {
			order := rand.Perm(len(s.Corpus.Rows))
			for _, idx := range order {
				r := s.Corpus.Rows[idx]
				if used[r.ID] || seen[label(r)] || (samePOS && r.POS != row.POS) {
					continue
				}
				values = append(values, r.ID)
				used[r.ID] = true
				seen[label(r)] = true
				if len(values) == 4 {
					break
				}
			}
			if len(values) == 4 {
				break
			}
		}
		if len(values) != 4 {
			return q, fmt.Errorf("insufficient distinct answer labels")
		}
	case "cloze":
		if row.cloze == nil {
			return q, bad("target has no safely located cloze span")
		}
		values = append(values, row.cloze.Start, row.cloze.Length)
	case "reading_review":
	default:
		return q, bad("unsupported quiz type")
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = strconv.Itoa(v)
	}
	q.Parameters = strings.Join(parts, ",")
	_, err := s.parseSpec(q)
	return q, err
}

type SessionRequest struct {
	QuizType   string  `json:"quiz_type"`
	Variant    string  `json:"variant"`
	SourceMode string  `json:"source_mode"`
	Filter     Filter  `json:"filter"`
	Action     *Action `json:"action,omitempty"`
}

func (r *SessionRequest) validate() error {
	if err := validQuiz(r.QuizType, r.Variant); err != nil {
		return err
	}
	if r.SourceMode != "new" && r.SourceMode != "review" {
		return bad("source_mode must be new or review")
	}
	if err := validateFilterAPI(&r.Filter); err != nil {
		return err
	}
	if r.SourceMode == "new" && (len(r.Filter.HistoryResult) > 0 || len(r.Filter.ConfidenceRange) > 0 || len(r.Filter.AttemptTimeRange) > 0) {
		return bad("history filters apply to Review only")
	}
	return nil
}

type candidate struct {
	Row  *Record
	Spec *QuestionSpec
}
type selectionItem struct {
	CorpusID int
	Spec     *QuestionSpec
}

func (s *Store) selectionItems(q querier, ids []ID) ([]selectionItem, error) {
	items := []selectionItem{}
	for _, id := range ids {
		var exists int
		if err := q.QueryRow("SELECT COUNT(*) FROM collections WHERE collection_id=?", id).Scan(&exists); err != nil {
			return nil, err
		}
		if exists == 0 {
			return nil, notFound("Collection " + id.String() + " not found")
		}
		rows, err := q.Query(`SELECT i.corpus_id,i.question_id,q.quiz_type,COALESCE(q.variant,''),q.parameters
FROM collection_items i LEFT JOIN question_specs q ON q.question_id=i.question_id WHERE i.collection_id=? ORDER BY i.item_id`, id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var corpusID, questionID sql.NullInt64
			var qt, param sql.NullString
			var variant string
			if err = rows.Scan(&corpusID, &questionID, &qt, &variant, &param); err != nil {
				rows.Close()
				return nil, err
			}
			item := selectionItem{}
			if corpusID.Valid {
				item.CorpusID = int(corpusID.Int64)
			} else {
				spec := QuestionSpec{QuestionID: ID(questionID.Int64), QuizType: qt.String, Variant: variant, Parameters: param.String}
				p, e := s.parseSpec(spec)
				if e != nil {
					rows.Close()
					return nil, e
				}
				item.CorpusID = p[0]
				item.Spec = &spec
			}
			items = append(items, item)
		}
		if err = rows.Err(); err != nil {
			rows.Close()
			return nil, err
		}
		rows.Close()
	}
	return items, nil
}

func (s *Store) corpusSelection(q querier, f Filter) (map[int]bool, error) {
	if len(f.CollectionIDs) == 0 {
		return nil, nil
	}
	items, err := s.selectionItems(q, f.CollectionIDs)
	if err != nil {
		return nil, err
	}
	set := map[int]bool{}
	for _, item := range items {
		set[item.CorpusID] = true
	}
	return set, nil
}

func (s *Store) Search(f Filter) ([]*Record, error) {
	if err := validateFilterAPI(&f); err != nil {
		return nil, err
	}
	set, err := s.corpusSelection(s.db, f)
	if err != nil {
		return nil, err
	}
	out := []*Record{}
	for _, r := range s.Corpus.Rows {
		if s.Corpus.Match(r, f) && (set == nil || set[r.ID]) {
			out = append(out, r)
		}
	}
	return out, nil
}

func (s *Store) candidates(q querier, r SessionRequest) ([]candidate, int, error) {
	if err := r.validate(); err != nil {
		return nil, 0, err
	}
	out := []candidate{}
	skipped := 0
	var selected []selectionItem
	var err error
	if len(r.Filter.CollectionIDs) > 0 {
		selected, err = s.selectionItems(q, r.Filter.CollectionIDs)
		if err != nil {
			return nil, 0, err
		}
	}
	if r.SourceMode == "new" {
		if len(r.Filter.CollectionIDs) == 0 {
			for _, row := range s.Corpus.Rows {
				selected = append(selected, selectionItem{CorpusID: row.ID})
			}
		}
		for _, item := range selected {
			row := s.Corpus.ByID[item.CorpusID]
			if row == nil {
				return nil, 0, fmt.Errorf("missing Corpus row")
			}
			if !s.Corpus.Match(row, r.Filter) {
				continue
			}
			if item.Spec != nil {
				if item.Spec.QuizType != r.QuizType || item.Spec.Variant != r.Variant {
					continue
				}
				out = append(out, candidate{row, item.Spec})
				continue
			}
			if r.QuizType == "cloze" && row.cloze == nil {
				skipped++
				continue
			}
			out = append(out, candidate{Row: row})
		}
		return out, skipped, nil
	}
	// Match all history constraints against ONE Attempt. Distinct specs by default.
	history, err := s.readAttempts(q)
	if err != nil {
		return nil, 0, err
	}
	eligible := map[ID]QuestionSpec{}
	order := []ID{}
	for _, a := range history {
		if a.QuizType != r.QuizType || a.Variant != r.Variant || !s.Corpus.Match(a.Corpus, r.Filter) || !matchesHistory(a.Attempt, r.Filter) {
			continue
		}
		if _, exists := eligible[a.QuestionID]; !exists {
			spec, err := s.getSpec(q, a.QuestionID)
			if err != nil {
				return nil, 0, err
			}
			eligible[a.QuestionID] = spec
			order = append(order, a.QuestionID)
		}
	}
	if len(r.Filter.CollectionIDs) == 0 {
		for _, id := range order {
			spec := eligible[id]
			p, _ := s.parseSpec(spec)
			out = append(out, candidate{s.Corpus.ByID[p[0]], &spec})
		}
	} else {
		for _, item := range selected {
			for _, id := range order {
				spec := eligible[id]
				p, _ := s.parseSpec(spec)
				if (item.Spec != nil && item.Spec.QuestionID == id) || (item.Spec == nil && item.CorpusID == p[0]) {
					out = append(out, candidate{s.Corpus.ByID[p[0]], &spec})
				}
			}
		}
	}
	return out, 0, nil
}

func (s *Store) Candidates(r SessionRequest) (int, int, error) {
	if err := r.validate(); err != nil {
		return 0, 0, err
	}
	items, skipped, err := s.candidates(s.db, r)
	return len(items), skipped, err
}

type SessionPosition struct {
	Position     int  `json:"position"`
	QuestionID   ID   `json:"question_id"`
	Answered     bool `json:"answered"`
	AttemptCount int  `json:"attempt_count"`
}
type Session struct {
	SessionID       ID                `json:"session_id"`
	QuizType        string            `json:"quiz_type"`
	Variant         *string           `json:"variant"`
	SourceMode      string            `json:"source_mode"`
	FilterSpec      string            `json:"filter_spec"`
	Filter          Filter            `json:"filter"`
	CurrentPosition int               `json:"current_position"`
	Status          string            `json:"status"`
	Total           int               `json:"total"`
	Answered        int               `json:"answered"`
	CreatedAt       *int64            `json:"created_at"`
	Questions       []SessionPosition `json:"questions,omitempty"`
	ClozeSkipped    int               `json:"cloze_skipped,omitempty"`
}

const sessionSelect = `SELECT s.session_id,s.quiz_type,s.variant,s.source_mode,s.filter_spec,s.current_position,s.status,
(SELECT COUNT(*) FROM session_questions sq WHERE sq.session_id=s.session_id),
(SELECT COUNT(DISTINCT a.position) FROM attempt_logs a WHERE a.session_id=s.session_id),
(SELECT MIN(l.occurred_at) FROM action_logs l WHERE l.session_id=s.session_id AND l.event='session_start') FROM sessions s`

func scanSession(row interface{ Scan(...any) error }) (Session, error) {
	v := Session{}
	err := row.Scan(&v.SessionID, &v.QuizType, &v.Variant, &v.SourceMode, &v.FilterSpec, &v.CurrentPosition, &v.Status, &v.Total, &v.Answered, &v.CreatedAt)
	if err != nil {
		return v, err
	}
	v.Filter, err = DecodeFilter(v.FilterSpec)
	return v, err
}

func readSession(q querier, id ID) (Session, error) {
	v, err := scanSession(q.QueryRow(sessionSelect+" WHERE s.session_id=?", id))
	if err != nil {
		return v, asNotFound(err, "Session")
	}
	v.Questions = []SessionPosition{}
	rows, err := q.Query(`SELECT sq.position,sq.question_id,COUNT(a.attempt_id) FROM session_questions sq
LEFT JOIN attempt_logs a ON a.session_id=sq.session_id AND a.position=sq.position
WHERE sq.session_id=? GROUP BY sq.position,sq.question_id ORDER BY sq.position`, id)
	if err != nil {
		return v, err
	}
	defer rows.Close()
	for rows.Next() {
		p := SessionPosition{}
		if err = rows.Scan(&p.Position, &p.QuestionID, &p.AttemptCount); err != nil {
			return v, err
		}
		p.Answered = p.AttemptCount > 0
		v.Questions = append(v.Questions, p)
	}
	return v, rows.Err()
}

func (s *Store) GetSession(id ID) (Session, error) { return readSession(s.db, id) }
func (s *Store) ListSessions(quiz string) ([]Session, error) {
	query := sessionSelect
	args := []any{}
	if quiz != "" {
		query += " WHERE s.quiz_type=?"
		args = append(args, quiz)
	}
	query += " ORDER BY s.session_id DESC"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Session{}
	for rows.Next() {
		v, e := scanSession(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) CreateSession(r SessionRequest) (Session, error) {
	if err := r.validate(); err != nil {
		return Session{}, err
	}
	var id ID
	skipped := 0
	err := s.transaction(func(tx *sql.Tx) error {
		items, n, err := s.candidates(tx, r)
		if err != nil {
			return err
		}
		skipped = n
		if len(items) == 0 {
			return &AppError{400, "no_candidates", "No matching questions. Change the filters or answer some questions before Review."}
		}
		rand.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
		if r.Filter.RequestedCount > 0 && len(items) > r.Filter.RequestedCount {
			items = items[:r.Filter.RequestedCount]
		}
		if len(items) > 100000 {
			return bad("A Session is limited to 100000 positions")
		}
		filterSpec, err := EncodeFilter(r.Filter)
		if err != nil {
			return err
		}
		id, err = allocate(tx, "sessions", "session_id")
		if err != nil {
			return err
		}
		_, err = tx.Exec("INSERT INTO sessions(session_id,quiz_type,variant,source_mode,filter_spec,current_position,status) VALUES(?,?,?,?,?,1,'active')", id, r.QuizType, nullableVariant(r.Variant), r.SourceMode, filterSpec)
		if err != nil {
			return err
		}
		for i, item := range items {
			var spec QuestionSpec
			if item.Spec != nil {
				spec = *item.Spec
			} else {
				spec, err = s.makeSpec(item.Row, r.QuizType, r.Variant)
				if err != nil {
					return err
				}
				spec.QuestionID, err = allocate(tx, "question_specs", "question_id")
				if err != nil {
					return err
				}
				_, err = tx.Exec("INSERT INTO question_specs(question_id,quiz_type,variant,parameters) VALUES(?,?,?,?)", spec.QuestionID, spec.QuizType, nullableVariant(spec.Variant), spec.Parameters)
				if err != nil {
					return err
				}
			}
			_, err = tx.Exec("INSERT INTO session_questions(session_id,position,question_id) VALUES(?,?,?)", id, i+1, spec.QuestionID)
			if err != nil {
				return err
			}
		}
		a, err := actionFor(tx, r.Action, "submode_selection", "session_start", "session:"+id.String(), id, 0)
		if err != nil {
			return err
		}
		_, err = insertAction(tx, a)
		return err
	})
	if err != nil {
		return Session{}, err
	}
	v, err := s.GetSession(id)
	v.ClozeSkipped = skipped
	return v, err
}

func (s *Store) UpdateSession(id ID, position int, status string) (Session, error) {
	err := s.transaction(func(tx *sql.Tx) error {
		v, err := readSession(tx, id)
		if err != nil {
			return err
		}
		if position == 0 {
			position = v.CurrentPosition
		}
		if position < 1 || position > v.Total {
			return bad("current_position outside Session")
		}
		if status == "" {
			status = v.Status
		}
		if status != "active" && status != "paused" && status != "completed" {
			return bad("invalid Session status")
		}
		if v.Answered == v.Total {
			status = "completed"
		} else if status == "completed" {
			return bad("unanswered positions prevent completion")
		}
		_, err = tx.Exec("UPDATE sessions SET current_position=?,status=? WHERE session_id=?", position, status, id)
		return err
	})
	if err != nil {
		return Session{}, err
	}
	return s.GetSession(id)
}

type Choice struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}
type Feedback struct {
	CorrectCorpusID int       `json:"correct_corpus_id"`
	ExpectedAnswer  *string   `json:"expected_answer"`
	Corpus          *Record   `json:"corpus"`
	Options         []*Record `json:"options"`
}
type Question struct {
	SessionID     ID        `json:"session_id"`
	Position      int       `json:"position"`
	Total         int       `json:"total"`
	QuestionID    ID        `json:"question_id"`
	QuizType      string    `json:"quiz_type"`
	Variant       *string   `json:"variant"`
	Corpus        *Record   `json:"corpus"`
	Prompt        string    `json:"prompt"`
	ClozeText     *string   `json:"cloze_text"`
	Options       []Choice  `json:"options"`
	LatestAttempt *Attempt  `json:"latest_attempt"`
	Feedback      *Feedback `json:"feedback"`
}

func choiceOrder(ids []int, sid ID, position int) []int {
	out := append([]int{}, ids...)
	seed := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:choices-v1", sid, position)))
	rng := rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(seed[:8]))))
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

func (s *Store) feedback(spec QuestionSpec, sid ID, position int) (Feedback, error) {
	p, err := s.parseSpec(spec)
	if err != nil {
		return Feedback{}, err
	}
	row := s.Corpus.ByID[p[0]]
	f := Feedback{CorrectCorpusID: p[0], Corpus: row, Options: []*Record{}}
	if spec.QuizType == "word_recall" {
		for _, id := range choiceOrder(p, sid, position) {
			f.Options = append(f.Options, s.Corpus.ByID[id])
		}
	}
	if spec.QuizType == "cloze" {
		a := string([]rune(row.Example)[p[1] : p[1]+p[2]])
		f.ExpectedAnswer = &a
	}
	return f, nil
}

func (s *Store) GetQuestion(sid ID, position int) (Question, error) {
	var qid ID
	if err := s.db.QueryRow("SELECT question_id FROM session_questions WHERE session_id=? AND position=?", sid, position).Scan(&qid); err != nil {
		return Question{}, asNotFound(err, "Session question")
	}
	spec, err := s.getSpec(s.db, qid)
	if err != nil {
		return Question{}, err
	}
	p, err := s.parseSpec(spec)
	if err != nil {
		return Question{}, err
	}
	row := s.Corpus.ByID[p[0]]
	v := Question{SessionID: sid, Position: position, QuestionID: qid, QuizType: spec.QuizType, Corpus: row, Options: []Choice{}}
	if spec.Variant != "" {
		v.Variant = &spec.Variant
	}
	if err = s.db.QueryRow("SELECT COUNT(*) FROM session_questions WHERE session_id=?", sid).Scan(&v.Total); err != nil {
		return v, err
	}
	switch spec.QuizType {
	case "word_recall":
		v.Prompt = row.Word
		if spec.Variant == "jp_to_en" {
			v.Prompt = row.JP
		}
		for _, id := range choiceOrder(p, sid, position) {
			r := s.Corpus.ByID[id]
			label := r.JP
			if spec.Variant == "jp_to_en" {
				label = r.Word
			}
			v.Options = append(v.Options, Choice{id, label})
		}
	case "cloze":
		ex := []rune(row.Example)
		text := string(ex[:p[1]]) + "________" + string(ex[p[1]+p[2]:])
		v.Prompt = text
		v.ClozeText = &text
	case "reading_review":
		v.Prompt = row.Word
		if spec.Variant == "jp_and_example" {
			v.Prompt = row.JP
		}
	}
	a, err := readLatestAttempt(s.db, sid, position)
	if err != nil {
		return v, err
	}
	v.LatestAttempt = a
	if a != nil {
		f, e := s.feedback(spec, sid, position)
		if e != nil {
			return v, e
		}
		v.Feedback = &f
	}
	return v, nil
}
