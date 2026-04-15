package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/collector"
	"github.com/narumina/cc-cost-dashboard/internal/storage"
)

func main() {
	ctx := context.Background()

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	// STORAGE=file (default) または cloudwatch で切り替え可能。
	// cloudwatch は LocalStack（AWS_ENDPOINT_URL=http://localstack:4566）と
	// 本番 AWS の両方をカバーする。
	writer, err := storage.NewWriter(ctx, dataDir)
	if err != nil {
		log.Fatalf("init storage writer: %v", err)
	}
	log.Printf("Storage backend: %s", os.Getenv("STORAGE"))

	mux := http.NewServeMux()

	// メインエンドポイント: OTel の protobuf ログを受信する
	mux.HandleFunc("POST /v1/logs", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: failed to read request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{}`))
			return
		}

		// デバッグ用に生ペイロードを保存する
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

		// protobuf をデコードする
		decoded, err := collector.DecodeLogs(body)
		if err != nil {
			log.Printf("ERROR: failed to decode logs: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{}`))
			return
		}

		// イベントを抽出する
		events := collector.ExtractEvents(decoded)
		log.Printf("Received %d event(s)", len(events))

		// 各イベントをストレージに書き込む
		var writeErrors int
		for _, event := range events {
			if err := writer.AppendEvent(r.Context(), "otel", event); err != nil {
				log.Printf("ERROR: failed to write event: %v", err)
				writeErrors++
			}
		}
		if writeErrors > 0 {
			log.Printf("WARN: %d/%d event(s) failed to write", writeErrors, len(events))
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	// traces と metrics は受理のみ行い内容は無視する
	mux.HandleFunc("POST /v1/traces", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})
	mux.HandleFunc("POST /v1/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	// ヘルスチェック
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "4318"
	}

	log.Printf("Collector listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
