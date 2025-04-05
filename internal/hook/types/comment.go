package types


type CommentStatus int

const (
	CommentCreated HookType = iota
	CommentUpdated
	CommentDeleted
)


type Comment struct {
	id int
	message string
	status CommentStatus
}

