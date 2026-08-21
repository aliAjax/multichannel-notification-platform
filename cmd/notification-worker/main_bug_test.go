package main

import (
	"testing"
	"time"
)

func TestShutdownWaitsUntilWorkerCompletes(t *testing.T) {
	start := make(chan struct{})
	release := make(chan struct{})
	done := make(chan bool, 2)
	wait := func() { <-start; done <- waitForShutdown(func() { <-release }, time.Second) }
	go wait()
	go wait()
	close(start)
	select {
	case <-done:
		t.Fatal("returned early")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	for i := 0; i < 2; i++ {
		if !<-done {
			t.Fatal("completion reported timeout")
		}
	}
}
func TestShutdownDeadlineReturnsForStuckWorker(t *testing.T) {
	if waitForShutdown(func() { select {} }, 10*time.Millisecond) {
		t.Fatal("blocked worker reported complete")
	}
}
