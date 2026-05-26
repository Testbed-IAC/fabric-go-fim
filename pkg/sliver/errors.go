package sliver

import "errors"

// ErrInvalidValue indicates a property value violates the ASM schema.
var ErrInvalidValue = errors.New("sliver: invalid value")

// ErrInvalidJSON indicates a JSON-valued property could not be decoded.
var ErrInvalidJSON = errors.New("sliver: invalid json property")

// ErrMissingProperty indicates a required property is absent.
var ErrMissingProperty = errors.New("sliver: missing required property")
