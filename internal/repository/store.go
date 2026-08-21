package repository

import (
	"encoding/json"
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type snapshot struct {
	Notifications map[string]domain.Notification    `json:"notifications"`
	Idempotency   map[string]string                 `json:"idempotency"`
	Events        map[string][]domain.TimelineEvent `json:"events"`
	Suppressions  map[string]Suppression            `json:"suppressions"`
	Receipts      map[string]bool                   `json:"receipts"`
	Outbox        []Outbox                          `json:"outbox"`
}
type Suppression struct {
	TenantID    string         `json:"tenant_id"`
	Channel     domain.Channel `json:"channel"`
	AddressHash string         `json:"address_hash"`
	Reason      string         `json:"reason"`
	CreatedAt   time.Time      `json:"created_at"`
}
type Outbox struct {
	ID          string          `json:"id"`
	Topic       string          `json:"topic"`
	AggregateID string          `json:"aggregate_id"`
	Payload     json.RawMessage `json:"payload"`
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt *time.Time      `json:"published_at,omitempty"`
}
type Store struct {
	mu   sync.RWMutex
	path string
	data snapshot
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: snapshot{Notifications: map[string]domain.Notification{}, Idempotency: map[string]string{}, Events: map[string][]domain.TimelineEvent{}, Suppressions: map[string]Suppression{}, Receipts: map[string]bool{}}}
	b, err := os.ReadFile(path)
	if err == nil && len(b) > 0 {
		if err = json.Unmarshal(b, &s.data); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s, nil
}
func (s *Store) persistLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0750); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
func (s *Store) Create(n domain.Notification) (domain.Notification, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := n.TenantID + ":" + n.IdempotencyKey
	if id, ok := s.data.Idempotency[key]; ok {
		return s.data.Notifications[id], false, nil
	}
	s.data.Notifications[n.ID] = n
	s.data.Idempotency[key] = n.ID
	s.addEventLocked(n.ID, n.Status, "", map[string]string{"source": "submit"})
	payload, _ := json.Marshal(n)
	s.data.Outbox = append(s.data.Outbox, Outbox{ID: n.ID + "-created", Topic: "notification.created", AggregateID: n.ID, Payload: payload, CreatedAt: time.Now().UTC()})
	return n, true, s.persistLocked()
}
func (s *Store) Get(id string) (domain.Notification, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.data.Notifications[id]
	if !ok {
		return domain.Notification{}, fmt.Errorf("notification lookup: %v", ErrNotFound)
	}
	return n, nil
}
func (s *Store) Update(n domain.Notification, expected int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.data.Notifications[n.ID]
	if !ok {
		return fmt.Errorf("notification update: %v", ErrNotFound)
	}
	if old.Version != expected {
		return errors.New("version conflict")
	}
	s.data.Notifications[n.ID] = n
	s.addEventLocked(n.ID, n.Status, n.Provider, map[string]string{"attempts": itoa(n.Attempts)})
	return s.persistLocked()
}
func (s *Store) addEventLocked(id string, status domain.Status, provider string, meta map[string]string) {
	events := s.data.Events[id]
	s.data.Events[id] = append(events, domain.TimelineEvent{ID: id + "-" + itoa(len(events)+1), NotificationID: id, Status: status, Provider: provider, Sequence: int64(len(events) + 1), OccurredAt: time.Now().UTC(), Metadata: meta})
}
func (s *Store) Timeline(id string) []domain.TimelineEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.TimelineEvent(nil), s.data.Events[id]...)
}
func (s *Store) List(tenant string, limit, offset int) []domain.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.Notification{}
	for _, n := range s.data.Notifications {
		if tenant == "" || n.TenantID == tenant {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if offset >= len(out) {
		return []domain.Notification{}
	}
	out = out[offset:]
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
func (s *Store) Queued() []domain.Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.Notification{}
	for _, n := range s.data.Notifications {
		if n.Status == domain.StatusQueued {
			out = append(out, n)
		}
	}
	return out
}
func (s *Store) RecordReceipt(key string, event domain.TimelineEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.data.Receipts[key] {
		return false
	}
	s.data.Receipts[key] = true
	s.data.Events[event.NotificationID] = append(s.data.Events[event.NotificationID], event)
	n, ok := s.data.Notifications[event.NotificationID]
	if ok {
		n.Status = event.Status
		n.Version++
		n.UpdatedAt = time.Now().UTC()
		s.data.Notifications[n.ID] = n
	}
	_ = s.persistLocked()
	return true
}
func (s *Store) Suppress(x Suppression) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Suppressions[x.TenantID+":"+string(x.Channel)+":"+x.AddressHash] = x
	return s.persistLocked()
}
func (s *Store) IsSuppressed(tenant string, ch domain.Channel, hash string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data.Suppressions[tenant+":"+string(ch)+":"+hash]
	return ok
}
func (s *Store) ListSuppressions(tenant string) []Suppression {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Suppression{}
	for _, x := range s.data.Suppressions {
		if tenant == "" || x.TenantID == tenant {
			out = append(out, x)
		}
	}
	return out
}
func (s *Store) Audit() []Outbox {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Outbox(nil), s.data.Outbox...)
}
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := make([]byte, 0, 20)
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
