package event

import (
	"io"
	"net/http"
)

// Provider is the interface for a Git hosting provider (Bitbucket, GitHub, etc.).
// It handles webhook parsing, event enrichment, and writing comments back to PRs.
// Implemented by BitbucketClient (same package) for the Bitbucket Cloud API.
type Provider interface {
	// ValidateWebhookToken verifies the webhook signature sent by the provider.
	// secret is the shared token configured via WEBHOOK_TOKEN.
	// body is the raw request body, needed for HMAC-based validation.
	// Returns nil if validation passes or if secret is empty.
	ValidateWebhookToken(secret string, headers http.Header, body []byte) error

	// ParseEvent reads the raw HTTP webhook and returns a structured, validated Event.
	// Parsing is the DTO→domain transformation for the slice, so implementations must
	// validate the event at this creation point and return the error; the EventCreate
	// use case maps it to a 400 response.
	ParseEvent(headers http.Header, body io.ReadCloser) (Event, error)

	// GetData enriches an event with additional data fetched from the provider API
	// (files changed, commits behind, etc.).
	GetData(Event) (Event, error)

	// WriteComment posts a comment on the pull request. If parentId > 0 the comment
	// is posted as a reply to that comment thread.
	WriteComment(repo string, prId int, parentId int, msg string) error

	// WriteEventResponse formats resp into a provider-specific comment and posts it
	// on the pull request. clusterName is used as a display label when the response
	// does not carry cluster information (single-cluster deployments).
	WriteEventResponse(repo string, prId int, parentId int, resp *EventResponse, clusterName string) error
}
