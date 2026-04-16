package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

const (
	flushMaxEvents   = 500
	flushInterval    = 5 * time.Second
	bufferMaxPerGroup = 10000 // これを超えたら古い方から捨てる
)

// ローカル JSONL を本番 CloudWatch Logs に差し替えるための Writer。
type CloudWatchWriter struct {
	client *cloudwatchlogs.Client

	mu            sync.Mutex
	streamByGroup map[string]string
	bufByGroup    map[string][]cwltypes.InputLogEvent

	stopOnce sync.Once
	done     chan struct{}
}

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

	w := &CloudWatchWriter{
		client:        cloudwatchlogs.NewFromConfig(cfg, opts...),
		streamByGroup: make(map[string]string),
		bufByGroup:    make(map[string][]cwltypes.InputLogEvent),
		done:          make(chan struct{}),
	}
	go w.flushLoop(ctx)
	return w, nil
}

// パイプラインイベントを CloudWatch Logs に永続化するためのメソッド。
func (w *CloudWatchWriter) AppendEvent(ctx context.Context, logGroup string, event any) error {
	logGroupName := normalizeLogGroup(logGroup)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	entry := cwltypes.InputLogEvent{
		Message:   aws.String(string(data)),
		Timestamp: aws.Int64(time.Now().UnixMilli()),
	}

	w.mu.Lock()
	w.bufByGroup[logGroupName] = append(w.bufByGroup[logGroupName], entry)
	shouldFlush := len(w.bufByGroup[logGroupName]) >= flushMaxEvents
	w.mu.Unlock()

	if shouldFlush {
		return w.flushGroup(ctx, logGroupName)
	}
	return nil
}

// graceful shutdown 時にバッファを確実に送りきるためのメソッド。
func (w *CloudWatchWriter) Flush(ctx context.Context) error {
	w.mu.Lock()
	groups := make([]string, 0, len(w.bufByGroup))
	for g := range w.bufByGroup {
		groups = append(groups, g)
	}
	w.mu.Unlock()

	var firstErr error
	for _, g := range groups {
		if err := w.flushGroup(ctx, g); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (w *CloudWatchWriter) Close(ctx context.Context) error {
	w.stopOnce.Do(func() { close(w.done) })
	return w.Flush(ctx)
}

func (w *CloudWatchWriter) flushLoop(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.Flush(ctx); err != nil {
				log.Printf("WARN: background flush: %v", err)
			}
		case <-w.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (w *CloudWatchWriter) flushGroup(ctx context.Context, logGroup string) error {
	w.mu.Lock()
	buf := w.bufByGroup[logGroup]
	if len(buf) == 0 {
		w.mu.Unlock()
		return nil
	}
	w.bufByGroup[logGroup] = nil
	w.mu.Unlock()

	stream, err := w.ensureStream(ctx, logGroup)
	if err != nil {
		w.requeue(logGroup, buf)
		return err
	}

	_, err = w.client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(stream),
		LogEvents:     buf,
	})
	if err != nil {
		w.requeue(logGroup, buf)
		return fmt.Errorf("put log events: %w", err)
	}
	return nil
}

func (w *CloudWatchWriter) requeue(logGroup string, buf []cwltypes.InputLogEvent) {
	w.mu.Lock()
	defer w.mu.Unlock()

	pending := w.bufByGroup[logGroup]
	merged := make([]cwltypes.InputLogEvent, 0, len(buf)+len(pending))
	merged = append(merged, buf...)
	merged = append(merged, pending...)
	if len(merged) > bufferMaxPerGroup {
		dropped := len(merged) - bufferMaxPerGroup
		merged = merged[dropped:]
		log.Printf("WARN: dropped %d oldest event(s) for %s (buffer capped at %d)", dropped, logGroup, bufferMaxPerGroup)
	}
	w.bufByGroup[logGroup] = merged
}

func (w *CloudWatchWriter) ensureStream(ctx context.Context, logGroup string) (string, error) {
	w.mu.Lock()
	if s, ok := w.streamByGroup[logGroup]; ok {
		w.mu.Unlock()
		return s, nil
	}
	w.mu.Unlock()

	host, _ := os.Hostname()
	if host == "" {
		host = "unknown"
	}
	stream := fmt.Sprintf("%s-%d", host, time.Now().Unix())

	// ロック外でネットワーク呼び出し
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

	// double-check: 別ゴルーチンが先に登録していたらそちらを使う
	w.mu.Lock()
	if s, ok := w.streamByGroup[logGroup]; ok {
		w.mu.Unlock()
		return s, nil
	}
	w.streamByGroup[logGroup] = stream
	w.mu.Unlock()
	return stream, nil
}

func normalizeLogGroup(group string) string {
	if group == "otel" {
		return "/otel/claude-code"
	}
	return group
}
