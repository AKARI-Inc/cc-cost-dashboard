package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/model"
)

// ReadOtelEvents は otel ログ グループ配下の JSONL ファイルから、
// [from, to] の範囲（両端含む）に該当するすべての OtelEvent を読み込む。
// 存在しないファイルは無言でスキップし、パースに失敗した行もスキップする。
// 戻り値のスライスはタイムスタンプ昇順でソートされる。
func ReadOtelEvents(dataDir string, from, to time.Time) ([]model.OtelEvent, error) {
	var events []model.OtelEvent

	for d := truncateToDay(from); !d.After(truncateToDay(to)); d = d.AddDate(0, 0, 1) {
		filename := filepath.Join(dataDir, "logs", "otel", d.Format("2006-01-02")+".jsonl")

		lines, err := readLines(filename)
		if err != nil {
			continue // ファイルが存在しない、あるいはオープンできない場合
		}

		for _, line := range lines {
			var ev model.OtelEvent
			if json.Unmarshal(line, &ev) == nil {
				events = append(events, ev)
			}
		}
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp < events[j].Timestamp
	})

	return events, nil
}

// truncateToDay は t を UTC におけるその日の 00:00 に丸めて返す。
func truncateToDay(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// readLines はファイルから空行以外のすべての行を読み込む。
// ファイルを開けない場合はエラーを返す。ファイルが存在しない場合の
// os.ErrNotExist は想定内のケース。
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
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		// scanner はバッファを使い回すためバイト列をコピーしておく。
		line := make([]byte, len(b))
		copy(line, b)
		lines = append(lines, line)
	}

	return lines, scanner.Err()
}
