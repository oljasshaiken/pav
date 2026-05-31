package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/pavillio/pav-edi/internal/api"
	"github.com/pavillio/pav-edi/internal/platform"
	"github.com/pavillio/pav-edi/internal/repository"
	"github.com/pavillio/pav-edi/internal/template"
	"github.com/pavillio/pav-edi/internal/validation"
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

	store := repository.New(pool)
	srv := &api.Server{
		Engine:     &template.StubRenderer{Store: store},
		EngineName: "template",
		Store:      store,
		Validate:   validation.NoopPipeline{},
	}

	httpServer := &http.Server{
		Addr:    ":" + cfg.TemplateEnginePort,
		Handler: srv.Routes(),
	}

	go func() {
		slog.Info("template engine listening", "port", cfg.TemplateEnginePort)
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
