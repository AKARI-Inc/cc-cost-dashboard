package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/model"
)

// mockReader は EventReader のテスト用スタブ。
type mockReader struct {
	events []model.OtelEvent
	err    error
}

func (m *mockReader) ReadOtelEvents(_ context.Context, _, _ time.Time) ([]model.OtelEvent, error) {
	return m.events, m.err
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	return &Handler{DataDir: t.TempDir()}
}

func newMockHandler(events []model.OtelEvent) *Handler {
	return &Handler{Reader: &mockReader{events: events}}
}

var sampleEvents = []model.OtelEvent{
	{
		Timestamp:      "2024-06-01T10:00:00Z",
		EventName:      "claude_code.api_request",
		UserEmail:      "alice@example.com",
		Model:          "claude-sonnet-4-20250514",
		InputTokens:    1000,
		OutputTokens:   200,
		CostUSD:        0.005,
		TerminalType:   "vscode",
		ServiceVersion: "1.0.0",
		Speed:          "normal",
	},
	{
		Timestamp:      "2024-06-01T11:00:00Z",
		EventName:      "claude_code.api_request",
		UserEmail:      "bob@example.com",
		Model:          "claude-sonnet-4-20250514",
		InputTokens:    2000,
		OutputTokens:   400,
		CostUSD:        0.010,
		TerminalType:   "terminal",
		ServiceVersion: "1.0.1",
		Speed:          "fast",
	},
	{
		Timestamp: "2024-06-01T12:00:00Z",
		EventName: "claude_code.conversation_start",
		UserEmail: "alice@example.com",
	},
}

func serve(h *Handler, method, url string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(method, url, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

func TestHealth(t *testing.T) {
	w := serve(newTestHandler(t), "GET", "/api/health")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", body["status"])
	}
}

func TestClaudeCodeUsageInvalidDate(t *testing.T) {
	w := serve(newTestHandler(t), "GET", "/api/claude-code/usage?from=bad-date")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["error"] == "" {
		t.Fatal("expected error message")
	}
}

func TestClaudeCodeUsageEmptyData(t *testing.T) {
	w := serve(newTestHandler(t), "GET", "/api/claude-code/usage?from=2024-01-01&to=2024-01-02")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Data []any          `json:"data"`
		Meta map[string]any `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 0 {
		t.Fatalf("expected empty data, got %d items", len(body.Data))
	}
	if body.Meta["groupBy"] != "day" {
		t.Fatalf("expected groupBy=day, got %v", body.Meta["groupBy"])
	}
}

func TestClaudeCodeUsageInvalidGroupBy(t *testing.T) {
	w := serve(newTestHandler(t), "GET", "/api/claude-code/usage?groupBy=invalid")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestClaudeCodeUsageWithMockData(t *testing.T) {
	h := newMockHandler(sampleEvents)

	tests := []struct {
		name     string
		groupBy  string
		wantLen  int
		checkKey string
	}{
		{"day", "day", 1, "date"},
		{"model", "model", 1, "model"},
		{"user", "user", 2, "user_email"},
		{"terminal", "terminal", 2, "key"},
		{"version", "version", 2, "key"},
		{"speed", "speed", 2, "key"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := serve(h, "GET", "/api/claude-code/usage?from=2024-06-01&to=2024-06-01&groupBy="+tt.groupBy)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
			}
			var body struct {
				Data []map[string]any `json:"data"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if len(body.Data) != tt.wantLen {
				t.Fatalf("expected %d rows, got %d", tt.wantLen, len(body.Data))
			}
			for _, row := range body.Data {
				if row[tt.checkKey] == nil || row[tt.checkKey] == "" {
					t.Errorf("expected %q to be set, row: %v", tt.checkKey, row)
				}
			}
		})
	}
}

func TestClaudeCodeUsageAggregation(t *testing.T) {
	h := newMockHandler(sampleEvents)
	w := serve(h, "GET", "/api/claude-code/usage?from=2024-06-01&to=2024-06-01&groupBy=day")

	var body struct {
		Data []struct {
			Date         string  `json:"date"`
			TotalCostUSD float64 `json:"total_cost_usd"`
			InputTokens  int     `json:"input_tokens"`
			OutputTokens int     `json:"output_tokens"`
			RequestCount int     `json:"request_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 day row, got %d", len(body.Data))
	}
	row := body.Data[0]
	if row.RequestCount != 2 {
		t.Errorf("expected 2 requests, got %d", row.RequestCount)
	}
	if row.InputTokens != 3000 {
		t.Errorf("expected 3000 input tokens, got %d", row.InputTokens)
	}
	if row.OutputTokens != 600 {
		t.Errorf("expected 600 output tokens, got %d", row.OutputTokens)
	}
	wantCost := 0.015
	if row.TotalCostUSD < wantCost-0.0001 || row.TotalCostUSD > wantCost+0.0001 {
		t.Errorf("expected cost ~%.4f, got %.4f", wantCost, row.TotalCostUSD)
	}
}

func TestClaudeCodeEvents(t *testing.T) {
	h := newMockHandler(sampleEvents)
	w := serve(h, "GET", "/api/claude-code/events?from=2024-06-01&to=2024-06-01")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body struct {
		Data []model.OtelEvent `json:"data"`
		Meta map[string]any    `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 3 {
		t.Fatalf("expected 3 events, got %d", len(body.Data))
	}
	// desc order by default: last event first
	if body.Data[0].EventName != "claude_code.conversation_start" {
		t.Errorf("expected desc order, first event: %s", body.Data[0].EventName)
	}
}

func TestClaudeCodeEventsFilterByEventName(t *testing.T) {
	h := newMockHandler(sampleEvents)
	w := serve(h, "GET", "/api/claude-code/events?from=2024-06-01&to=2024-06-01&eventName=claude_code.api_request")

	var body struct {
		Data []model.OtelEvent `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 api_request events, got %d", len(body.Data))
	}
}

func TestClaudeCodeEventsFilterByUser(t *testing.T) {
	h := newMockHandler(sampleEvents)
	w := serve(h, "GET", "/api/claude-code/events?from=2024-06-01&to=2024-06-01&userEmail=alice@example.com")

	var body struct {
		Data []model.OtelEvent `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 events for alice, got %d", len(body.Data))
	}
}

func TestClaudeCodeEventsLimit(t *testing.T) {
	h := newMockHandler(sampleEvents)
	w := serve(h, "GET", "/api/claude-code/events?from=2024-06-01&to=2024-06-01&limit=1")

	var body struct {
		Data []model.OtelEvent `json:"data"`
		Meta map[string]any    `json:"meta"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("expected 1 event with limit=1, got %d", len(body.Data))
	}
}

func TestClaudeCodeEventsAscOrder(t *testing.T) {
	h := newMockHandler(sampleEvents)
	w := serve(h, "GET", "/api/claude-code/events?from=2024-06-01&to=2024-06-01&order=asc")

	var body struct {
		Data []model.OtelEvent `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.Data) != 3 {
		t.Fatalf("expected 3 events, got %d", len(body.Data))
	}
	if body.Data[0].EventName != "claude_code.api_request" {
		t.Errorf("expected asc order, first event: %s", body.Data[0].EventName)
	}
}

func TestClaudeCodeEventsInvalidLimit(t *testing.T) {
	w := serve(newMockHandler(nil), "GET", "/api/claude-code/events?from=2024-06-01&to=2024-06-01&limit=abc")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
