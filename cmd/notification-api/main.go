package main

import (
	"context"
	"errors"
	"example.com/notification-platform/internal/application"
	"example.com/notification-platform/internal/config"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/observability"
	"example.com/notification-platform/internal/provider"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/rate"
	"example.com/notification-platform/internal/receipt"
	"example.com/notification-platform/internal/repository"
	"example.com/notification-platform/internal/routing"
	tpl "example.com/notification-platform/internal/template"
	"example.com/notification-platform/internal/transport/httpapi"
	"example.com/notification-platform/internal/worker"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	logger := observability.Logger()
	cfg, err := config.Load(os.Getenv("NOTIFY_CONFIG"))
	if err != nil {
		logger.Error("config load failed", "error", err)
		os.Exit(1)
	}
	store, err := repository.Open(cfg.DataFile)
	if err != nil {
		logger.Error("store open failed", "error", err)
		os.Exit(1)
	}
	q := queue.New(cfg.QueueCapacity)
	templates := tpl.NewService()
	app := application.New(store, q, templates)
	router := routing.New()
	router.Add(&provider.Mock{ProviderName: "mock-email", ProviderChannel: domain.ChannelEmail})
	router.Add(&provider.Mock{ProviderName: "mock-sms", ProviderChannel: domain.ChannelSMS})
	router.Add(&provider.Mock{ProviderName: "mock-webhook", ProviderChannel: domain.ChannelWebhook})
	router.Add(&provider.Mock{ProviderName: "mock-push", ProviderChannel: domain.ChannelPush})
	app.Recover()
	dispatcher := &worker.Dispatcher{Store: store, Queue: q, Router: router, Limiter: rate.New(100, 100), Concurrency: cfg.WorkerConcurrency, Timeout: cfg.ProviderTimeout, MaxAttempts: cfg.MaxAttempts, Logger: logger}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	dispatcher.Run(ctx)
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: buildHandler(httpapi.New(app, receipt.New(store), templates, logger, cfg.WebhookSecret).Handler(), "web"), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("notification api started", "addr", cfg.HTTPAddr)
		if e := srv.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			logger.Error("http server failed", "error", e)
			stop()
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	dispatcher.Wait()
	logger.Info("notification api stopped")
}
func buildHandler(api http.Handler, dir string) http.Handler { return api }
