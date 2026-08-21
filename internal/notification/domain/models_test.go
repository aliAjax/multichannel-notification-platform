package domain

import "testing"

func TestNotificationTransition(t *testing.T) {
	n := Notification{TenantID: "t", IdempotencyKey: "k", Channel: ChannelEmail, Targets: []Target{{Address: "a@example.com"}}, Status: StatusQueued, Version: 1}
	if err := n.Transition(StatusProcessing); err != nil {
		t.Fatal(err)
	}
	if err := n.Transition(StatusDelivered); err == nil {
		t.Fatal("processing must not jump to delivered")
	}
}

func TestNormalizeTarget(t *testing.T) {
	n := Notification{TenantID: "t", IdempotencyKey: "k", Channel: ChannelEmail, Targets: []Target{{Address: " A@Example.COM "}}}
	if err := n.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := n.Targets[0].Address; got != "a@example.com" {
		t.Fatalf("normalized target = %q", got)
	}
}
