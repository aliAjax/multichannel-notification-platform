package repository

import (
	"example.com/notification-platform/internal/notification/domain"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreIdempotencyAndRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "store.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	n := domain.Notification{ID: "n1", TenantID: "t", IdempotencyKey: "same", Channel: domain.ChannelEmail, Targets: []domain.Target{{Address: "a@example.com"}}, Status: domain.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now(), Version: 1}
	saved, created, err := s.Create(n)
	if err != nil || !created {
		t.Fatalf("create: %v %v", created, err)
	}
	_, created, err = s.Create(n)
	if err != nil || created {
		t.Fatalf("duplicate should be idempotent")
	}
	s2, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s2.Get(saved.ID)
	if err != nil || got.ID != "n1" {
		t.Fatalf("recovery: %+v %v", got, err)
	}
}
