package routing

import (
	"context"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/provider"
	"time"
)

type probeProvider struct{ router *Router }
type cancellationProbe struct{}

func (*probeProvider) Name() string            { return "probe" }
func (*probeProvider) Channel() domain.Channel { return domain.ChannelEmail }
func (*probeProvider) Send(context.Context, provider.Request) (provider.Result, error) {
	return provider.Result{}, nil
}
func (p *probeProvider) Health(context.Context) error { _ = p.router.Snapshot(); return nil }
func (*cancellationProbe) Name() string               { return "cancel" }
func (*cancellationProbe) Channel() domain.Channel    { return domain.ChannelEmail }
func (*cancellationProbe) Send(context.Context, provider.Request) (provider.Result, error) {
	return provider.Result{}, nil
}
func (*cancellationProbe) Health(context.Context) error { return nil }

func ProbeHealthReentry() bool {
	r := New()
	p := &probeProvider{router: r}
	r.Add(p)
	r.mu.Lock()
	r.providers[domain.ChannelEmail][0].state = StateHalfOpen
	r.mu.Unlock()
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	go func() { <-start; _, _ = r.Choose(context.Background(), domain.ChannelEmail); done <- struct{}{} }()
	go func() { <-start; _ = r.Snapshot(); done <- struct{}{} }()
	close(start)
	select {
	case <-done:
		return true
	case <-time.After(100 * time.Millisecond):
		return false
	}
}
func ProbePolicyMatches() bool {
	p := NewPolicySet()
	r := Rule{ID: "r", TenantID: "t", Channel: domain.ChannelEmail, ProviderNames: []string{"p"}, Weight: 1, Enabled: true, Matches: []Match{{Subject: "tenant", Pattern: "acme"}}}
	if p.Put(r) != nil {
		return false
	}
	start := make(chan struct{})
	done := make(chan struct{}, 2)
	go func() {
		<-start
		for i := 0; i < 1000; i++ {
			r.Matches[0].Pattern = "evil"
		}
		done <- struct{}{}
	}()
	go func() {
		<-start
		for i := 0; i < 1000; i++ {
			_ = p.List("t")[0].Matches[0].Pattern
		}
		done <- struct{}{}
	}()
	close(start)
	<-done
	<-done
	return p.List("t")[0].Matches[0].Pattern == "acme"
}
func ProbePolicyProviders() bool {
	p := NewPolicySet()
	r := Rule{ID: "p", TenantID: "t", Channel: domain.ChannelEmail, ProviderNames: []string{"primary"}, Weight: 1, Enabled: true}
	if p.Put(r) != nil {
		return false
	}
	r.ProviderNames[0] = "mutated"
	return p.List("t")[0].ProviderNames[0] == "primary"
}
func ProbeHealthCancellation() bool {
	r := New()
	p := &cancellationProbe{}
	r.Add(p)
	r.mu.Lock()
	r.providers[domain.ChannelEmail][0].state = StateHalfOpen
	r.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Choose(ctx, domain.ChannelEmail)
	return err == context.Canceled
}
