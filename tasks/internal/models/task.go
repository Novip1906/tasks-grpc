package models

import "time"

type Task struct {
	Text       string
	AuthorName string
	CreatedAt  time.Time
}
