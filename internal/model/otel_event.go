package model

const APIRequestEvent = "claude_code.api_request"

// ExtractDate は "2006-01-02T..." 形式のタイムスタンプから日付部分を返す。
func ExtractDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}

// OTel テレメトリから抽出した 1 件のイベント。頻出フィールドは型付きカラムに昇格済み。
type OtelEvent struct {
	Timestamp string `json:"timestamp"`
	EventName string `json:"event_name"`

	SessionID       string `json:"session_id,omitempty"`
	UserEmail       string `json:"user_email,omitempty"`
	UserID          string `json:"user_id,omitempty"`
	UserAccountID   string `json:"user_account_id,omitempty"`
	UserAccountUUID string `json:"user_account_uuid,omitempty"`
	OrganizationID  string `json:"organization_id,omitempty"`

	TerminalType   string `json:"terminal_type,omitempty"`
	ServiceVersion string `json:"service_version,omitempty"`
	HostArch       string `json:"host_arch,omitempty"`
	OSType         string `json:"os_type,omitempty"`
	OSVersion      string `json:"os_version,omitempty"`

	Model               string  `json:"model,omitempty"`
	InputTokens         int     `json:"input_tokens,omitempty"`
	OutputTokens        int     `json:"output_tokens,omitempty"`
	CacheReadTokens     int     `json:"cache_read_tokens,omitempty"`
	CacheCreationTokens int     `json:"cache_creation_tokens,omitempty"`
	CostUSD             float64 `json:"cost_usd,omitempty"`
	DurationMs          int     `json:"duration_ms,omitempty"`
	Speed               string  `json:"speed,omitempty"`
	PromptID            string  `json:"prompt_id,omitempty"`

	CharCount int `json:"char_count,omitempty"`

	ToolName string `json:"tool_name,omitempty"`

	// skill_activated イベント専用。
	SkillName   string `json:"skill_name,omitempty"`
	SkillSource string `json:"skill_source,omitempty"`
	PluginName  string `json:"plugin_name,omitempty"`

	EventSequence int `json:"event_sequence,omitempty"`

	// 型付きカラムに昇格していないフィールドも UI で参照できるようにするためのダンプ。
	RawAttributes map[string]any `json:"raw_attributes,omitempty"`
}
