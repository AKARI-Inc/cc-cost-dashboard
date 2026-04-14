package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/narumina/cc-cost-dashboard/internal/storage"
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

func handler(ctx context.Context) error {
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)

	log.Printf("Generating dashboard data: %s → %s", from.Format("2006-01-02"), now.Format("2006-01-02"))

	// CloudWatch Logs からイベントを読み取る
	events, err := reader.ReadOtelEvents(ctx, from, now)
	if err != nil {
		return fmt.Errorf("read otel events: %w", err)
	}
	log.Printf("Read %d events from CloudWatch Logs", len(events))

	// 集計
	daily := storage.AggregateByDay(events)
	byModel := storage.AggregateByModel(events)
	byUser := storage.AggregateByUser(events)

	// S3 に書き出し
	summaries := map[string]any{
		"data/summary/daily.json": map[string]any{
			"data":       daily,
			"meta":       meta(from, now, "day"),
			"generated":  now.Format(time.RFC3339),
			"eventCount": len(events),
		},
		"data/summary/by-model.json": map[string]any{
			"data":       byModel,
			"meta":       meta(from, now, "model"),
			"generated":  now.Format(time.RFC3339),
			"eventCount": len(events),
		},
		"data/summary/by-user.json": map[string]any{
			"data":       byUser,
			"meta":       meta(from, now, "user"),
			"generated":  now.Format(time.RFC3339),
			"eventCount": len(events),
		},
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
		CacheControl: aws.String("public, max-age=300"),
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
