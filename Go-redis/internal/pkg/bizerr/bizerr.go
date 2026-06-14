package bizerr

import "errors"

type Error struct {
	message string
}

func (e *Error) Error() string {
	return e.message
}

func New(message string) error {
	return &Error{message: message}
}

func Is(err error) bool {
	var target *Error
	return errors.As(err, &target)
}
