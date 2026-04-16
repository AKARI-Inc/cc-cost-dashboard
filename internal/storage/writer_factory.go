package storage

import (
	"context"
	"fmt"
	"os"
)

// NewWriter は STORAGE 環境変数に応じて Writer を返す。backendName はログ表示用。
func NewWriter(ctx context.Context, dataDir string) (w Writer, backendName string, err error) {
	switch os.Getenv("STORAGE") {
	case "", "file":
		return NewFileWriter(dataDir), "file", nil
	case "cloudwatch":
		cw, err := NewCloudWatchWriter(ctx)
		return cw, "cloudwatch", err
	default:
		return nil, "", fmt.Errorf("unknown STORAGE value: %q", os.Getenv("STORAGE"))
	}
}
