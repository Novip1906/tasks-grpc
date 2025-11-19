package models

import "time"

type Task struct {
	Id         int64
	Text       string
	AuthorName string
	CreatedAt  time.Time
}
