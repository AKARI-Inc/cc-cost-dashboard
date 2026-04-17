package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/narumina/cc-cost-dashboard/internal/model"
)

// ローカル JSONL を本番 CloudWatch Logs に差し替えるための Reader。
type CloudWatchReader struct {
	client *cloudwatchlogs.Client
}

func NewCloudWatchReader(ctx context.Context) (*CloudWatchReader, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	var opts []func(*cloudwatchlogs.Options)
	if ep := os.Getenv("AWS_ENDPOINT_URL"); ep != "" {
		opts = append(opts, func(o *cloudwatchlogs.Options) {
			o.BaseEndpoint = aws.String(ep)
		})
	}

	return &CloudWatchReader{
		client: cloudwatchlogs.NewFromConfig(cfg, opts...),
	}, nil
}

func (r *CloudWatchReader) ReadOtelEvents(ctx context.Context, from, to time.Time, opts *ReadOptions) ([]model.OtelEvent, error) {
	logGroup := LogGroupOtel
	var events []model.OtelEvent

	var nextToken *string
	for {
		input := &cloudwatchlogs.FilterLogEventsInput{
			LogGroupName: aws.String(logGroup),
			StartTime:    aws.Int64(from.UnixMilli()),
			EndTime:      aws.Int64(to.UnixMilli()),
			NextToken:    nextToken,
		}

		if fp := buildFilterPattern(opts); fp != "" {
			input.FilterPattern = aws.String(fp)
		}

		out, err := r.client.FilterLogEvents(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("filter log events from %s: %w", logGroup, err)
		}

		for _, ev := range out.Events {
			if ev.Message == nil {
				continue
			}
			var otelEv model.OtelEvent
			if err := json.Unmarshal([]byte(*ev.Message), &otelEv); err != nil {
				log.Printf("WARN: skip unparseable event: %v", err)
				continue
			}
			events = append(events, otelEv)
		}

		nextToken = out.NextToken
		if nextToken == nil {
			break
		}
		if opts != nil && opts.Limit > 0 && len(events) >= opts.Limit {
			break
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp < events[j].Timestamp
	})

	if opts != nil && opts.Limit > 0 && len(events) > opts.Limit {
		events = events[len(events)-opts.Limit:]
	}

	return events, nil
}

// buildFilterPattern は CloudWatch Logs の JSON フィルタパターンを組み立てる。
// https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html
func buildFilterPattern(opts *ReadOptions) string {
	if opts == nil {
		return ""
	}
	var clauses []string
	if opts.EventName != "" {
		clauses = append(clauses, fmt.Sprintf(`$.event_name = %q`, opts.EventName))
	}
	if opts.UserEmail != "" {
		clauses = append(clauses, fmt.Sprintf(`$.user_email = %q`, opts.UserEmail))
	}
	if len(clauses) == 0 {
		return ""
	}
	return "{ " + strings.Join(clauses, " && ") + " }"
}
