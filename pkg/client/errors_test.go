package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/auth"
)

func responseWithBody(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}
}

func genericErr(message string) error {
	return errors.New(message)
}

func TestMapHTTPErrStatusSentinels(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		code int
		want error
	}{
		{name: "401", code: http.StatusUnauthorized, want: ErrUnauthorized},
		{name: "403", code: http.StatusForbidden, want: ErrForbidden},
		{name: "404", code: http.StatusNotFound, want: ErrNotFound},
		{name: "400", code: http.StatusBadRequest, want: ErrBadRequest},
		{name: "500", code: http.StatusInternalServerError, want: ErrServerError},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := mapHTTPErr(responseWithBody(tc.code, `{"errors":[{"details":"body"}]}`), genericErr("http error"))
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if !strings.Contains(err.Error(), "body") {
				t.Fatalf("err = %v, want response body", err)
			}
		})
	}
}

func TestMapHTTPErrOtherStatusIncludesCode(t *testing.T) {
	t.Parallel()
	err := mapHTTPErr(responseWithBody(http.StatusTeapot, "oops"), genericErr("weird status"))
	if err == nil || !strings.Contains(err.Error(), "HTTP") {
		t.Fatalf("err = %v, want HTTP code in message", err)
	}
}

func TestMapHTTPErrUndefinedResponseType(t *testing.T) {
	t.Parallel()
	in := errors.New("undefined response type")
	err := mapHTTPErr(nil, in)
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "orchestrator_url") {
		t.Fatalf("err = %v, want it to mention orchestrator_url", err)
	}
	if !errors.Is(err, in) {
		t.Fatalf("err = %v, should wrap original", err)
	}
}

func TestMapHTTPErrUndefinedResponseTypeWithHTTPRespIncludesBody(t *testing.T) {
	t.Parallel()
	resp := &http.Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
		Body:       io.NopCloser(strings.NewReader("<html><body>Please sign in</body></html>")),
	}
	err := mapHTTPErr(resp, genericErr("undefined response type"))
	if err == nil {
		t.Fatal("err = nil, want non-nil")
	}
	msg := err.Error()
	for _, sub := range []string{"HTTP 200", "text/html", "Please sign in"} {
		if !strings.Contains(msg, sub) {
			t.Fatalf("msg missing %q; full msg:\n%s", sub, msg)
		}
	}
}

// fakeTokenSource satisfies auth.TokenSource for httptest-backed client tests.
type fakeTokenSource struct{}

func (fakeTokenSource) IDToken(context.Context) (string, error) { return "test-token", nil }
func (fakeTokenSource) ProjectID() string                       { return "" }
func (fakeTokenSource) Claims() *auth.Claims                    { return nil }

func TestMapHTTPErrPrefersStructuredModel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		status      int
		body        string
		wantErr     error
		wantContain []string
		wantAbsent  []string
	}{
		{
			name:        "400 message and details joined",
			status:      http.StatusBadRequest,
			body:        `{"errors":[{"message":"invalid GraphML","details":"node vm1 has no site"}]}`,
			wantErr:     ErrBadRequest,
			wantContain: []string{"invalid GraphML: node vm1 has no site"},
			wantAbsent:  []string{`{"errors"`},
		},
		{
			name:        "403 message only",
			status:      http.StatusForbidden,
			body:        `{"errors":[{"message":"project lacks Component.FPGA tag"}]}`,
			wantErr:     ErrForbidden,
			wantContain: []string{"project lacks Component.FPGA tag"},
			wantAbsent:  []string{`{"errors"`},
		},
		{
			name:        "500 multiple entries joined",
			status:      http.StatusInternalServerError,
			body:        `{"errors":[{"message":"first"},{"message":"second"}]}`,
			wantErr:     ErrServerError,
			wantContain: []string{"first; second"},
		},
		{
			name:        "undecodable body falls back to raw text",
			status:      http.StatusBadRequest,
			body:        `plain failure text`,
			wantErr:     ErrBadRequest,
			wantContain: []string{"plain failure text"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			c := New(srv.URL, fakeTokenSource{})
			_, err := c.GetSlice(context.Background(), "3f4a7b1e-aaaa-bbbb-cccc-000000000000")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			msg := err.Error()
			for _, sub := range tc.wantContain {
				if !strings.Contains(msg, sub) {
					t.Fatalf("msg missing %q; full msg:\n%s", sub, msg)
				}
			}
			for _, sub := range tc.wantAbsent {
				if strings.Contains(msg, sub) {
					t.Fatalf("msg should not contain raw %q; full msg:\n%s", sub, msg)
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	if got, want := truncate("abcdefghij", 5), "abcde...(truncated)"; got != want {
		t.Fatalf("truncate = %q, want %q", got, want)
	}
}
