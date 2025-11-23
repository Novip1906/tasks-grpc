package models

type EventMessage struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Type        string `json:"type"`
	TaskText    string `json:"task_text"`
	TaskOldText string `json:"task_old_text"`
}
