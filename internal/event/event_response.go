package event

// EventResponse holds the result of processing an event, including a success
// flag, an optional error message, and a per-app status summary.
// When Environments is non-nil the response is a help reply; the provider
// formats it as a help comment instead of an operation status.
type EventResponse struct {
	Success      bool
	Message      string
	Summary      []EventAppStatus
	Environments []string // non-nil → help response listing available environments
}

// EventAppStatus describes the outcome of a lock or unlock operation on a single app.
type EventAppStatus struct {
	Name    string
	Cluster string
	Message string
}
