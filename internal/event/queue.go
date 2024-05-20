package event

import (
	"gitbot/internal/event/queue"
	. "gitbot/types"
)

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

func NewMemoryQueue() *queue.MemoryQueue[QueueItem] {
	return &queue.MemoryQueue[QueueItem]{}
}
