package routing

import (
	"context"
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/provider"
	"sync"
	"time"
)

type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half_open"
)

type candidate struct {
	p        provider.Provider
	failures int
	opened   time.Time
	state    CircuitState
}
type Router struct {
	mu        sync.Mutex
	providers map[domain.Channel][]*candidate
	threshold int
	cooldown  time.Duration
}

func New() *Router {
	return &Router{providers: map[domain.Channel][]*candidate{}, threshold: 3, cooldown: 10 * time.Second}
}
func (r *Router) Add(p provider.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Channel()] = append(r.providers[p.Channel()], &candidate{p: p, state: StateClosed})
}
func (r *Router) Choose(ctx context.Context, ch domain.Channel) (provider.Provider, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	type pick struct {
		p     provider.Provider
		state CircuitState
	}
	var picks []pick
	func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, c := range r.providers[ch] {
			if c.state == StateOpen && time.Since(c.opened) < r.cooldown {
				continue
			}
			if c.state == StateOpen {
				c.state = StateHalfOpen
			}
			picks = append(picks, pick{p: c.p, state: c.state})
		}
	}()
	for _, c := range picks {
		if c.state == StateHalfOpen {
			if err := c.p.Health(ctx); err != nil {
				continue
			}
		}
		return c.p, nil
	}
	return nil, errors.New("no healthy provider")
}
func (r *Router) Record(p provider.Provider, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, list := range r.providers {
		for _, c := range list {
			if c.p.Name() != p.Name() {
				continue
			}
			if err == nil {
				c.failures = 0
				c.state = StateClosed
				return
			}
			c.failures++
			if c.failures >= r.threshold {
				c.state = StateOpen
				c.opened = time.Now()
			}
		}
	}
}
func (r *Router) Snapshot() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]string{}
	for ch, list := range r.providers {
		for _, c := range list {
			out[string(ch)+":"+c.p.Name()] = string(c.state)
		}
	}
	return out
}
