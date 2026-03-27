package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"mudgame/server/internal/app"
	"mudgame/server/internal/engine"
	"mudgame/server/internal/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "4009"
	}

	dataDir := os.Getenv("MUD_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join("..", "data", "servers", "server1")
	}

	var repo storage.Repository = storage.NewFileRepository(dataDir)
	log.Printf("storage backend: file (%s)", dataDir)

	svc := engine.NewService(repo)
	srv := app.NewServer(svc)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("mudgame go server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
