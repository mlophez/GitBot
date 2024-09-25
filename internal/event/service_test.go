package event

import (
	"fmt"
	"testing"
)

func TestGetAction(t *testing.T) {
	e := Event{
		Type:    EventTypeCommented,
		Comment: "#argo deploy dev",
		//CommentId   int
	}
	action := getActionFromEvent(e, "demo")

	fmt.Printf("%d\n\n", action)
}
