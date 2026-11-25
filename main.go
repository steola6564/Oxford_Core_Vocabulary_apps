package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	csvPath := flag.String("csv", "data/all_with_ID.csv", "external authoritative Corpus CSV")
	dbPath := flag.String("db", "data/learning.db", "external SQLite learning database (created on first start)")
	check := flag.Bool("check-corpus", false, "validate CSV and print its metadata, without opening a database")
	version := flag.Bool("version", false, "print application version")
	flag.Parse()
	if *version {
		fmt.Println("Oxford Core Vocabulary 1.0")
		return
	}
	corpus, err := LoadCorpus(*csvPath, 6100)
	if err != nil {
		log.Fatal(err)
	}
	if *check {
		json.NewEncoder(os.Stdout).Encode(map[string]any{"count": len(corpus.Rows), "sha256": corpus.SHA256, "expected_sha256": CorpusSHA256, "matches_reference": corpus.SHA256 == CorpusSHA256, "cloze_eligible": corpus.ClozeEligible, "cloze_skipped": len(corpus.Rows) - corpus.ClozeEligible})
		return
	}
	if corpus.SHA256 != CorpusSHA256 {
		log.Fatal("Corpus SHA-256 differs from the v1.0 reference. Restore the original all_with_ID.csv to preserve stored question meanings.")
	}
	store, err := OpenStore(*dbPath, corpus)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	server := &http.Server{Addr: *addr, Handler: store.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	log.Printf("Oxford Core Vocabulary v1.0 at http://%s | Corpus %d rows | Cloze %d eligible | DB %s", *addr, len(corpus.Rows), corpus.ClozeEligible, *dbPath)
	if err = server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
