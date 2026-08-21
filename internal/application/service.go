package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/repository"
	"example.com/notification-platform/internal/security"
	tpl "example.com/notification-platform/internal/template"
	"time"
)

type Service struct {
	Store     *repository.Store
	Queue     *queue.Queue
	Templates *tpl.Service
}

func New(store *repository.Store, q *queue.Queue, t *tpl.Service) *Service {
	return &Service{Store: store, Queue: q, Templates: t}
}
func (a *Service) Submit(ctx context.Context, n domain.Notification) (domain.Notification, bool, error) {
	if err := ctx.Err(); err != nil {
		return n, false, err
	}
	if err := n.Validate(); err != nil {
		return n, false, err
	}
	now := time.Now().UTC()
	if n.ID == "" {
		n.ID = newID("ntf")
	}
	n.Status = domain.StatusQueued
	n.CreatedAt = now
	n.UpdatedAt = now
	n.Version = 1
	for _, target := range n.Targets {
		if a.Store.IsSuppressed(n.TenantID, n.Channel, security.HashAddress(n.TenantID, target.Address)) {
			return n, false, errors.New("target is suppressed")
		}
	}
	saved, created, err := a.Store.Create(n)
	if err != nil || !created {
		return saved, created, err
	}
	if n.ScheduleAt == nil || !n.ScheduleAt.After(now) {
		if err = a.Queue.Enqueue(&saved); err != nil {
			return saved, true, err
		}
	}
	return saved, true, nil
}
func (a *Service) Cancel(id string, expected int64) error {
	return a.transition(id, expected, domain.StatusCanceled)
}
func (a *Service) Pause(id string, expected int64) error {
	return a.transition(id, expected, domain.StatusPaused)
}
func (a *Service) Replay(id string) (domain.Notification, error) {
	old, err := a.Store.Get(id)
	if err != nil {
		return old, err
	}
	old.ID = ""
	old.IdempotencyKey = old.IdempotencyKey + ":" + newID("replay")
	old.Attempts = 0
	old.LastError = ""
	old.Provider = ""
	n, _, err := a.Submit(context.Background(), old)
	return n, err
}
func (a *Service) transition(id string, expected int64, next domain.Status) error {
	n, err := a.Store.Get(id)
	if err != nil {
		return err
	}
	if expected > 0 && n.Version != expected {
		return errors.New("etag version conflict")
	}
	version := n.Version
	if err = n.Transition(next); err != nil {
		return err
	}
	return a.Store.Update(n, version)
}
func (a *Service) Recover() {
	for _, n := range a.Store.Queued() {
		copy := n
		_ = a.Queue.Enqueue(&copy)
	}
}
func newID(prefix string) string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}
