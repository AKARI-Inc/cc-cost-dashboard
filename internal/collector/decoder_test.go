package collector

import (
	"testing"

	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	logspb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/proto"
)

func TestDecodeLogs_EmptyBody(t *testing.T) {
	// 空のボディは空のリクエストとしてデコードされるべき（protobuf はこれを許容する）。
	req, err := DecodeLogs([]byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.ResourceLogs) != 0 {
		t.Fatalf("expected 0 resource logs, got %d", len(req.ResourceLogs))
	}
}

func TestDecodeLogs_NilBody(t *testing.T) {
	req, err := DecodeLogs(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.ResourceLogs) != 0 {
		t.Fatalf("expected 0 resource logs, got %d", len(req.ResourceLogs))
	}
}

func TestDecodeLogs_InvalidBody(t *testing.T) {
	_, err := DecodeLogs([]byte("not valid protobuf"))
	if err == nil {
		t.Fatal("expected error for invalid protobuf")
	}
}

func TestDecodeLogs_ValidRequest(t *testing.T) {
	orig := &collogspb.ExportLogsServiceRequest{
		ResourceLogs: []*logspb.ResourceLogs{{}},
	}
	data, err := proto.Marshal(orig)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	req, err := DecodeLogs(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(req.ResourceLogs) != 1 {
		t.Fatalf("expected 1 resource log, got %d", len(req.ResourceLogs))
	}
}
