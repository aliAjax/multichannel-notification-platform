package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestConsoleRouteServesStaticPage(t *testing.T) {
	d := t.TempDir()
	if err := os.WriteFile(filepath.Join(d, "index.html"), []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	buildHandler(http.NotFoundHandler(), d).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/console/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}
