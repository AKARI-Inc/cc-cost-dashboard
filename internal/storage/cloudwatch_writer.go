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

const (
	flushMaxEvents  = 500           // PutLogEvents の上限は 10,000 だが余裕を持たせる
	flushInterval   = 5 * time.Second
)

// ローカル JSONL を本番 CloudWatch Logs に差し替えるための Writer。
// AppendEventはバッファに溜め、一定件数 or 一定間隔でバッチ送信する。
type CloudWatchWriter struct {
	client *cloudwatchlogs.Client

	mu            sync.Mutex
	streamByGroup map[string]string
	bufByGroup    map[string][]cwltypes.InputLogEvent

	stopOnce sync.Once
	done     chan struct{}
}

// CloudWatch Logs クライアントを構築し、バックグラウンド flush を開始するためのコンストラクタ。
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

// パイプラインイベントを JSON 化してバッファに追加するためのメソッド。
// バッファが flushMaxEvents に達した場合は即座に flush する。
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

// バッファに残っている全イベントを送信する。graceful shutdown 用。
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

// flush ループを停止し、残りバッファを送信する。
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
			_ = w.Flush(ctx)
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
		// 送信失敗分をバッファに戻す
		w.mu.Lock()
		w.bufByGroup[logGroup] = append(buf, w.bufByGroup[logGroup]...)
		w.mu.Unlock()
		return err
	}

	_, err = w.client.PutLogEvents(ctx, &cloudwatchlogs.PutLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		LogStreamName: aws.String(stream),
		LogEvents:     buf,
	})
	if err != nil {
		// 送信失敗分をバッファに戻す
		w.mu.Lock()
		w.bufByGroup[logGroup] = append(buf, w.bufByGroup[logGroup]...)
		w.mu.Unlock()
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
