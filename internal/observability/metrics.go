package observability

import (
	"fmt"
	"sync"
	"time"
)

type Counter struct {
	mu    sync.Mutex
	value uint64
}

func (c *Counter) Inc()          { c.mu.Lock(); c.value++; c.mu.Unlock() }
func (c *Counter) Add(n uint64)  { c.mu.Lock(); c.value += n; c.mu.Unlock() }
func (c *Counter) Value() uint64 { c.mu.Lock(); defer c.mu.Unlock(); return c.value }

type Histogram struct {
	mu      sync.Mutex
	samples []time.Duration
	head    int
}

func (h *Histogram) Observe(d time.Duration) {
	h.mu.Lock()
	if len(h.samples) < 10000 {
		h.samples = append(h.samples, d)
	} else {
		h.samples[h.head] = d
		h.head++
		if h.head >= len(h.samples) {
			h.head = 0
		}
	}
	h.mu.Unlock()
}
func (h *Histogram) Snapshot() map[string]time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.samples) == 0 {
		return map[string]time.Duration{}
	}
	copySamples := append([]time.Duration(nil), h.samples...)
	sortDurations(copySamples)
	return map[string]time.Duration{"p50": copySamples[len(copySamples)/2], "p95": copySamples[len(copySamples)*95/100], "p99": copySamples[len(copySamples)*99/100]}
}
func sortDurations(v []time.Duration) {
	for i := 1; i < len(v); i++ {
		for j := i; j > 0 && v[j] < v[j-1]; j-- {
			v[j], v[j-1] = v[j-1], v[j]
		}
	}
}

type Registry struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	histograms map[string]*Histogram
}

func NewRegistry() *Registry {
	return &Registry{counters: map[string]*Counter{}, histograms: map[string]*Histogram{}}
}
func (r *Registry) Counter(name string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &Counter{}
	r.counters[name] = c
	return c
}
func (r *Registry) Histogram(name string) *Histogram {
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.histograms[name]; ok {
		return h
	}
	h := &Histogram{}
	r.histograms[name] = h
	return h
}
func (r *Registry) Prometheus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := ""
	for n, c := range r.counters {
		out += fmt.Sprintf("notification_%s %d\n", n, c.Value())
	}
	return out
}
