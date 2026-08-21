package routeprobe

import (
	"example.com/notification-platform/internal/routing"
	"testing"
)

func parallel(t *testing.T, check func() bool) {
	start := make(chan struct{})
	done := make(chan bool, 2)
	go func() { <-start; done <- check() }()
	go func() { <-start; done <- check() }()
	close(start)
	if !<-done || !<-done {
		t.Fatal("parallel probe failed")
	}
}

func TestRouterProbeReentry(t *testing.T) {
	start := make(chan struct{})
	done := make(chan bool, 2)
	go func() { <-start; done <- routing.ProbeHealthReentry() }()
	go func() { <-start; done <- routing.ProbeHealthReentry() }()
	close(start)
	if !<-done || !<-done {
		t.Fatal("parallel probe failed")
	}
}
func TestPolicyProbeMatches(t *testing.T) {
	start := make(chan struct{})
	done := make(chan bool, 2)
	go func() { <-start; done <- routing.ProbePolicyMatches() }()
	go func() { <-start; done <- routing.ProbePolicyMatches() }()
	close(start)
	if !<-done || !<-done {
		t.Fatal("parallel probe failed")
	}
}
func TestPolicyProbeProviders(t *testing.T) {
	start := make(chan struct{})
	done := make(chan bool, 2)
	go func() { <-start; done <- routing.ProbePolicyProviders() }()
	go func() { <-start; done <- routing.ProbePolicyProviders() }()
	close(start)
	if !<-done || !<-done {
		t.Fatal("parallel probe failed")
	}
}
func TestRouterProbeCancellation(t *testing.T) {
	start := make(chan struct{})
	done := make(chan bool, 2)
	go func() { <-start; done <- routing.ProbeHealthCancellation() }()
	go func() { <-start; done <- routing.ProbeHealthCancellation() }()
	close(start)
	if !<-done || !<-done {
		t.Fatal("parallel probe failed")
	}
}
