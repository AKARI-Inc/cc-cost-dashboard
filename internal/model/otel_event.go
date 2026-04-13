package model

// OtelEvent は Claude Code の OTel テレメトリから抽出した 1 件の構造化イベントを表す。
//
// 設計方針: 頻繁に使うフィールドは型付きカラムに昇格させて集計を高速化しつつ、
// 生の protobuf を再デコードしなくても UI が追加フィールドを表示できるよう、
// 全属性のペイロードを RawAttributes に保持する。
type OtelEvent struct {
	// イベントの基本情報
	Timestamp string `json:"timestamp"`
	EventName string `json:"event_name"`

	// セッション / ユーザー識別情報
	SessionID       string `json:"session_id,omitempty"`
	UserEmail       string `json:"user_email,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	UserAccountID   string `json:"user_account_id,omitempty"`
	UserAccountUUID string `json:"user_account_uuid,omitempty"`
	OrganizationID  string `json:"organization_id,omitempty"`

	// 実行環境
	TerminalType   string `json:"terminal_type,omitempty"`    // vscode / iterm / ghostty など
	ServiceVersion string `json:"service_version,omitempty"`  // Claude Code のバージョン
	HostArch       string `json:"host_arch,omitempty"`
	OSType         string `json:"os_type,omitempty"`
	OSVersion      string `json:"os_version,omitempty"`

	// api_request 用フィールド
	Model               string  `json:"model,omitempty"`
	InputTokens         int     `json:"input_tokens,omitempty"`
	OutputTokens        int     `json:"output_tokens,omitempty"`
	CacheReadTokens     int     `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int     `json:"cache_creation_tokens,omitempty"`
	CostUSD             float64 `json:"cost_usd,omitempty"`
	DurationMs          int     `json:"duration_ms,omitempty"`
	Speed               string  `json:"speed,omitempty"`   // normal / fast など
	PromptID            string  `json:"prompt_id,omitempty"`

	// user_prompt 用フィールド
	CharCount int `json:"char_count,omitempty"`

	// tool_decision / tool_result 用フィールド
	ToolName string `json:"tool_name,omitempty"`

	// セッション内のイベント順序
	EventSequence int `json:"event_sequence,omitempty"`

	// 全属性のダンプ（resource 属性 + log record 属性をマージしたもの）。
	// 型付きカラムに昇格していないフィールドも UI で表示できるようにする。
	RawAttributes map[string]any `json:"raw_attributes,omitempty"`
}
