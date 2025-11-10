package errors

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrWrongPassword     = errors.New("wrong password")
	ErrInvalidToken      = errors.New("token invalid")
	ErrExpiredToken      = errors.New("token expired")
	ErrInternal          = errors.New("internal error")
)
