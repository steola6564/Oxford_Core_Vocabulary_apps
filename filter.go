package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// ID is stored as an integer and is always represented by eight digits over HTTP.
type ID int64

func (i ID) String() string { return fmt.Sprintf("%08d", i) }
func (i ID) MarshalJSON() ([]byte, error) {
	if i == 0 {
		return []byte("null"), nil
	}
	return json.Marshal(i.String())
}
func (i *ID) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		*i = 0
		return nil
	}
	s := string(b)
	if len(s) > 0 && s[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v < 0 || v > 99999999 {
		return fmt.Errorf("invalid eight-digit ID")
	}
	*i = ID(v)
	return nil
}

type Filter struct {
	FreeText         string  `json:"free_text"`
	CorpusIDs        []int   `json:"corpus_ids"`
	POSCodes         []int   `json:"pos_codes"`
	LevelCodes       []int   `json:"level_codes"`
	SectionCodes     []int   `json:"section_codes"`
	CollectionIDs    []ID    `json:"collection_ids"`
	InitialLetter    int     `json:"initial_letter"`
	HistoryResult    []int   `json:"history_result"`
	ConfidenceRange  []int   `json:"confidence_range"`
	ScoreRange       []int   `json:"score_range,omitempty"`
	AttemptTimeRange []int64 `json:"attempt_time_range"`
	RequestedCount   int     `json:"requested_count"`
}

func (f *Filter) defaults() {
	if f.RequestedCount == 0 {
		f.RequestedCount = 20
	}
}
func (f Filter) Validate() error {
	if len([]rune(f.FreeText)) > 2048 {
		return fmt.Errorf("search text is limited to 2048 characters")
	}
	for _, set := range []struct {
		name   string
		values []int
		max    int
	}{{"corpus_ids", f.CorpusIDs, 6100}, {"pos_codes", f.POSCodes, 16}, {"level_codes", f.LevelCodes, 3}, {"section_codes", f.SectionCodes, 19}, {"history_result", f.HistoryResult, 5}} {
		if len(set.values) > 10000 {
			return fmt.Errorf("too many %s", set.name)
		}
		for _, v := range set.values {
			if v < 1 || v > set.max {
				return fmt.Errorf("invalid %s code %d", set.name, v)
			}
		}
	}
	for _, v := range f.CollectionIDs {
		if v < 1 || v > 99999999 {
			return fmt.Errorf("invalid collection ID")
		}
	}
	if f.InitialLetter < 0 || f.InitialLetter > 26 {
		return fmt.Errorf("initial_letter must be 0..26")
	}
	for _, bounds := range [][]int{f.ConfidenceRange, f.ScoreRange} {
		if len(bounds) > 0 && (len(bounds) != 2 || bounds[0] < 0 || bounds[1] > 100 || bounds[0] > bounds[1]) {
			return fmt.Errorf("confidence/score range requires lower,upper in 0..100")
		}
	}
	if len(f.AttemptTimeRange) > 0 {
		if len(f.AttemptTimeRange) != 2 {
			return fmt.Errorf("attempt_time_range requires start,end")
		}
		for _, v := range f.AttemptTimeRange {
			if v < -1 {
				return fmt.Errorf("time range endpoints must be Unix milliseconds or -1")
			}
		}
		if f.AttemptTimeRange[0] >= 0 && f.AttemptTimeRange[1] >= 0 && f.AttemptTimeRange[0] > f.AttemptTimeRange[1] {
			return fmt.Errorf("time range start exceeds end")
		}
	}
	if f.RequestedCount != -1 && (f.RequestedCount < 1 || f.RequestedCount > 100000) {
		return fmt.Errorf("requested_count must be -1 or 1..100000")
	}
	return nil
}

