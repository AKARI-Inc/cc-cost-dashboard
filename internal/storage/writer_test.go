package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/model"
)

func TestFileWriter_WritesValidJSONL(t *testing.T) {
	dataDir := t.TempDir()
	w := NewFileWriter(dataDir)

	ev := model.OtelEvent{
		Timestamp:   "2025-06-01T10:00:00Z",
		EventName:   "claude_code.api_request",
		SessionID:   "sess-1",
		UserEmail:   "alice@example.com",
		Model:       "claude-sonnet-4-20250514",
		InputTokens: 100,
		CostUSD:     0.05,
	}

	if err := w.AppendEvent(context.Background(), "otel", ev); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	filename := filepath.Join(dataDir, "logs", "otel", today+".jsonl")

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	line := strings.TrimSpace(string(data))
	var got model.OtelEvent
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("failed to unmarshal JSONL line: %v", err)
	}

	if got.EventName != ev.EventName {
		t.Errorf("EventName = %q, want %q", got.EventName, ev.EventName)
	}
	if got.UserEmail != ev.UserEmail {
		t.Errorf("UserEmail = %q, want %q", got.UserEmail, ev.UserEmail)
	}
	if got.InputTokens != ev.InputTokens {
		t.Errorf("InputTokens = %d, want %d", got.InputTokens, ev.InputTokens)
	}
}

func TestFileWriter_CreatesDirectories(t *testing.T) {
	dataDir := t.TempDir()
	w := NewFileWriter(dataDir)

	logGroup := "custom-group"
	dir := filepath.Join(dataDir, "logs", logGroup)

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("expected directory %s to not exist before AppendEvent", dir)
	}

	ev := model.OtelEvent{Timestamp: "2025-06-01T10:00:00Z", EventName: "test"}
	if err := w.AppendEvent(context.Background(), logGroup, ev); err != nil {
		t.Fatalf("AppendEvent returned error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected directory %s to exist after AppendEvent: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %s to be a directory", dir)
	}
}

func TestFileWriter_MultipleCallsAppendToSameFile(t *testing.T) {
	dataDir := t.TempDir()
	w := NewFileWriter(dataDir)

	events := []model.OtelEvent{
		{Timestamp: "2025-06-01T10:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.01},
		{Timestamp: "2025-06-01T10:01:00Z", EventName: "claude_code.api_request", CostUSD: 0.02},
		{Timestamp: "2025-06-01T10:02:00Z", EventName: "claude_code.api_request", CostUSD: 0.03},
	}

	for _, ev := range events {
		if err := w.AppendEvent(context.Background(), "otel", ev); err != nil {
			t.Fatalf("AppendEvent returned error: %v", err)
		}
	}

	today := time.Now().UTC().Format("2006-01-02")
	filename := filepath.Join(dataDir, "logs", "otel", today+".jsonl")

	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(events) {
		t.Fatalf("expected %d lines, got %d", len(events), len(lines))
	}

	for i, line := range lines {
		var got model.OtelEvent
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Errorf("line %d: failed to unmarshal: %v", i, err)
			continue
		}
		if got.CostUSD != events[i].CostUSD {
			t.Errorf("line %d: CostUSD = %f, want %f", i, got.CostUSD, events[i].CostUSD)
		}
	}
}
