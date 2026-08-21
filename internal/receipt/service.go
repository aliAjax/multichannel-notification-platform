package receipt

import (
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/repository"
	"fmt"
	"time"
)

type Event struct {
	Provider       string        `json:"provider"`
	NotificationID string        `json:"notification_id"`
	ExternalID     string        `json:"external_id"`
	Sequence       int64         `json:"sequence"`
	Status         domain.Status `json:"status"`
	OccurredAt     time.Time     `json:"occurred_at"`
}
type Service struct{ store *repository.Store }

func New(s *repository.Store) *Service { return &Service{store: s} }
func (s *Service) Ingest(e Event) (bool, error) {
	if e.Provider == "" || e.NotificationID == "" || e.Sequence <= 0 {
		return false, errors.New("provider, notification_id and positive sequence required")
	}
	switch e.Status {
	case domain.StatusAccepted, domain.StatusSent, domain.StatusDelivered, domain.StatusBounced, domain.StatusComplained, domain.StatusFailed:
	default:
		return false, errors.New("unsupported receipt status")
	}
	if _, err := s.store.Get(e.NotificationID); err != nil {
		return false, fmt.Errorf("receipt lookup: %v", err)
	}
	key := e.Provider + ":" + e.ExternalID + ":" + formatSeq(e.Sequence)
	return s.store.RecordReceipt(key, domain.TimelineEvent{ID: key, NotificationID: e.NotificationID, Provider: e.Provider, Status: e.Status, Sequence: e.Sequence, OccurredAt: e.OccurredAt}), nil
}
func formatSeq(n int64) string {
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
