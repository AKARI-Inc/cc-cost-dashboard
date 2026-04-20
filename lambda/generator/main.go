package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/AKARI-Inc/cc-cost-dashboard/internal/model"
	"github.com/AKARI-Inc/cc-cost-dashboard/internal/storage"
)

const (
	// 集計対象の遡及日数。Frontend は client-side で再フィルタするので
	// 最大の検索期間 (1 年) をカバー。
	lookbackDays = 365

	// Raw Events の出力上限。ペイロードサイズを抑える。
	maxRawEvents = 10000
)

var (
	s3Client *s3.Client
	reader   *storage.CloudWatchReader
	bucket   string
)

func init() {
	ctx := context.Background()

	bucket = os.Getenv("S3_BUCKET")
	if bucket == "" {
		log.Fatal("S3_BUCKET is required")
	}

	var err error
	reader, err = storage.NewCloudWatchReader(ctx)
	if err != nil {
		log.Fatalf("init cloudwatch reader: %v", err)
	}

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("load aws config: %v", err)
	}
	s3Client = s3.NewFromConfig(cfg)
}

// Bucket は (date, key) でグループ化した 1 行のメトリクス。
// frontend は date で範囲フィルタ、key で再集計する。
type Bucket struct {
	Date                string  `json:"date"`
	Key                 string  `json:"key"`
	TotalCostUSD        float64 `json:"total_cost_usd"`
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	CacheReadTokens     int     `json:"cache_read_tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens"`
	RequestCount        int     `json:"request_count"`
}

// UserModelBucket は (date, user_email, model) 単位の明細。
// ユーザー詳細のモデル別内訳に使う。
type UserModelBucket struct {
	Date                string  `json:"date"`
	UserEmail           string  `json:"user_email"`
	Model               string  `json:"model"`
	TotalCostUSD        float64 `json:"total_cost_usd"`
	InputTokens         int     `json:"input_tokens"`
	OutputTokens        int     `json:"output_tokens"`
	CacheReadTokens     int     `json:"cache_read_tokens"`
	CacheCreationTokens int     `json:"cache_creation_tokens"`
	RequestCount        int     `json:"request_count"`
}

// UserToolBucket は (date, user_email, tool_name) 単位の明細。
// tool_decision / tool_result イベントのみが対象。
type UserToolBucket struct {
	Date         string `json:"date"`
	UserEmail    string `json:"user_email"`
	ToolName     string `json:"tool_name"`
	RequestCount int    `json:"request_count"`
}

