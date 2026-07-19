package domain

import "errors"

var (
	// ErrValidation is returned when input validation fails.
	ErrValidation = errors.New("validation error")
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when a resource cannot be created due to a conflict.
	ErrConflict = errors.New("conflict")
)
