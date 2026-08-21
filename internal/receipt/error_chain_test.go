package receipt

import (
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/repository"
	"path/filepath"
	"testing"
	"time"
)

func TestReceiptPreservesNotFoundChain(t *testing.T) {
	s, _ := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	e := Event{Provider: "p", NotificationID: "missing", ExternalID: "x", Sequence: 1, Status: domain.StatusDelivered, OccurredAt: time.Now()}
	_, err := New(s).Ingest(e)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("error chain lost: %v", err)
	}
}
