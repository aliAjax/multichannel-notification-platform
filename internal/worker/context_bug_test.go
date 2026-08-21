package worker

import (
	"context"
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/repository"
	"path/filepath"
	"testing"
	"time"
)

func TestRetryStopsAfterContextCancel(t *testing.T) {
	q := queue.New(2)
	s, _ := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	n := &domain.Notification{ID: "n", TenantID: "t", IdempotencyKey: "k", Channel: domain.ChannelEmail, Targets: []domain.Target{{Address: "a@example.com"}}, Status: domain.StatusProcessing, Attempts: 1, Version: 1}
	_, _, _ = s.Create(*n)
	d := &Dispatcher{Queue: q, Store: s, MaxAttempts: 1}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d.retry(ctx, n, errors.New("failed"))
	time.Sleep(250 * time.Millisecond)
	if q.Len() != 0 {
		t.Fatalf("retry queued after terminal attempt: %d", q.Len())
	}
}
func TestScheduledRetryHonorsCancel(t *testing.T) {
	q := queue.New(2)
	s, _ := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	n := &domain.Notification{ID: "n", TenantID: "t", IdempotencyKey: "k", Channel: domain.ChannelEmail, Targets: []domain.Target{{Address: "a@example.com"}}, Status: domain.StatusProcessing, Version: 1}
	_, _, _ = s.Create(*n)
	d := &Dispatcher{Queue: q, Store: s, MaxAttempts: 3}
	ctx, cancel := context.WithCancel(context.Background())
	d.retry(ctx, n, errors.New("failed"))
	cancel()
	time.Sleep(250 * time.Millisecond)
	if q.Len() != 0 {
		t.Fatal("canceled retry was enqueued")
	}
}
