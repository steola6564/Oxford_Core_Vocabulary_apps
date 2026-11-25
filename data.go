package main

import (
	"sort"
)

type DataRequest struct {
	Filter       Filter `json:"filter"`
	QuizType     string `json:"quiz_type"`
	Variant      string `json:"variant"`
	Offset       int    `json:"offset"`
	Limit        int    `json:"limit"`
	ActionOffset int    `json:"action_offset"`
	ActionLimit  int    `json:"action_limit"`
	Screen       string `json:"screen"`
	Event        string `json:"event"`
}
type Stats struct {
	Attempts          int      `json:"attempts"`
	ObjectiveCount    int      `json:"objective_count"`
	Correct           int      `json:"correct"`
	Incorrect         int      `json:"incorrect"`
	Unknown           int      `json:"unknown"`
	ReadingYes        int      `json:"reading_yes"`
	ReadingNo         int      `json:"reading_no"`
	Accuracy          *float64 `json:"accuracy"`
	ConfidenceAverage *float64 `json:"confidence_average"`
}
type ScreenUsage struct {
	Screen     string `json:"screen"`
	DurationMS int64  `json:"duration_ms"`
	Intervals  int    `json:"intervals"`
}
type Usage struct {
	TotalMS          int64         `json:"total_ms"`
	ByScreen         []ScreenUsage `json:"by_screen"`
	UnknownIntervals int           `json:"unknown_intervals"`
	Approximate      bool          `json:"approximate"`
}
type DataResponse struct {
	Attempts    []AttemptRecord `json:"attempts"`
	Total       int             `json:"total"`
	Actions     []Action        `json:"actions"`
	ActionTotal int             `json:"action_total"`
	Stats       Stats           `json:"stats"`
	Usage       Usage           `json:"usage"`
}

func limits(offset, limit int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	return offset, limit
}
func page[T any](values []T, offset, limit int) []T {
	offset, limit = limits(offset, limit)
	if offset >= len(values) {
		return []T{}
	}
	end := offset + limit
	if end > len(values) {
		end = len(values)
	}
	return values[offset:end]
}

func (s *Store) Data(r DataRequest) (DataResponse, error) {
	out := DataResponse{Attempts: []AttemptRecord{}, Actions: []Action{}}
	if err := validateFilterAPI(&r.Filter); err != nil {
		return out, err
	}
	history, err := s.readAttempts(s.db)
	if err != nil {
		return out, err
	}
	selected, err := s.selectionItems(s.db, r.Filter.CollectionIDs)
	if err != nil {
		return out, err
	}
	matched := []AttemptRecord{}
	confidenceTotal, confidenceN := 0, 0
	for _, a := range history {
		if r.QuizType != "" && a.QuizType != r.QuizType {
			continue
		}
		if r.Variant != "" && a.Variant != r.Variant {
			continue
		}
		if !s.Corpus.Match(a.Corpus, r.Filter) || !matchesHistory(a.Attempt, r.Filter) {
			continue
		}
		if len(r.Filter.CollectionIDs) > 0 {
			ok := false
			for _, item := range selected {
				if (item.Spec != nil && item.Spec.QuestionID == a.QuestionID) || (item.Spec == nil && item.CorpusID == a.Corpus.ID) {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		matched = append(matched, a)
		st := &out.Stats
		st.Attempts++
		if a.Outcome == "unknown" {
			st.Unknown++
		}
		if a.Correct != nil {
			st.ObjectiveCount++
			if *a.Correct == 1 {
				st.Correct++
			} else if a.Outcome == "answered" {
				st.Incorrect++
			}
		}
		if a.Recalled != nil {
			if *a.Recalled == 1 {
				st.ReadingYes++
			} else {
				st.ReadingNo++
			}
		}
		if a.Confidence != nil {
			confidenceTotal += *a.Confidence
			confidenceN++
		}
	}
	if out.Stats.ObjectiveCount > 0 {
		v := 100 * float64(out.Stats.Correct) / float64(out.Stats.ObjectiveCount)
		out.Stats.Accuracy = &v
	}
	if confidenceN > 0 {
		v := float64(confidenceTotal) / float64(confidenceN)
		out.Stats.ConfidenceAverage = &v
	}
	out.Total = len(matched)
	out.Attempts = page(matched, r.Offset, r.Limit)
	rows, err := s.db.Query("SELECT action_id,occurred_at,activity_id,screen,event,target,value,session_id,attempt_id FROM action_logs ORDER BY occurred_at,action_id")
	if err != nil {
		return out, err
	}
	all := []Action{}
	for rows.Next() {
		a, e := scanAction(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		all = append(all, a)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	rows.Close()
	out.Usage = calculateUsage(all, r.Filter.AttemptTimeRange)
	filtered := []Action{}
	for i := len(all) - 1; i >= 0; i-- {
		a := all[i]
		if r.Screen != "" && a.Screen != r.Screen {
			continue
		}
		if r.Event != "" && a.Event != r.Event {
			continue
		}
		if !timeWithin(a.OccurredAt, r.Filter.AttemptTimeRange) {
			continue
		}
		filtered = append(filtered, a)
	}
	out.ActionTotal = len(filtered)
	out.Actions = page(filtered, r.ActionOffset, r.ActionLimit)
	return out, nil
}

func timeWithin(t int64, bounds []int64) bool {
	if len(bounds) != 2 {
		return true
	}
	return (bounds[0] < 0 || t >= bounds[0]) && (bounds[1] < 0 || t <= bounds[1])
}

func calculateUsage(actions []Action, bounds []int64) Usage {
	u := Usage{ByScreen: []ScreenUsage{}, Approximate: true}
	open := map[ID]Action{}
	closed := map[ID]string{}
	byScreen := map[string]*ScreenUsage{}
	for _, a := range actions {
		switch a.Event {
		case "enter", "visible":
			delete(closed, a.ActivityID)
			if previous, ok := open[a.ActivityID]; ok && timeWithin(previous.OccurredAt, bounds) {
				u.UnknownIntervals++
			}
			open[a.ActivityID] = a
		case "leave", "hidden":
			previous, ok := open[a.ActivityID]
			// visibilitychange(hidden) is often followed by pagehide(leave).
			// Both end the same interval; the second is not a missing interval.
			if !ok && closed[a.ActivityID] == a.Screen {
				continue
			}
			if !ok || previous.Screen != a.Screen {
				if timeWithin(a.OccurredAt, bounds) {
					u.UnknownIntervals++
				}
				continue
			}
			delete(open, a.ActivityID)
			closed[a.ActivityID] = a.Screen
			start, end := previous.OccurredAt, a.OccurredAt
			if len(bounds) == 2 {
				if bounds[0] >= 0 && start < bounds[0] {
					start = bounds[0]
				}
				if bounds[1] >= 0 && end > bounds[1] {
					end = bounds[1]
				}
			}
			if end < start {
				continue
			}
			entry := byScreen[a.Screen]
			if entry == nil {
				entry = &ScreenUsage{Screen: a.Screen}
				byScreen[a.Screen] = entry
			}
			entry.DurationMS += end - start
			entry.Intervals++
			u.TotalMS += end - start
		}
	}
	for _, a := range open {
		if len(bounds) != 2 || bounds[1] < 0 || a.OccurredAt <= bounds[1] {
			u.UnknownIntervals++
		}
	}
	for _, entry := range byScreen {
		u.ByScreen = append(u.ByScreen, *entry)
	}
	sort.Slice(u.ByScreen, func(i, j int) bool { return u.ByScreen[i].Screen < u.ByScreen[j].Screen })
	return u
}
