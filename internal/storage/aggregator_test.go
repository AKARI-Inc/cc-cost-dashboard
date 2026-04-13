package storage

import (
	"math"
	"testing"

	"github.com/narumina/cc-cost-dashboard/internal/model"
)

func TestAggregateByDay_GroupsAndSumsCorrectly(t *testing.T) {
	events := []model.OtelEvent{
		{Timestamp: "2025-06-01T10:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.01, InputTokens: 100, OutputTokens: 50},
		{Timestamp: "2025-06-01T11:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.02, InputTokens: 200, OutputTokens: 100},
		{Timestamp: "2025-06-02T09:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.05, InputTokens: 500, OutputTokens: 250},
	}

	result := AggregateByDay(events)

	if len(result) != 2 {
		t.Fatalf("expected 2 daily summaries, got %d", len(result))
	}

	// 結果は日付の昇順でソートされているはず。
	if result[0].Date != "2025-06-01" {
		t.Errorf("result[0].Date = %q, want %q", result[0].Date, "2025-06-01")
	}
	if result[1].Date != "2025-06-02" {
		t.Errorf("result[1].Date = %q, want %q", result[1].Date, "2025-06-02")
	}

	// 1 日目: 2 リクエスト、コスト 0.03、入力 300、出力 150。
	if result[0].RequestCount != 2 {
		t.Errorf("day1 RequestCount = %d, want 2", result[0].RequestCount)
	}
	if !floatClose(result[0].TotalCostUSD, 0.03) {
		t.Errorf("day1 TotalCostUSD = %f, want 0.03", result[0].TotalCostUSD)
	}
	if result[0].InputTokens != 300 {
		t.Errorf("day1 InputTokens = %d, want 300", result[0].InputTokens)
	}
	if result[0].OutputTokens != 150 {
		t.Errorf("day1 OutputTokens = %d, want 150", result[0].OutputTokens)
	}

	// 2 日目: 1 リクエスト。
	if result[1].RequestCount != 1 {
		t.Errorf("day2 RequestCount = %d, want 1", result[1].RequestCount)
	}
	if !floatClose(result[1].TotalCostUSD, 0.05) {
		t.Errorf("day2 TotalCostUSD = %f, want 0.05", result[1].TotalCostUSD)
	}
}

func TestAggregateByModel_GroupsByModelCorrectly(t *testing.T) {
	events := []model.OtelEvent{
		{Timestamp: "2025-06-01T10:00:00Z", EventName: "claude_code.api_request", Model: "claude-sonnet-4-20250514", CostUSD: 0.01, InputTokens: 100, OutputTokens: 50},
		{Timestamp: "2025-06-01T11:00:00Z", EventName: "claude_code.api_request", Model: "claude-sonnet-4-20250514", CostUSD: 0.02, InputTokens: 200, OutputTokens: 100},
		{Timestamp: "2025-06-01T12:00:00Z", EventName: "claude_code.api_request", Model: "claude-opus-4-20250514", CostUSD: 0.10, InputTokens: 500, OutputTokens: 250},
	}

	result := AggregateByModel(events)

	if len(result) != 2 {
		t.Fatalf("expected 2 model summaries, got %d", len(result))
	}

	// コストの降順でソートされるため、opus が先頭に来るはず。
	if result[0].Model != "claude-opus-4-20250514" {
		t.Errorf("result[0].Model = %q, want opus", result[0].Model)
	}
	if result[0].RequestCount != 1 {
		t.Errorf("opus RequestCount = %d, want 1", result[0].RequestCount)
	}
	if !floatClose(result[0].TotalCostUSD, 0.10) {
		t.Errorf("opus TotalCostUSD = %f, want 0.10", result[0].TotalCostUSD)
	}

	if result[1].Model != "claude-sonnet-4-20250514" {
		t.Errorf("result[1].Model = %q, want sonnet", result[1].Model)
	}
	if result[1].RequestCount != 2 {
		t.Errorf("sonnet RequestCount = %d, want 2", result[1].RequestCount)
	}
}

func TestAggregateByUser_GroupsByUserCorrectly(t *testing.T) {
	events := []model.OtelEvent{
		{Timestamp: "2025-06-01T10:00:00Z", EventName: "claude_code.api_request", UserEmail: "alice@example.com", CostUSD: 0.05, InputTokens: 100, OutputTokens: 50},
		{Timestamp: "2025-06-01T11:00:00Z", EventName: "claude_code.api_request", UserEmail: "bob@example.com", CostUSD: 0.10, InputTokens: 200, OutputTokens: 100},
		{Timestamp: "2025-06-01T12:00:00Z", EventName: "claude_code.api_request", UserEmail: "alice@example.com", CostUSD: 0.03, InputTokens: 150, OutputTokens: 75},
	}

	result := AggregateByUser(events)

	if len(result) != 2 {
		t.Fatalf("expected 2 user summaries, got %d", len(result))
	}

	// コストの降順でソートされる。Bob が 0.10、Alice が 0.08。
	if result[0].UserEmail != "bob@example.com" {
		t.Errorf("result[0].UserEmail = %q, want bob", result[0].UserEmail)
	}
	if !floatClose(result[0].TotalCostUSD, 0.10) {
		t.Errorf("bob TotalCostUSD = %f, want 0.10", result[0].TotalCostUSD)
	}

	if result[1].UserEmail != "alice@example.com" {
		t.Errorf("result[1].UserEmail = %q, want alice", result[1].UserEmail)
	}
	if !floatClose(result[1].TotalCostUSD, 0.08) {
		t.Errorf("alice TotalCostUSD = %f, want 0.08", result[1].TotalCostUSD)
	}
	if result[1].RequestCount != 2 {
		t.Errorf("alice RequestCount = %d, want 2", result[1].RequestCount)
	}
	if result[1].InputTokens != 250 {
		t.Errorf("alice InputTokens = %d, want 250", result[1].InputTokens)
	}
}

func TestAggregateByDay_OnlyCountsApiRequestEvents(t *testing.T) {
	events := []model.OtelEvent{
		{Timestamp: "2025-06-01T10:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.05, InputTokens: 100},
		{Timestamp: "2025-06-01T11:00:00Z", EventName: "claude_code.tool_use", CostUSD: 0.00, InputTokens: 50},
		{Timestamp: "2025-06-01T12:00:00Z", EventName: "claude_code.conversation", CostUSD: 0.00, InputTokens: 30},
		{Timestamp: "2025-06-01T13:00:00Z", EventName: "claude_code.api_request", CostUSD: 0.03, InputTokens: 200},
	}

	byDay := AggregateByDay(events)
	if len(byDay) != 1 {
		t.Fatalf("expected 1 daily summary, got %d", len(byDay))
	}
	if byDay[0].RequestCount != 2 {
		t.Errorf("RequestCount = %d, want 2 (only api_request events)", byDay[0].RequestCount)
	}
	if byDay[0].InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300 (only api_request events)", byDay[0].InputTokens)
	}

	byModel := AggregateByModel(events)
	totalRequests := 0
	for _, m := range byModel {
		totalRequests += m.RequestCount
	}
	if totalRequests != 2 {
		t.Errorf("AggregateByModel total RequestCount = %d, want 2", totalRequests)
	}

	byUser := AggregateByUser(events)
	totalRequests = 0
	for _, u := range byUser {
		totalRequests += u.RequestCount
	}
	if totalRequests != 2 {
		t.Errorf("AggregateByUser total RequestCount = %d, want 2", totalRequests)
	}
}

func floatClose(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
