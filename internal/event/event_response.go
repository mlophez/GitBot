package event

import "fmt"

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

// Validate reports whether the response is structurally coherent: it never carries
// a non-nil Environments (help reply) together with a Summary (operation result),
// as those are mutually exclusive modes. It does not require any field to be
// present, so all the responses the use case builds today pass; the check only
// guards against future corruption. It is called by the worker right before the
// response is rendered into a comment.
func (r EventResponse) Validate() error {
	if r.Environments != nil && len(r.Summary) > 0 {
		return fmt.Errorf("event response mixes help and operation modes")
	}
	return nil
}

// EventAppStatus describes the outcome of a lock or unlock operation on a single app.
type EventAppStatus struct {
	Name    string
	Cluster string
	Message string
}
