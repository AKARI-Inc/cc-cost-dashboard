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

// Writer は 1 件のイベントを指定の logGroup に書き込む抽象。
// ローカル開発では FileWriter（JSONL 追記）、
// 本番 / LocalStack では CloudWatchWriter（PutLogEvents）を使う。
type Writer interface {
	AppendEvent(ctx context.Context, logGroup string, event any) error
}

// ---------- FileWriter ----------

// FileWriter は {DataDir}/logs/{logGroup}/YYYY-MM-DD.jsonl に 1 行として追記する。
// ローカル開発のデフォルト実装。JSONL ファイルは jq や cat で直接デバッグできる。
type FileWriter struct {
	DataDir string
}

// NewFileWriter はローカルファイルベースの Writer を生成する。
func NewFileWriter(dataDir string) *FileWriter {
	return &FileWriter{DataDir: dataDir}
}

// AppendEvent は event を JSON にマーシャルし当日分の JSONL に追記する。
// syscall.Flock でファイルロックを取り、同時書き込みから保護する。
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

// ---------- 後方互換エイリアス ----------

// AppendEvent はパッケージレベルの旧インタフェース。
// 既存コードとテストを壊さないために FileWriter を介して実装する。
// 新規コードは Writer インタフェースを使うこと。
func AppendEvent(dataDir string, logGroup string, event any) error {
	return NewFileWriter(dataDir).AppendEvent(context.Background(), logGroup, event)
}
