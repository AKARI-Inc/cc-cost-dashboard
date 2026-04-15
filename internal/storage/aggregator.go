package storage

import (
	"sort"

	"github.com/narumina/cc-cost-dashboard/internal/model"
)

const apiRequestEvent = "claude_code.api_request"

// AggregateByDay は OtelEvent を日付単位でグループ化し、コスト・トークン数・
// リクエスト数を合算する。event_name が "claude_code.api_request" のイベントのみ
// 集計対象となる。結果は日付の昇順でソートされる。
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

// AggregateByModel は OtelEvent をモデル単位でグループ化し、コスト・トークン数・
// リクエスト数を合算する。api_request イベントのみ集計対象となる。
// 結果は total_cost_usd の降順でソートされる。
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

// AggregateByUser は OtelEvent を user_email 単位でグループ化し、コスト・
// トークン数・リクエスト数を合算する。api_request イベントのみ集計対象となる。
// 結果は total_cost_usd の降順でソートされる。
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

// extractDate はタイムスタンプ文字列の先頭 10 文字（YYYY-MM-DD）を返す。
// タイムスタンプが 10 文字未満の場合はそのまま全体を返す。
func extractDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
