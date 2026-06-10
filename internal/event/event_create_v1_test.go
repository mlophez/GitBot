package event

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubProvider is a Provider test double whose ParseEvent returns a preset event
// (or a preset error), letting the EventCreate use case be exercised without a real
// webhook payload. The error models a validation failure raised at the creation
// point inside a real provider's ParseEvent.
type stubProvider struct {
	parsed   Event
	parseErr error
}

func (s stubProvider) ValidateWebhookToken(secret string, headers http.Header, body []byte) error {
	return nil
}
func (s stubProvider) ParseEvent(headers http.Header, body io.ReadCloser) (Event, error) {
	return s.parsed, s.parseErr
}
func (s stubProvider) GetData(e Event) (Event, error) { return e, nil }
func (s stubProvider) WriteComment(repo string, prId int, parentId int, msg string) error {
	return nil
}
func (s stubProvider) WriteEventResponse(repo string, prId int, parentId int, resp *EventResponse, clusterName string) error {
	return nil
}

// stubQueue is a Queue test double that only counts enqueued items.
type stubQueue struct{ enqueued int }

func (q *stubQueue) Enqueue(item QueueItem) { q.enqueued++ }
func (q *stubQueue) NextItem() *QueueItem   { return nil }
func (q *stubQueue) Dequeue() *QueueItem    { return nil }
func (q *stubQueue) Size() int              { return q.enqueued }

// newWebhookRequest builds a minimal POST request to the Bitbucket webhook endpoint
// with a valid-enough body to pass io.ReadAll; actual parsing is done by the stub.
func newWebhookRequest() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/api/v1/webhook/bitbucket", strings.NewReader("{}"))
}

// TestEventCreatePropagatesParseError checks the use case contract: when the
// provider's ParseEvent fails (e.g. the event failed validation at its creation
// point), EventCreate answers 400 Bad Request and does not enqueue anything; when
// ParseEvent succeeds, the event is enqueued and answered with 200.
func TestEventCreatePropagatesParseError(t *testing.T) {
	t.Run("parse/validation error returns 400 and is not enqueued", func(t *testing.T) {
		provider := stubProvider{parseErr: errors.New("invalid event")}
		queue := &stubQueue{}

		rec := httptest.NewRecorder()
		EventCreate(queue, provider, "")(rec, newWebhookRequest())

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
		if queue.enqueued != 0 {
			t.Errorf("enqueued = %d, want 0", queue.enqueued)
		}
	})

	t.Run("valid event is enqueued with 200", func(t *testing.T) {
		provider := stubProvider{parsed: Event{Type: EventTypeOpened, Repository: "https://bitbucket.org/org/repo.git", PullRequest: PullRequest{Id: 5}}}
		queue := &stubQueue{}

		rec := httptest.NewRecorder()
		EventCreate(queue, provider, "")(rec, newWebhookRequest())

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
		if queue.enqueued != 1 {
			t.Errorf("enqueued = %d, want 1", queue.enqueued)
		}
	})
}
