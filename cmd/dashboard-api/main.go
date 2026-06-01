package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pavillio/pav-edi/internal/api/dashboard"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
)

func main() {
	cfg := platform.LoadConfig()
	ctx := context.Background()

	pool, err := platform.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	var sfnClient *dashboard.SFNClient
	if endpoint := cfg.LocalStackEndpoint; endpoint != "" {
		sfnClient, err = dashboard.NewSFNClient(ctx, endpoint, cfg.AWSRegion)
		if err != nil {
			slog.Warn("step functions client unavailable", "err", err)
		}
	}

	srv := &dashboard.Server{
		Store:       repository.New(pool),
		RulesURL:    cfg.RulesEngineURL,
		SFN:         sfnClient,
		S3Bucket:    envOr("OUTBOUND_BUCKET", "pav-edi-outbound"),
		S3KeyPrefix: "",
	}

	httpServer := &http.Server{
		Addr:    ":" + cfg.DashboardAPIPort,
		Handler: srv.Routes(),
	}

	go func() {
		slog.Info("dashboard api listening", "port", cfg.DashboardAPIPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	_ = httpServer.Shutdown(context.Background())
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
