package provider

import (
	"context"
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Request struct {
	Notification domain.Notification
	Body         string
	Subject      string
}
type Result struct {
	ExternalID string
	Status     domain.Status
	Retryable  bool
	ErrorClass string
	Message    string
}
type Provider interface {
	Name() string
	Channel() domain.Channel
	Send(context.Context, Request) (Result, error)
	Health(context.Context) error
}
type Mock struct {
	ProviderName    string
	ProviderChannel domain.Channel
	mu              sync.Mutex
	Calls           int
	Failures        int
	Delay           time.Duration
}

func (m *Mock) Name() string            { return m.ProviderName }
func (m *Mock) Channel() domain.Channel { return m.ProviderChannel }
func (m *Mock) Send(ctx context.Context, r Request) (Result, error) {
	m.mu.Lock()
	m.Calls++
	call := m.Calls
	m.mu.Unlock()
	if m.Delay > 0 {
		select {
		case <-time.After(m.Delay):
		case <-ctx.Done():
			return Result{Retryable: true, ErrorClass: "timeout"}, ctx.Err()
		}
	}
	if call <= m.Failures {
		return Result{Retryable: true, ErrorClass: "temporary", Message: "injected failure"}, errors.New("provider temporary failure")
	}
	return Result{ExternalID: fmt.Sprintf("%s-%d", m.ProviderName, call), Status: domain.StatusAccepted}, nil
}
func (m *Mock) Health(context.Context) error { return nil }

type HTTP struct {
	ProviderName    string
	ProviderChannel domain.Channel
	URL             string
	Client          *http.Client
}

func (h *HTTP) Name() string            { return h.ProviderName }
func (h *HTTP) Channel() domain.Channel { return h.ProviderChannel }
func (h *HTTP) client() *http.Client {
	if h.Client == nil {
		return http.DefaultClient
	}
	return h.Client
}
func (h *HTTP) Send(ctx context.Context, r Request) (Result, error) {
	if h.URL == "" {
		return Result{}, errors.New("provider URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, strings.NewReader(r.Body))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client().Do(req)
	if err != nil {
		return Result{Retryable: true, ErrorClass: "network"}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return Result{Retryable: true, ErrorClass: "server"}, fmt.Errorf("provider status %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return Result{Retryable: false, ErrorClass: "rejected"}, fmt.Errorf("provider status %d", resp.StatusCode)
	}
	return Result{ExternalID: resp.Header.Get("X-Message-ID"), Status: domain.StatusAccepted}, nil
}
func (h *HTTP) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.URL, nil)
	if err != nil {
		return err
	}
	resp, err := h.client().Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("unhealthy: %s", resp.Status)
	}
	return nil
}
