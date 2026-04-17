package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/AKARI-Inc/cc-cost-dashboard/internal/model"
)

// ReadOptions はサーバーサイドのフィルタリングとリミットを制御する。
// nil を渡した場合はフィルタなし・リミットなしとして扱う。
type ReadOptions struct {
	EventName string
	UserEmail string
	Limit     int // 0 = 制限なし
}

type Reader interface {
	ReadOtelEvents(ctx context.Context, from, to time.Time, opts *ReadOptions) ([]model.OtelEvent, error)
}

// ローカル JSONL からイベントを読み取るための Reader。
type FileReader struct {
	DataDir string
}

func NewFileReader(dataDir string) *FileReader {
	return &FileReader{DataDir: dataDir}
}

func (r *FileReader) ReadOtelEvents(_ context.Context, from, to time.Time, opts *ReadOptions) ([]model.OtelEvent, error) {
	var events []model.OtelEvent

	for d := truncateToDay(from); !d.After(truncateToDay(to)); d = d.AddDate(0, 0, 1) {
		filename := filepath.Join(r.DataDir, "logs", "otel", d.Format("2006-01-02")+".jsonl")

		lines, err := readLines(filename)
		if err != nil {
			continue
		}

		for _, line := range lines {
			var ev model.OtelEvent
			if json.Unmarshal(line, &ev) == nil {
				if opts != nil {
					if opts.EventName != "" && ev.EventName != opts.EventName {
						continue
					}
					if opts.UserEmail != "" && ev.UserEmail != opts.UserEmail {
						continue
					}
				}
				events = append(events, ev)
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp < events[j].Timestamp
	})

	if opts != nil && opts.Limit > 0 && len(events) > opts.Limit {
		events = events[len(events)-opts.Limit:]
	}

	return events, nil
}

func truncateToDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func readLines(filename string) ([][]byte, error) {
	f, err := os.Open(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		return nil, err
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20) // RawAttributes が大きい行に対応
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		line := make([]byte, len(b))
		copy(line, b)
		lines = append(lines, line)
	}

	return lines, scanner.Err()
}
