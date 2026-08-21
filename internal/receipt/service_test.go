package receipt

import (
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/repository"
	"path/filepath"
	"testing"
	"time"
)

func TestReceiptDeduplication(t *testing.T) {
	s, err := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.Create(domain.Notification{ID: "n", TenantID: "t", IdempotencyKey: "k", Channel: domain.ChannelWebhook, Targets: []domain.Target{{Address: "https://example.test"}}, Status: domain.StatusQueued, CreatedAt: time.Now(), UpdatedAt: time.Now(), Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	svc := New(s)
	e := Event{Provider: "p", NotificationID: "n", ExternalID: "x", Sequence: 1, Status: domain.StatusDelivered, OccurredAt: time.Now()}
	ok, err := svc.Ingest(e)
	if err != nil || !ok {
		t.Fatalf("first receipt: %v %v", ok, err)
	}
	ok, err = svc.Ingest(e)
	if err != nil || ok {
		t.Fatalf("duplicate receipt: %v %v", ok, err)
	}
}
