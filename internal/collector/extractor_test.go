package collector

import (
	"testing"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	commonpb "go.opentelemetry.io/proto/otlp/common/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	resourcepb "go.opentelemetry.io/proto/otlp/resource/v1"
)

func TestExtractEvents_Nil(t *testing.T) {
	events := ExtractEvents(nil)
	if events != nil {
		t.Fatalf("expected nil, got %v", events)
	}
}

func TestExtractEvents_Empty(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{}
	events := ExtractEvents(req)
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestExtractEvents_APIRequest(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{
			{
				Resource: &resourcepb.Resource{
					Attributes: []*commonpb.KeyValue{
						strKV("user.email", "test@example.com"),
					},
				},
				ScopeLogs: []*logspb.ScopeLogs{
					{
						LogRecords: []*logspb.LogRecord{
							{
								TimeUnixNano: 1700000000000000000,
								Attributes: []*commonpb.KeyValue{
									strKV("event.name", "claude_code.api_request"),
									strKV("session_id", "sess-123"),
									strKV("model", "claude-sonnet-4-20250514"),
									intKV("input_tokens", 100),
									intKV("output_tokens", 50),
									doubleKV("cost_usd", 0.005),
									intKV("duration_ms", 1200),
									intKV("cache_read_tokens", 10),
									intKV("cache_creation_tokens", 5),
								},
							},
						},
					},
				},
			},
		},
	}

	events := ExtractEvents(req)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	ev := events[0]
	if ev.UserEmail != "test@example.com" {
		t.Errorf("user_email = %q, want %q", ev.UserEmail, "test@example.com")
	}
	if ev.EventName != "claude_code.api_request" {
		t.Errorf("event_name = %q", ev.EventName)
	}
	if ev.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q", ev.Model)
	}
	if ev.InputTokens != 100 {
		t.Errorf("input_tokens = %d", ev.InputTokens)
	}
	if ev.OutputTokens != 50 {
		t.Errorf("output_tokens = %d", ev.OutputTokens)
	}
	if ev.CostUSD != 0.005 {
		t.Errorf("cost_usd = %f", ev.CostUSD)
	}
	if ev.CacheReadTokens != 10 {
		t.Errorf("cache_read_tokens = %d", ev.CacheReadTokens)
	}
	if ev.CacheCreationTokens != 5 {
		t.Errorf("cache_creation_tokens = %d", ev.CacheCreationTokens)
	}
}

func TestExtractEvents_UserPrompt(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{
			{
				ScopeLogs: []*logspb.ScopeLogs{
					{
						LogRecords: []*logspb.LogRecord{
							{
								Attributes: []*commonpb.KeyValue{
									strKV("event.name", "claude_code.user_prompt"),
									intKV("char_count", 42),
								},
							},
						},
					},
				},
			},
		},
	}

	events := ExtractEvents(req)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].CharCount != 42 {
		t.Errorf("char_count = %d, want 42", events[0].CharCount)
	}
}

func TestExtractEvents_ToolDecision(t *testing.T) {
	req := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{
			{
				ScopeLogs: []*logspb.ScopeLogs{
					{
						LogRecords: []*logspb.LogRecord{
							{
								Attributes: []*commonpb.KeyValue{
									strKV("event.name", "claude_code.tool_decision"),
									strKV("tool_name", "bash"),
								},
							},
						},
					},
				},
			},
		},
	}

	events := ExtractEvents(req)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].ToolName != "bash" {
		t.Errorf("tool_name = %q, want %q", events[0].ToolName, "bash")
	}
}

// テスト用属性を組み立てるためのヘルパー関数群。

func strKV(key, val string) *commonpb.KeyValue {
	return &commonpb.KeyValue{
		Key:   key,
		Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_StringValue{StringValue: val}},
	}
}

func intKV(key string, val int64) *commonpb.KeyValue {
	return &commonpb.KeyValue{
		Key:   key,
		Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_IntValue{IntValue: val}},
	}
}

func doubleKV(key string, val float64) *commonpb.KeyValue {
	return &commonpb.KeyValue{
		Key:   key,
		Value: &commonpb.AnyValue{Value: &commonpb.AnyValue_DoubleValue{DoubleValue: val}},
	}
}
