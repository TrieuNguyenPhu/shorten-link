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

	"shortlink/internal/adapters/dynamostore"
	"shortlink/internal/adapters/httpapi"
	"shortlink/internal/application"
	"shortlink/internal/localconfig"
)

func run() error {
	cfg, err := localconfig.Load()
	if err != nil {
		return err
	}
	repo := dynamostore.New(cfg.Client(), cfg.Table)
	server := &http.Server{Addr: cfg.Address, Handler: httpapi.New(application.New(repo), cfg.Origin), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			slog.Error("shutdown API", "error", err)
		}
	}()
	slog.Info("single-user local API listening", "address", cfg.Address, "webOrigin", cfg.Origin)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func main() {
	if err := run(); err != nil {
		slog.Error("API stopped", "error", err)
		os.Exit(1)
	}
}
