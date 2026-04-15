package model

// DailySummary は 1 日分の集計メトリクスを保持する。
type DailySummary struct {
	Date         string  `json:"date"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}

// ModelSummary は単一モデル分の集計メトリクスを保持する。
type ModelSummary struct {
	Model        string  `json:"model"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}

// UserSummary は単一ユーザー分の集計メトリクスを保持する。
type UserSummary struct {
	UserEmail    string  `json:"user_email"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	RequestCount int     `json:"request_count"`
}
