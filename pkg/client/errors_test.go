package client

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
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

func TestTruncate(t *testing.T) {
	t.Parallel()
	if got, want := truncate("abcdefghij", 5), "abcde...(truncated)"; got != want {
		t.Fatalf("truncate = %q, want %q", got, want)
	}
}
