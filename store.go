package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

type Store struct {
	db     *sql.DB
	Corpus *Corpus
	mu     sync.Mutex
}
type querier interface {
	Exec(string, ...any) (sql.Result, error)
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
}
type AppError struct {
	Status        int
	Code, Message string
}

func (e *AppError) Error() string   { return e.Message }
func bad(message string) error      { return &AppError{400, "invalid_request", message} }
func notFound(message string) error { return &AppError{404, "not_found", message} }

func OpenStore(path string, c *Corpus) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0700); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: abs}
	q := u.Query()
	q.Set("_foreign_keys", "on")
	q.Set("_busy_timeout", "5000")
	q.Set("_journal_mode", "WAL")
	u.RawQuery = q.Encode()
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	schema, err := schemaFS.ReadFile("schema.sql")
	if err == nil {
		_, err = db.Exec(string(schema))
	}
	if err != nil {
		db.Close()
		return nil, err
	}
	os.Chmod(abs, 0600)
	s := &Store{db: db, Corpus: c}
	if err = s.validateStoredReferences(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }
func (s *Store) transaction(fn func(*sql.Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func allocate(q querier, table, key string) (ID, error) {
	var id int64
	query := "SELECT COALESCE(MAX(" + key + "),0)+1 FROM " + table
	// Deleted collection registrations retain their identity in semantic Actions.
	if table == "collections" || table == "collection_items" {
		prefix := "collection:"
		if table == "collection_items" {
			prefix = "collection_item:"
		}
		query = fmt.Sprintf("SELECT COALESCE(MAX(v),0)+1 FROM (SELECT %s AS v FROM %s UNION ALL SELECT CAST(SUBSTR(target,%d) AS INTEGER) FROM action_logs WHERE target LIKE '%s%%')", key, table, len(prefix)+1, prefix)
	}
	if err := q.QueryRow(query).Scan(&id); err != nil {
		return 0, err
	}
	if id < 1 || id > 99999999 {
		return 0, fmt.Errorf("eight-digit %s ID space exhausted", table)
	}
	return ID(id), nil
}

func nullID(id ID) any {
	if id == 0 {
		return nil
	}
	return int64(id)
}
func ptrString(s string) *string { return &s }
func nullableVariant(v string) any {
	if v == "" {
		return nil
	}
	return v
}

type Action struct {
	ActionID   ID      `json:"action_id,omitempty"`
	OccurredAt int64   `json:"occurred_at"`
	ActivityID ID      `json:"activity_id"`
	Screen     string  `json:"screen"`
	Event      string  `json:"event"`
	Target     *string `json:"target"`
	Value      *string `json:"value"`
	SessionID  ID      `json:"session_id"`
	AttemptID  ID      `json:"attempt_id"`
}

var screens = map[string]bool{"hero": true, "view_selection": true, "quiz_selection": true, "submode_selection": true, "dictionary": true, "data": true, "word_recall": true, "cloze_test": true, "cloze": true, "reading_review": true, "sentence_generation": true}

func validateAction(a Action) error {
	if a.ActivityID < 1 || a.ActivityID > 99999999 {
		return bad("activity_id must be an allocated eight-digit ID")
	}
	if a.OccurredAt < 0 || a.OccurredAt > 253402300799999 {
		return bad("occurred_at must be Unix milliseconds")
	}
	if !screens[a.Screen] {
		return bad("invalid screen")
	}
	if a.Event == "" || len(a.Event) > 64 {
		return bad("event must contain 1..64 characters")
	}
	if a.Target != nil && len(*a.Target) > 512 {
		return bad("action target is too long")
	}
	if a.Value != nil && len(*a.Value) > 8192 {
		return bad("action value is too long")
	}
	return nil
}

func insertAction(q querier, a Action) (bool, error) {
	if err := validateAction(a); err != nil {
		return false, err
	}
	if a.AttemptID != 0 {
		var sid ID
		if err := q.QueryRow("SELECT session_id FROM attempt_logs WHERE attempt_id=?", a.AttemptID).Scan(&sid); err != nil {
			return false, bad("action refers to an unknown attempt")
		}
		if a.SessionID == 0 {
			a.SessionID = sid
		}
		if a.SessionID != sid {
			return false, bad("action/attempt session mismatch")
		}
	}
	id, err := allocate(q, "action_logs", "action_id")
	if err != nil {
		return false, err
	}
	res, err := q.Exec("INSERT INTO action_logs(action_id,occurred_at,activity_id,screen,event,target,value,session_id,attempt_id) VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT DO NOTHING", id, a.OccurredAt, a.ActivityID, a.Screen, a.Event, a.Target, a.Value, nullID(a.SessionID), nullID(a.AttemptID))
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

func (s *Store) NewActivity() (ID, error) {
	var id ID
	err := s.transaction(func(tx *sql.Tx) error {
		if err := tx.QueryRow("SELECT COALESCE(MAX(activity_id),0)+1 FROM action_logs").Scan(&id); err != nil {
			return err
		}
		_, err := insertAction(tx, Action{OccurredAt: time.Now().UnixMilli(), ActivityID: id, Screen: "hero", Event: "activity_start"})
		return err
	})
	return id, err
}

func actionFor(q querier, provided *Action, screen, event, target string, sessionID, attemptID ID) (Action, error) {
	a := Action{}
	if provided != nil {
		a = *provided
	} else {
		a.OccurredAt = time.Now().UnixMilli()
		if err := q.QueryRow("SELECT COALESCE(MAX(activity_id),0)+1 FROM action_logs").Scan(&a.ActivityID); err != nil {
			return a, err
		}
	}
	a.Screen = screen
	a.Event = event
	a.Target = ptrString(target)
	a.SessionID = sessionID
	a.AttemptID = attemptID
	// Answers belong in attempt_logs; never duplicate their input in Action.value.
	if attemptID != 0 {
		a.Value = nil
	}
	return a, validateAction(a)
}

func (s *Store) SaveActions(events []Action) (int, error) {
	if len(events) > 500 {
		return 0, bad("maximum 500 actions per batch")
	}
	for _, a := range events {
		if err := validateAction(a); err != nil {
			return 0, err
		}
	}
	saved := 0
	err := s.transaction(func(tx *sql.Tx) error {
		for _, a := range events {
			ok, err := insertAction(tx, a)
			if err != nil {
				return err
			}
			if ok {
				saved++
			}
		}
		return nil
	})
	return saved, err
}

func scanAction(rows interface{ Scan(...any) error }) (Action, error) {
	a := Action{}
	var sid, aid sql.NullInt64
	err := rows.Scan(&a.ActionID, &a.OccurredAt, &a.ActivityID, &a.Screen, &a.Event, &a.Target, &a.Value, &sid, &aid)
	if sid.Valid {
		a.SessionID = ID(sid.Int64)
	}
	if aid.Valid {
		a.AttemptID = ID(aid.Int64)
	}
	return a, err
}

func (s *Store) validateStoredReferences() error {
	rows, err := s.db.Query("SELECT question_id,quiz_type,COALESCE(variant,''),parameters FROM question_specs")
	if err != nil {
		return err
	}
	for rows.Next() {
		var q QuestionSpec
		if err = rows.Scan(&q.QuestionID, &q.QuizType, &q.Variant, &q.Parameters); err != nil {
			rows.Close()
			return err
		}
		if _, err = s.parseSpec(q); err != nil {
			rows.Close()
			return fmt.Errorf("stored question %s invalid for Corpus: %w", q.QuestionID, err)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = s.db.Query("SELECT filter_spec FROM sessions")
	if err != nil {
		return err
	}
	for rows.Next() {
		var f string
		if err = rows.Scan(&f); err != nil {
			rows.Close()
			return err
		}
		if _, err = DecodeFilter(f); err != nil {
			rows.Close()
			return fmt.Errorf("stored filter_spec invalid: %w", err)
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	rows, err = s.db.Query("SELECT corpus_id FROM collection_items WHERE corpus_id IS NOT NULL")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			return err
		}
		if s.Corpus.ByID[id] == nil {
			return fmt.Errorf("collection refers to missing Corpus ID %d", id)
		}
	}
	return rows.Err()
}

func asNotFound(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return notFound(what + " not found")
	}
	return err
}
func validateFilterAPI(f *Filter) error {
	f.defaults()
	if err := f.Validate(); err != nil {
		return bad(err.Error())
	}
	if len(f.ScoreRange) > 0 {
		return bad("Score filters are reserved and unavailable in v1.0")
	}
	return nil
}
func validQuiz(quiz, variant string) error {
	switch quiz {
	case "word_recall":
		if variant != "en_to_jp" && variant != "jp_to_en" {
			return bad("Word Recall requires en_to_jp or jp_to_en")
		}
	case "reading_review":
		if variant != "word_and_example" && variant != "jp_and_example" {
			return bad("Reading Review requires word_and_example or jp_and_example")
		}
	case "cloze":
		if variant != "" {
			return bad("Cloze variant must be null")
		}
	case "sentence_generation":
		return bad("Sentence Generation is Coming soon; no Session is created")
	default:
		return bad("unsupported quiz_type")
	}
	return nil
}

func placeholders(n int) string {
	if n == 0 {
		return "NULL"
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