// UserTerminalBucket は (date, user_email, terminal_type, os_type) 単位の明細。
// ユーザーが普段どの環境で使っているかの内訳に使う。
type UserTerminalBucket struct {
	Date         string  `json:"date"`
	UserEmail    string  `json:"user_email"`
	TerminalType string  `json:"terminal_type"`
	OSType       string  `json:"os_type,omitempty"`
	RequestCount int     `json:"request_count"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}

func handler(ctx context.Context) error {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -lookbackDays)

	log.Printf("Generating dashboard data: %s → %s (%d days)", from.Format("2006-01-02"), now.Format("2006-01-02"), lookbackDays)

	events, err := reader.ReadOtelEvents(ctx, from, now, nil)
	if err != nil {
		return fmt.Errorf("read otel events: %w", err)
	}
	log.Printf("Read %d events from CloudWatch Logs", len(events))

	// per-day × per-key の breakdown を出力。
	// frontend は date で範囲フィルタ → key で再集計する。
	wrap := func(data any, groupBy string) any {
		return map[string]any{
			"data":       data,
			"meta":       meta(from, now, groupBy),
			"generated":  now.Format(time.RFC3339),
			"eventCount": len(events),
		}
	}

	summaries := map[string]any{
		"data/summary/per-day-per-model.json":      wrap(bucketize(events, func(e model.OtelEvent) string { return e.Model }), "model"),
		"data/summary/per-day-per-user.json":       wrap(bucketize(events, func(e model.OtelEvent) string { return e.UserEmail }), "user"),
		"data/summary/per-day-per-terminal.json":   wrap(bucketize(events, func(e model.OtelEvent) string { return e.TerminalType }), "terminal"),
		"data/summary/per-day-per-version.json":    wrap(bucketize(events, func(e model.OtelEvent) string { return e.ServiceVersion }), "version"),
		"data/summary/per-day-per-speed.json":      wrap(bucketize(events, func(e model.OtelEvent) string { return e.Speed }), "speed"),
		"data/summary/per-day-per-user-model.json":    wrap(bucketizeUserModel(events), "user-model"),
		"data/summary/per-day-per-user-tool.json":     wrap(bucketizeUserTool(events), "user-tool"),
		"data/summary/per-day-per-user-terminal.json": wrap(bucketizeUserTerminal(events), "user-terminal"),
	}

	for key, data := range summaries {
		if err := putJSON(ctx, key, data); err != nil {
			log.Printf("ERROR: failed to write %s: %v", key, err)
			return err
		}
		log.Printf("Wrote s3://%s/%s", bucket, key)
	}

	// Raw Events を S3 に書き出し (フロントエンドで直接配信)
	slim := slimEvents(events, maxRawEvents)
	eventsPayload := map[string]any{
		"data": slim,
		"meta": map[string]any{
			"from":  from.Format("2006-01-02"),
			"to":    now.Format("2006-01-02"),
			"count": len(slim),
		},
		"generated": now.Format(time.RFC3339),
	}
	if err := putJSON(ctx, "data/events/recent.json", eventsPayload); err != nil {
		log.Printf("ERROR: failed to write events: %v", err)
		return err
	}
	log.Printf("Wrote s3://%s/data/events/recent.json (%d events)", bucket, len(slim))

	log.Printf("Dashboard generation complete: %d events → %d summaries + events", len(events), len(summaries))
	return nil
}

// bucketize は events を (date, key) でグループ化して合算する。
// api_request 以外のイベントは無視。key が空文字のレコードは "(unknown)" として
// 1 つにまとめる。
func bucketize(events []model.OtelEvent, keyFn func(model.OtelEvent) string) []Bucket {
	type bucketKey struct {
		date string
		key  string
	}
	m := make(map[bucketKey]*Bucket)

	for _, ev := range events {
		if ev.EventName != model.APIRequestEvent {
			continue
		}
		date := model.ExtractDate(ev.Timestamp)
		k := keyFn(ev)
		if k == "" {
			k = "(unknown)"
		}
		bk := bucketKey{date: date, key: k}
		b, ok := m[bk]
		if !ok {
			b = &Bucket{Date: date, Key: k}
			m[bk] = b
		}
		b.TotalCostUSD += ev.CostUSD
		b.InputTokens += ev.InputTokens
		b.OutputTokens += ev.OutputTokens
		b.CacheReadTokens += ev.CacheReadTokens
		b.CacheCreationTokens += ev.CacheCreationTokens
		b.RequestCount++
	}

	result := make([]Bucket, 0, len(m))
	for _, b := range m {
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		return result[i].Key < result[j].Key
	})
	return result
}

// bucketizeUserModel は api_request イベントを (date, user, model) で集計する。
func bucketizeUserModel(events []model.OtelEvent) []UserModelBucket {
	type k struct{ date, user, mdl string }
	m := make(map[k]*UserModelBucket)

	for _, ev := range events {
		if ev.EventName != model.APIRequestEvent {
			continue
		}
		user := ev.UserEmail
		if user == "" {
			user = "(unknown)"
		}
		mdl := ev.Model
		if mdl == "" {
			mdl = "(unknown)"
		}
		key := k{date: model.ExtractDate(ev.Timestamp), user: user, mdl: mdl}
		b, ok := m[key]
		if !ok {
			b = &UserModelBucket{Date: key.date, UserEmail: user, Model: mdl}
			m[key] = b
		}
		b.TotalCostUSD += ev.CostUSD
		b.InputTokens += ev.InputTokens
		b.OutputTokens += ev.OutputTokens
		b.CacheReadTokens += ev.CacheReadTokens
		b.CacheCreationTokens += ev.CacheCreationTokens
		b.RequestCount++
	}

	result := make([]UserModelBucket, 0, len(m))
	for _, b := range m {
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		if result[i].UserEmail != result[j].UserEmail {
			return result[i].UserEmail < result[j].UserEmail
		}
		return result[i].Model < result[j].Model
	})
	return result
}

// bucketizeUserTool は tool_decision / tool_result イベントを (date, user, tool) で集計する。
func bucketizeUserTool(events []model.OtelEvent) []UserToolBucket {
	type k struct{ date, user, tool string }
	m := make(map[k]*UserToolBucket)

	for _, ev := range events {
		if ev.EventName != "claude_code.tool_decision" && ev.EventName != "claude_code.tool_result" {
			continue
		}
		user := ev.UserEmail
		if user == "" {
			user = "(unknown)"
		}
		tool := ev.ToolName
		if tool == "" {
			tool = "(unknown)"
		}
		key := k{date: model.ExtractDate(ev.Timestamp), user: user, tool: tool}
		b, ok := m[key]
		if !ok {
			b = &UserToolBucket{Date: key.date, UserEmail: user, ToolName: tool}
			m[key] = b
		}
		b.RequestCount++
	}

	result := make([]UserToolBucket, 0, len(m))
	for _, b := range m {
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		if result[i].UserEmail != result[j].UserEmail {
			return result[i].UserEmail < result[j].UserEmail
		}
		return result[i].ToolName < result[j].ToolName
	})
	return result
}

// bucketizeUserTerminal は api_request イベントを (date, user, terminal, os) で集計する。
func bucketizeUserTerminal(events []model.OtelEvent) []UserTerminalBucket {
	type k struct{ date, user, term, os string }
	m := make(map[k]*UserTerminalBucket)

	for _, ev := range events {
		if ev.EventName != model.APIRequestEvent {
			continue
		}
		user := ev.UserEmail
		if user == "" {
			user = "(unknown)"
		}
		term := ev.TerminalType
		if term == "" {
			term = "(unknown)"
		}
		key := k{date: model.ExtractDate(ev.Timestamp), user: user, term: term, os: ev.OSType}
		b, ok := m[key]
		if !ok {
			b = &UserTerminalBucket{Date: key.date, UserEmail: user, TerminalType: term, OSType: ev.OSType}
			m[key] = b
		}
		b.RequestCount++
		b.TotalCostUSD += ev.CostUSD
	}

	result := make([]UserTerminalBucket, 0, len(m))
	for _, b := range m {
		result = append(result, *b)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		if result[i].UserEmail != result[j].UserEmail {
			return result[i].UserEmail < result[j].UserEmail
		}
		if result[i].TerminalType != result[j].TerminalType {
			return result[i].TerminalType < result[j].TerminalType
		}
		return result[i].OSType < result[j].OSType
	})
	return result
}

// slimEvents は raw_attributes を除外し、新しい順で上限を適用する。
func slimEvents(events []model.OtelEvent, limit int) []model.OtelEvent {
	out := make([]model.OtelEvent, len(events))
	copy(out, events)

	for i := range out {
		out[i].RawAttributes = nil
	}

	// 新しい順 (timestamp desc)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp > out[j].Timestamp
	})

	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func putJSON(ctx context.Context, key string, data any) error {
	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", key, err)
	}

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(bucket),
		Key:          aws.String(key),
		Body:         bytes.NewReader(body),
		ContentType:  aws.String("application/json"),
		CacheControl: aws.String("public, max-age=60"), // 1 分キャッシュ (5分ごとに generator 走るので余裕)
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", bucket, key, err)
	}
	return nil
}

func meta(from, to time.Time, groupBy string) map[string]string {
	return map[string]string{
		"from":    from.Format("2006-01-02"),
		"to":      to.Format("2006-01-02"),
		"groupBy": groupBy,
	}
}

func main() {
	lambda.Start(handler)
}
