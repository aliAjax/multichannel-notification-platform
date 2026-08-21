package security

import (
	"testing"
	"time"
)

func TestFutureReceiptTimeDenied(t *testing.T) {
	ts := time.Now().Add(time.Hour).Unix()
	sig := Sign("secret", ts, []byte("body"))
	if e := Verify("secret", sig, ts, []byte("body"), time.Minute); e == nil {
		t.Fatal("future timestamp accepted")
	}
}
func TestEmptyReceiptSecretDenied(t *testing.T) {
	ts := time.Now().Unix()
	if e := Verify("", Sign("", ts, []byte("body")), ts, []byte("body"), time.Minute); e == nil {
		t.Fatal("empty secret accepted")
	}
}
func TestMissingReceiptTimeDenied(t *testing.T) {
	if e := Verify("secret", Sign("secret", 0, []byte("body")), 0, []byte("body"), time.Minute); e == nil {
		t.Fatal("missing timestamp accepted")
	}
}
