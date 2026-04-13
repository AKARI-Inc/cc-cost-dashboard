package model

// ClaudeAiEvent は claude.ai の会話エクスポートから抽出した 1 件のメッセージを表す。
type ClaudeAiEvent struct {
	Timestamp         string `json:"timestamp"`
	UserEmail         string `json:"user_email,omitempty"`
	ConversationID    string `json:"conversation_id,omitempty"`
	MessageRole       string `json:"message_role,omitempty"`
	CharCount         int    `json:"char_count,omitempty"`
	EstimatedTokens   int    `json:"estimated_tokens,omitempty"`
	Model             string `json:"model,omitempty"`
	ConversationTitle string `json:"conversation_title,omitempty"`
}
