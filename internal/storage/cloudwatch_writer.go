package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
)

// CloudWatchWriter は CloudWatch Logs にイベントを PutLogEvents で書き込む。
// LocalStack と本番 AWS の両方で動作する（AWS_ENDPOINT_URL で切り替え）。
//
// 本番では logGroup 名は "/otel/claude-code" や "/claude-ai/usage" のような
// スラッシュ始まりのパスに揃える。
type CloudWatchWriter struct {
	client *cloudwatchlogs.Client

	mu           sync.Mutex
	streamByGroup map[string]string // logGroup -> logStream (1 プロセス 1 stream でバッチ効率化)
}

// NewCloudWatchWriter は AWS SDK の設定を読んで CloudWatchWriter を生成する。
// AWS_ENDPOINT_URL が設定されていれば LocalStack を指すようになる。
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

// AppendEvent は event を JSON 化して PutLogEvents で CloudWatch Logs に送信する。
// logGroup は "/otel/claude-code" のようなフルパス名を使う。
// logStream は 1 プロセス 1 本で使い回す（ホスト名 + プロセス起動時刻）。
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

// ensureStream は logGroup ごとに 1 本 logStream を確保する。
// 既に作成済みの場合はキャッシュから返す。
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
		// 既存 stream はエラーとしない
		var existsErr *cwltypes.ResourceAlreadyExistsException
		if !errorAs(err, &existsErr) {
			return "", fmt.Errorf("create log stream %s/%s: %w", logGroup, stream, err)
		}
	}

	w.streamByGroup[logGroup] = stream
	return stream, nil
}

// normalizeLogGroup は "otel" のような短縮名を "/otel/claude-code" にマップする。
// 既にスラッシュ始まりならそのまま返す（柔軟性を保つ）。
func normalizeLogGroup(group string) string {
	if group == "otel" {
		return "/otel/claude-code"
	}
	return group
}

// errorAs は errors.As の薄いラッパー（循環依存を避けるため別ファイルに分けない）。
func errorAs(err error, target any) bool {
	for err != nil {
		if t, ok := target.(**cwltypes.ResourceAlreadyExistsException); ok {
			if e, ok := err.(*cwltypes.ResourceAlreadyExistsException); ok {
				*t = e
				return true
			}
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}
