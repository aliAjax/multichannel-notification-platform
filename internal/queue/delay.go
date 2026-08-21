package queue

import (
	"container/heap"
	"context"
	"example.com/notification-platform/internal/notification/domain"
	"sync"
	"time"
)

type delayedItem struct {
	notification *domain.Notification
	due          time.Time
	index        int
}
type delayedHeap []*delayedItem

func (h delayedHeap) Len() int           { return len(h) }
func (h delayedHeap) Less(i, j int) bool { return h[i].due.Before(h[j].due) }
func (h delayedHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i]; h[i].index = i; h[j].index = j }
func (h *delayedHeap) Push(x any)        { it := x.(*delayedItem); it.index = len(*h); *h = append(*h, it) }
func (h *delayedHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}

type DelayQueue struct {
	mu     sync.Mutex
	items  delayedHeap
	notify chan struct{}
	limit  int
}

func NewDelayQueue(limit int) *DelayQueue {
	q := &DelayQueue{notify: make(chan struct{}, 1), limit: limit}
	heap.Init(&q.items)
	return q
}
func (q *DelayQueue) Add(n *domain.Notification, due time.Time) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) >= q.limit {
		return false
	}
	heap.Push(&q.items, &delayedItem{notification: n, due: due})
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return true
}
func (q *DelayQueue) Run(ctx context.Context, target *Queue) {
	for {
		q.mu.Lock()
		if len(q.items) == 0 {
			q.mu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-q.notify:
				continue
			}
		}
		if len(q.items) == 0 {
			continue
		}
		item := q.items[0]
		wait := time.Until(item.due)
		if wait > 0 {
			q.mu.Unlock()
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-q.notify:
				timer.Stop()
				continue
			case <-timer.C:
				continue
			}
		}
		heap.Pop(&q.items)
		q.mu.Unlock()
		_ = target.Enqueue(item.notification)
	}
}
