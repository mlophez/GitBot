package event

import (
	. "gitbot/types"
	"io"
	"net/http"
)

type Provider interface {
	//Name() string
	ParseEvent(headers http.Header, body io.ReadCloser) (Event, error)
	WriteComment(repo string, prId int, parentId int, msg string) error
	//	RespondEvent(e GitEvent, msg string) error
}
