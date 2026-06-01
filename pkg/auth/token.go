// Package auth provides FABRIC token sources and claim parsing.
package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

const refreshSkew = 5 * time.Minute

// ErrTokenExpired indicates a FABRIC token has expired.
var ErrTokenExpired = errors.New("fabric auth: token expired")

// TokenSource supplies FABRIC ID tokens and project metadata.
type TokenSource interface {
	IDToken(ctx context.Context) (string, error)
	ProjectID() string
	Claims() *Claims
}

// Project describes one FABRIC project claim.
type Project struct {
	Name string   `json:"name"`
	UUID string   `json:"uuid"`
	Tags []string `json:"tags"`
}

// Claims contains the FABRIC JWT claims consumed by tooling.
type Claims struct {
	Projects []Project `json:"projects"`
	Exp      int64     `json:"exp"`
}

// Project returns the first project claim, or a zero value when absent.
func (c *Claims) Project() Project {
	if c == nil || len(c.Projects) == 0 {
		return Project{}
	}
	return c.Projects[0]
}

// ProjectID returns the first project UUID.
func (c *Claims) ProjectID() string {
	return c.Project().UUID
}

// HasTag reports whether the first project has tag.
func (c *Claims) HasTag(tag string) bool {
	if c == nil || tag == "" {
		return false
	}
	for _, have := range c.Project().Tags {
		if have == tag {
			return true
		}
	}
	return false
}

// ExpiresAt returns the token expiration time from exp.
func (c *Claims) ExpiresAt() time.Time {
	if c == nil || c.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(c.Exp, 0)
}

