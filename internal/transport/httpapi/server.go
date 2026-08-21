package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"example.com/notification-platform/internal/application"
	"example.com/notification-platform/internal/notification/domain"
	"example.com/notification-platform/internal/receipt"
	"example.com/notification-platform/internal/repository"
	tpl "example.com/notification-platform/internal/template"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	App           *application.Service
	Receipts      *receipt.Service
	Templates     *tpl.Service
	Logger        *slog.Logger
	WebhookSecret string
}

func New(a *application.Service, r *receipt.Service, t *tpl.Service, l *slog.Logger, secret string) *Server {
	return &Server{App: a, Receipts: r, Templates: t, Logger: l, WebhookSecret: secret}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/readyz", s.health)
	mux.HandleFunc("/api/v1/notifications", s.notifications)
	mux.HandleFunc("/api/v1/notifications/", s.notificationByID)
	mux.HandleFunc("/api/v1/templates", s.templates)
	mux.HandleFunc("/api/v1/templates/", s.templateByID)
	mux.HandleFunc("/api/v1/webhooks/", s.webhook)
	mux.HandleFunc("/api/v1/suppressions", s.suppressions)
	mux.HandleFunc("/api/v1/audit/export", s.audit)
	return requestID(mux)
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"status": "ok", "time": time.Now().UTC()})
}
func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(r.URL.Path, "/batch") {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var input []domain.Notification
		if err := decode(r, &input); err != nil {
			errorJSON(w, http.StatusBadRequest, "invalid_request", err)
			return
		}
		if len(input) == 0 || len(input) > 1000 {
			errorJSON(w, http.StatusBadRequest, "invalid_request", errors.New("batch must contain 1..1000 notifications"))
			return
		}
		result := make([]domain.Notification, 0, len(input))
		for i := range input {
			if input[i].TenantID == "" {
				input[i].TenantID = r.Header.Get("X-Tenant-ID")
			}
			saved, _, err := s.App.Submit(r.Context(), input[i])
			if err != nil {
				errorJSON(w, http.StatusBadRequest, "batch_item_failed", fmt.Errorf("item %d: %w", i, err))
				return
			}
			result = append(result, saved)
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"items": result})
		return
	}
	if r.Method == http.MethodGet {
		tenant := r.Header.Get("X-Tenant-ID")
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		writeJSON(w, 200, map[string]any{"items": s.App.Store.List(tenant, limit, offset)})
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var n domain.Notification
	if err := decode(r, &n); err != nil {
		errorJSON(w, 400, "invalid_request", err)
		return
	}
	if n.TenantID == "" {
		n.TenantID = r.Header.Get("X-Tenant-ID")
	}
	saved, created, err := s.App.Submit(r.Context(), n)
	if err != nil {
		errorJSON(w, 400, "submit_failed", err)
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	w.Header().Set("ETag", strconv.FormatInt(saved.Version, 10))
	writeJSON(w, status, saved)
}
func (s *Server) notificationByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		errorJSON(w, 404, "not_found", repository.ErrNotFound)
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		n, err := s.App.Store.Get(id)
		if err != nil {
			errorJSON(w, 404, "not_found", err)
			return
		}
		w.Header().Set("ETag", strconv.FormatInt(n.Version, 10))
		writeJSON(w, 200, n)
		return
	}
	if len(parts) == 2 && parts[1] == "timeline" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		writeJSON(w, 200, map[string]any{"items": s.App.Store.Timeline(id)})
		return
	}
	if len(parts) == 2 && (parts[1] == "cancel" || parts[1] == "pause" || parts[1] == "replay") {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		n, err := s.App.Store.Get(id)
		if err != nil {
			errorJSON(w, 404, "not_found", err)
			return
		}
		switch parts[1] {
		case "cancel":
			err = s.App.Cancel(id, n.Version)
		case "pause":
			err = s.App.Pause(id, n.Version)
		case "replay":
			var x domain.Notification
			x, err = s.App.Replay(id)
			if err == nil {
				writeJSON(w, 202, x)
				return
			}
		}
		if err != nil {
			errorJSON(w, 409, "transition_failed", err)
			return
		}
		writeJSON(w, 202, map[string]string{"status": "accepted"})
		return
	}
	errorJSON(w, 404, "not_found", repository.ErrNotFound)
}
func (s *Server) templates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var d tpl.Definition
	if err := decode(r, &d); err != nil {
		errorJSON(w, 400, "invalid_request", err)
		return
	}
	if err := s.Templates.Create(d); err != nil {
		errorJSON(w, 409, "create_failed", err)
		return
	}
	writeJSON(w, 201, d)
}
func (s *Server) templateByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/templates/"), "/")
	if len(parts) < 2 {
		errorJSON(w, 404, "not_found", repository.ErrNotFound)
		return
	}
	id := parts[0]
	switch parts[1] {
	case "versions":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		var v tpl.Version
		if err := decode(r, &v); err != nil {
			errorJSON(w, 400, "invalid_request", err)
			return
		}
		if err := s.Templates.AddVersion(id, v); err != nil {
			errorJSON(w, 400, "version_failed", err)
			return
		}
		writeJSON(w, 201, v)
	case "publish":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		v, _ := strconv.Atoi(r.URL.Query().Get("version"))
		if err := s.Templates.Publish(id, v); err != nil {
			errorJSON(w, 400, "publish_failed", err)
			return
		}
		writeJSON(w, 202, map[string]any{"published": v})
	default:
		errorJSON(w, 404, "not_found", repository.ErrNotFound)
	}
}
func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		errorJSON(w, 400, "read_failed", err)
		return
	}
	var e receipt.Event
	if err = json.Unmarshal(body, &e); err != nil {
		errorJSON(w, 400, "invalid_receipt", err)
		return
	}
	accepted, err := s.Receipts.Ingest(e)
	if err != nil {
		errorJSON(w, 400, "receipt_rejected", err)
		return
	}
	writeJSON(w, 202, map[string]bool{"accepted": accepted})
}
func (s *Server) suppressions(w http.ResponseWriter, r *http.Request) {
	tenant := r.Header.Get("X-Tenant-ID")
	if r.Method == http.MethodGet {
		writeJSON(w, 200, map[string]any{"items": s.App.Store.ListSuppressions(tenant)})
		return
	}
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	var x struct {
		Channel domain.Channel `json:"channel"`
		Address string         `json:"address"`
		Reason  string         `json:"reason"`
	}
	if err := decode(r, &x); err != nil {
		errorJSON(w, 400, "invalid_request", err)
		return
	}
	if tenant == "" || x.Address == "" {
		errorJSON(w, 400, "invalid_request", errors.New("tenant and address required"))
		return
	}
	sp := repository.Suppression{TenantID: tenant, Channel: x.Channel, AddressHash: hash(tenant, x.Address), Reason: x.Reason, CreatedAt: time.Now().UTC()}
	if err := s.App.Store.Suppress(sp); err != nil {
		errorJSON(w, 500, "persist_failed", err)
		return
	}
	writeJSON(w, 201, sp)
}
func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	writeJSON(w, 200, map[string]any{"items": s.App.Store.Audit()})
}
func decode(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func errorJSON(w http.ResponseWriter, status int, code string, err error) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": err.Error()}})
}
func methodNotAllowed(w http.ResponseWriter) {
	errorJSON(w, 405, "method_not_allowed", fmt.Errorf("method not allowed"))
}
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = hash(time.Now().String(), r.RemoteAddr)
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}
func hash(v ...string) string {
	h := sha256.New()
	for _, x := range v {
		h.Write([]byte(x))
	}
	return hex.EncodeToString(h.Sum(nil))
}
