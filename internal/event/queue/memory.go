package queue

import (
	"sync"
)

type MemoryQueue[T any] struct {
	items []interface{}
	mu    sync.Mutex
}

func (q *MemoryQueue[T]) Enqueue(item T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = append(q.items, item)
}

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

func (q *MemoryQueue[T]) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *MemoryQueue[T]) IsEmpty() bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items) == 0
}
