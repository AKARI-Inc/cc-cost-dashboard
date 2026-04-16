package main

import (
	"context"
	"encoding/base64"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/narumina/cc-cost-dashboard/internal/collector"
	"github.com/narumina/cc-cost-dashboard/internal/storage"
)

var writer storage.Writer

func init() {
	ctx := context.Background()
	var err error
	writer, err = storage.NewCloudWatchWriter(ctx)
	if err != nil {
		log.Fatalf("init cloudwatch writer: %v", err)
	}
}

func handler(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	var body []byte
	if req.IsBase64Encoded {
		var err error
		body, err = base64.StdEncoding.DecodeString(req.Body)
		if err != nil {
			log.Printf("ERROR: base64 decode: %v", err)
			return respond(400, `{"error":"base64 decode failed"}`), nil
		}
	} else {
		body = []byte(req.Body)
	}

	// POST /v1/traces, /v1/metrics は受理のみ
	path := req.RawPath
	if path == "/v1/traces" || path == "/v1/metrics" {
		return respond(200, `{}`), nil
	}

	decoded, err := collector.DecodeLogs(body)
	if err != nil {
		log.Printf("ERROR: decode logs: %v", err)
		return respond(400, `{"error":"decode failed"}`), nil
	}

	otelEvents := collector.ExtractEvents(decoded)
	log.Printf("Received %d event(s)", len(otelEvents))

	var writeErrors int
	for _, e := range otelEvents {
		if err := writer.AppendEvent(ctx, "otel", e); err != nil {
			log.Printf("ERROR: write event: %v", err)
			writeErrors++
		}
	}
	if writeErrors > 0 {
		log.Printf("WARN: %d/%d event(s) failed to write", writeErrors, len(otelEvents))
	}

	if writeErrors == len(otelEvents) && len(otelEvents) > 0 {
		return respond(500, `{"error":"all events failed to write"}`), nil
	}

	return respond(200, `{}`), nil
}

func respond(code int, body string) events.LambdaFunctionURLResponse {
	return events.LambdaFunctionURLResponse{
		StatusCode: code,
		Body:       body,
		Headers:    map[string]string{"Content-Type": "application/json"},
	}
}

func main() {
	lambda.Start(handler)
}
