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

	"extra-badges/internal/badge"
	"extra-badges/internal/config"
	"extra-badges/internal/httpapi"
	"extra-badges/internal/storage/cached"
	"extra-badges/internal/storage/sqlite"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := sqlite.Open(context.Background(), cfg.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var repo badge.Repository = db
	if cfg.CacheTTL > 0 {
		repo = cached.New(db, cfg.CacheTTL)
	}
	svc := badge.NewService(repo)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewRouter(svc, log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Info("server starting", "addr", cfg.Addr, "db", cfg.DBPath, "cache_ttl", cfg.CacheTTL)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
