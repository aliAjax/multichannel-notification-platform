package rate

import (
	"testing"
	"time"
)

func TestRollbackReleasesReservation(t *testing.T) {
	l := NewLedger()
	_ = l.Configure(Quota{TenantID: "t", DailyLimit: 10, MonthlyLimit: 20, BurstLimit: 10})
	_, e := l.Reserve("r", "t", 5, time.Hour)
	if e != nil {
		t.Fatal(e)
	}
	if e = l.Rollback("r"); e != nil {
		t.Fatal(e)
	}
	if got := l.Usage("t").Reserved; got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}
func TestWaitReportsRemainingTokens(t *testing.T) {
	b := New(1, 1)
	if !b.Allow(1) {
		t.Fatal("initial token missing")
	}
	if got := b.Wait(1); got < 900*time.Millisecond {
		t.Fatalf("wait=%s", got)
	}
}
func TestDailyRolloverKeepsReservation(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewLedger()
	l.clock = func() time.Time { return now }
	_ = l.Configure(Quota{TenantID: "t", DailyLimit: 10, MonthlyLimit: 20, BurstLimit: 10})
	_, _ = l.Reserve("r", "t", 5, 48*time.Hour)
	now = now.Add(24 * time.Hour)
	if l.Usage("t").Reserved != 5 {
		t.Fatal("reservation lost at rollover")
	}
}
func TestExactExpiryReleasesReservation(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	l := NewLedger()
	l.clock = func() time.Time { return now }
	_ = l.Configure(Quota{TenantID: "t", DailyLimit: 10, MonthlyLimit: 20, BurstLimit: 10})
	_, _ = l.Reserve("r", "t", 5, time.Hour)
	now = now.Add(time.Hour)
	if l.Usage("t").Reserved != 0 {
		t.Fatal("exactly expired reservation retained")
	}
}
