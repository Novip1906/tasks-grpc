package models

type EmailVerificationMessage struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Username string `json:"username,omitempty"`
}
