package queue

import (
	"container/heap"
	"errors"
	"example.com/notification-platform/internal/notification/domain"
	"sync"
	"time"
)

type Item struct {
	N        *domain.Notification
	index    int
	enqueued time.Time
}
type pq []*Item

func (p pq) Len() int { return len(p) }
func (p pq) Less(i, j int) bool {
	if p[i].N.Priority == p[j].N.Priority {
		return p[i].enqueued.Before(p[j].enqueued)
	}
	return p[i].N.Priority > p[j].N.Priority
}
func (p pq) Swap(i, j int) { p[i], p[j] = p[j], p[i]; p[i].index = i; p[j].index = j }
func (p *pq) Push(x any)   { it := x.(*Item); it.index = len(*p); *p = append(*p, it) }
func (p *pq) Pop() any     { old := *p; n := len(old); it := old[n-1]; *p = old[:n-1]; return it }

type Queue struct {
	mu    sync.Mutex
	items pq
	wake  chan struct{}
	max   int
}

func New(max int) *Queue {
	q := &Queue{wake: make(chan struct{}, 1), max: max}
	heap.Init(&q.items)
	return q
}
func (q *Queue) Enqueue(n *domain.Notification) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) >= q.max {
		return errors.New("queue capacity reached")
	}
	heap.Push(&q.items, &Item{N: n, enqueued: time.Now()})
	select {
	case q.wake <- struct{}{}:
	default:
	}
	return nil
}
func (q *Queue) Dequeue() (*domain.Notification, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil, false
	}
	return heap.Pop(&q.items).(*Item).N, true
}
func (q *Queue) Wake() <-chan struct{} { return q.wake }
func (q *Queue) Len() int              { q.mu.Lock(); defer q.mu.Unlock(); return len(q.items) }