func EncodeFilter(f Filter) (string, error) {
	if err := f.Validate(); err != nil {
		return "", err
	}
	blocks := [][]int64{}
	add := func(tag int, values []int64) {
		if len(values) > 0 {
			block := []int64{int64(tag), int64(len(values))}
			block = append(block, values...)
			blocks = append(blocks, block)
		}
	}
	int64s := func(values []int) []int64 {
		out := make([]int64, len(values))
		for i, v := range values {
			out[i] = int64(v)
		}
		return out
	}
	runes := []int64{}
	for _, r := range f.FreeText {
		runes = append(runes, int64(r))
	}
	add(1, runes)
	add(2, int64s(f.CorpusIDs))
	add(3, int64s(f.POSCodes))
	add(4, int64s(f.LevelCodes))
	add(5, int64s(f.SectionCodes))
	ids := []int64{}
	for _, id := range f.CollectionIDs {
		ids = append(ids, int64(id))
	}
	add(6, ids)
	if f.InitialLetter > 0 {
		add(7, []int64{int64(f.InitialLetter)})
	}
	add(8, int64s(f.HistoryResult))
	add(9, int64s(f.ConfidenceRange))
	add(10, int64s(f.ScoreRange))
	add(11, f.AttemptTimeRange)
	add(12, []int64{int64(f.RequestedCount)})
	values := []int64{1, int64(len(blocks))}
	for _, block := range blocks {
		values = append(values, block...)
	}
	stringsOut := make([]string, len(values))
	for i, v := range values {
		stringsOut[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(stringsOut, ","), nil
}

func DecodeFilter(raw string) (Filter, error) {
	f := Filter{}
	if len(raw) > 150000 {
		return f, fmt.Errorf("filter_spec too long")
	}
	parts := strings.Split(raw, ",")
	values := make([]int64, len(parts))
	for i, p := range parts {
		if p == "" || strings.TrimSpace(p) != p || strings.HasPrefix(p, "+") {
			return f, fmt.Errorf("invalid filter_spec integer")
		}
		for j, r := range p {
			if r >= '0' && r <= '9' {
				continue
			}
			if j == 0 && r == '-' {
				continue
			}
			return f, fmt.Errorf("invalid filter_spec character")
		}
		v, e := strconv.ParseInt(p, 10, 64)
		if e != nil {
			return f, e
		}
		values[i] = v
	}
	if len(values) < 2 || values[0] != 1 || values[1] < 1 || values[1] > 12 {
		return f, fmt.Errorf("unsupported filter_spec version/block count")
	}
	seen := map[int64]bool{}
	pos := 2
	toInts := func(a []int64) ([]int, error) {
		out := make([]int, len(a))
		for i, v := range a {
			if v < -1 || v > 99999999 {
				return nil, fmt.Errorf("filter code out of range")
			}
			out[i] = int(v)
		}
		return out, nil
	}
	for n := int64(0); n < values[1]; n++ {
		if pos+2 > len(values) {
			return f, fmt.Errorf("truncated filter_spec block")
		}
		tag, count := values[pos], values[pos+1]
		pos += 2
		if tag < 1 || tag > 12 || seen[tag] || count < 1 || count > 100000 || count > int64(len(values)-pos) {
			return f, fmt.Errorf("invalid or duplicate filter_spec tag/count")
		}
		seen[tag] = true
		v := values[pos : pos+int(count)]
		pos += int(count)
		switch tag {
		case 1:
			runes := make([]rune, len(v))
			for i, r := range v {
				if r < 0 || r > utf8.MaxRune || (r >= 0xD800 && r <= 0xDFFF) {
					return f, fmt.Errorf("invalid Unicode code point")
				}
				runes[i] = rune(r)
			}
			f.FreeText = string(runes)
		case 6:
			for _, id := range v {
				f.CollectionIDs = append(f.CollectionIDs, ID(id))
			}
		case 11:
			f.AttemptTimeRange = append([]int64{}, v...)
		default:
			ints, e := toInts(v)
			if e != nil {
				return f, e
			}
			switch tag {
			case 2:
				f.CorpusIDs = ints
			case 3:
				f.POSCodes = ints
			case 4:
				f.LevelCodes = ints
			case 5:
				f.SectionCodes = ints
			case 8:
				f.HistoryResult = ints
			case 9:
				f.ConfidenceRange = ints
			case 10:
				f.ScoreRange = ints
			case 7:
				if len(ints) != 1 || ints[0] < 1 || ints[0] > 26 {
					return f, fmt.Errorf("initial letter tag requires one value in 1..26; omit for All")
				}
				f.InitialLetter = ints[0]
			case 12:
				if len(ints) != 1 {
					return f, fmt.Errorf("requested_count requires one value")
				}
				f.RequestedCount = ints[0]
			}
		}
	}
	if pos != len(values) || !seen[12] {
		return f, fmt.Errorf("filter_spec trailing values or missing requested_count")
	}
	return f, f.Validate()
}
