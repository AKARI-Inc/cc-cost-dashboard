package main

import (
	"log"
	"net/http"
	"os"

	"github.com/narumina/cc-cost-dashboard/internal/api"
)

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	h := &api.Handler{DataDir: dataDir}
	mux := http.NewServeMux()
	h.Register(mux)

	log.Printf("API listening on :%s (data: %s)", port, dataDir)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
