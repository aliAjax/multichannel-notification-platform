package application

import (
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/repository"
	tpl "example.com/notification-platform/internal/template"
	"path/filepath"
	"testing"
)

func TestPausedNotificationCanResume(t *testing.T){n:=domain.Notification{Status:domain.StatusPaused,Version:3};if e:=n.Transition(domain.StatusQueued);e!=nil{t.Fatal(e)}}
func TestQueuedNotificationCanPause(t *testing.T){n:=domain.Notification{Status:domain.StatusQueued,Version:1};if e:=n.Transition(domain.StatusPaused);e!=nil{t.Fatal(e)}}
func TestProcessingNotificationCanPause(t *testing.T){n:=domain.Notification{Status:domain.StatusProcessing,Version:1};if e:=n.Transition(domain.StatusPaused);e!=nil{t.Fatal(e)}}
func TestAcceptedNotificationCanDeliver(t *testing.T){n:=domain.Notification{Status:domain.StatusAccepted,Version:1};if e:=n.Transition(domain.StatusDelivered);e!=nil{t.Fatal(e)}}

func TestReplayResetsDeliveryState(t *testing.T) {
	s, _ := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	n := domain.Notification{ID: "f", TenantID: "t", IdempotencyKey: "k", Channel: domain.ChannelEmail, Targets: []domain.Target{{Address: "a@example.com"}}, Status: domain.StatusFailed, Attempts: 4, Provider: "old", LastError: "timeout", Version: 2}
	_, _, _ = s.Create(n)
	r, err := New(s, queue.New(10), tpl.NewService()).Replay(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r.Attempts != 0 || r.Provider != "" || r.LastError != "" {
		t.Fatalf("stale delivery state: %+v", r)
	}
}
