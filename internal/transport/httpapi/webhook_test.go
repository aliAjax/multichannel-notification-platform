package httpapi

import (
	"bytes"
	"example.com/notification-platform/internal/application"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/queue"
	"example.com/notification-platform/internal/receipt"
	"example.com/notification-platform/internal/repository"
	"example.com/notification-platform/internal/security"
	tpl "example.com/notification-platform/internal/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestBadWebhookSignatureDenied(t *testing.T) {
	s, _ := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	_, _, _ = s.Create(domain.Notification{ID: "missing", TenantID: "t", Channel: domain.ChannelEmail, Targets: []domain.Target{{Address: "a@example.com"}}, Status: domain.StatusQueued})
	h := New(application.New(s, queue.New(2), tpl.NewService()), receipt.New(s), tpl.NewService(), slog.Default(), "secret").Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/p/receipts", bytes.NewBufferString(`{"provider":"p","notification_id":"missing","external_id":"x","sequence":1,"status":"delivered"}`))
	req.Header.Set("X-Signature", "bad")
	req.Header.Set("X-Timestamp", "1700000000")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}
func TestPathProviderMismatchDenied(t *testing.T) {
	s, _ := repository.Open(filepath.Join(t.TempDir(), "s.json"))
	_, _, _ = s.Create(domain.Notification{ID: "missing", TenantID: "t", Channel: domain.ChannelEmail, Targets: []domain.Target{{Address: "a@example.com"}}, Status: domain.StatusQueued})
	h := New(application.New(s, queue.New(2), tpl.NewService()), receipt.New(s), tpl.NewService(), slog.Default(), "secret").Handler()
	body := []byte(`{"provider":"other","notification_id":"missing","external_id":"x","sequence":1,"status":"delivered"}`)
	ts := time.Now().Unix()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/path-provider/receipts", bytes.NewReader(body))
	req.Header.Set("X-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Signature", security.Sign("secret", ts, body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}
