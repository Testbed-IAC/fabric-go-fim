package client

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	openapi "github.com/Testbed-IAC/fabric-orchestrator-go-client"
)

var (
	// ErrNotFound indicates an orchestrator 404 response.
	ErrNotFound = errors.New("fabric client: resource not found (404)")
	// ErrUnauthorized indicates an orchestrator 401 response.
	ErrUnauthorized = errors.New("fabric client: unauthorized - check your FABRIC token (401)")
	// ErrForbidden indicates an orchestrator 403 response.
	ErrForbidden = errors.New("fabric client: forbidden: check project permissions (403)")
	// ErrBadRequest indicates an orchestrator 400 response.
	ErrBadRequest = errors.New("fabric client: bad request - GraphML or parameters rejected by orchestrator (400)")
	// ErrServerError indicates an orchestrator 500 response.
	ErrServerError = errors.New("fabric client: orchestrator internal server error (500)")
)

func mapHTTPErr(httpResp *http.Response, err error) error {
	if err == nil {
		return nil
	}
	body := ""
	var apiErr *openapi.GenericOpenAPIError
	if errors.As(err, &apiErr) {
		body = string(apiErr.Body())
	}
	if body == "" && httpResp != nil && httpResp.Body != nil {
		respBody, readErr := io.ReadAll(httpResp.Body)
		if readErr == nil {
			body = string(respBody)
		}
	}
	// Prefer the decoded error entries (stashed in Model()) over the raw body.
	detail := structuredDetail(apiErr)
	if detail == "" {
		detail = body
	}
	if httpResp != nil {
		switch httpResp.StatusCode {
		case http.StatusUnauthorized:
			return fmt.Errorf("%w: %s", ErrUnauthorized, truncate(detail, 300))
		case http.StatusForbidden:
			return fmt.Errorf("%w: %s", ErrForbidden, truncate(detail, 300))
		case http.StatusNotFound:
			return fmt.Errorf("%w: %s", ErrNotFound, truncate(detail, 300))
		case http.StatusBadRequest:
			return fmt.Errorf("%w: %s", ErrBadRequest, truncate(detail, 300))
		case http.StatusInternalServerError:
			// 500s carry the orchestrator's traceback; keep enough to diagnose.
			return fmt.Errorf("%w: %s", ErrServerError, truncate(detail, 4000))
		default:
			if httpResp.StatusCode >= 300 {
				return fmt.Errorf("orchestrator returned HTTP %d: %s",
					httpResp.StatusCode, truncate(detail, 300))
			}
		}
	}
	if err.Error() == "undefined response type" {
		if httpResp == nil {
			return fmt.Errorf("orchestrator client could not parse response - "+
				"this usually means the request never reached the orchestrator "+
				"(check orchestrator_url) or auth failed before a response was sent: %w", err)
		}
		contentType := httpResp.Header.Get("Content-Type")
		bodyPart := "<empty body>"
		if body != "" {
			bodyPart = truncate(body, 500)
		}
		return fmt.Errorf("orchestrator returned a response the generated client "+
			"could not decode (HTTP %d, Content-Type %q). The body usually tells "+
			"you what happened - an HTML login page means the token expired or "+
			"orchestrator_url is wrong; a plain-text gateway error means a proxy/LB "+
			"is in between; a JSON shape the client does not know means the API "+
			"schema drifted. Body: %s",
			httpResp.StatusCode, contentType, bodyPart)
	}
	return fmt.Errorf("orchestrator error: %w", err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}

// errorEntry is the shape shared by every generated StatusNNNErrors model.
type errorEntry interface {
	GetMessage() string
	GetDetails() string
}

// structuredDetail joins the message/details pairs decoded from an error
// response. Returns "" when no structured model is available.
func structuredDetail(apiErr *openapi.GenericOpenAPIError) string {
	if apiErr == nil {
		return ""
	}
	switch m := apiErr.Model().(type) {
	case openapi.Status400BadRequest:
		return joinEntries(m.Errors)
	case openapi.Status401Unauthorized:
		return joinEntries(m.Errors)
	case openapi.Status403Forbidden:
		return joinEntries(m.Errors)
	case openapi.Status404NotFound:
		return joinEntries(m.Errors)
	case openapi.Status500InternalServerError:
		return joinEntries(m.Errors)
	default:
		return ""
	}
}

// joinEntries renders decoded error entries as "message: details" fragments.
// The generated getters have pointer receivers, hence the pointer constraint.
func joinEntries[T any, P interface {
	*T
	errorEntry
}](entries []T) string {
	parts := make([]string, 0, len(entries))
	for i := range entries {
		entry := P(&entries[i])
		message, details := entry.GetMessage(), entry.GetDetails()
		switch {
		case message != "" && details != "" && message != details:
			parts = append(parts, message+": "+details)
		case details != "":
			parts = append(parts, details)
		case message != "":
			parts = append(parts, message)
		}
	}
	return strings.Join(parts, "; ")
}
