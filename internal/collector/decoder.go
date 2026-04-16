package collector

import (
	collogspb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	"google.golang.org/protobuf/proto"
)

func DecodeLogs(body []byte) (*collogspb.ExportLogsServiceRequest, error) {
	req := &collogspb.ExportLogsServiceRequest{}
	if err := proto.Unmarshal(body, req); err != nil {
		return nil, err
	}
	return req, nil
}
