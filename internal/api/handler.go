package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/model"
	"github.com/narumina/cc-cost-dashboard/internal/storage"
)

// EventReader は OTel イベントを読み取る抽象。
// ローカルでは FileReader (JSONL)、本番では CloudWatchReader を使う。
type EventReader interface {
	ReadOtelEvents(ctx context.Context, from, to time.Time) ([]model.OtelEvent, error)
}

// fileReader は既存の storage.ReadOtelEvents をラップする。
type fileReader struct {
	dataDir string
}

func (r *fileReader) ReadOtelEvents(_ context.Context, from, to time.Time) ([]model.OtelEvent, error) {
	return storage.ReadOtelEvents(r.dataDir, from, to)
}

// Handler はコストダッシュボード API の HTTP ハンドラを提供する。
type Handler struct {
	DataDir string
	Reader  EventReader // nil の場合は DataDir からファイル読み取り
}

func (h *Handler) reader() EventReader {
	if h.Reader != nil {
		return h.Reader
	}
	return &fileReader{dataDir: h.DataDir}
}

// Register は全ルートを指定された ServeMux に登録する。
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/claude-code/usage", h.ClaudeCodeUsage)
	mux.HandleFunc("GET /api/claude-ai/usage", h.ClaudeAiUsage)
	mux.HandleFunc("GET /api/claude-code/events", h.ClaudeCodeEvents)
	mux.HandleFunc("GET /api/health", h.Health)
}

// Health は簡易なヘルスチェックレスポンスを返す。
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	setCORS(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ClaudeCodeUsage は Claude Code の集計済み利用データを返す。
func (h *Handler) ClaudeCodeUsage(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	from, to, err := parseDateRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	groupBy := r.URL.Query().Get("groupBy")
	if groupBy == "" {
		groupBy = "day"
	}

	events, err := h.reader().ReadOtelEvents(r.Context(), from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var data any
	switch groupBy {
	case "day":
		data = storage.AggregateByDay(events)
	case "model":
		data = storage.AggregateByModel(events)
	case "user":
		data = storage.AggregateByUser(events)
	case "terminal":
		data = aggregateOtelByKey(events, func(e model.OtelEvent) string { return e.TerminalType })
	case "version":
		data = aggregateOtelByKey(events, func(e model.OtelEvent) string { return e.ServiceVersion })
	case "speed":
		data = aggregateOtelByKey(events, func(e model.OtelEvent) string { return e.Speed })
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid groupBy: " + groupBy})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]string{
			"from":    from.Format("2006-01-02"),
			"to":      to.Format("2006-01-02"),
			"groupBy": groupBy,
		},
	})
}

// ClaudeAiUsage は Claude AI の集計済み利用データを返す。
func (h *Handler) ClaudeAiUsage(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	from, to, err := parseDateRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	groupBy := r.URL.Query().Get("groupBy")
	if groupBy == "" {
		groupBy = "day"
	}

	events, err := storage.ReadClaudeAiEvents(h.DataDir, from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var data any
	switch groupBy {
	case "day":
		data = aggregateClaudeAiByDay(events)
	case "user":
		data = aggregateClaudeAiByUser(events)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid groupBy: " + groupBy})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]string{
			"from":    from.Format("2006-01-02"),
			"to":      to.Format("2006-01-02"),
			"groupBy": groupBy,
		},
	})
}

// ClaudeCodeEvents は任意のフィルタを適用した生の OtelEvent データを返す。
func (h *Handler) ClaudeCodeEvents(w http.ResponseWriter, r *http.Request) {
	setCORS(w)

	from, to, err := parseDateRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	limit := 100
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		if n > 1000 {
			n = 1000
		}
		limit = n
	}

	eventName := r.URL.Query().Get("eventName")

	events, err := h.reader().ReadOtelEvents(r.Context(), from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var filtered []model.OtelEvent
	for _, ev := range events {
		if eventName != "" && ev.EventName != eventName {
			continue
		}
		filtered = append(filtered, ev)
		if len(filtered) >= limit {
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": filtered,
		"meta": map[string]any{
			"from":      from.Format("2006-01-02"),
			"to":        to.Format("2006-01-02"),
			"eventName": eventName,
			"limit":     limit,
			"count":     len(filtered),
		},
	})
}

// --- ヘルパー ---

func setCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

// jst は from/to 解釈に使うタイムゾーン。
// クライアント (ダッシュボード) は JST 基準の YYYY-MM-DD を送ってくるので、
// API 内では JST → UTC に変換してから CloudWatch に問い合わせる。
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	to := now

	if s := r.URL.Query().Get("from"); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, jst)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from date: %s", s)
		}
		from = t.UTC()
	}
	if s := r.URL.Query().Get("to"); s != "" {
		t, err := time.ParseInLocation("2006-01-02", s, jst)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to date: %s", s)
		}
		// 当日を含めるため 23:59:59.999 まで拡張 (JST)
		to = t.Add(24*time.Hour - time.Nanosecond).UTC()
	}

	return from, to, nil
}

// KeySummary はカスタム groupBy 向けの汎用集計結果。
type KeySummary struct {
	Key          string  `json:"key"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}

func aggregateOtelByKey(events []model.OtelEvent, keyFn func(model.OtelEvent) string) []KeySummary {
	m := make(map[string]*KeySummary)
	for _, ev := range events {
		if ev.EventName != "claude_code.api_request" {
			continue
		}
		k := keyFn(ev)
		s, ok := m[k]
		if !ok {
			s = &KeySummary{Key: k}
			m[k] = s
		}
		s.TotalCostUSD += ev.CostUSD
		s.InputTokens += ev.InputTokens
		s.OutputTokens += ev.OutputTokens
		s.RequestCount++
	}

	result := make([]KeySummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCostUSD > result[j].TotalCostUSD
	})
	return result
}

// ClaudeAiDaySummary は Claude AI イベントの日次集計結果を保持する。
type ClaudeAiDaySummary struct {
	Date            string `json:"date"`
	EstimatedTokens int    `json:"estimated_tokens"`
	MessageCount    int    `json:"message_count"`
}

func aggregateClaudeAiByDay(events []model.ClaudeAiEvent) []ClaudeAiDaySummary {
	m := make(map[string]*ClaudeAiDaySummary)
	for _, ev := range events {
		date := ev.Timestamp
		if len(date) >= 10 {
			date = date[:10]
		}
		s, ok := m[date]
		if !ok {
			s = &ClaudeAiDaySummary{Date: date}
			m[date] = s
		}
		s.EstimatedTokens += ev.EstimatedTokens
		s.MessageCount++
	}

	result := make([]ClaudeAiDaySummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})
	return result
}

// ClaudeAiUserSummary は Claude AI イベントのユーザー別集計結果を保持する。
type ClaudeAiUserSummary struct {
	UserEmail       string `json:"user_email"`
	EstimatedTokens int    `json:"estimated_tokens"`
	MessageCount    int    `json:"message_count"`
}

func aggregateClaudeAiByUser(events []model.ClaudeAiEvent) []ClaudeAiUserSummary {
	m := make(map[string]*ClaudeAiUserSummary)
	for _, ev := range events {
		s, ok := m[ev.UserEmail]
		if !ok {
			s = &ClaudeAiUserSummary{UserEmail: ev.UserEmail}
			m[ev.UserEmail] = s
		}
		s.EstimatedTokens += ev.EstimatedTokens
		s.MessageCount++
	}

	result := make([]ClaudeAiUserSummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].EstimatedTokens > result[j].EstimatedTokens
	})
	return result
}
