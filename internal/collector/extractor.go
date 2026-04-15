package collector

import (
	"strconv"
	"strings"
	"time"

	"github.com/narumina/cc-cost-dashboard/internal/model"
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
)

// ExtractEvents は ExportLogsServiceRequest を OtelEvent のスライスに変換する。
//
// 方針: よく使われる属性は型付きカラムに昇格させて集計を高速化しつつ、
// 全属性を RawAttributes にマージすることで、再デコードやコード変更なしに
// UI が任意のフィールドを参照できるようにする。
//
// 実際の Claude Code テレメトリの特徴（2026-04 時点で確認）:
//   - イベント名は logRecord.body.stringValue に入る（例: "claude_code.api_request"）。
//     属性 "event.name" にはプレフィクスなし（例: "api_request"）でも入る
//   - セッションキーは "session.id"（ドット区切り）
//   - 数値フィールド（input_tokens, cost_usd, duration_ms）は stringValue として到着する
//   - プロンプト文字数は "prompt_length" に入る
func ExtractEvents(req *collogspb.ExportLogsServiceRequest) []model.OtelEvent {
	if req == nil {
		return nil
	}

	var events []model.OtelEvent

	for _, rl := range req.ResourceLogs {
		// ResourceLogs ごとに一度だけリソース属性をまとめて取得する。
		resAttrs := map[string]*commonpb.AnyValue{}
		if rl.Resource != nil {
			resAttrs = attrMap(rl.Resource.Attributes)
		}

		for _, sl := range rl.ScopeLogs {
			for _, lr := range sl.LogRecords {
				recAttrs := attrMap(lr.Attributes)

				// イベント名: body（"claude_code.api_request"）を優先し、
				// 無い場合は "event.name" 属性（"api_request"）にフォールバックして
				// 標準プレフィクスを前置して正規化する。
				eventName := ""
				if lr.Body != nil {
					eventName = lr.Body.GetStringValue()
				}
				if eventName == "" {
					eventName = strAttr(recAttrs, "event.name")
					if eventName != "" && !strings.Contains(eventName, ".") {
						eventName = "claude_code." + eventName
					}
				}
				if eventName == "" {
					continue
				}

				// resource 属性と log record 属性を JSON として扱いやすい単一の map にマージする。
				raw := make(map[string]any, len(resAttrs)+len(recAttrs))
				for k, v := range resAttrs {
					raw[k] = anyValueToGo(v)
				}
				for k, v := range recAttrs {
					raw[k] = anyValueToGo(v)
				}

				// user.email はどちらのスコープにも出現しうる。record スコープを優先する。
				userEmail := firstNonEmpty(strAttr(recAttrs, "user.email"), strAttr(resAttrs, "user.email"))

				// タイムスタンプ: OTel の timeUnixNano を優先し、なければ event.timestamp を使う。
				ts := nanoToISO(lr.TimeUnixNano)
				if ts == "" {
					ts = strAttr(recAttrs, "event.timestamp")
				}

				ev := model.OtelEvent{
					Timestamp:       ts,
					EventName:       eventName,
					SessionID:       firstNonEmpty(strAttr(recAttrs, "session.id"), strAttr(recAttrs, "session_id")),
					UserEmail:       userEmail,
					UserID:          strAttr(recAttrs, "user.id"),
					UserAccountID:   strAttr(recAttrs, "user.account_id"),
					UserAccountUUID: strAttr(recAttrs, "user.account_uuid"),
					OrganizationID:  strAttr(recAttrs, "organization.id"),
					TerminalType:    strAttr(recAttrs, "terminal.type"),
					ServiceVersion:  strAttr(resAttrs, "service.version"),
					HostArch:        strAttr(resAttrs, "host.arch"),
					OSType:          strAttr(resAttrs, "os.type"),
					OSVersion:       strAttr(resAttrs, "os.version"),
					EventSequence:   numAttr(recAttrs, "event.sequence"),
					PromptID:        strAttr(recAttrs, "prompt.id"),
					RawAttributes:   raw,
				}

				switch eventName {
				case "claude_code.api_request":
					ev.Model = strAttr(recAttrs, "model")
					ev.InputTokens = numAttr(recAttrs, "input_tokens")
					ev.OutputTokens = numAttr(recAttrs, "output_tokens")
					ev.CacheReadTokens = numAttr(recAttrs, "cache_read_tokens")
					ev.CacheCreationTokens = numAttr(recAttrs, "cache_creation_tokens")
					ev.CostUSD = numAttrFloat(recAttrs, "cost_usd")
					ev.DurationMs = numAttr(recAttrs, "duration_ms")
					ev.Speed = strAttr(recAttrs, "speed")
				case "claude_code.user_prompt":
					ev.CharCount = firstNonZero(
						numAttr(recAttrs, "prompt_length"),
						numAttr(recAttrs, "char_count"),
					)
				case "claude_code.tool_decision", "claude_code.tool_result":
					ev.ToolName = strAttr(recAttrs, "tool_name")
					if eventName == "claude_code.tool_result" {
						ev.DurationMs = numAttr(recAttrs, "duration_ms")
					}
				}

				events = append(events, ev)
			}
		}
	}

	return events
}

func attrMap(attrs []*commonpb.KeyValue) map[string]*commonpb.AnyValue {
	m := make(map[string]*commonpb.AnyValue, len(attrs))
	for _, kv := range attrs {
		m[kv.Key] = kv.Value
	}
	return m
}

func strAttr(m map[string]*commonpb.AnyValue, key string) string {
	if v, ok := m[key]; ok {
		return v.GetStringValue()
	}
	return ""
}

// numAttr は、整数フィールドが stringValue として届く Claude Code の慣習に対応する。
func numAttr(m map[string]*commonpb.AnyValue, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if s := v.GetStringValue(); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return int(f)
		}
	}
	if iv := v.GetIntValue(); iv != 0 {
		return int(iv)
	}
	if dv := v.GetDoubleValue(); dv != 0 {
		return int(dv)
	}
	return 0
}

func numAttrFloat(m map[string]*commonpb.AnyValue, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	if s := v.GetStringValue(); s != "" {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	if dv := v.GetDoubleValue(); dv != 0 {
		return dv
	}
	if iv := v.GetIntValue(); iv != 0 {
		return float64(iv)
	}
	return 0
}

// anyValueToGo は OTel の AnyValue を JSON 互換の素朴な Go の値に変換する。
func anyValueToGo(v *commonpb.AnyValue) any {
	if v == nil {
		return nil
	}
	switch x := v.Value.(type) {
	case *commonpb.AnyValue_StringValue:
		return x.StringValue
	case *commonpb.AnyValue_BoolValue:
		return x.BoolValue
	case *commonpb.AnyValue_IntValue:
		return x.IntValue
	case *commonpb.AnyValue_DoubleValue:
		return x.DoubleValue
	case *commonpb.AnyValue_ArrayValue:
		out := make([]any, 0, len(x.ArrayValue.Values))
		for _, e := range x.ArrayValue.Values {
			out = append(out, anyValueToGo(e))
		}
		return out
	case *commonpb.AnyValue_KvlistValue:
		out := make(map[string]any, len(x.KvlistValue.Values))
		for _, kv := range x.KvlistValue.Values {
			out[kv.Key] = anyValueToGo(kv.Value)
		}
		return out
	case *commonpb.AnyValue_BytesValue:
		return x.BytesValue
	}
	return nil
}

func nanoToISO(ns uint64) string {
	if ns == 0 {
		return ""
	}
	t := time.Unix(0, int64(ns))
	return t.UTC().Format(time.RFC3339)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
