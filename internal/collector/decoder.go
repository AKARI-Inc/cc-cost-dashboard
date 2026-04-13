package collector

import (
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"
)

// DecodeLogs は渡されたバイト列を protobuf としてデコードし、
// ExportLogsServiceRequest を返す。
func DecodeLogs(body []byte) (*collogspb.ExportLogsServiceRequest, error) {
	req := &collogspb.ExportLogsServiceRequest{}
	if err := proto.Unmarshal(body, req); err != nil {
		return nil, err
	}
	return req, nil
}
