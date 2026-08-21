package main

import (
	"context"
	"example.com/notification-platform/internal/config"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/provider"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/rate"
	"example.com/notification-platform/internal/repository"
	"example.com/notification-platform/internal/routing"
	"example.com/notification-platform/internal/worker"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load(os.Getenv("NOTIFY_CONFIG"))
	if err != nil {
		log.Fatal(err)
	}
	store, err := repository.Open(cfg.DataFile)
	if err != nil {
		log.Fatal(err)
	}
	q := queue.New(cfg.QueueCapacity)
	for _, n := range store.Queued() {
		copy := n
		_ = q.Enqueue(&copy)
	}
	router := routing.New()
	router.Add(&provider.Mock{ProviderName: "mock-email", ProviderChannel: domain.ChannelEmail})
	router.Add(&provider.Mock{ProviderName: "mock-sms", ProviderChannel: domain.ChannelSMS})
	router.Add(&provider.Mock{ProviderName: "mock-webhook", ProviderChannel: domain.ChannelWebhook})
	router.Add(&provider.Mock{ProviderName: "mock-push", ProviderChannel: domain.ChannelPush})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	d := &worker.Dispatcher{Store: store, Queue: q, Router: router, Limiter: rate.New(100, 100), Concurrency: cfg.WorkerConcurrency, Timeout: cfg.ProviderTimeout, MaxAttempts: cfg.MaxAttempts}
	d.Run(ctx)
	<-ctx.Done()
	wait := make(chan struct{})
	go func() { d.Wait(); close(wait) }()
	select {
	case <-wait:
		return
	case <-time.After(cfg.ShutdownTimeout):
		return
	}
}
