package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPProviderUsesDefaultClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer srv.Close()
	h := &HTTP{ProviderName: "p", URL: srv.URL}
	if e := h.Health(context.Background()); e != nil {
		t.Fatal(e)
	}
}
func TestRegistryMissingConfigReturnsError(t *testing.T) {
	r := NewRegistry()
	if _, e := r.Get("missing"); e == nil {
		t.Fatal("missing config accepted")
	}
}
