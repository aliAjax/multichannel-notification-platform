package domain

import (
	"errors"
	"strings"
	"time"
)

type Channel string

const (
	ChannelEmail   Channel = "email"
	ChannelSMS     Channel = "sms"
	ChannelWebhook Channel = "webhook"
	ChannelPush    Channel = "push"
)

type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusAccepted   Status = "accepted"
	StatusSent       Status = "sent"
	StatusDelivered  Status = "delivered"
	StatusBounced    Status = "bounced"
	StatusComplained Status = "complained"
	StatusFailed     Status = "failed"
	StatusExpired    Status = "expired"
	StatusCanceled   Status = "canceled"
	StatusPaused     Status = "paused"
)

type Trigger string

const (
	TriggerImmediate Trigger = "immediate"
	TriggerScheduled Trigger = "scheduled"
	TriggerRecurring Trigger = "recurring"
)

type Target struct {
	Address string `json:"address"`
	Name    string `json:"name,omitempty"`
}

func (t Target) Normalize(ch Channel) (Target, error) {
	t.Address = strings.TrimSpace(strings.ToLower(t.Address))
	if t.Address == "" {
		return Target{}, errors.New("target address is required")
	}
	if len(t.Address) > 512 {
		return Target{}, errors.New("target address too long")
	}
	if ch == ChannelEmail && !strings.Contains(t.Address, "@") {
		return Target{}, errors.New("invalid email target")
	}
	if ch == ChannelSMS && len(t.Address) < 7 {
		return Target{}, errors.New("invalid phone target")
	}
	return t, nil
}

type Notification struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	IdempotencyKey  string         `json:"idempotency_key"`
	Channel         Channel        `json:"channel"`
	Targets         []Target       `json:"targets"`
	TemplateID      string         `json:"template_id,omitempty"`
	TemplateVersion int            `json:"template_version,omitempty"`
	Variables       map[string]any `json:"variables,omitempty"`
	Subject         string         `json:"subject,omitempty"`
	Body            string         `json:"body,omitempty"`
	Priority        int            `json:"priority"`
	Trigger         Trigger        `json:"trigger"`
	ScheduleAt      *time.Time     `json:"schedule_at,omitempty"`
	ExpiresAt       *time.Time     `json:"expires_at,omitempty"`
	BusinessID      string         `json:"business_id,omitempty"`
	Status          Status         `json:"status"`
	Attempts        int            `json:"attempts"`
	Provider        string         `json:"provider,omitempty"`
	LastError       string         `json:"last_error,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	Version         int64          `json:"version"`
}

type Attempt struct {
	ID             string     `json:"id"`
	NotificationID string     `json:"notification_id"`
	Provider       string     `json:"provider"`
	Status         Status     `json:"status"`
	ErrorClass     string     `json:"error_class,omitempty"`
	ExternalID     string     `json:"external_id,omitempty"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at,omitempty"`
}
type TimelineEvent struct {
	ID             string            `json:"id"`
	NotificationID string            `json:"notification_id"`
	Status         Status            `json:"status"`
	Provider       string            `json:"provider,omitempty"`
	Sequence       int64             `json:"sequence"`
	OccurredAt     time.Time         `json:"occurred_at"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func (n *Notification) Validate() error {
	if strings.TrimSpace(n.TenantID) == "" {
		return errors.New("tenant_id is required")
	}
	if strings.TrimSpace(n.IdempotencyKey) == "" {
		return errors.New("idempotency_key is required")
	}
	if n.Channel != ChannelEmail && n.Channel != ChannelSMS && n.Channel != ChannelWebhook && n.Channel != ChannelPush {
		return errors.New("unsupported channel")
	}
	if len(n.Targets) == 0 || len(n.Targets) > 1000 {
		return errors.New("targets must contain 1..1000 items")
	}
	if n.Priority < 0 || n.Priority > 100 {
		return errors.New("priority must be between 0 and 100")
	}
	if n.Trigger == "" {
		n.Trigger = TriggerImmediate
	}
	if n.Trigger == TriggerScheduled && n.ScheduleAt == nil {
		return errors.New("schedule_at required")
	}
	for i, t := range n.Targets {
		normalized, err := t.Normalize(n.Channel)
		if err != nil {
			return err
		}
		n.Targets[i] = normalized
	}
	return nil
}

func (n *Notification) Transition(next Status) error {
	allowed := map[Status]map[Status]bool{
		StatusQueued:     {StatusProcessing: true, StatusCanceled: true, StatusPaused: true, StatusExpired: true},
		StatusProcessing: {StatusAccepted: true, StatusSent: true, StatusFailed: true, StatusQueued: true, StatusPaused: true},
		StatusAccepted:   {StatusSent: true, StatusDelivered: true, StatusBounced: true, StatusFailed: true},
		StatusSent:       {StatusDelivered: true, StatusBounced: true, StatusComplained: true, StatusFailed: true},
		StatusPaused:     {StatusQueued: true, StatusCanceled: true},
	}
	if n.Status == next {
		return nil
	}
	if !allowed[n.Status][next] {
		return errors.New("invalid status transition: " + string(n.Status) + " -> " + string(next))
	}
	n.Status = next
	n.Version++
	n.UpdatedAt = time.Now().UTC()
	return nil
}
