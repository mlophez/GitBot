package event

// Queue is the interface for the event processing queue.
// Implemented by queue.MemoryQueue for in-memory use.
type Queue interface {
	Enqueue(item QueueItem)
	NextItem() *QueueItem
	Dequeue() *QueueItem
	Size() int
}
