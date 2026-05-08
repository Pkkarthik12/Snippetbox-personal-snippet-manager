package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/config"
	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/handlers"
	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/models"
	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/render"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := models.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	if err := store.Ping(ctx); err != nil {
		slog.Error("ping database", "error", err)
		os.Exit(1)
	}

	renderer, err := render.New("templates")
	if err != nil {
		slog.Error("parse templates", "error", err)
		os.Exit(1)
	}

	app := handlers.New(handlers.Dependencies{
		Config:   cfg,
		Store:    store,
		Renderer: renderer,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           app.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("snippetbox listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}
