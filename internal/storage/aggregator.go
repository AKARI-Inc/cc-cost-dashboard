package storage

import (
	"sort"

	"github.com/AKARI-Inc/cc-cost-dashboard/internal/model"
)

// ダッシュボードの日別コスト推移チャートを描画するための集計。
func AggregateByDay(events []model.OtelEvent) []model.DailySummary {
	m := make(map[string]*model.DailySummary)

	for _, ev := range events {
		if ev.EventName != model.APIRequestEvent {
			continue
		}
		date := model.ExtractDate(ev.Timestamp)
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
		if ev.EventName != model.APIRequestEvent {
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
		if ev.EventName != model.APIRequestEvent {
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

// KeySummary は任意のキーで groupBy した汎用集計結果。
type KeySummary struct {
	Key          string  `json:"key"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}

// AggregateByKey は keyFn で抽出したキーごとにイベントを集計する。
func AggregateByKey(events []model.OtelEvent, keyFn func(model.OtelEvent) string) []KeySummary {
	m := make(map[string]*KeySummary)
	for _, ev := range events {
		if ev.EventName != model.APIRequestEvent {
			continue
		}
		k := keyFn(ev)
		s, ok := m[k]
		if !ok {
			s = &KeySummary{Key: k}
			m[k] = s
		}
		s.TotalCostUSD += ev.CostUSD
		s.InputTokens += ev.InputTokens
		s.OutputTokens += ev.OutputTokens
		s.RequestCount++
	}

	result := make([]KeySummary, 0, len(m))
	for _, s := range m {
		result = append(result, *s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCostUSD > result[j].TotalCostUSD
	})
	return result
}

