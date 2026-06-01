package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type contentTypeFixTransport struct {
	rt http.RoundTripper
}

func (t contentTypeFixTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.rt.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	contentType := resp.Header.Get("Content-Type")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp, nil
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "text/html") {
		return resp, nil
	}
	body, readErr := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); readErr == nil {
		readErr = closeErr
	}
	if readErr != nil {
		resp.Body = io.NopCloser(bytes.NewReader(nil))
		return resp, readErr
	}
	if json.Valid(body) {
		resp.Header.Set("Content-Type", "application/json")
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, nil
}

// WithContentTypeFix wraps c's transport so mislabelled text/html-but-JSON
// orchestrator responses get a Content-Type rewrite before decoding.
func WithContentTypeFix(c *http.Client) *http.Client {
	if c == nil {
		c = &http.Client{}
	}
	rt := c.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	c.Transport = contentTypeFixTransport{rt: rt}
	return c
}
