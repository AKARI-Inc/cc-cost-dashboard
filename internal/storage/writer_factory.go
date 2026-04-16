package storage

import (
	"context"
	"fmt"
	"os"
)

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

func NewReader(ctx context.Context, dataDir string) (Reader, error) {
	switch os.Getenv("STORAGE") {
	case "", "file":
		return NewFileReader(dataDir), nil
	case "cloudwatch":
		return NewCloudWatchReader(ctx)
	default:
		return nil, fmt.Errorf("unknown STORAGE value: %q", os.Getenv("STORAGE"))
	}
}
