package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"mudgame/server/internal/app"
	"mudgame/server/internal/manager"
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

	repo := manager.NewRuntimeManager(dataDir)
	log.Printf("storage backend: file (%s)", dataDir)

	srv := app.NewServer(repo)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("mudgame go server listening on %s", addr)

	httpServer := &http.Server{Addr: addr, Handler: mux}
	errCh := make(chan error, 1)
	go func() {
		errCh <- httpServer.ListenAndServe()
	}()

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var serveErr error

	select {
	case <-sigCtx.Done():
		log.Printf("shutdown signal received")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr = err
			log.Printf("server exited: %v", err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("http shutdown error: %v", err)
	}

	if err := repo.Close(); err != nil {
		log.Printf("final persistence flush failed: %v", err)
	} else {
		log.Printf("dynamic data flushed to disk")
	}

	if serveErr != nil {
		log.Fatalf("server exited with error: %v", serveErr)
	}
}
