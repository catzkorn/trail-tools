package store

import "errors"

// ErrNotFound is returned when the resource could not be found.
// It should only be used for Get or Update operations.
var ErrNotFound = errors.New("not found")
