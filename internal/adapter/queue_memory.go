package adapter

import (
	"sync"
)

// MemoryQueue is a generic thread-safe in-memory FIFO queue.
// It implements types.Queue for any item type T.
type MemoryQueue[T any] struct {
	items []interface{}
	mu    sync.Mutex
}

// NewMemoryQueue creates an empty MemoryQueue ready for use.
func NewMemoryQueue[T any]() *MemoryQueue[T] {
	return &MemoryQueue[T]{}
}

// Enqueue adds item to the back of the queue.
func (q *MemoryQueue[T]) Enqueue(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
}

// NextItem returns a pointer to the front item without removing it, or nil if empty.
func (q *MemoryQueue[T]) NextItem() *T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	if item, ok := q.items[0].(T); ok {
		return &item
	}
	return nil
}

// Dequeue removes and returns the front item, or nil if the queue is empty.
func (q *MemoryQueue[T]) Dequeue() *T {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	if item, ok := q.items[0].(T); ok {
		q.items = q.items[1:]
		return &item
	}
	return nil
}

// Size returns the number of items currently in the queue.
func (q *MemoryQueue[T]) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// IsEmpty reports whether the queue has no items.
func (q *MemoryQueue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}
