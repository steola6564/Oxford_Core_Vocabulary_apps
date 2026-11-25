package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testCorpus(t *testing.T) *Corpus {
	t.Helper()
	c, err := LoadCorpus("data/all_with_ID.csv", 6100)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := OpenStore(filepath.Join(t.TempDir(), "learning.db"), testCorpus(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestFilterSpecificationExamples(t *testing.T) {
	cases := []struct {
		f    Filter
		want string
	}{
		{Filter{POSCodes: []int{12, 16}, LevelCodes: []int{1, 2}, InitialLetter: 18, RequestedCount: 50}, "1,4,3,2,12,16,4,2,1,2,7,1,18,12,1,50"},
		{Filter{FreeText: "cat", POSCodes: []int{12}, RequestedCount: 10}, "1,3,1,3,99,97,116,3,1,12,12,1,10"},
		{Filter{HistoryResult: []int{2, 3}, ConfidenceRange: []int{0, 30}, RequestedCount: -1}, "1,3,8,2,2,3,9,2,0,30,12,1,-1"},
		{Filter{RequestedCount: -1}, "1,1,12,1,-1"},
		{Filter{FreeText: " ＣＡＴ  猫😀 ", CorpusIDs: []int{1, 6100}, POSCodes: []int{1, 12}, LevelCodes: []int{3}, SectionCodes: []int{4, 11, 19}, CollectionIDs: []ID{1, 12}, InitialLetter: 18, HistoryResult: []int{2, 3}, ConfidenceRange: []int{0, 100}, AttemptTimeRange: []int64{-1, 1793836800000}, RequestedCount: 10}, ""},
	}
	for _, tc := range cases {
		raw, err := EncodeFilter(tc.f)
		if err != nil {
			t.Fatal(err)
		}
		if tc.want != "" && raw != tc.want {
			t.Fatalf("got %s; want %s", raw, tc.want)
		}
		decoded, err := DecodeFilter(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(tc.f, decoded) {
			t.Fatalf("round trip changed filter: %#v => %#v", tc.f, decoded)
		}
	}
}

func TestMalformedFilterSpecifications(t *testing.T) {
	for _, raw := range []string{"", "2,1,12,1,2", "1,1,12,1,0", "1,2,7,1,0,12,1,2", "1,2,7,1,27,12,1,2", "1,1,3,1,12", "1,2,12,1,2,12,1,3", "1,1,12,2,1", "1,1,12,1,1,4", "1,1,12,1,+1", "1,1,12,1, 1", "1,2,1,1,55296,12,1,1", "1,2,9,2,80,20,12,1,1", "1,2,3,1,17,12,1,1", "1,2,11,2,200,100,12,1,1"} {
		if _, err := DecodeFilter(raw); err == nil {
			t.Errorf("accepted invalid filter_spec %q", raw)
		}
	}
}

func TestCSVCorpusAndSearchSemantics(t *testing.T) {
	c := testCorpus(t)
	if c.SHA256 != CorpusSHA256 || len(c.Rows) != 6100 || c.ByID[1].Word != "a, an" {
		t.Fatal("Corpus changed")
	}
	count := func(f Filter) int {
		n := 0
		for _, r := range c.Rows {
			if c.Match(r, f) {
				n++
			}
		}
		return n
	}
	if count(Filter{}) != 6100 || count(Filter{InitialLetter: 24}) != 0 {
		t.Fatal("All/X filtering is incorrect")
	}
	if count(Filter{LevelCodes: []int{3}}) != 151 {
		t.Fatal("blank LEVEL not distinguished")
	}
	if count(Filter{FreeText: "ＡＢＩＬＩＴＹ  能力", POSCodes: []int{12}, InitialLetter: 1}) == 0 {
		t.Fatal("NFKC + cross-field token AND failed")
	}
	if count(Filter{FreeText: "ability 非存在検索語"}) != 0 {
		t.Fatal("tokens are not ANDed")
	}
	if count(Filter{POSCodes: []int{12, 16}}) != count(Filter{POSCodes: []int{12}})+count(Filter{POSCodes: []int{16}}) {
		t.Fatal("POS within-attribute OR failed")
	}
	row := c.ByID[3]
	if !c.Match(row, Filter{SectionCodes: []int{4, 18}}) || c.Match(row, Filter{SectionCodes: []int{1, 19}}) {
		t.Fatal("SECTION exact-code membership failed")
	}
}

func TestClozeSpansAndOriginalSurfaceForms(t *testing.T) {
	c := testCorpus(t)
	if c.ClozeEligible != 6100 {
		t.Fatalf("expected verified span for all current Corpus records; got %d", c.ClozeEligible)
	}
	for _, r := range c.Rows {
		if r.cloze == nil {
			t.Fatalf("missing span at %d", r.ID)
		}
		ex := []rune(r.Example)
		p := r.cloze
		if p.Start < 0 || p.Length < 1 || p.Start+p.Length > len(ex) || strings.TrimSpace(string(ex[p.Start:p.Start+p.Length])) == "" {
			t.Fatalf("invalid span for %d", r.ID)
		}
	}
	for _, tc := range []struct {
		id, start, length int
		answer            string
	}{{2, 10, 7, "abandon"}, {315, -1, -1, "arose"}, {709, -1, -1, "bought"}, {902, -1, -1, "classified"}, {1870, -1, -1, "entities"}, {3070, -1, -1, "leapt"}, {5725, -1, -1, "underwent"}} {
		r := c.ByID[tc.id]
		span := r.cloze
		actual := string([]rune(r.Example)[span.Start : span.Start+span.Length])
		if actual != tc.answer {
			t.Errorf("ID%d expected surface %q; got %q", tc.id, tc.answer, actual)
		}
		if tc.start >= 0 && (span.Start != tc.start || span.Length != tc.length) {
			t.Errorf("ID%d incorrect rune span", tc.id)
		}
	}
	if locateCloze(&Record{Word: "nonexistent", POS: "verb", Example: "We went to the river."}) != nil {
		t.Fatal("unsafe fallback invented a cloze")
	}
}

func TestAttemptsAreAtomicAndNullable(t *testing.T) {
	s := testStore(t)
	session, err := s.CreateSession(SessionRequest{QuizType: "reading_review", Variant: "word_and_example", SourceMode: "new", Filter: Filter{CorpusIDs: []int{3}, RequestedCount: 1}})
	if err != nil {
		t.Fatal(err)
	}
	activity, err := s.NewActivity()
	if err != nil {
		t.Fatal(err)
	}
	action := &Action{OccurredAt: 123456789, ActivityID: activity, Screen: "reading_review", Event: "submit"}
	invalid := *action
	invalid.ActivityID = 0
	if _, err = s.SubmitAttempt(session.SessionID, AttemptRequest{Position: 1, Outcome: "answered", Recalled: intPointer(1), Action: &invalid}); err == nil {
		t.Fatal("invalid answer Action accepted")
	}
	var n int
	if err = s.db.QueryRow("SELECT COUNT(*) FROM attempt_logs").Scan(&n); err != nil || n != 0 {
		t.Fatal("failed Action left a partial Attempt")
	}
	result, err := s.SubmitAttempt(session.SessionID, AttemptRequest{Position: 1, Outcome: "answered", Recalled: intPointer(0), Confidence: intPointer(0), Action: action})
	if err != nil {
		t.Fatal(err)
	}
	if result.Attempt.Correct != nil || result.Attempt.Confidence == nil || *result.Attempt.Confidence != 0 || result.Session.Status != "completed" {
		t.Fatal("reading self-assessment/zero confidence/completion lost")
	}
	if result.Attempt.OccurredAt == nil || *result.Attempt.OccurredAt != 123456789 {
		t.Fatal("event time replaced by server time")
	}
	action.OccurredAt++
	result, err = s.SubmitAttempt(session.SessionID, AttemptRequest{Position: 1, Outcome: "unknown", Action: action})
	if err != nil {
		t.Fatal(err)
	}
	if result.Attempt.Correct != nil || result.Attempt.Recalled != nil || result.Attempt.Confidence != nil {
		t.Fatal("NULL converted to zero")
	}
	if err = s.db.QueryRow("SELECT COUNT(*) FROM attempt_logs").Scan(&n); err != nil || n != 2 {
		t.Fatal("repeat answer replaced previous Attempt")
	}
	if err = s.db.QueryRow("SELECT COUNT(*) FROM action_logs WHERE attempt_id IS NOT NULL").Scan(&n); err != nil || n != 2 {
		t.Fatal("Attempt/Action linkage incomplete")
	}
}

func TestHistoryConstraintsUseOneAttempt(t *testing.T) {
	zero, one, low, high := 0, 1, 10, 90
	early, late := int64(1000), int64(2000)
	a := Attempt{Outcome: "answered", Correct: &zero, Confidence: &high, OccurredAt: &early}
	b := Attempt{Outcome: "answered", Correct: &one, Confidence: &low, OccurredAt: &late}
	f := Filter{HistoryResult: []int{2}, ConfidenceRange: []int{0, 30}}
	if matchesHistory(a, f) || matchesHistory(b, f) {
		t.Fatal("independent attempts improperly satisfy combined conditions")
	}
	if matchesHistory(Attempt{Outcome: "unknown", Correct: &zero}, Filter{HistoryResult: []int{2}}) {
		t.Fatal("Unknown treated as Incorrect")
	}
	if matchesHistory(Attempt{Outcome: "answered", Correct: &zero}, Filter{ConfidenceRange: []int{0, 100}}) {
		t.Fatal("NULL confidence treated as numeric")
	}
}

func TestUsageHasKnownPairsAndUnknownEnds(t *testing.T) {
	actions := []Action{{ActivityID: 1, Screen: "dictionary", Event: "enter", OccurredAt: 1000}, {ActivityID: 1, Screen: "dictionary", Event: "search", OccurredAt: 2000}, {ActivityID: 1, Screen: "dictionary", Event: "leave", OccurredAt: 61000}, {ActivityID: 1, Screen: "data", Event: "enter", OccurredAt: 62000}}
	u := calculateUsage(actions, nil)
	if u.TotalMS != 60000 || u.UnknownIntervals != 1 || len(u.ByScreen) != 1 || !u.Approximate {
		t.Fatalf("incorrect usage: %#v", u)
	}
	u = calculateUsage(actions, []int64{2000, 12000})
	if u.TotalMS != 10000 {
		t.Fatal("date range clipping incorrect")
	}
	closed := []Action{{ActivityID: 1, Screen: "dictionary", Event: "enter", OccurredAt: 1000}, {ActivityID: 1, Screen: "dictionary", Event: "hidden", OccurredAt: 2000}, {ActivityID: 1, Screen: "dictionary", Event: "leave", OccurredAt: 2001}}
	u = calculateUsage(closed, nil)
	if u.TotalMS != 1000 || u.UnknownIntervals != 0 {
		t.Fatal("hidden/pagehide duplicate closure produced an invented unknown interval")
	}
}

func TestSevenTablesNineActionColumns(t *testing.T) {
	s := testStore(t)
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'").Scan(&n); err != nil || n != 7 {
		t.Fatal("schema has other than seven application tables")
	}
	rows, err := s.db.Query("PRAGMA table_info(action_logs)")
	if err != nil {
		t.Fatal(err)
	}
	n = 0
	for rows.Next() {
		var a, b, c, d, e, f any
		if err = rows.Scan(&a, &b, &c, &d, &e, &f); err != nil {
			t.Fatal(err)
		}
		n++
	}
	rows.Close()
	if n != 9 {
		t.Fatalf("action_logs has %d columns", n)
	}
	var result string
	if err = s.db.QueryRow("PRAGMA integrity_check").Scan(&result); err != nil || result != "ok" {
		t.Fatal("database integrity check failed")
	}
}
