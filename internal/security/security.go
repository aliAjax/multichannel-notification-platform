package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

func HashAddress(tenant, address string) string {
	h := sha256.Sum256([]byte(tenant + ":" + strings.ToLower(strings.TrimSpace(address))))
	return hex.EncodeToString(h[:])
}
func Sign(secret string, timestamp int64, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(formatInt(timestamp)))
	m.Write([]byte("."))
	m.Write(body)
	return hex.EncodeToString(m.Sum(nil))
}
func Verify(secret, signature string, timestamp int64, body []byte, maxSkew time.Duration) error {
	if time.Since(time.Unix(timestamp, 0)) > maxSkew {
		return errors.New("receipt timestamp outside replay window")
	}
	got, err := hex.DecodeString(signature)
	if err != nil {
		return errors.New("invalid signature encoding")
	}
	expected, _ := hex.DecodeString(Sign(secret, timestamp, body))
	if !hmac.Equal(got, expected) {
		return errors.New("invalid receipt signature")
	}
	return nil
}
func Redact(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + "****" + value[len(value)-2:]
}
func formatInt(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
