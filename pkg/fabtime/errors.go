package fabtime

import "errors"

// ErrInvalidTime indicates a timestamp is not a supported FABRIC time value.
var ErrInvalidTime = errors.New("invalid fabric time")
