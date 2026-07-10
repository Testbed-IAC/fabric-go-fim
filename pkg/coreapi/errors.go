package coreapi

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrNoRecord indicates the Core API returned an empty results list where a
	// record was expected (unknown user, or a token without a person record).
	ErrNoRecord = errors.New("fabric coreapi: no record returned")
	// ErrNoBastionLogin indicates the caller's person record has no bastion
	// login assigned. FABRIC assigns one after bastion keys are generated on
	// the portal.
	ErrNoBastionLogin = errors.New("fabric coreapi: no bastion login for this account - generate bastion keys on the FABRIC portal, then retry")
	// ErrUnauthorized indicates a Core API 401 response.
	ErrUnauthorized = errors.New("fabric coreapi: unauthorized - check your FABRIC token (401)")
	// ErrForbidden indicates a Core API 403 response.
	ErrForbidden = errors.New("fabric coreapi: forbidden - check account permissions (403)")
	// ErrNotFound indicates a Core API 404 response.
	ErrNotFound = errors.New("fabric coreapi: resource not found (404)")
)

// httpError maps a non-2xx Core API response to a typed error, preserving a
// bounded slice of the response body for diagnosis.
func httpError(statusCode int, body string) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrUnauthorized, truncate(body, 300))
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrForbidden, truncate(body, 300))
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrNotFound, truncate(body, 300))
	default:
		return fmt.Errorf("fabric coreapi: unexpected HTTP %d: %s", statusCode, truncate(body, 500))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
