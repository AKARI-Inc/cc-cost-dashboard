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
	Date         string  `json:"date"`
	Key          string  `json:"key"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}

func handler(ctx context.Context) error {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -lookbackDays)

	log.Printf("Generating dashboard data: %s → %s (%d days)", from.Format("2006-01-02"), now.Format("2006-01-02"), lookbackDays)

	events, err := reader.ReadOtelEvents(ctx, from, now)
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
		"data/summary/per-day-per-model.json":    wrap(bucketize(events, func(e model.OtelEvent) string { return e.Model }), "model"),
		"data/summary/per-day-per-user.json":     wrap(bucketize(events, func(e model.OtelEvent) string { return e.UserEmail }), "user"),
		"data/summary/per-day-per-terminal.json": wrap(bucketize(events, func(e model.OtelEvent) string { return e.TerminalType }), "terminal"),
		"data/summary/per-day-per-version.json":  wrap(bucketize(events, func(e model.OtelEvent) string { return e.ServiceVersion }), "version"),
		"data/summary/per-day-per-speed.json":    wrap(bucketize(events, func(e model.OtelEvent) string { return e.Speed }), "speed"),
	}

	for key, data := range summaries {
		if err := putJSON(ctx, key, data); err != nil {
			log.Printf("ERROR: failed to write %s: %v", key, err)
			return err
		}
		log.Printf("Wrote s3://%s/%s", bucket, key)
	}

	log.Printf("Dashboard generation complete: %d events → %d summaries", len(events), len(summaries))
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
