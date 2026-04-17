package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/AKARI-Inc/cc-cost-dashboard/internal/api"
	"github.com/AKARI-Inc/cc-cost-dashboard/internal/storage"
)

func main() {
	ctx := context.Background()

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/tmp/data"
	}

	h := &api.Handler{DataDir: dataDir}

	switch s := os.Getenv("STORAGE"); s {
	case "cloudwatch":
		reader, err := storage.NewCloudWatchReader(ctx)
		if err != nil {
			log.Fatalf("init cloudwatch reader: %v", err)
		}
		h.Reader = reader
		log.Println("API Lambda using CloudWatch reader")
	default:
		log.Printf("WARN: STORAGE=%q — falling back to local file reader (DATA_DIR=%s). Set STORAGE=cloudwatch for production.", s, dataDir)
	}

	mux := http.NewServeMux()
	h.Register(mux)

	log.Println("API Lambda starting")
	lambda.Start(httpadapter.NewV2(mux).ProxyWithContext)
}
