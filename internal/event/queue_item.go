package event

// QueueItem groups an event with the provider that generated it so the worker
// can use the same provider to enrich the event and write comments back.
type QueueItem struct {
	Event    Event
	Provider Provider
}
