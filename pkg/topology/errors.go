package topology

import "errors"

// ErrDuplicateName is returned when a name that must be unique within its
// scope (e.g., two components with the same name on the same node) is used
// more than once.
var ErrDuplicateName = errors.New("topology: duplicate name")

// ErrNotFound is returned when a lookup by name fails to find the requested
// topology element.
var ErrNotFound = errors.New("topology: not found")

// ErrInvalidOption is returned when a builder option value is syntactically or
// semantically invalid (e.g., a name that does not match the allowed pattern).
var ErrInvalidOption = errors.New("topology: invalid option")

// ErrConstraintViolation is returned when a valid option would violate an ASM
// construction constraint (e.g., connecting more interfaces than a service
// type allows, or requesting a cross-site service with same-site interfaces).
var ErrConstraintViolation = errors.New("topology: constraint violation")

// Diagnostic is an error that carries field-level detail for Terraform provider
// diagnostics.  All errors returned by topology builder methods implement this
// interface, so providers can display field-specific messages and suggestions
// directly in the Terraform plan output.
//
// Use errors.Is to test the underlying sentinel error:
//
//	var diag topology.Diagnostic
//	if errors.As(err, &diag) {
//	    fmt.Println("field:", diag.Field())
//	    fmt.Println("hint:", diag.Suggestion())
//	}
type Diagnostic interface {
	error
	// Field returns the name of the option field that caused the error
	// (e.g., "Name", "Site", "Interfaces").
	Field() string
	// Suggestion returns an optional human-readable hint to resolve the error.
	// It may be empty.
	Suggestion() string
}

type diagnosticError struct {
	err        error
	field      string
	suggestion string
}

func (e diagnosticError) Error() string      { return e.err.Error() }
func (e diagnosticError) Unwrap() error      { return e.err }
func (e diagnosticError) Field() string      { return e.field }
func (e diagnosticError) Suggestion() string { return e.suggestion }

// diagnostic wraps err in a diagnosticError with field-level metadata.
// All internal topology errors should be created through this function so that
// callers can always retrieve field and suggestion information.
func diagnostic(err error, field, suggestion string) error {
	return diagnosticError{err: err, field: field, suggestion: suggestion}
}