// TokenFile is the FABRIC portal token JSON shape.
type TokenFile struct {
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	State        string `json:"state,omitempty"`
	TokenHash    string `json:"token_hash,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	CreatedFrom  string `json:"created_from,omitempty"`
}

// StaticToken is a token source backed by one in-memory ID token.
type StaticToken struct {
	token  string
	claims *Claims
}

// NewStaticToken returns a static token source after parsing token claims.
func NewStaticToken(token string) (*StaticToken, error) {
	claims, err := ParseJWT(token)
	if err != nil {
		return nil, fmt.Errorf("parsing static token: %w", err)
	}
	return &StaticToken{token: token, claims: claims}, nil
}

// IDToken returns the configured ID token if it is still valid.
func (s *StaticToken) IDToken(context.Context) (string, error) {
	if err := validateTokenTime(s.claims, false); err != nil {
		return "", err
	}
	return s.token, nil
}

// ProjectID returns the token's first project UUID.
func (s *StaticToken) ProjectID() string {
	return s.claims.ProjectID()
}

// Claims returns parsed token claims.
func (s *StaticToken) Claims() *Claims {
	return s.claims
}

// FileToken is a token source backed by a FABRIC portal token file.
type FileToken struct {
	path       string
	credmgrURL string
	httpClient *http.Client
	mu         sync.Mutex
	tokenFile  TokenFile
	claims     *Claims
}

// NewFileToken returns a file-backed token source with automatic refresh.
func NewFileToken(path, credmgrURL string, httpClient *http.Client) (*FileToken, error) {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	tokenFile, err := ParseTokenFile(path)
	if err != nil {
		return nil, err
	}
	claims, err := ParseJWT(tokenFile.IDToken)
	if err != nil {
		return nil, fmt.Errorf("parsing token file id_token: %w", err)
	}
	return &FileToken{
		path:       path,
		credmgrURL: credmgrURL,
		httpClient: httpClient,
		tokenFile:  tokenFile,
		claims:     claims,
	}, nil
}

// IDToken returns a valid ID token, refreshing and rewriting the token file when needed.
func (f *FileToken) IDToken(ctx context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if err := validateTokenTime(f.claims, true); err != nil && !errors.Is(err, ErrTokenExpired) {
		return "", err
	}
	if time.Until(f.claims.ExpiresAt()) > refreshSkew {
		return f.tokenFile.IDToken, nil
	}
	if f.tokenFile.RefreshToken == "" {
		return "", fmt.Errorf("refreshing token file %s: missing refresh_token", f.path)
	}
	refreshed, err := refreshToken(ctx, f.httpClient, f.credmgrURL, f.tokenFile.RefreshToken, f.claims.ProjectID(), f.claims.Project().Name, "all")
	if err != nil {
		return "", fmt.Errorf("refreshing token file %s: %w", f.path, err)
	}
	claims, err := ParseJWT(refreshed.IDToken)
	if err != nil {
		return "", fmt.Errorf("parsing refreshed id_token: %w", err)
	}
	body, err := json.MarshalIndent(refreshed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshalling refreshed token file: %w", err)
	}
	if err := os.WriteFile(f.path, append(body, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("writing refreshed token file %s: %w", f.path, err)
	}
	f.tokenFile = refreshed
	f.claims = claims
	return f.tokenFile.IDToken, nil
}

// ProjectID returns the first project UUID.
func (f *FileToken) ProjectID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.claims.ProjectID()
}

// Claims returns a deep copy of the parsed claims.
func (f *FileToken) Claims() *Claims {
	f.mu.Lock()
	defer f.mu.Unlock()
	claimsCopy := *f.claims
	claimsCopy.Projects = append([]Project(nil), f.claims.Projects...)
	for i := range claimsCopy.Projects {
		claimsCopy.Projects[i].Tags = append([]string(nil), f.claims.Projects[i].Tags...)
	}
	return &claimsCopy
}

// ParseTokenFile parses a FABRIC portal token JSON file.
func ParseTokenFile(path string) (TokenFile, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return TokenFile{}, fmt.Errorf("reading token file %s: %w", path, err)
	}
	var out TokenFile
	if err := json.Unmarshal(body, &out); err != nil {
		return TokenFile{}, fmt.Errorf("parsing token file %s: %w", path, err)
	}
	if out.IDToken == "" {
		return TokenFile{}, fmt.Errorf("parsing token file %s: missing id_token", path)
	}
	return out, nil
}

// ParseJWT parses the FABRIC claims from a JWT without verifying its signature.
func ParseJWT(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("jwt: malformed token: expected 3 dot-separated segments, got %d", len(parts))
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, fmt.Errorf("jwt: malformed token: decoding payload: %w", err)
		}
	}
	var claims Claims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, fmt.Errorf("jwt: malformed token: parsing payload: %w", err)
	}
	return &claims, nil
}

func validateTokenTime(claims *Claims, refreshable bool) error {
	expiresAt := claims.ExpiresAt()
	if expiresAt.IsZero() {
		return nil
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		if refreshable {
			return ErrTokenExpired
		}
		return fmt.Errorf("%w: request a fresh token from https://portal.fabric-testbed.net/experiments#tokens", ErrTokenExpired)
	}
	return nil
}

func refreshToken(ctx context.Context, client *http.Client, credmgrURL, refreshToken, projectID, projectName, scope string) (TokenFile, error) {
	endpoint, err := refreshEndpoint(credmgrURL)
	if err != nil {
		return TokenFile{}, err
	}
	body := map[string]string{
		"refresh_token": refreshToken,
		"scope":         scope,
	}
	if projectID != "" {
		body["project_id"] = projectID
	} else if projectName != "" {
		body["project_name"] = projectName
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return TokenFile{}, fmt.Errorf("marshalling refresh request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return TokenFile{}, fmt.Errorf("building refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return TokenFile{}, fmt.Errorf("calling credmgr refresh: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenFile{}, fmt.Errorf("credmgr refresh returned HTTP %d", resp.StatusCode)
	}
	var refreshed TokenFile
	if err := json.NewDecoder(resp.Body).Decode(&refreshed); err != nil {
		return TokenFile{}, fmt.Errorf("decoding credmgr refresh response: %w", err)
	}
	if refreshed.IDToken == "" {
		return TokenFile{}, errors.New("credmgr refresh response missing id_token")
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = refreshToken
	}
	return refreshed, nil
}

func refreshEndpoint(rawURL string) (string, error) {
	if rawURL == "" {
		rawURL = "cm.fabric-testbed.net"
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing credmgr_url: %w", err)
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/credmgr/tokens/refresh"
	return parsed.String(), nil
}
