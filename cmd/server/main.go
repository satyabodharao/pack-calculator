// Command server is the entrypoint for the Pack Calculator HTTP service.
//
// It performs dependency wiring (composition root): it constructs the in-memory
// repository, the service layer, and the API handlers, then starts an HTTP
// server that exposes both the JSON API and the static web UI. It also handles
// graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/satyabodharao/pack-calculator/internal/api"
	"github.com/satyabodharao/pack-calculator/internal/repository"
	"github.com/satyabodharao/pack-calculator/internal/service"
)

func main() {
	// Structured JSON logging to stdout — friendly for both local dev and the
	// log aggregation used by container platforms such as Heroku.
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	// Configuration via environment variables (12-factor friendly).
	// PORT is provided by Heroku at runtime; default to 8080 locally.
	port := getenv("PORT", "8080")
	// WEB_DIR lets us relocate the static UI in different deployments.
	webDir := getenv("WEB_DIR", "web")

	// Composition root: wire the layers together.
	repo := repository.NewMemoryRepository()
	svc := service.New(repo, logger)
	handler := api.NewHandler(svc, logger)
	router := api.NewRouter(handler, webDir, logger)

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Start the server in a goroutine so main can wait for shutdown signals.
	go func() {
		logger.Info("server starting", "port", port, "web_dir", webDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Block until we receive an interrupt or termination signal.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Graceful shutdown with a timeout so in-flight requests can complete.
	logger.Info("shutdown signal received, draining connections")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped cleanly")
}

// getenv returns the value of the environment variable key, or fallback if unset.
func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
