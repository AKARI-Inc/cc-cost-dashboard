package storage

// CloudWatch Logs の log group 名。writer / reader で共有するための定数。
const (
	LogGroupOtel     = "/otel/claude-code"
	LogGroupClaudeAI = "/claude-ai/usage"
)

// 短縮名 → 正式 log group 名のマッピング。
var logGroupAliases = map[string]string{
	"otel":      LogGroupOtel,
	"claude-ai": LogGroupClaudeAI,
}

func resolveLogGroup(name string) string {
	if g, ok := logGroupAliases[name]; ok {
		return g
	}
	return name
}
