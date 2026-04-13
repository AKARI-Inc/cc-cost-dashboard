package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// AppendEvent は event を JSON にマーシャルし、
// {dataDir}/logs/{logGroup}/YYYY-MM-DD.jsonl という当日分の JSONL ファイルへ
// 1 行として追記する。
func AppendEvent(dataDir string, logGroup string, event any) error {
	now := time.Now().UTC()
	dir := filepath.Join(dataDir, "logs", logGroup)
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
