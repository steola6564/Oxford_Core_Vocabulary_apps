#!/usr/bin/env python3
"""Independent end-to-end API/SQLite verification using only Python's stdlib.

Launches the supplied compiled Go application with its own temporary database.
Never opens or modifies the user's learning.db. Browser behavior is explicitly
outside this script's evidence; see the separate browser verification report.
"""

from __future__ import annotations

import argparse
import collections
import contextlib
import csv
import hashlib
import json
import os
from pathlib import Path
import signal
import socket
import sqlite3
import subprocess
import tempfile
import time
import traceback
import unicodedata
import urllib.error
import urllib.request


POS = ["adjective", "adverb", "auxiliary verb", "conjunction", "definite article",
       "determiner", "exclam", "indefinite article", "infinitive marker", "linking verb",
       "modal verb", "noun", "number", "preposition", "pronoun", "verb"]
LEVEL = ["3000", "5000", ""]
SECTION = [*(f"S{i}" for i in range(1, 7)), *(f"W{i}" for i in range(1, 13)), ""]
TABLES = {"question_specs", "sessions", "session_questions", "attempt_logs",
          "action_logs", "collections", "collection_items"}
ACTION_COLUMNS = ["action_id", "occurred_at", "activity_id", "screen", "event",
                  "target", "value", "session_id", "attempt_id"]
EXPECTED_SHA = "9c871e809b082836339c665aa611476602301f3e6425eca0ded34185d8093f72"


def norm(text):
    return unicodedata.normalize("NFKC", text).lower()


def encode_filter(f):
    """Independent implementation of specification §9.6, not imported Go code."""
    blocks = []
    for tag, key in [(1, "free_text"), (2, "corpus_ids"), (3, "pos_codes"),
                     (4, "level_codes"), (5, "section_codes"), (6, "collection_ids"),
                     (7, "initial_letter"), (8, "history_result"),
                     (9, "confidence_range"), (11, "attempt_time_range"),
                     (12, "requested_count")]:
        value = f.get(key)
        if tag == 12:
            values = [value if value is not None else 20]
        elif value is None or value == "" or value == [] or (tag == 7 and value == 0):
            continue
        elif tag == 1:
            values = [ord(c) for c in value]
        elif tag == 7:
            values = [value]
        else:
            values = [int(v) for v in value]
        blocks.extend([tag, len(values), *values])
    block_count = 0
    i = 0
    while i < len(blocks):
        block_count += 1
        i += 2 + blocks[i + 1]
    return ",".join(str(v) for v in [1, block_count, *blocks])


def decode_filter(text):
    parts = [int(v) for v in text.split(",")]
    assert parts[0] == 1
    result = {}
    index = 2
    for _ in range(parts[1]):
        tag, count = parts[index:index + 2]
        assert tag not in result and count > 0
        index += 2
        result[tag] = parts[index:index + count]
        assert len(result[tag]) == count
        index += count
    assert index == len(parts)
    return result


