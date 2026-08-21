package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPSendUsesDefaultClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("X-Message-ID", "x"); w.WriteHeader(200) }))
	defer srv.Close()
	h := &HTTP{ProviderName: "p", URL: srv.URL}
	if _, e := h.Send(context.Background(), Request{Body: "ok"}); e != nil {
		t.Fatal(e)
	}
}
