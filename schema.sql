-- Seven application tables. Corpus bodies remain exclusively in the external CSV.
PRAGMA foreign_keys = ON;
CREATE TABLE IF NOT EXISTS question_specs (
  question_id INTEGER PRIMARY KEY CHECK(question_id BETWEEN 1 AND 99999999),
  quiz_type TEXT NOT NULL CHECK(quiz_type IN ('word_recall','cloze','reading_review')),
  variant TEXT,
  parameters TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  session_id INTEGER PRIMARY KEY CHECK(session_id BETWEEN 1 AND 99999999),
  quiz_type TEXT NOT NULL CHECK(quiz_type IN ('word_recall','cloze','reading_review')),
  variant TEXT,
  source_mode TEXT NOT NULL CHECK(source_mode IN ('new','review')),
  filter_spec TEXT NOT NULL,
  current_position INTEGER NOT NULL CHECK(current_position >= 1),
  status TEXT NOT NULL CHECK(status IN ('active','paused','completed'))
);
CREATE TABLE IF NOT EXISTS session_questions (
  session_id INTEGER NOT NULL REFERENCES sessions(session_id),
  position INTEGER NOT NULL CHECK(position >= 1),
  question_id INTEGER NOT NULL REFERENCES question_specs(question_id),
  PRIMARY KEY(session_id,position)
);
CREATE TABLE IF NOT EXISTS attempt_logs (
  attempt_id INTEGER PRIMARY KEY CHECK(attempt_id BETWEEN 1 AND 99999999),
  session_id INTEGER NOT NULL,
  position INTEGER NOT NULL,
  outcome TEXT NOT NULL CHECK(outcome IN ('answered','unknown')),
  selected_corpus_id INTEGER CHECK(selected_corpus_id BETWEEN 1 AND 6100),
  answer_text TEXT,
  recalled INTEGER CHECK(recalled IN (0,1)),
  correct INTEGER CHECK(correct IN (0,1)),
  confidence INTEGER CHECK(confidence BETWEEN 0 AND 100),
  response_time_ms INTEGER CHECK(response_time_ms >= 0),
  FOREIGN KEY(session_id,position) REFERENCES session_questions(session_id,position)
);
CREATE TABLE IF NOT EXISTS action_logs (
  action_id INTEGER PRIMARY KEY CHECK(action_id BETWEEN 1 AND 99999999),
  occurred_at INTEGER NOT NULL CHECK(occurred_at >= 0),
  activity_id INTEGER NOT NULL CHECK(activity_id BETWEEN 1 AND 99999999),
  screen TEXT NOT NULL,
  event TEXT NOT NULL,
  target TEXT,
  value TEXT,
  session_id INTEGER REFERENCES sessions(session_id),
  attempt_id INTEGER REFERENCES attempt_logs(attempt_id)
);
CREATE TABLE IF NOT EXISTS collections (
  collection_id INTEGER PRIMARY KEY CHECK(collection_id BETWEEN 1 AND 99999999),
  name TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('vocabulary','question'))
);
CREATE TABLE IF NOT EXISTS collection_items (
  item_id INTEGER PRIMARY KEY CHECK(item_id BETWEEN 1 AND 99999999),
  collection_id INTEGER NOT NULL REFERENCES collections(collection_id) ON DELETE CASCADE,
  corpus_id INTEGER CHECK(corpus_id BETWEEN 1 AND 6100),
  question_id INTEGER REFERENCES question_specs(question_id),
  CHECK((corpus_id IS NOT NULL AND question_id IS NULL) OR
        (corpus_id IS NULL AND question_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS attempts_by_session ON attempt_logs(session_id,position);
CREATE INDEX IF NOT EXISTS actions_by_attempt ON action_logs(attempt_id);
CREATE INDEX IF NOT EXISTS actions_by_activity ON action_logs(activity_id,occurred_at);
CREATE UNIQUE INDEX IF NOT EXISTS actions_retry_dedup ON action_logs(
  activity_id,occurred_at,screen,event,COALESCE(target,''),COALESCE(value,''),
  COALESCE(session_id,0),COALESCE(attempt_id,0)
);
