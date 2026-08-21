package queueprobe

import (
	"context"
	"testing"
	"time"

	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/queue"
)

func TestPrioritySnapshotIsolation(t *testing.T) {
	q := queue.New(2)
	n := &domain.Notification{ID: "q", Status: domain.StatusQueued, Targets: []domain.Target{{Address: "a@example.com"}}, Variables: map[string]any{"k": "v"}}
	if err := q.Enqueue(n); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	var got *domain.Notification
	go func() {
		<-start
		for i := 0; i < 1000; i++ {
			n.Status = domain.StatusFailed
			n.Targets[0].Address = "changed"
			n.Variables["k"] = "changed"
		}
		done <- struct{}{}
	}()
	go func() { <-start; got, _ = q.Dequeue(); done <- struct{}{} }()
	close(start)
	<-done
	<-done
	if got.Status != domain.StatusQueued || got.Targets[0].Address != "a@example.com" || got.Variables["k"] != "v" {
		t.Fatalf("queue alias: %s", got.Status)
	}
}

func TestDelayedSnapshotIsolation(t *testing.T) {
	d := queue.NewDelayQueue(2)
	q := queue.New(2)
	n := &domain.Notification{ID: "d", Status: domain.StatusQueued, Targets: []domain.Target{{Address: "d@example.com"}}, Variables: map[string]any{"k": "v"}}
	if !d.Add(n, time.Now().Add(-time.Millisecond)) {
		t.Fatal("add failed")
	}
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	go func() {
		<-start
		for i := 0; i < 1000; i++ {
			n.Status = domain.StatusFailed
			n.Targets[0].Address = "changed"
			n.Variables["k"] = "changed"
		}
		done <- struct{}{}
	}()
	go func() { <-start; time.Sleep(time.Millisecond); done <- struct{}{} }()
	close(start)
	<-done
	<-done
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go d.Run(ctx, q)
	deadline := time.Now().Add(time.Second)
	for q.Len() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	got, ok := q.Dequeue()
	if !ok {
		t.Fatal("delayed item not promoted")
	}
	if got.Status != domain.StatusQueued || got.Targets[0].Address != "d@example.com" || got.Variables["k"] != "v" {
		t.Fatalf("delay alias: %s", got.Status)
	}
}
