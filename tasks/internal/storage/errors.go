package storage

import "errors"

var (
	ErrTaskNotFound  = errors.New("task not found")
	ErrNotTaskAuthor = errors.New("user is not the task author")
)
