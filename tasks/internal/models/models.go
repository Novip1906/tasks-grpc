package models

import "time"

type Task struct {
	Id         int64
	Text       string
	AuthorName string
	CreatedAt  time.Time
}

type TokenClaims struct {
	UserId   int64  `json:"sub"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type EventMessage struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Type        string `json:"type"`
	TaskText    string `json:"task_text"`
	TaskOldText string `json:"task_old_text"`
}
