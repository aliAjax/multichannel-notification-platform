package rate

import (
	"sync"
	"time"
)

type Bucket struct {
	mu                       sync.Mutex
	tokens, capacity, refill float64
	last                     time.Time
}

func New(capacity, perSecond float64) *Bucket {
	return &Bucket{tokens: capacity, capacity: capacity, refill: perSecond, last: time.Now()}
}
func (b *Bucket) Allow(n float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * b.refill
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens < n {
		return false
	}
	b.tokens -= n
	return true
}
func (b *Bucket) Wait(n float64) time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens += now.Sub(b.last).Seconds() * b.refill
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens >= n {
		return 0
	}
	needed := n - b.tokens
	return time.Duration(needed / b.refill * float64(time.Second))
}
