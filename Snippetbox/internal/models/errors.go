package models

import (
	"errors"
)

var (
	ErrNoRecord = errors.New("models:no matching record found")

	ErrIncalidCredentials = errors.New("models:invalid vredentials")

	ErrDuplicateEmail = errors.New("models:duplicate email")
)
