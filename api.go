package main

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed web/index.html
var webFS embed.FS

type apiHandler func(http.ResponseWriter, *http.Request) error

func jsonResponse(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	return json.NewEncoder(w).Encode(v)
}
func api(fn apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			status, code := 500, "internal_error"
			var ae *AppError
			if errors.As(err, &ae) {
				status, code = ae.Status, ae.Code
			} else {
				log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error(), "code": code})
		}
	}
}
func body(r *http.Request, v any, optional bool) error {
	if r.Header.Get("Content-Type") != "" && !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		return bad("Content-Type must be application/json")
	}
	d := json.NewDecoder(io.LimitReader(r.Body, 6<<20))
	d.DisallowUnknownFields()
	err := d.Decode(v)
	if optional && err == io.EOF {
		return nil
	}
	if err != nil {
		return bad("Invalid JSON: " + err.Error())
	}
	if d.Decode(new(any)) != io.EOF {
		return bad("Only one JSON object is allowed")
	}
	return nil
}
func routeID(r *http.Request, key string) (ID, error) {
	v, err := strconv.ParseInt(r.PathValue(key), 10, 64)
	if err != nil || v < 1 || v > 99999999 {
		return 0, bad("Invalid " + key)
	}
	return ID(v), nil
}
func routeInt(r *http.Request, key string) (int, error) {
	v, err := strconv.Atoi(r.PathValue(key))
	if err != nil || v < 1 {
		return 0, bad("Invalid " + key)
	}
	return v, nil
}

