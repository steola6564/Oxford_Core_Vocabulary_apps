package main

import (
	"database/sql"
	"errors"
	"strings"
)

type Attempt struct {
	AttemptID        ID      `json:"attempt_id"`
	SessionID        ID      `json:"session_id"`
	Position         int     `json:"position"`
	Outcome          string  `json:"outcome"`
	SelectedCorpusID *int    `json:"selected_corpus_id"`
	AnswerText       *string `json:"answer_text"`
	Recalled         *int    `json:"recalled"`
	Correct          *int    `json:"correct"`
	Confidence       *int    `json:"confidence"`
	ResponseTimeMS   *int64  `json:"response_time_ms"`
	OccurredAt       *int64  `json:"occurred_at"`
}

type AttemptRecord struct {
	Attempt
	QuestionID ID      `json:"question_id"`
	QuizType   string  `json:"quiz_type"`
	Variant    string  `json:"variant"`
	Corpus     *Record `json:"corpus"`
}

const attemptColumns = `a.attempt_id,a.session_id,a.position,a.outcome,a.selected_corpus_id,a.answer_text,a.recalled,a.correct,a.confidence,a.response_time_ms,
(SELECT MIN(l.occurred_at) FROM action_logs l WHERE l.attempt_id=a.attempt_id AND l.event IN ('submit','unknown'))`

func scanAttempt(row interface{ Scan(...any) error }) (Attempt, error) {
	a := Attempt{}
	err := row.Scan(&a.AttemptID, &a.SessionID, &a.Position, &a.Outcome, &a.SelectedCorpusID, &a.AnswerText, &a.Recalled, &a.Correct, &a.Confidence, &a.ResponseTimeMS, &a.OccurredAt)
	return a, err
}

