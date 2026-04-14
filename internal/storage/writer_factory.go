package storage

import (
	"context"
	"fmt"
	"os"
)

// NewWriter は環境変数 STORAGE に応じて実装を選択する。
//
//	STORAGE=file       (default) -> FileWriter  : data/logs/{group}/YYYY-MM-DD.jsonl
//	STORAGE=cloudwatch           -> CloudWatchWriter : LocalStack / 本番 AWS
//
// AWS_ENDPOINT_URL が設定されていれば自動的に LocalStack を指す。
func NewWriter(ctx context.Context, dataDir string) (Writer, error) {
	switch os.Getenv("STORAGE") {
	case "", "file":
		return NewFileWriter(dataDir), nil
	case "cloudwatch":
		return NewCloudWatchWriter(ctx)
	default:
		return nil, fmt.Errorf("unknown STORAGE value: %q", os.Getenv("STORAGE"))
	}
}
