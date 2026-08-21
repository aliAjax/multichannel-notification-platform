package rate

import (
	"errors"
	"sync"
	"time"
)

type Quota struct {
	TenantID     string    `json:"tenant_id"`
	DailyLimit   int64     `json:"daily_limit"`
	MonthlyLimit int64     `json:"monthly_limit"`
	BurstLimit   int64     `json:"burst_limit"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Usage struct {
	TenantID     string    `json:"tenant_id"`
	Day          string    `json:"day"`
	Month        string    `json:"month"`
	DailyUsed    int64     `json:"daily_used"`
	MonthlyUsed  int64     `json:"monthly_used"`
	Reserved     int64     `json:"reserved"`
	Committed    int64     `json:"committed"`
	LastModified time.Time `json:"last_modified"`
}

type Reservation struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Committed bool      `json:"committed"`
}

type Ledger struct {
	mu           sync.Mutex
	quotas       map[string]Quota
	usage        map[string]Usage
	reservations map[string]Reservation
	clock        func() time.Time
}

func NewLedger() *Ledger {
	return &Ledger{
		quotas:       make(map[string]Quota),
		usage:        make(map[string]Usage),
		reservations: make(map[string]Reservation),
		clock:        time.Now,
	}
}

func (l *Ledger) Configure(q Quota) error {
	if q.TenantID == "" {
		return errors.New("tenant is required")
	}
	if q.DailyLimit < 1 || q.MonthlyLimit < q.DailyLimit || q.BurstLimit < 1 {
		return errors.New("invalid quota limits")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	q.UpdatedAt = l.clock().UTC()
	l.quotas[q.TenantID] = q
	return nil
}

func (l *Ledger) Reserve(id, tenant string, amount int64, ttl time.Duration) (Reservation, error) {
	if id == "" || tenant == "" || amount < 1 {
		return Reservation{}, errors.New("reservation id, tenant and positive amount required")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.reservations[id]; ok {
		return existing, nil
	}
	q, ok := l.quotas[tenant]
	if !ok {
		return Reservation{}, errors.New("quota not configured")
	}
	now := l.clock().UTC()
	l.expireLocked(now)
	u := l.currentUsageLocked(tenant, now)
	if u.DailyUsed+u.Reserved+amount > q.DailyLimit {
		return Reservation{}, errors.New("daily quota exceeded")
	}
	if u.MonthlyUsed+u.Reserved+amount > q.MonthlyLimit {
		return Reservation{}, errors.New("monthly quota exceeded")
	}
	if u.Reserved+amount > q.BurstLimit {
		return Reservation{}, errors.New("burst quota exceeded")
	}
	r := Reservation{ID: id, TenantID: tenant, Amount: amount, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	u.Reserved += amount
	u.LastModified = now
	l.usage[tenant] = u
	l.reservations[id] = r
	return r, nil
}

func (l *Ledger) Commit(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	r, ok := l.reservations[id]
	if !ok {
		return errors.New("reservation not found")
	}
	if r.Committed {
		return nil
	}
	now := l.clock().UTC()
	if now.After(r.ExpiresAt) {
		l.releaseLocked(r, now)
		return errors.New("reservation expired")
	}
	u := l.currentUsageLocked(r.TenantID, now)
	u.Reserved -= r.Amount
	u.DailyUsed += r.Amount
	u.MonthlyUsed += r.Amount
	u.Committed += r.Amount
	u.LastModified = now
	l.usage[r.TenantID] = u
	r.Committed = true
	l.reservations[id] = r
	return nil
}

func (l *Ledger) Rollback(id string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	r, ok := l.reservations[id]
	if !ok {
		return errors.New("reservation not found")
	}
	if r.Committed {
		return errors.New("committed reservation cannot be rolled back")
	}
	now := l.clock().UTC()
	l.releaseLocked(r, now)
	return nil
}

func (l *Ledger) Usage(tenant string) Usage {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock().UTC()
	l.expireLocked(now)
	return l.currentUsageLocked(tenant, now)
}

func (l *Ledger) currentUsageLocked(tenant string, now time.Time) Usage {
	u := l.usage[tenant]
	day := now.Format("2006-01-02")
	month := now.Format("2006-01")
	if u.Day != day {
		u.DailyUsed = 0
		u.Day = day
	}
	if u.Month != month {
		u.MonthlyUsed = 0
		u.Month = month
	}
	u.TenantID = tenant
	return u
}

func (l *Ledger) expireLocked(now time.Time) {
	for _, r := range l.reservations {
		if !r.Committed && !now.Before(r.ExpiresAt) {
			l.releaseLocked(r, now)
		}
	}
}

func (l *Ledger) releaseLocked(r Reservation, now time.Time) {
	u := l.currentUsageLocked(r.TenantID, now)
	u.Reserved -= r.Amount
	if u.Reserved < 0 {
		u.Reserved = 0
	}
	u.LastModified = now
	l.usage[r.TenantID] = u
	delete(l.reservations, r.ID)
}
