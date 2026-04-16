package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
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
	mux.HandleFunc("GET /api/claude-code/events", h.ClaudeCodeEvents)
	mux.HandleFunc("GET /api/health", h.Health)
}

// Health は簡易なヘルスチェックレスポンスを返す。
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ClaudeCodeUsage は Claude Code の集計済み利用データを返す。
func (h *Handler) ClaudeCodeUsage(w http.ResponseWriter, r *http.Request) {
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
		data = storage.AggregateByKey(events, func(e model.OtelEvent) string { return e.TerminalType })
	case "version":
		data = storage.AggregateByKey(events, func(e model.OtelEvent) string { return e.ServiceVersion })
	case "speed":
		data = storage.AggregateByKey(events, func(e model.OtelEvent) string { return e.Speed })
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
	from, to, err := parseDateRange(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	limit := 500
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
		if n > 5000 {
			n = 5000
		}
		limit = n
	}

	eventName := r.URL.Query().Get("eventName")
	userEmail := r.URL.Query().Get("userEmail")
	order := r.URL.Query().Get("order")
	if order != "asc" {
		order = "desc"
	}

	events, err := h.reader().ReadOtelEvents(r.Context(), from, to)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if order == "desc" {
		slices.Reverse(events)
	}

	var filtered []model.OtelEvent
	for _, ev := range events {
		if eventName != "" && ev.EventName != eventName {
			continue
		}
		if userEmail != "" && ev.UserEmail != userEmail {
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
			"userEmail": userEmail,
			"order":     order,
			"limit":     limit,
			"count":     len(filtered),
		},
	})
}

// --- ヘルパー ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: encode error: %v", err)
	}
}

// jst は from/to 解釈に使うタイムゾーン。
// クライアント (ダッシュボード) は JST 基準の YYYY-MM-DD を送ってくるので、
// API 内では JST → UTC に変換してから CloudWatch に問い合わせる。
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	nowJST := time.Now().In(jst)
	todayStart := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day(), 0, 0, 0, 0, jst)
	from := todayStart.AddDate(0, 0, -30).UTC()
	to := todayStart.Add(24 * time.Hour).UTC() // exclusive upper bound (翌日 00:00:00)

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
		to = t.Add(24 * time.Hour).UTC() // exclusive upper bound
	}

	return from, to, nil
}


