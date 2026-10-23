// Command httpserver runs a small JSON REST API for managing tasks.
//
// Usage:
//
//	go run . [-addr :8080] [-static ./static]
//
// The address can also be set with the PORT environment variable.
// See README.md for the full API reference.
package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"httpserver/internal/handler"
	"httpserver/internal/middleware"
	"httpserver/internal/store"
)

func main() {
	defaultAddr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		defaultAddr = ":" + p
	}
	addr := flag.String("addr", defaultAddr, "address to listen on (host:port)")
	staticDir := flag.String("static", "static", "directory of static files served at / (empty to disable)")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Build the app: store -> handlers -> middleware.
	h := handler.New(store.New())
	app := middleware.Chain(
		h.Routes(*staticDir),
		middleware.Recover(log),
		middleware.Logger(log),
	)

	// Timeouts protect the server from slow or stuck clients (e.g. Slowloris).
	srv := &http.Server{
		Addr:              *addr,
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start the server in a goroutine so main can wait for a shutdown signal.
	go func() {
		log.Info("server starting", "addr", *addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown: on Ctrl+C or SIGTERM, stop accepting new connections
	// and give in-flight requests up to 10 seconds to finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}
	log.Info("server stopped")
}