class Harness:
    def __init__(self, args, tmp):
        self.args = args
        self.tmp = Path(tmp)
        self.db = self.tmp / "verification.db"
        self.log_path = self.tmp / "server.log"
        self.server = None
        self.log = None
        self.results = []
        self.context = {}
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            self.port = sock.getsockname()[1]
        self.url = f"http://127.0.0.1:{self.port}"
        self.corpus = list(csv.DictReader(args.corpus.open(encoding="utf-8-sig", newline="")))
        self.by_id = {int(r["ID"]): r for r in self.corpus}
        self.event_time = int(time.time() * 1000) - 600_000

    def start(self):
        self.log = self.log_path.open("ab")
        self.server = subprocess.Popen(
            [str(self.args.binary), "-addr", f"127.0.0.1:{self.port}",
             "-csv", str(self.args.corpus), "-db", str(self.db)],
            stdout=self.log, stderr=subprocess.STDOUT, cwd=str(self.args.binary.parent),
        )
        deadline = time.monotonic() + 40
        while time.monotonic() < deadline:
            if self.server.poll() is not None:
                raise RuntimeError("Server exited: " + self.log_path.read_text(errors="replace"))
            try:
                self.api("GET", "/api/meta")
                return
            except (OSError, urllib.error.URLError):
                time.sleep(0.1)
        raise RuntimeError("Server did not become available within 40 seconds")

    def stop(self):
        if self.server is not None and self.server.poll() is None:
            self.server.send_signal(signal.SIGTERM)
            try:
                self.server.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.server.kill()
                self.server.wait(timeout=5)
        if self.log is not None:
            self.log.close()
        self.server = None

    def api(self, method, path, data=None, status=(200, 201)):
        body = None if data is None else json.dumps(data, ensure_ascii=False).encode()
        request = urllib.request.Request(self.url + path, data=body, method=method,
                                         headers={"Content-Type": "application/json"})
        try:
            with urllib.request.urlopen(request, timeout=60) as response:
                code, raw = response.status, response.read()
        except urllib.error.HTTPError as error:
            code, raw = error.code, error.read()
        expected = (status,) if isinstance(status, int) else status
        assert code in expected, f"{method} {path}: HTTP {code}, expected {expected}: {raw[:1000]!r}"
        if not raw:
            return None
        try:
            return json.loads(raw)
        except json.JSONDecodeError:
            return raw.decode("utf-8", errors="replace")

    def query(self, sql, params=()):
        with sqlite3.connect(self.db) as con:
            con.row_factory = sqlite3.Row
            return [dict(r) for r in con.execute(sql, params)]

    def scalar(self, sql, params=()):
        with sqlite3.connect(self.db) as con:
            return con.execute(sql, params).fetchone()[0]

    def counts(self):
        return {name: self.scalar(f"SELECT COUNT(*) FROM {name}") for name in sorted(TABLES)}

    def action(self, screen="word_recall", event="submit", at=None):
        self.event_time += 1
        return {"occurred_at": at if at is not None else self.event_time,
                "activity_id": self.context["activity"], "screen": screen,
                "event": event, "target": None, "value": None, "session_id": None}

    def session(self, quiz="word_recall", variant="en_to_jp", mode="new", f=None):
        return self.api("POST", "/api/sessions", {
            "quiz_type": quiz, "variant": variant, "source_mode": mode,
            "filter": f if f is not None else {"requested_count": 1},
            "action": self.action("submode_selection", "start"),
        })

    def question(self, session, position=1):
        return self.api("GET", f"/api/sessions/{session['session_id']}/questions/{position}")

    def submit(self, session, position=1, outcome="answered", **fields):
        action = fields.pop("action", self.action(session["quiz_type"]))
        payload = {"position": position, "outcome": outcome, "confidence": None,
                   "selected_corpus_id": None, "answer_text": None, "recalled": None,
                   "response_time_ms": 1234, "action": action}
        payload.update(fields)
        return self.api("POST", f"/api/sessions/{session['session_id']}/attempts", payload)

    def candidates(self, quiz, variant, mode, f):
        return self.api("POST", "/api/candidates", {
            "quiz_type": quiz, "variant": variant, "source_mode": mode, "filter": f})

    def check(self, name, criteria, fn):
        started = time.monotonic()
        result = {"name": name, "spec_22_items": criteria, "kind": "API/SQLite execution"}
        try:
            result["details"] = fn() or {}
            result["status"] = "passed"
        except Exception as exc:
            result["status"] = "failed"
            result["error"] = f"{type(exc).__name__}: {exc}"
            result["traceback"] = traceback.format_exc()
        result["duration_ms"] = round((time.monotonic() - started) * 1000)
        self.results.append(result)
        print(f"[{result['status'].upper()}] {name}", flush=True)

    def startup(self):
        html = self.api("GET", "/")
        assert isinstance(html, str) and "<html" in html.lower() and "<script" in html.lower()
        tables = {r["name"] for r in self.query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")}
        assert tables == TABLES, tables
        columns = [r["name"] for r in self.query("PRAGMA table_info(action_logs)")]
        assert columns == ACTION_COLUMNS, columns
        forbidden = {"word", "jp", "example", "expected_answer", "spec_json", "answer_json"}
        for name in tables:
            actual = {r["name"].lower() for r in self.query(f"PRAGMA table_info({name})")}
            assert not (forbidden & actual), (name, forbidden & actual)
        self.context["activity"] = self.api("POST", "/api/activity", {})["activity_id"]
        assert len(self.context["activity"]) == 8 and self.context["activity"].isdigit()
        return {"tables": sorted(tables), "action_columns": columns, "served_html_bytes": len(html.encode())}

    def corpus_check(self):
        sha = hashlib.sha256(self.args.corpus.read_bytes()).hexdigest()
        assert sha == EXPECTED_SHA
        assert len(self.corpus) == 6100
        assert list(self.corpus[0]) == ["ID", "WORD", "POS", "LEVEL", "SECTION", "JP", "EXAMPLE"]
        assert [int(r["ID"]) for r in self.corpus] == list(range(1, 6101))
        meta = self.api("GET", "/api/meta")
        assert meta["count"] == 6100 and meta["sha256"] == sha
        assert len(meta["pos"]) == 16
        for ident in [1, 2, 3, 315, 483, 6100]:
            record = self.api("GET", f"/api/corpus/{ident}")
            for field in ["WORD", "POS", "LEVEL", "SECTION", "JP", "EXAMPLE"]:
                assert record[field.lower()] == self.by_id[ident][field], (ident, field)
        assert self.api("GET", "/api/corpus/1")["word"] == "a, an"
        self.api("GET", "/api/corpus/99999999", status=404)
        return {"records": 6100, "sha256": sha, "comma_cell": "a, an", "cloze_eligible": meta.get("cloze_eligible")}

    def expected_ids(self, f):
        out = []
        tokens = norm(f.get("free_text", "")).split()
        for row in self.corpus:
            rid = int(row["ID"])
            if f.get("corpus_ids") and rid not in f["corpus_ids"]:
                continue
            if f.get("initial_letter") and ord(row["WORD"][0].upper()) - 64 != f["initial_letter"]:
                continue
            if f.get("pos_codes") and POS.index(row["POS"]) + 1 not in f["pos_codes"]:
                continue
            if f.get("level_codes") and LEVEL.index(row["LEVEL"]) + 1 not in f["level_codes"]:
                continue
            if f.get("section_codes") and not any(SECTION.index(s) + 1 in f["section_codes"] for s in row["SECTION"].split(" | ")):
                continue
            haystack = norm(" ".join(row[k] for k in ["WORD", "POS", "LEVEL", "SECTION", "JP", "EXAMPLE"]))
            if not all(token in haystack for token in tokens):
                continue
            out.append(rid)
        return out

    def search_check(self):
        filters = [{}, {"free_text": "ＡＢＡＮＤＯＮ　VERB"}, {"free_text": "theo"},
                   {"free_text": "能力"}, {"free_text": "café noun"}, {"free_text": "Ｓ４ 5000"},
                   {"pos_codes": [12, 16]}, {"level_codes": [1, 2]}, {"level_codes": [3]},
                   {"section_codes": [4, 11]}, {"section_codes": [19]},
                   {"pos_codes": [12, 16], "level_codes": [1, 2], "initial_letter": 18},
                   {"pos_codes": [1, 2], "level_codes": [2], "section_codes": [4, 11]},
                   {"free_text": "nonexistentword_231093"},
                   *({"initial_letter": n} for n in range(1, 27))]
        for f in filters:
            expected = self.expected_ids(f)
            result = self.api("POST", "/api/search", {"filter": f, "offset": 0, "limit": 75})
            assert result["total"] == len(expected), (f, result["total"], len(expected))
            assert {r["id"] for r in result["items"]} <= set(expected), f
        expected = set(self.by_id)
        received = []
        offset = 0
        while len(received) < 6100:
            page = self.api("POST", "/api/search", {"filter": {}, "offset": offset, "limit": 500})
            assert page["items"], offset
            received.extend(r["id"] for r in page["items"])
            offset += len(page["items"])
        assert len(received) == len(set(received)) == 6100 and set(received) == expected
        return {"oracle_filter_cases": len(filters), "full_pagination_ids": len(received), "initial_X": 0}

    def filter_encoding(self):
        f = {"pos_codes": [12, 16], "level_codes": [1, 2], "initial_letter": 18, "requested_count": 50}
        sess = self.session("reading_review", "word_and_example", f=f)
        expected = "1,4,3,2,12,16,4,2,1,2,7,1,18,12,1,50"
        assert sess["filter_spec"] == expected == encode_filter(f), sess["filter_spec"]
        assert sess["total"] == 50
        assert sess["filter"]["initial_letter"] == 18
        for q in sess["questions"]:
            params = self.scalar("SELECT parameters FROM question_specs WHERE question_id=?", (q["question_id"],))
            assert self.by_id[int(params)]["WORD"].lower().startswith("r")
        f2 = {"free_text": "  ＣＡＴ  ", "pos_codes": [12], "requested_count": 2}
        s2 = self.session("reading_review", "word_and_example", f=f2)
        assert s2["filter_spec"] == encode_filter(f2)
        assert "".join(chr(v) for v in decode_filter(s2["filter_spec"])[1]) == f2["free_text"]
        assert s2["filter"]["free_text"] == f2["free_text"]
        s3 = self.session("reading_review", "word_and_example", f={"corpus_ids": [3], "requested_count": -1})
        assert s3["filter_spec"] == "1,2,2,1,3,12,1,-1"
        assert 7 not in decode_filter(s3["filter_spec"])
        self.context["encoding_session"] = sess
        return {"spec_example_A": expected, "raw_unicode_and_spaces_preserved": True, "all_tag_7_omitted": True}

    def word_recall(self):
        sess = self.session(f={"corpus_ids": [302], "requested_count": 1})
        q = self.question(sess)
        ids = [o["id"] for o in q["options"]]
        assert len(ids) == len(set(ids)) == 4 and 302 in ids
        assert len({norm(o["label"]) for o in q["options"]}) == 4
        params = self.scalar("SELECT parameters FROM question_specs WHERE question_id=?", (q["question_id"],))
        assert [int(v) for v in params.split(",")][0] == 302
        assert set(map(int, params.split(","))) == set(ids)
        wrong = next(v for v in ids if v != 302)
        bad = self.submit(sess, selected_corpus_id=wrong, confidence=0)
        assert bad["attempt"]["selected_corpus_id"] == wrong
        assert bad["attempt"]["correct"] == 0 and bad["attempt"]["confidence"] == 0
        assert len(bad["feedback"]["options"]) == 4
        for option in bad["feedback"]["options"]:
            assert option["word"] and option["pos"] and option["jp"] and option["example"]
        good = self.submit(sess, selected_corpus_id=302, confidence=None)
        assert good["attempt"]["correct"] == 1 and good["attempt"]["confidence"] is None
        assert good["attempt"]["attempt_id"] != bad["attempt"]["attempt_id"]
        assert good["session"]["status"] == "completed"
        self.context.update(recall_session=sess, recall_question=q, recall_wrong=bad["attempt"], recall_good=good["attempt"])
        back = self.session(variant="jp_to_en", f={"corpus_ids": [3], "requested_count": 1})
        bq = self.question(back)
        assert bq["prompt"] == self.by_id[3]["JP"]
        assert len({norm(o["label"]) for o in bq["options"]}) == 4
        ba = self.submit(back, selected_corpus_id=3, confidence=100)
        assert ba["attempt"]["correct"] == 1 and ba["attempt"]["selected_corpus_id"] == 3
        unknown_session = self.session(f={"corpus_ids": [20], "requested_count": 1})
        unknown = self.submit(unknown_session, outcome="unknown")
        assert unknown["attempt"]["correct"] == 0
        assert unknown["attempt"]["selected_corpus_id"] is None and unknown["attempt"]["confidence"] is None
        assert unknown["attempt"]["outcome"] == "unknown"
        assert unknown["session"]["status"] == "completed"
        self.context["unknown_session"] = unknown_session
        return {"directions": ["en_to_jp", "jp_to_en"], "selected_corpus_id_retained": wrong,
                "repeated_attempts_preserved": True, "confidence_values": [0, None, 100]}

    def cloze(self):
        sess = self.session("cloze", None, f={"corpus_ids": [2], "requested_count": 1})
        q = self.question(sess)
        params = self.scalar("SELECT parameters FROM question_specs WHERE question_id=?", (q["question_id"],))
        assert params == "2,10,7", params
        assert "abandon" not in q["prompt"].lower()
        answer = "  ABANDON  "
        result = self.submit(sess, answer_text=answer, confidence=81)
        assert result["attempt"]["correct"] == 1
        assert result["attempt"]["answer_text"] == answer
        assert result["feedback"]["expected_answer"] == "abandon"
        unknown = self.submit(sess, outcome="unknown")
        assert unknown["attempt"]["correct"] == 0 and unknown["attempt"]["outcome"] == "unknown"
        all_candidates = self.candidates("cloze", None, "new", {"requested_count": -1})
        all_session = self.session("cloze", None, f={"requested_count": -1})
        assert all_session["total"] == all_candidates["total"]
        rows = self.query("SELECT q.parameters FROM session_questions s JOIN question_specs q ON q.question_id=s.question_id WHERE s.session_id=?", (all_session["session_id"],))
        corpus_ids = []
        for row in rows:
            cid, start, length = map(int, row["parameters"].split(","))
            example = self.by_id[cid]["EXAMPLE"]
            assert 0 <= start < len(example) and length > 0 and start + length <= len(example)
            text = example[start:start + length]
            assert text.strip(), row
            assert not (start > 0 and example[start - 1].isalnum() and text[0].isalnum()), (cid, text, "left boundary")
            end = start + length
            assert not (end < len(example) and example[end].isalnum() and text[-1].isalnum()), (cid, text, "right boundary")
            corpus_ids.append(cid)
        assert len(corpus_ids) == len(set(corpus_ids))
        self.context["cloze_session"] = sess
        return {"example_parameters": params, "input_preserved": answer, "all_generated_spans_checked": len(rows),
                "cloze_skipped": all_candidates.get("cloze_skipped"), "rune_bounds_and_token_boundaries": True}

    def reading(self):
        summaries = []
        for variant, recalled, confidence in [("word_and_example", 1, 0), ("jp_and_example", 0, None)]:
            sess = self.session("reading_review", variant, f={"corpus_ids": [3], "requested_count": 1})
            q = self.question(sess)
            assert q["corpus"]["example"] == self.by_id[3]["EXAMPLE"]
            assert q["prompt"] == self.by_id[3]["WORD" if variant == "word_and_example" else "JP"]
            result = self.submit(sess, recalled=recalled, confidence=confidence)
            assert result["attempt"]["correct"] is None
            assert result["attempt"]["recalled"] == recalled
            assert result["attempt"]["confidence"] == confidence
            summaries.append({"variant": variant, "recalled": recalled, "correct": None, "confidence": confidence})
            self.context[f"reading_{recalled}"] = sess
        unknown = self.submit(self.context["reading_0"], outcome="unknown")
        assert unknown["attempt"]["correct"] is None and unknown["attempt"]["recalled"] is None
        assert unknown["attempt"]["outcome"] == "unknown"
        return {"variants": summaries, "explicit_unknown_distinct": True,
                "scope": "API storage only; touch/keyboard input requires separate browser evidence"}

    def completion_and_resume(self):
        sess = self.session("reading_review", "word_and_example", f={"corpus_ids": [4, 5, 6], "requested_count": -1})
        sid = sess["session_id"]
        before = self.counts()
        moved = self.api("PATCH", f"/api/sessions/{sid}", {"current_position": 3, "status": "paused"})
        assert moved["answered"] == 0 and moved["status"] == "paused"
        restored = self.api("GET", f"/api/sessions/{sid}")
        assert restored["session_id"] == sid and restored["questions"] == sess["questions"]
        assert restored["current_position"] == 3
        after = self.counts()
        for key in ["question_specs", "sessions", "attempt_logs"]:
            assert before[key] == after[key]
        r3 = self.submit(sess, 3, outcome="unknown")
        assert r3["session"]["answered"] == 1 and r3["session"]["status"] != "completed"
        r1 = self.submit(sess, 1, recalled=1)
        assert r1["session"]["answered"] == 2 and r1["session"]["status"] != "completed"
        r2 = self.submit(sess, 2, outcome="unknown")
        assert r2["session"]["answered"] == 3 and r2["session"]["status"] == "completed"
        self.context["persist_session"] = sess
        return {"positions": 3, "movement_attempt_delta": 0, "resume_spec_delta": 0,
                "completed_after_positions": [3, 1, 2], "unknown_counts_as_answer": True}

    def review(self):
        original = self.context["recall_question"]["question_id"]
        before = self.counts()
        review = self.session(mode="review", f={"corpus_ids": [302], "history_result": [2], "requested_count": -1})
        assert original in [q["question_id"] for q in review["questions"]]
        after = self.counts()
        assert after["question_specs"] == before["question_specs"]
        assert after["attempt_logs"] == before["attempt_logs"]
        assert review["session_id"] != self.context["recall_session"]["session_id"]
        u = self.candidates("word_recall", "en_to_jp", "review", {"corpus_ids": [20], "history_result": [3], "requested_count": -1})
        inc = self.candidates("word_recall", "en_to_jp", "review", {"corpus_ids": [20], "history_result": [2], "requested_count": -1})
        assert u["total"] == 1 and inc["total"] == 0
        read_no = self.candidates("reading_review", "jp_and_example", "review", {"corpus_ids": [3], "history_result": [5], "requested_count": -1})
        read_incorrect = self.candidates("reading_review", "jp_and_example", "review", {"corpus_ids": [3], "history_result": [2], "requested_count": -1})
        assert read_no["total"] >= 1 and read_incorrect["total"] == 0
        return {"review_reuses_question_id": original, "question_spec_delta": 0,
                "earlier_wrong_survives_later_correct": True, "unknown_and_reading_not_incorrect": True}

    def same_attempt_history(self):
        sess = self.session(f={"corpus_ids": [2], "requested_count": 1})
        q = self.question(sess)
        wrong = next(o["id"] for o in q["options"] if o["id"] != 2)
        t1, t2 = self.event_time + 100, self.event_time + 200
        self.event_time = t2 + 1
        self.submit(sess, selected_corpus_id=wrong, confidence=20, action=self.action(at=t1))
        self.submit(sess, selected_corpus_id=2, confidence=80, action=self.action(at=t2))
        base = {"corpus_ids": [2], "requested_count": -1}
        cases = [({"history_result": [2], "confidence_range": [70, 100]}, 0),
                 ({"history_result": [1], "confidence_range": [0, 30]}, 0),
                 ({"history_result": [2], "confidence_range": [0, 30], "attempt_time_range": [t1, t1]}, 1),
                 ({"history_result": [2], "attempt_time_range": [t2, t2]}, 0),
                 ({"history_result": [1], "confidence_range": [70, 100], "attempt_time_range": [t2, t2]}, 1)]
        for extra, count in cases:
            result = self.candidates("word_recall", "en_to_jp", "review", {**base, **extra})
            assert result["total"] == count, (extra, result, count)
        self.context.update(history_session=sess, history_time=t1, history_question=q)
        return {"same_attempt_cases": len(cases), "wrong_at": t1, "correct_at": t2, "inclusive_time_endpoints": True}

    def all_filter_tags(self):
        col = self.api("POST", "/api/collections", {"name": "Filter round trip", "kind": "vocabulary"})
        self.api("POST", f"/api/collections/{col['collection_id']}/items", {"corpus_id": 2})
        row = self.by_id[2]
        f = {"free_text": "  Ａｂａｎｄｏｎ  vErB  ", "corpus_ids": [2], "pos_codes": [16],
             "level_codes": [LEVEL.index(row["LEVEL"]) + 1],
             "section_codes": [SECTION.index(row["SECTION"].split(" | ")[0]) + 1],
             "collection_ids": [col["collection_id"]], "initial_letter": 1,
             "history_result": [2], "confidence_range": [20, 20],
             "attempt_time_range": [self.context["history_time"], self.context["history_time"]], "requested_count": -1}
        sess = self.session(mode="review", f=f)
        assert sess["filter_spec"] == encode_filter(f), (sess["filter_spec"], encode_filter(f))
        blocks = decode_filter(sess["filter_spec"])
        assert set(blocks) == {1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12}
        assert sess["total"] == 1
        assert sess["questions"][0]["question_id"] == self.context["history_question"]["question_id"]
        assert sess["filter"]["free_text"] == f["free_text"]
        assert sess["filter"]["initial_letter"] == 1
        x = self.candidates("word_recall", "en_to_jp", "review", {**f, "initial_letter": 24})
        assert x["total"] == 0
        return {"implemented_tags_round_tripped": sorted(blocks), "tag10_reserved_not_used": True,
                "new_and_review_initial_filter": True, "filter_spec": sess["filter_spec"]}

    def collections(self):
        col = self.api("POST", "/api/collections", {"name": "Duplicate vocabulary", "kind": "vocabulary"})
        path = f"/api/collections/{col['collection_id']}/items"
        first = self.api("POST", path, {"corpus_id": 3})
        duplicate = self.api("POST", path, {"corpus_id": 3}, status=409)
        assert duplicate["code"] == "duplicate"
        second = self.api("POST", path, {"corpus_id": 3, "allow_duplicate": True})
        assert first["item_id"] != second["item_id"]
        viewed = self.api("GET", f"/api/collections/{col['collection_id']}")
        assert len(viewed["items"]) == 2
        filtered = self.api("POST", "/api/search", {"filter": {"collection_ids": [col["collection_id"]]}, "limit": 75})
        assert filtered["total"] == 1 and filtered["items"][0]["id"] == 3
        sess = self.session("reading_review", "word_and_example", f={"collection_ids": [col["collection_id"]], "requested_count": -1})
        assert sess["total"] == 2
        for question in sess["questions"]:
            assert self.scalar("SELECT parameters FROM question_specs WHERE question_id=?", (question["question_id"],)) == "3"
        self.api("DELETE", path + "/" + first["item_id"])
        remaining = self.api("GET", f"/api/collections/{col['collection_id']}")
        assert len(remaining["items"]) == 1 and remaining["items"][0]["item_id"] == second["item_id"]
        qcol = self.api("POST", "/api/collections", {"name": "Duplicate questions", "kind": "question"})
        qpath = f"/api/collections/{qcol['collection_id']}/items"
        qid = self.context["recall_question"]["question_id"]
        self.api("POST", qpath, {"question_id": qid})
        self.api("POST", qpath, {"question_id": qid}, status=409)
        self.api("POST", qpath, {"question_id": qid, "allow_duplicate": True})
        before = self.counts()
        qsess = self.session(f={"collection_ids": [qcol["collection_id"]], "requested_count": -1})
        assert qsess["total"] == 2 and [q["question_id"] for q in qsess["questions"]] == [qid, qid]
        assert self.counts()["question_specs"] == before["question_specs"]
        return {"duplicate_warning_http": 409, "vocabulary_repetition_count": 2,
                "question_repetition_count": 2, "single_registration_deletion": True,
                "question_collection_spec_delta": 0}

    def actions_and_data(self):
        activity = self.api("POST", "/api/activity", {})["activity_id"]
        base = int(time.time() * 1000) - 3_600_000
        events = []
        for offset, screen, event in [(0, "dictionary", "enter"), (500, "dictionary", "search"),
                                     (1000, "dictionary", "reveal"), (2000, "dictionary", "leave"),
                                     (4000, "data", "enter")]:
            events.append({"occurred_at": base + offset, "activity_id": activity, "screen": screen,
                           "event": event, "target": "corpus:00000003" if event == "reveal" else None,
                           "value": "ability" if event == "search" else None, "session_id": None})
        sent_at = int(time.time() * 1000)
        result = self.api("POST", "/api/actions", {"events": events, "sent_at": sent_at})
        assert result["saved"] == 5 and result["received_at"] >= sent_at
        found = self.query("SELECT occurred_at,event FROM action_logs WHERE activity_id=? AND event IN ('enter','leave','search','reveal') ORDER BY occurred_at", (activity,))
        assert [r["occurred_at"] for r in found] == [e["occurred_at"] for e in events]
        before = self.scalar("SELECT COUNT(*) FROM action_logs")
        self.api("POST", "/api/actions", {"events": events, "sent_at": sent_at + 1})
        assert self.scalar("SELECT COUNT(*) FROM action_logs") == before
        missing = self.scalar("SELECT COUNT(*) FROM attempt_logs a LEFT JOIN action_logs l ON l.attempt_id=a.attempt_id WHERE l.action_id IS NULL")
        assert missing == 0
        double = self.query("SELECT attempt_id,COUNT(*) AS n FROM action_logs WHERE attempt_id IS NOT NULL GROUP BY attempt_id HAVING COUNT(*)<>1")
        assert double == []
        data = self.api("POST", "/api/data", {"filter": {}, "limit": 500, "action_limit": 500})
        stats = data["stats"]
        objective = self.scalar("SELECT COUNT(*) FROM attempt_logs WHERE correct IS NOT NULL")
        correct = self.scalar("SELECT COUNT(*) FROM attempt_logs WHERE correct=1")
        unknown = self.scalar("SELECT COUNT(*) FROM attempt_logs WHERE outcome='unknown'")
        assert stats["objective_count"] == objective and stats["correct"] == correct and stats["unknown"] == unknown
        assert abs(stats["accuracy"] - correct / objective * 100) < 0.00001
        assert stats["attempts"] == self.scalar("SELECT COUNT(*) FROM attempt_logs")
        assert all(a["occurred_at"] is not None for a in data["attempts"])
        usage = data["usage"]
        assert usage["approximate"] is True
        assert usage["unknown_intervals"] >= 1
        dictionary = next(s for s in usage["by_screen"] if s["screen"] == "dictionary")
        assert dictionary["duration_ms"] == 2000, usage
        filtered = self.api("POST", "/api/data", {"filter": {"corpus_ids": [302], "history_result": [2]}, "limit": 50})
        assert filtered["attempts"] and all(a["corpus"]["id"] == 302 and a["correct"] == 0 and a["outcome"] == "answered" for a in filtered["attempts"])
        assert all(len(a["attempt_id"]) == 8 for a in data["attempts"])
        return {"batch_events": len(events), "occurred_at_distinct_from_sent_at": True,
                "retry_no_duplicate_rows": True, "missing_answer_action_links": missing,
                "objective_count": objective, "unknown_count": unknown, "dictionary_usage_ms": 2000,
                "unclosed_interval_marked_unknown": True}

    def invalid_requests(self):
        sess = self.context["recall_session"]
        before = self.counts()
        for payload in [
            {"position": 1, "outcome": "answered", "selected_corpus_id": 302, "confidence": -1},
            {"position": 1, "outcome": "answered", "selected_corpus_id": 302, "confidence": 101},
            {"position": 1, "outcome": "answered", "selected_corpus_id": 99999999},
            {"position": 999, "outcome": "unknown"},
        ]:
            self.api("POST", f"/api/sessions/{sess['session_id']}/attempts", {**payload, "action": self.action()}, status=(400, 404))
        self.api("POST", "/api/sessions", {"quiz_type": "sentence_generation", "source_mode": "new", "filter": {"requested_count": 1}}, status=400)
        self.api("POST", "/api/sessions", {"quiz_type": "reading_review", "variant": "word_and_example", "source_mode": "new", "filter": {"initial_letter": 24, "requested_count": 1}}, status=400)
        after = self.counts()
        for key in ["question_specs", "sessions", "attempt_logs"]:
            assert before[key] == after[key], (key, before[key], after[key])
        return {"invalid_confidence_or_selection_or_position_rejected": True,
                "sentence_generation_no_session": True, "empty_initial_X_no_session": True}

    def restart(self):
        sess = self.context["persist_session"]
        sid = sess["session_id"]
        before = self.api("GET", f"/api/sessions/{sid}")
        counts_before = self.counts()
        self.stop()
        self.start()
        after = self.api("GET", f"/api/sessions/{sid}")
        assert before == after, (before, after)
        assert counts_before == self.counts()
        history = self.api("POST", "/api/data", {"filter": {}, "limit": 500, "action_limit": 500})
        assert history["total"] == counts_before["attempt_logs"]
        return {"same_session_id": sid, "same_question_order_and_attempt_counts": True,
                "all_table_row_counts_retained": counts_before}

    def run(self):
        for name, criteria, fn in [
            ("Startup / seven tables / nine Action columns / embedded HTML", [1], self.startup),
            ("Authoritative CSV / stable IDs / comma-containing cell", [2], self.corpus_check),
            ("Independent NFKC search oracle and full pagination", [5, 6], self.search_check),
            ("Filter encoding examples and raw Unicode round trip", [6, 7], self.filter_encoding),
            ("Recall both directions / correctness / NULL and zero / Unknown", [8, 11, 12], self.word_recall),
            ("Cloze exact span, original input, all generated bounds", [9], self.cloze),
            ("Reading two variants and subjective-only outcomes", [10, 11], self.reading),
            ("Skipped movement, completion and nonmutating Resume", [12, 14], self.completion_and_resume),
            ("Review historical incorrect and outcome distinctions", [13], self.review),
            ("History conditions intersect on the same Attempt", [13, 17], self.same_attempt_history),
            ("All implemented filter tags and Review initial-letter constraint", [6, 7, 13], self.all_filter_tags),
            ("Collections warning, multiplicity, individual deletion, spec reuse", [15], self.collections),
            ("Action batching, timestamps, links, Data and bounded usage evidence", [16, 17], self.actions_and_data),
            ("Invalid submissions and deferred generation do not create rows", [11, 12, 18], self.invalid_requests),
            ("Real process restart preserves DB, Session and history", [14, 17], self.restart),
        ]:
            self.check(name, criteria, fn)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--corpus", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    args.binary = args.binary.resolve()
    args.corpus = args.corpus.resolve()
    args.output = args.output.resolve()
    assert args.binary.is_file(), args.binary
    assert args.corpus.is_file(), args.corpus
    started = time.time()
    report = {
        "scope": "Executed Go HTTP API and SQLite; browser scenarios require separate verification",
        "binary": str(args.binary), "corpus": str(args.corpus),
        "started_at_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime(started)),
        "browser_not_tested_here": [3, 4, "6 controls", "10 swipes/cancel/PC buttons", "11 before-reveal UI", "16 frontend buffering", "17 rendered UI", "18 Coming soon UI"],
    }
    with tempfile.TemporaryDirectory(prefix="oxford-core-api-verify-") as tmp:
        h = Harness(args, tmp)
        try:
            h.start()
            h.run()
        except Exception as exc:
            report["fatal_error"] = f"{type(exc).__name__}: {exc}"
            report["fatal_traceback"] = traceback.format_exc()
        finally:
            h.stop()
        report["checks"] = h.results
        report["server_log"] = h.log_path.read_text(errors="replace") if h.log_path.exists() else ""
    report["passed"] = sum(r["status"] == "passed" for r in report["checks"])
    report["failed"] = sum(r["status"] == "failed" for r in report["checks"])
    report["duration_seconds"] = round(time.time() - started, 3)
    report["status"] = "passed" if report["failed"] == 0 and not report.get("fatal_error") else "failed"
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({k: report[k] for k in ["status", "passed", "failed", "duration_seconds"]}), flush=True)
    return 0 if report["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
