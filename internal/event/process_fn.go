package event

// ProcessFn is the function signature for processing a single event.
// Returns the response to post back to the PR, and a bool indicating whether
// the event should be retried (e.g. for app-of-apps double-lock).
type ProcessFn func(Event) (*EventResponse, bool)
