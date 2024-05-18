package event

import "gitbot/internal/event/queue"

type Queue interface {
	Enqueue(item QueueItem)
	NextItem() *QueueItem
	Dequeue() *QueueItem
	Size() int
}

type QueueItem struct {
	event    Event
	provider Provider
}

type QueueManager struct {
	queue Queue
}

func NewQueueManager() *QueueManager {
	return &QueueManager{
		queue: &queue.MemoryQueue[QueueItem]{},
	}
}

func NewMemoryQueue() *queue.MemoryQueue[QueueItem] {
	return &queue.MemoryQueue[QueueItem]{}
}
