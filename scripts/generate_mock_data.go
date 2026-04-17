package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/AKARI-Inc/cc-cost-dashboard/internal/model"
)

// 100 万トークンあたりのモデル料金（入力 / 出力）。
var modelPricing = map[string][2]float64{
	"claude-opus-4-20250514":    {15.0, 75.0},
	"claude-sonnet-4-20250514":  {3.0, 15.0},
	"claude-haiku-4-5-20251001": {0.80, 4.0},
}

// 重み付きのモデル選択: opus 20%、sonnet 60%、haiku 20%。
var modelWeights = []struct {
	name   string
	weight int
}{
	{"claude-sonnet-4-20250514", 60},
	{"claude-opus-4-20250514", 20},
	{"claude-haiku-4-5-20251001", 20},
}

var toolNames = []string{
	"Read", "Edit", "Write", "Bash", "Glob", "Grep",
	"WebSearch", "WebFetch", "NotebookEdit",
}

func pickModel() string {
	r := rand.IntN(100)
	cumulative := 0
	for _, mw := range modelWeights {
		cumulative += mw.weight
		if r < cumulative {
			return mw.name
		}
	}
	return modelWeights[0].name
}

func calcCost(modelName string, inputTokens, outputTokens int) float64 {
	pricing := modelPricing[modelName]
	cost := float64(inputTokens)/1_000_000*pricing[0] + float64(outputTokens)/1_000_000*pricing[1]
	// 浮動小数点誤差を抑えるため小数点以下 8 桁に丸める。
	return float64(int(cost*1e8+0.5)) / 1e8
}

func randBetween(min, max int) int {
	return min + rand.IntN(max-min+1)
}

// appendToDateFile は JSON 化したイベントを data/logs/otel/YYYY-MM-DD.jsonl に追記する。
func appendToDateFile(dataDir string, date time.Time, event any) error {
	dir := filepath.Join(dataDir, "logs", "otel")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	filename := filepath.Join(dir, date.Format("2006-01-02")+".jsonl")

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

func randomTimeInDay(date time.Time) time.Time {
	hour := randBetween(8, 22)   // 稼働時間帯 08:00-22:59
	minute := rand.IntN(60)
	second := rand.IntN(60)
	return time.Date(date.Year(), date.Month(), date.Day(), hour, minute, second, 0, time.UTC)
}

func main() {
	dataDir := "data"
	now := time.Now().UTC()
	users := make([]string, 10)
	for i := range users {
		users[i] = fmt.Sprintf("user%d@example.com", i+1)
	}

	totalEvents := 0

	for dayOffset := 29; dayOffset >= 0; dayOffset-- {
		date := now.AddDate(0, 0, -dayOffset)
		date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		dayStr := date.Format("2006-01-02")
		dayEvents := 0

		for _, userEmail := range users {
			// 1 ユーザー 1 日あたり 10〜30 セッション。
			numSessions := randBetween(10, 30)

			for s := 0; s < numSessions; s++ {
				sessionID := fmt.Sprintf("sess-%s-%s-%04d", userEmail[:5], dayStr, s)

				// 各セッションには複数の api_request イベントが含まれる。
				requestsInSession := randBetween(2, 8)
				for r := 0; r < requestsInSession; r++ {
					modelName := pickModel()
					inputTokens := randBetween(500, 5000)
					outputTokens := randBetween(200, 3000)
					cacheRead := randBetween(0, inputTokens/2)
					cacheCreation := randBetween(0, inputTokens/4)
					cost := calcCost(modelName, inputTokens, outputTokens)
					dur := randBetween(500, 15000)
					ts := randomTimeInDay(date)

					event := model.OtelEvent{
						Timestamp:           ts.Format(time.RFC3339),
						EventName:           "claude_code.api_request",
						SessionID:           sessionID,
						UserEmail:           userEmail,
						Model:               modelName,
						InputTokens:         inputTokens,
						OutputTokens:        outputTokens,
						CacheReadTokens:     cacheRead,
						CacheCreationTokens: cacheCreation,
						CostUSD:             cost,
						DurationMs:          dur,
					}
					if err := appendToDateFile(dataDir, date, event); err != nil {
						fmt.Fprintf(os.Stderr, "ERROR writing api_request: %v\n", err)
						os.Exit(1)
					}
					dayEvents++
				}

				// セッションあたり 1〜3 件の user_prompt イベント。
				numPrompts := randBetween(1, 3)
				for p := 0; p < numPrompts; p++ {
					ts := randomTimeInDay(date)
					event := model.OtelEvent{
						Timestamp: ts.Format(time.RFC3339),
						EventName: "claude_code.user_prompt",
						SessionID: sessionID,
						UserEmail: userEmail,
						CharCount: randBetween(50, 500),
					}
					if err := appendToDateFile(dataDir, date, event); err != nil {
						fmt.Fprintf(os.Stderr, "ERROR writing user_prompt: %v\n", err)
						os.Exit(1)
					}
					dayEvents++
				}

				// 0〜3 ペアの tool_decision と tool_result イベント。
				numTools := randBetween(0, 3)
				for t := 0; t < numTools; t++ {
					toolName := toolNames[rand.IntN(len(toolNames))]
					ts := randomTimeInDay(date)

					decision := model.OtelEvent{
						Timestamp: ts.Format(time.RFC3339),
						EventName: "claude_code.tool_decision",
						SessionID: sessionID,
						UserEmail: userEmail,
						ToolName:  toolName,
					}
					if err := appendToDateFile(dataDir, date, decision); err != nil {
						fmt.Fprintf(os.Stderr, "ERROR writing tool_decision: %v\n", err)
						os.Exit(1)
					}
					dayEvents++

					// tool_decision の直後に tool_result が続く。
					resultTs := ts.Add(time.Duration(randBetween(100, 5000)) * time.Millisecond)
					result := model.OtelEvent{
						Timestamp:  resultTs.Format(time.RFC3339),
						EventName:  "claude_code.tool_result",
						SessionID:  sessionID,
						UserEmail:  userEmail,
						ToolName:   toolName,
						DurationMs: randBetween(50, 3000),
					}
					if err := appendToDateFile(dataDir, date, result); err != nil {
						fmt.Fprintf(os.Stderr, "ERROR writing tool_result: %v\n", err)
						os.Exit(1)
					}
					dayEvents++
				}
			}
		}

		totalEvents += dayEvents
		fmt.Printf("[%s] generated %d events\n", dayStr, dayEvents)
	}

	fmt.Printf("\nDone! Total events: %d\n", totalEvents)
	fmt.Printf("Data written to %s/logs/otel/\n", dataDir)
}