func readLatestAttempt(q querier, sid ID, position int) (*Attempt, error) {
	a, err := scanAttempt(q.QueryRow("SELECT "+attemptColumns+" FROM attempt_logs a WHERE a.session_id=? AND a.position=? ORDER BY a.attempt_id DESC LIMIT 1", sid, position))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) readAttempts(q querier) ([]AttemptRecord, error) {
	rows, err := q.Query("SELECT " + attemptColumns + `,q.question_id,q.quiz_type,COALESCE(q.variant,''),q.parameters
FROM attempt_logs a JOIN session_questions sq ON sq.session_id=a.session_id AND sq.position=a.position
JOIN question_specs q ON q.question_id=sq.question_id ORDER BY a.attempt_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []AttemptRecord{}
	for rows.Next() {
		a := AttemptRecord{}
		var params string
		if err = rows.Scan(&a.AttemptID, &a.SessionID, &a.Position, &a.Outcome, &a.SelectedCorpusID, &a.AnswerText, &a.Recalled, &a.Correct, &a.Confidence, &a.ResponseTimeMS, &a.OccurredAt, &a.QuestionID, &a.QuizType, &a.Variant, &params); err != nil {
			return nil, err
		}
		p, e := s.parseSpec(QuestionSpec{a.QuestionID, a.QuizType, a.Variant, params})
		if e != nil {
			return nil, e
		}
		a.Corpus = s.Corpus.ByID[p[0]]
		out = append(out, a)
	}
	return out, rows.Err()
}

func matchesHistory(a Attempt, f Filter) bool {
	if len(f.HistoryResult) > 0 {
		match := false
		for _, code := range f.HistoryResult {
			switch code {
			case 1:
				match = match || (a.Correct != nil && *a.Correct == 1)
			case 2:
				match = match || (a.Outcome == "answered" && a.Correct != nil && *a.Correct == 0)
			case 3:
				match = match || a.Outcome == "unknown"
			case 4:
				match = match || (a.Recalled != nil && *a.Recalled == 1)
			case 5:
				match = match || (a.Recalled != nil && *a.Recalled == 0)
			}
		}
		if !match {
			return false
		}
	}
	if len(f.ConfidenceRange) == 2 && (a.Confidence == nil || *a.Confidence < f.ConfidenceRange[0] || *a.Confidence > f.ConfidenceRange[1]) {
		return false
	}
	if len(f.AttemptTimeRange) == 2 {
		if a.OccurredAt == nil {
			return false
		}
		if f.AttemptTimeRange[0] >= 0 && *a.OccurredAt < f.AttemptTimeRange[0] {
			return false
		}
		if f.AttemptTimeRange[1] >= 0 && *a.OccurredAt > f.AttemptTimeRange[1] {
			return false
		}
	}
	return true
}

type AttemptRequest struct {
	Position         int     `json:"position"`
	Outcome          string  `json:"outcome"`
	SelectedCorpusID *int    `json:"selected_corpus_id"`
	AnswerText       *string `json:"answer_text"`
	Recalled         *int    `json:"recalled"`
	Confidence       *int    `json:"confidence"`
	ResponseTimeMS   *int64  `json:"response_time_ms"`
	Action           *Action `json:"action"`
}
type AttemptResponse struct {
	Attempt  Attempt  `json:"attempt"`
	Feedback Feedback `json:"feedback"`
	Session  Session  `json:"session"`
}

func intPointer(v int) *int            { return &v }
func normalizedAnswer(v string) string { return strings.Join(strings.Fields(normalize(v)), " ") }

func (s *Store) SubmitAttempt(sid ID, r AttemptRequest) (AttemptResponse, error) {
	if r.Outcome != "answered" && r.Outcome != "unknown" {
		return AttemptResponse{}, bad("outcome must be answered or unknown")
	}
	if r.Confidence != nil && (*r.Confidence < 0 || *r.Confidence > 100) {
		return AttemptResponse{}, bad("confidence must be 0..100 or null")
	}
	if r.ResponseTimeMS != nil && *r.ResponseTimeMS < 0 {
		return AttemptResponse{}, bad("response_time_ms must be nonnegative")
	}
	if r.AnswerText != nil && len(*r.AnswerText) > 4096 {
		return AttemptResponse{}, bad("answer_text is too long")
	}
	var result AttemptResponse
	err := s.transaction(func(tx *sql.Tx) error {
		var qid ID
		if err := tx.QueryRow("SELECT question_id FROM session_questions WHERE session_id=? AND position=?", sid, r.Position).Scan(&qid); err != nil {
			return asNotFound(err, "Session question")
		}
		spec, err := s.getSpec(tx, qid)
		if err != nil {
			return err
		}
		p, err := s.parseSpec(spec)
		if err != nil {
			return err
		}
		a := Attempt{SessionID: sid, Position: r.Position, Outcome: r.Outcome, Confidence: r.Confidence, ResponseTimeMS: r.ResponseTimeMS}
		if r.Outcome == "unknown" {
			if spec.QuizType != "reading_review" {
				a.Correct = intPointer(0)
			}
		} else {
			switch spec.QuizType {
			case "word_recall":
				if r.SelectedCorpusID == nil || !contains(p, *r.SelectedCorpusID) {
					return bad("choose one of the four Corpus IDs")
				}
				if r.AnswerText != nil || r.Recalled != nil {
					return bad("Word Recall accepts selected_corpus_id only")
				}
				a.SelectedCorpusID = r.SelectedCorpusID
				a.Correct = intPointer(0)
				if *r.SelectedCorpusID == p[0] {
					a.Correct = intPointer(1)
				}
			case "cloze":
				if r.AnswerText == nil || strings.TrimSpace(*r.AnswerText) == "" {
					return bad("enter a Cloze answer or choose Unknown")
				}
				if r.SelectedCorpusID != nil || r.Recalled != nil {
					return bad("Cloze accepts answer_text only")
				}
				a.AnswerText = r.AnswerText
				expected := string([]rune(s.Corpus.ByID[p[0]].Example)[p[1] : p[1]+p[2]])
				a.Correct = intPointer(0)
				if normalizedAnswer(*r.AnswerText) == normalizedAnswer(expected) {
					a.Correct = intPointer(1)
				}
			case "reading_review":
				if r.Recalled == nil || (*r.Recalled != 0 && *r.Recalled != 1) {
					return bad("Reading Review requires recalled 0 or 1")
				}
				if r.SelectedCorpusID != nil || r.AnswerText != nil {
					return bad("Reading Review accepts recalled only")
				}
				a.Recalled = r.Recalled
			}
		}
		a.AttemptID, err = allocate(tx, "attempt_logs", "attempt_id")
		if err != nil {
			return err
		}
		_, err = tx.Exec(`INSERT INTO attempt_logs(attempt_id,session_id,position,outcome,selected_corpus_id,answer_text,recalled,correct,confidence,response_time_ms)
VALUES(?,?,?,?,?,?,?,?,?,?)`, a.AttemptID, sid, r.Position, a.Outcome, a.SelectedCorpusID, a.AnswerText, a.Recalled, a.Correct, a.Confidence, a.ResponseTimeMS)
		if err != nil {
			return err
		}
		screen := spec.QuizType
		if screen == "cloze" {
			screen = "cloze_test"
		}
		event := "submit"
		if r.Outcome == "unknown" {
			event = "unknown"
		}
		action, err := actionFor(tx, r.Action, screen, event, "question:"+qid.String(), sid, a.AttemptID)
		if err != nil {
			return err
		}
		if _, err = insertAction(tx, action); err != nil {
			return err
		}
		a.OccurredAt = &action.OccurredAt
		_, err = tx.Exec(`UPDATE sessions SET current_position=?,status=CASE WHEN
(SELECT COUNT(DISTINCT position) FROM attempt_logs WHERE session_id=?)=(SELECT COUNT(*) FROM session_questions WHERE session_id=?)
THEN 'completed' ELSE 'active' END WHERE session_id=?`, r.Position, sid, sid, sid)
		if err != nil {
			return err
		}
		result.Attempt = a
		result.Feedback, err = s.feedback(spec, sid, r.Position)
		return err
	})
	if err != nil {
		return AttemptResponse{}, err
	}
	result.Session, err = s.GetSession(sid)
	return result, err
}