func (s *Store) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/meta", api(func(w http.ResponseWriter, r *http.Request) error {
		type option struct {
			Code  int    `json:"code"`
			Label string `json:"label"`
		}
		pos := []option{}
		for i, label := range PosLabels {
			pos = append(pos, option{i + 1, label})
		}
		sections := []option{}
		for i := 1; i <= 6; i++ {
			sections = append(sections, option{i, fmt.Sprintf("S%d", i)})
		}
		for i := 1; i <= 12; i++ {
			sections = append(sections, option{i + 6, fmt.Sprintf("W%d", i)})
		}
		sections = append(sections, option{19, "Blank"})
		return jsonResponse(w, map[string]any{"count": len(s.Corpus.Rows), "sha256": s.Corpus.SHA256, "pos": pos, "levels": []option{{1, "3000"}, {2, "5000"}, {3, "Blank"}}, "sections": sections, "cloze_eligible": s.Corpus.ClozeEligible, "version": "1.0"})
	}))
	mux.HandleFunc("GET /api/health", api(func(w http.ResponseWriter, r *http.Request) error {
		if err := s.db.Ping(); err != nil {
			return err
		}
		return jsonResponse(w, map[string]any{"ok": true, "corpus_count": len(s.Corpus.Rows)})
	}))
	mux.HandleFunc("POST /api/activity", api(func(w http.ResponseWriter, r *http.Request) error {
		var req struct{}
		if err := body(r, &req, true); err != nil {
			return err
		}
		id, err := s.NewActivity()
		if err != nil {
			return err
		}
		return jsonResponse(w, map[string]ID{"activity_id": id})
	}))
	mux.HandleFunc("POST /api/actions", api(func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Events []Action `json:"events"`
			SentAt int64    `json:"sent_at"`
		}
		if err := body(r, &req, false); err != nil {
			return err
		}
		saved, err := s.SaveActions(req.Events)
		if err != nil {
			return err
		}
		return jsonResponse(w, map[string]any{"saved": saved, "received_at": time.Now().UnixMilli()})
	}))
	mux.HandleFunc("POST /api/search", api(func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Filter Filter `json:"filter"`
			Offset int    `json:"offset"`
			Limit  int    `json:"limit"`
		}
		if err := body(r, &req, false); err != nil {
			return err
		}
		rows, err := s.Search(req.Filter)
		if err != nil {
			return err
		}
		offset, limit := limits(req.Offset, req.Limit)
		return jsonResponse(w, map[string]any{"items": page(rows, offset, limit), "total": len(rows), "offset": offset, "limit": limit})
	}))
	mux.HandleFunc("GET /api/corpus/{id}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeInt(r, "id")
		if err != nil {
			return err
		}
		row := s.Corpus.ByID[id]
		if row == nil {
			return notFound("Corpus record not found")
		}
		return jsonResponse(w, row)
	}))
	mux.HandleFunc("POST /api/candidates", api(func(w http.ResponseWriter, r *http.Request) error {
		var req SessionRequest
		if err := body(r, &req, false); err != nil {
			return err
		}
		total, skipped, err := s.Candidates(req)
		if err != nil {
			return err
		}
		return jsonResponse(w, map[string]int{"total": total, "cloze_skipped": skipped})
	}))
	mux.HandleFunc("POST /api/sessions", api(func(w http.ResponseWriter, r *http.Request) error {
		var req SessionRequest
		if err := body(r, &req, false); err != nil {
			return err
		}
		v, err := s.CreateSession(req)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("GET /api/sessions", api(func(w http.ResponseWriter, r *http.Request) error {
		v, err := s.ListSessions(r.URL.Query().Get("quiz_type"))
		if err != nil {
			return err
		}
		return jsonResponse(w, map[string]any{"items": v})
	}))
	mux.HandleFunc("GET /api/sessions/{id}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		v, err := s.GetSession(id)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("PATCH /api/sessions/{id}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		var req struct {
			CurrentPosition int    `json:"current_position"`
			Status          string `json:"status"`
		}
		if err = body(r, &req, false); err != nil {
			return err
		}
		v, err := s.UpdateSession(id, req.CurrentPosition, req.Status)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("GET /api/sessions/{id}/questions/{position}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		position, err := routeInt(r, "position")
		if err != nil {
			return err
		}
		v, err := s.GetQuestion(id, position)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("POST /api/sessions/{id}/attempts", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		var req AttemptRequest
		if err = body(r, &req, false); err != nil {
			return err
		}
		v, err := s.SubmitAttempt(id, req)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("GET /api/collections", api(func(w http.ResponseWriter, r *http.Request) error {
		v, err := s.ListCollections()
		if err != nil {
			return err
		}
		return jsonResponse(w, map[string]any{"items": v})
	}))
	mux.HandleFunc("POST /api/collections", api(func(w http.ResponseWriter, r *http.Request) error {
		var req CollectionRequest
		if err := body(r, &req, false); err != nil {
			return err
		}
		v, err := s.CreateCollection(req)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("GET /api/collections/{id}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		v, err := s.GetCollection(id)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("POST /api/collections/{id}/items", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		var req ItemRequest
		if err = body(r, &req, false); err != nil {
			return err
		}
		v, err := s.AddCollectionItem(id, req)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	mux.HandleFunc("DELETE /api/collections/{id}/items/{item_id}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		itemID, err := routeID(r, "item_id")
		if err != nil {
			return err
		}
		var req struct {
			Action *Action `json:"action"`
		}
		if err = body(r, &req, true); err != nil {
			return err
		}
		if err = s.DeleteCollectionItem(id, itemID, req.Action); err != nil {
			return err
		}
		return jsonResponse(w, map[string]bool{"deleted": true})
	}))
	mux.HandleFunc("DELETE /api/collections/{id}", api(func(w http.ResponseWriter, r *http.Request) error {
		id, err := routeID(r, "id")
		if err != nil {
			return err
		}
		var req struct {
			Action *Action `json:"action"`
		}
		if err = body(r, &req, true); err != nil {
			return err
		}
		if err = s.DeleteCollection(id, req.Action); err != nil {
			return err
		}
		return jsonResponse(w, map[string]bool{"deleted": true})
	}))
	mux.HandleFunc("POST /api/data", api(func(w http.ResponseWriter, r *http.Request) error {
		var req DataRequest
		if err := body(r, &req, false); err != nil {
			return err
		}
		v, err := s.Data(req)
		if err != nil {
			return err
		}
		return jsonResponse(w, v)
	}))
	serveIndex := func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/index.html")
		if err != nil {
			http.Error(w, "Frontend unavailable", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(b)
	}
	mux.HandleFunc("GET /{$}", serveIndex)
	mux.HandleFunc("GET /index.html", serveIndex)
	mux.HandleFunc("GET /manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		io.WriteString(w, `{"name":"Oxford Core Vocabulary","short_name":"Oxford Core","start_url":"/","display":"standalone","background_color":"#f7f5f0","theme_color":"#c64b2c","icons":[{"src":"/icon.svg","sizes":"any","type":"image/svg+xml","purpose":"any"}]}`)
	})
	mux.HandleFunc("GET /icon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		io.WriteString(w, `<svg xmlns="http://www.w3.org/2000/svg" width="192" height="192" viewBox="0 0 192 192"><rect width="192" height="192" rx="40" fill="#c64b2c"/><circle cx="96" cy="96" r="48" fill="none" stroke="#fffaf4" stroke-width="15"/><path d="M132 133L154 154" stroke="#fffaf4" stroke-width="15" stroke-linecap="round"/></svg>`)
	})
	mux.HandleFunc("GET /sw.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		io.WriteString(w, `// Installation support only. No offline response or learning-data cache.
self.addEventListener('install',()=>self.skipWaiting());
self.addEventListener('activate',e=>e.waitUntil(self.clients.claim()));`)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		mux.ServeHTTP(w, r)
	})
}
