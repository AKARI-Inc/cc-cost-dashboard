package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type Writer interface {
	AppendEvent(ctx context.Context, logGroup string, event any) error
	Close(ctx context.Context) error
}

// ローカル開発で JSONL ファイルにイベントを追記するための Writer。
type FileWriter struct {
	DataDir string
}

func NewFileWriter(dataDir string) *FileWriter {
	return &FileWriter{DataDir: dataDir}
}

func (w *FileWriter) AppendEvent(_ context.Context, logGroup string, event any) error {
	now := time.Now().UTC()
	dir := filepath.Join(w.DataDir, "logs", logGroup)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	filename := filepath.Join(dir, now.Format("2006-01-02")+".jsonl")

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	data = append(data, '\n')

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open file %s: %w", filename, err)
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock file %s: %w", filename, err)
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write to %s: %w", filename, err)
	}

	return nil
}

func (w *FileWriter) Close(_ context.Context) error { return nil }

