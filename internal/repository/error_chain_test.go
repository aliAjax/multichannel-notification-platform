package repository

import (
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"path/filepath"
	"testing"
)

func TestGetPreservesNotFoundChain(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "s.json"))
	_, e := s.Get("missing")
	if !errors.Is(e, ErrNotFound) {
		t.Fatalf("error chain lost: %v", e)
	}
}
func TestUpdatePreservesNotFoundChain(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "s.json"))
	e := s.Update(domain.Notification{ID: "missing"}, 1)
	if !errors.Is(e, ErrNotFound) {
		t.Fatalf("error chain lost: %v", e)
	}
}
