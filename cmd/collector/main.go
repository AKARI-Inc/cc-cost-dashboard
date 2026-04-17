package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AKARI-Inc/cc-cost-dashboard/internal/collector"
	"github.com/AKARI-Inc/cc-cost-dashboard/internal/storage"
)

func main() {
	ctx := context.Background()

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	saveRaw := os.Getenv("SAVE_RAW") == "true"

	writer, backend, err := storage.NewWriter(ctx, dataDir)
	if err != nil {
		log.Fatalf("init storage writer: %v", err)
	}
	log.Printf("Storage backend: %s", backend)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /v1/logs", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20))
		if err != nil {
			log.Printf("ERROR: failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{}`))
			return
		}

		if saveRaw {
			rawDir := filepath.Join(dataDir, "logs", "otel", "raw")
			if err := os.MkdirAll(rawDir, 0o755); err != nil {
				log.Printf("ERROR: failed to create raw dir: %v", err)
			} else {
				ts := time.Now().UTC().Format("20060102T150405.000000000")
				rawPath := filepath.Join(rawDir, fmt.Sprintf("%s.bin", ts))
				if err := os.WriteFile(rawPath, body, 0o644); err != nil {
					log.Printf("ERROR: failed to write raw payload: %v", err)
				}
			}
		}

		decoded, err := collector.DecodeLogs(body)
		if err != nil {
			log.Printf("ERROR: failed to decode logs: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{}`))
			return
		}

		events := collector.ExtractEvents(decoded)
		log.Printf("Received %d event(s)", len(events))

		var writeErrors int
		for _, event := range events {
			if err := writer.AppendEvent(r.Context(), "otel", event); err != nil {
				log.Printf("ERROR: failed to write event: %v", err)
				writeErrors++
			}
		}

		switch {
		case writeErrors == 0:
			w.WriteHeader(http.StatusOK)
		case writeErrors < len(events):
			log.Printf("WARN: %d/%d event(s) failed to write", writeErrors, len(events))
			w.WriteHeader(http.StatusOK)
		default:
			// 全件失敗 → 500 で OTel SDK にリトライさせる
			log.Printf("ERROR: all %d event(s) failed to write", len(events))
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte(`{}`))
	})

	mux.HandleFunc("POST /v1/traces", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	mux.HandleFunc("POST /v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "4318"
	}

	srv := &http.Server{Addr: ":" + port, Handler: mux}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		log.Println("shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		srv.Shutdown(shutdownCtx)
		if err := writer.Close(shutdownCtx); err != nil {
			log.Printf("ERROR: writer close: %v", err)
		}
	}()

	log.Printf("Collector listening on :%s", port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
