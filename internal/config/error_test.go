package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigRejectsInvalidProviderTimeout(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c")
	if e := os.WriteFile(p, []byte("provider_timeout: nope\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(p); e == nil {
		t.Fatal("invalid timeout accepted")
	}
}
func TestBadGracePeriodConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c")
	if e := os.WriteFile(p, []byte("shutdown_timeout: nope\n"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := Load(p); e == nil {
		t.Fatal("invalid shutdown timeout accepted")
	}
}
