package observability

import (
	"testing"
	"time"
)

func TestHistogramSnapshotIsImmutable(t *testing.T) {
	h := &Histogram{}
	h.Observe(9 * time.Second)
	h.Observe(time.Second)
	h.Observe(5 * time.Second)
	s := h.Snapshot()
	if s["p50"] != 5*time.Second {
		t.Fatalf("p50=%s", s["p50"])
	}
}
func TestHistogramRotatesBoundedSamples(t *testing.T) {
	h := &Histogram{}
	for i := 0; i < 10002; i++ {
		h.Observe(time.Duration(i))
	}
	if h.samples[1] == 1 {
		t.Fatal("bounded histogram never advanced replacement slot")
	}
}
