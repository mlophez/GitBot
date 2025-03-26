package comment

import "errors"

type Comment struct {
	id         int
	message    string
	repository string
}

func NewComment(id int, message string) (*Comment, error) {
	if id < 1 {
		return nil, errors.New("id cannot be < 1")
	}
	if message == "" {
		return nil, errors.New("message cannot be empty")
	}
	if len(message) > 4096 {
		return nil, errors.New("message exceeds the maximum length of 4096 characters")
	}
	return &Comment{
		id:      id,
		message: message,
	}, nil
}

func (c Comment) ID() int {
	return c.id
}

func (c Comment) Message() string {
	return c.message
}

func (c Comment) Repository() string {
	return c.repository
}
