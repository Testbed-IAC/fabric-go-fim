package userdata

import "errors"

var (
	// ErrInvalid indicates malformed user-data JSON or an invalid envelope element.
	ErrInvalid = errors.New("invalid user-data")
	// ErrTooLarge indicates the encoded envelope exceeds MaxBytes.
	ErrTooLarge = errors.New("user-data too large")
)
