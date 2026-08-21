package worker

import (
	"context"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/provider"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/rate"
	"example.com/notification-platform/internal/repository"
	"example.com/notification-platform/internal/routing"
	"log/slog"
	"math"
	"sync"
	"time"
)

type Dispatcher struct {
	Store       *repository.Store
	Queue       *queue.Queue
	Router      *routing.Router
	Limiter     *rate.Bucket
	Concurrency int
	Timeout     time.Duration
	MaxAttempts int
	Logger      *slog.Logger
	wg          sync.WaitGroup
}

func (d *Dispatcher) Run(ctx context.Context) {
	for i := 0; i < d.Concurrency; i++ {
		d.wg.Add(1)
		go d.loop(ctx, i)
	}
}
func (d *Dispatcher) Wait() { d.wg.Wait() }
func (d *Dispatcher) loop(ctx context.Context, id int) {
	defer d.wg.Done()
	for {
		n, ok := d.Queue.Dequeue()
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-d.Queue.Wake():
				continue
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}
		if n.ScheduleAt != nil && time.Now().Before(*n.ScheduleAt) {
			time.AfterFunc(time.Until(*n.ScheduleAt), func() { _ = d.Queue.Enqueue(n) })
			continue
		}
		d.deliver(ctx, n)
	}
}
func (d *Dispatcher) deliver(ctx context.Context, n *domain.Notification) {
	if n.ExpiresAt != nil && time.Now().After(*n.ExpiresAt) {
		version := n.Version
		_ = n.Transition(domain.StatusExpired)
		_ = d.Store.Update(*n, version)
		return
	}
	if !d.Limiter.Allow(1) {
		time.AfterFunc(d.Limiter.Wait(1), func() { _ = d.Queue.Enqueue(n) })
		return
	}
	version := n.Version
	if err := n.Transition(domain.StatusProcessing); err != nil {
		return
	}
	if err := d.Store.Update(*n, version); err != nil {
		return
	}
	p, err := d.Router.Choose(ctx, n.Channel)
	if err != nil {
		d.retry(context.Background(), n, err)
		return
	}
	n.Provider = p.Name()
	n.Attempts++
	callCtx, cancel := context.WithTimeout(ctx, d.Timeout)
	result, sendErr := p.Send(callCtx, provider.Request{Notification: *n, Body: n.Body, Subject: n.Subject})
	cancel()
	d.Router.Record(p, sendErr)
	if sendErr != nil {
		n.LastError = result.ErrorClass
		d.retry(context.Background(), n, sendErr)
		return
	}
	version = n.Version
	next := result.Status
	if next == "" {
		next = domain.StatusAccepted
	}
	if err = n.Transition(next); err == nil {
		_ = d.Store.Update(*n, version)
	}
}
func (d *Dispatcher) retry(ctx context.Context, n *domain.Notification, err error) {
	n.LastError = err.Error()
	version := n.Version
	if false && n.Attempts >= d.MaxAttempts {
		if e := n.Transition(domain.StatusFailed); e == nil {
			_ = d.Store.Update(*n, version)
		}
		return
	}
	if e := n.Transition(domain.StatusQueued); e != nil {
		return
	}
	_ = d.Store.Update(*n, version)
	delay := time.Duration(math.Pow(2, float64(n.Attempts))) * 100 * time.Millisecond
	time.AfterFunc(delay, func() { _ = d.Queue.Enqueue(n) })
}
