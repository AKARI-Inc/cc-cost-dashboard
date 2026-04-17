package storage

import (
	"sort"

	"github.com/AKARI-Inc/cc-cost-dashboard/internal/model"
)

const apiRequestEvent = "claude_code.api_request"

// ダッシュボードの日別コスト推移チャートを描画するための集計。
func AggregateByDay(events []model.OtelEvent) []model.DailySummary {
	m := make(map[string]*model.DailySummary)

	for _, ev := range events {
		if ev.EventName != apiRequestEvent {
			continue
		}
		date := extractDate(ev.Timestamp)
		s, ok := m[date]
		if !ok {
			s = &model.DailySummary{Date: date}
			m[date] = s
		}
		s.TotalCostUSD += ev.CostUSD
		s.InputTokens += ev.InputTokens
		s.OutputTokens += ev.OutputTokens
		s.RequestCount++
	}

	result := make([]model.DailySummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date < result[j].Date
	})

	return result
}

// ダッシュボードのモデル別利用比率チャートを描画するための集計。
func AggregateByModel(events []model.OtelEvent) []model.ModelSummary {
	m := make(map[string]*model.ModelSummary)

	for _, ev := range events {
		if ev.EventName != apiRequestEvent {
			continue
		}
		s, ok := m[ev.Model]
		if !ok {
			s = &model.ModelSummary{Model: ev.Model}
			m[ev.Model] = s
		}
		s.TotalCostUSD += ev.CostUSD
		s.InputTokens += ev.InputTokens
		s.OutputTokens += ev.OutputTokens
		s.RequestCount++
	}

	result := make([]model.ModelSummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCostUSD > result[j].TotalCostUSD
	})

	return result
}

// ダッシュボードのユーザー別消費量テーブルを描画するための集計。
func AggregateByUser(events []model.OtelEvent) []model.UserSummary {
	m := make(map[string]*model.UserSummary)

	for _, ev := range events {
		if ev.EventName != apiRequestEvent {
			continue
		}
		s, ok := m[ev.UserEmail]
		if !ok {
			s = &model.UserSummary{UserEmail: ev.UserEmail}
			m[ev.UserEmail] = s
		}
		s.TotalCostUSD += ev.CostUSD
		s.InputTokens += ev.InputTokens
		s.OutputTokens += ev.OutputTokens
		s.RequestCount++
	}

	result := make([]model.UserSummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCostUSD > result[j].TotalCostUSD
	})

	return result
}

func extractDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
