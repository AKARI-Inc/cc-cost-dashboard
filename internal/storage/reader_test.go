package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/model"
)

// writeJSONLFile は、イベントのスライスを指定パスへ JSONL として書き込む
// テスト用ヘルパー。必要に応じて親ディレクトリを作成する。
func writeJSONLFile(t *testing.T, path string, events []any) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create dir for %s: %v", path, err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}
	defer f.Close()

	for _, ev := range events {
		data, err := json.Marshal(ev)
		if err != nil {
			t.Fatalf("failed to marshal event: %v", err)
		}
		f.Write(data)
		f.Write([]byte("\n"))
	}
}

func TestReadOtelEvents_AcrossMultipleDates(t *testing.T) {
	dataDir := t.TempDir()

	day1Events := []any{
		model.OtelEvent{Timestamp: "2025-06-01T10:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.01},
		model.OtelEvent{Timestamp: "2025-06-01T11:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.02},
	}
	day2Events := []any{
		model.OtelEvent{Timestamp: "2025-06-02T09:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.03},
	}

	writeJSONLFile(t, filepath.Join(dataDir, "logs", "otel", "2025-06-01.jsonl"), day1Events)
	writeJSONLFile(t, filepath.Join(dataDir, "logs", "otel", "2025-06-02.jsonl"), day2Events)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 2, 23, 59, 59, 0, time.UTC)

	events, err := ReadOtelEvents(dataDir, from, to)
	if err != nil {
		t.Fatalf("ReadOtelEvents returned error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// タイムスタンプ昇順でソートされていることを検証する。
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp < events[i-1].Timestamp {
			t.Errorf("events not sorted: events[%d].Timestamp=%q < events[%d].Timestamp=%q",
				i, events[i].Timestamp, i-1, events[i-1].Timestamp)
		}
	}
}

func TestReadOtelEvents_SkipsMissingDates(t *testing.T) {
	dataDir := t.TempDir()

	// 1 日目のデータのみ書き込み、2 日目のファイルは存在しない状態にする。
	day1Events := []any{
		model.OtelEvent{Timestamp: "2025-06-01T10:00:00Z", EventName: "test"},
	}
	writeJSONLFile(t, filepath.Join(dataDir, "logs", "otel", "2025-06-01.jsonl"), day1Events)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 3, 0, 0, 0, 0, time.UTC) // 3 日間にまたがるが存在するのは 1 日のみ

	events, err := ReadOtelEvents(dataDir, from, to)
	if err != nil {
		t.Fatalf("ReadOtelEvents returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestReadOtelEvents_SkipsMalformedJSON(t *testing.T) {
	dataDir := t.TempDir()

	dir := filepath.Join(dataDir, "logs", "otel")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 有効な JSON 行と無効な行が混在するファイルを書き込む。
	content := `{"timestamp":"2025-06-01T10:00:00Z","event_name":"test","cost_usd":0.01}
this is not valid json
{"timestamp":"2025-06-01T11:00:00Z","event_name":"test","cost_usd":0.02}
{broken json
`
	if err := os.WriteFile(filepath.Join(dir, "2025-06-01.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 1, 23, 59, 59, 0, time.UTC)

	events, err := ReadOtelEvents(dataDir, from, to)
	if err != nil {
		t.Fatalf("ReadOtelEvents returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 valid events (skipping malformed lines), got %d", len(events))
	}
}

func TestReadClaudeAiEvents_AcrossMultipleDates(t *testing.T) {
	dataDir := t.TempDir()

	day1Events := []any{
		model.ClaudeAiEvent{Timestamp: "2025-06-01T10:00:00Z", UserEmail: "alice@example.com", MessageRole: "human"},
		model.ClaudeAiEvent{Timestamp: "2025-06-01T11:00:00Z", UserEmail: "alice@example.com", MessageRole: "assistant"},
	}
	day2Events := []any{
		model.ClaudeAiEvent{Timestamp: "2025-06-02T09:00:00Z", UserEmail: "bob@example.com", MessageRole: "human"},
	}

	writeJSONLFile(t, filepath.Join(dataDir, "logs", "claude-ai", "2025-06-01.jsonl"), day1Events)
	writeJSONLFile(t, filepath.Join(dataDir, "logs", "claude-ai", "2025-06-02.jsonl"), day2Events)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 2, 23, 59, 59, 0, time.UTC)

	events, err := ReadClaudeAiEvents(dataDir, from, to)
	if err != nil {
		t.Fatalf("ReadClaudeAiEvents returned error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// タイムスタンプ昇順でソートされていることを検証する。
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp < events[i-1].Timestamp {
			t.Errorf("events not sorted: events[%d].Timestamp=%q < events[%d].Timestamp=%q",
				i, events[i].Timestamp, i-1, events[i-1].Timestamp)
		}
	}
}

func TestReadClaudeAiEvents_SkipsMissingDates(t *testing.T) {
	dataDir := t.TempDir()

	day1Events := []any{
		model.ClaudeAiEvent{Timestamp: "2025-06-01T10:00:00Z", UserEmail: "alice@example.com"},
	}
	writeJSONLFile(t, filepath.Join(dataDir, "logs", "claude-ai", "2025-06-01.jsonl"), day1Events)

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 5, 0, 0, 0, 0, time.UTC) // 5 日間にまたがるがデータがあるのは 1 日のみ

	events, err := ReadClaudeAiEvents(dataDir, from, to)
	if err != nil {
		t.Fatalf("ReadClaudeAiEvents returned error: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestReadClaudeAiEvents_SkipsMalformedJSON(t *testing.T) {
	dataDir := t.TempDir()

	dir := filepath.Join(dataDir, "logs", "claude-ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `{"timestamp":"2025-06-01T10:00:00Z","user_email":"alice@example.com"}
not json at all
{"timestamp":"2025-06-01T11:00:00Z","user_email":"bob@example.com"}
`
	if err := os.WriteFile(filepath.Join(dir, "2025-06-01.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 6, 1, 23, 59, 59, 0, time.UTC)

	events, err := ReadClaudeAiEvents(dataDir, from, to)
	if err != nil {
		t.Fatalf("ReadClaudeAiEvents returned error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 valid events (skipping malformed line), got %d", len(events))
	}
}
