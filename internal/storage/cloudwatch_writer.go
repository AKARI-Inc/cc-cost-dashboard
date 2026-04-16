package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// ローカル JSONL を本番 CloudWatch Logs に差し替えるための Writer。
type CloudWatchWriter struct {
	client *cloudwatchlogs.Client

	mu            sync.Mutex
	streamByGroup map[string]string
}

// CloudWatch Logs クライアントを構築するためのコンストラクタ。
func NewCloudWatchWriter(ctx context.Context) (*CloudWatchWriter, error) {
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

	return &CloudWatchWriter{
		client:        cloudwatchlogs.NewFromConfig(cfg, opts...),
		streamByGroup: make(map[string]string),
	}, nil
}

// パイプラインイベントを JSON 化して CloudWatch Logs に永続化するためのメソッド。
func (w *CloudWatchWriter) AppendEvent(ctx context.Context, logGroup string, event any) error {
	logGroupName := normalizeLogGroup(logGroup)

	stream, err := w.ensureStream(ctx, logGroupName)
	if err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	_, err = w.client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(stream),
		LogEvents: []cwltypes.InputLogEvent{{
			Message:   aws.String(string(data)),
			Timestamp: aws.Int64(time.Now().UnixMilli()),
		}},
	})
	if err != nil {
		return fmt.Errorf("put log events: %w", err)
	}
	return nil
}

func (w *CloudWatchWriter) ensureStream(ctx context.Context, logGroup string) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if s, ok := w.streamByGroup[logGroup]; ok {
		return s, nil
	}

	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	stream := fmt.Sprintf("%s-%d", host, time.Now().Unix())

	_, err := w.client.CreateLogStream(ctx, &cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(stream),
	})
	if err != nil {
		var existsErr *cwltypes.ResourceAlreadyExistsException
		if !errors.As(err, &existsErr) {
			return "", fmt.Errorf("create log stream %s/%s: %w", logGroup, stream, err)
		}
	}

	w.streamByGroup[logGroup] = stream
	return stream, nil
}

func normalizeLogGroup(group string) string {
	if group == "otel" {
		return "/otel/claude-code"
	}
	return group
}

