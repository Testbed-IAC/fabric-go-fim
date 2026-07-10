package coreapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/auth"
)

// DefaultBaseURL is the FABRIC Core API (UIS) base URL.
const DefaultBaseURL = "https://uis.fabric-testbed.net"

// FABRIC SSH key types. A bastion key authenticates to the FABRIC bastion; a
// sliver key is installed on slice nodes.
const (
	KeyTypeBastion = "bastion"
	KeyTypeSliver  = "sliver"
)

// Client is a FABRIC Core API client. It adapts the raw Core API to a small,
// tool-friendly surface and reuses a pkg/auth token source for bearer auth.
type Client struct {
	baseURL    string
	ts         auth.TokenSource
	httpClient *http.Client
}

// New returns a Core API client. An empty baseURL selects DefaultBaseURL; a
// nil httpClient selects http.DefaultClient.
func New(baseURL string, ts auth.TokenSource, httpClient *http.Client) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		ts:         ts,
		httpClient: httpClient,
	}
}

// WhoAmI is the caller's identity as returned by GET /whoami.
type WhoAmI struct {
	UUID  string
	Email string
}

// Person is the subset of a Core API person record consumed by tooling.
type Person struct {
	UUID         string
	Email        string
	Name         string
	BastionLogin string
}

// Identity is the caller's resolved identity, including the bastion login used
// to SSH through the FABRIC bastion.
type Identity struct {
	UUID         string
	Email        string
	Name         string
	BastionLogin string
}

// WhoAmI returns the caller's UUID and email from GET /whoami.
func (c *Client) WhoAmI(ctx context.Context) (WhoAmI, error) {
	var out struct {
		Results []struct {
			UUID  string `json:"uuid"`
			Email string `json:"email"`
		} `json:"results"`
	}
	if err := c.get(ctx, "/whoami", &out); err != nil {
		return WhoAmI{}, err
	}
	if len(out.Results) == 0 {
		return WhoAmI{}, fmt.Errorf("whoami: %w", ErrNoRecord)
	}
	return WhoAmI{UUID: out.Results[0].UUID, Email: out.Results[0].Email}, nil
}

// Person returns the person record for uuid via GET /people/{uuid}?as_self=true.
// The as_self flag includes the caller's own private fields, notably
// bastion_login.
func (c *Client) Person(ctx context.Context, uuid string) (Person, error) {
	var out struct {
		Results []struct {
			UUID         string `json:"uuid"`
			Email        string `json:"email"`
			Name         string `json:"name"`
			BastionLogin string `json:"bastion_login"`
		} `json:"results"`
	}
	path := "/people/" + url.PathEscape(uuid) + "?as_self=true"
	if err := c.get(ctx, path, &out); err != nil {
		return Person{}, err
	}
	if len(out.Results) == 0 {
		return Person{}, fmt.Errorf("people/%s: %w", uuid, ErrNoRecord)
	}
	r := out.Results[0]
	return Person{UUID: r.UUID, Email: r.Email, Name: r.Name, BastionLogin: r.BastionLogin}, nil
}

// ResolveIdentity resolves the caller's full identity: WhoAmI for the UUID,
// then Person for the bastion login. It returns ErrNoBastionLogin when the
// account has no bastion login assigned.
func (c *Client) ResolveIdentity(ctx context.Context) (Identity, error) {
	who, err := c.WhoAmI(ctx)
	if err != nil {
		return Identity{}, err
	}
	person, err := c.Person(ctx, who.UUID)
	if err != nil {
		return Identity{}, err
	}
	if person.BastionLogin == "" {
		return Identity{}, ErrNoBastionLogin
	}
	email := person.Email
	if email == "" {
		email = who.Email
	}
	return Identity{
		UUID:         who.UUID,
		Email:        email,
		Name:         person.Name,
		BastionLogin: person.BastionLogin,
	}, nil
}

// SSHKey is a registered FABRIC SSH key as returned by GET /sshkeys. Only the
// fields consumed by tooling are surfaced.
type SSHKey struct {
	UUID          string
	Comment       string
	Description   string
	Fingerprint   string
	FabricKeyType string // "bastion" or "sliver"
	PublicKey     string // the stored public key (no comment)
	ExpiresOn     string // RFC3339; empty when not provided
}

// SSHKeyPair is a freshly generated keypair returned by POST /sshkeys. FABRIC
// generates the key server-side, registers the public half against the caller's
// account, and returns both halves once — the private key is not retrievable
// afterward, so callers must persist it.
type SSHKeyPair struct {
	PrivateOpenSSH string
	PublicOpenSSH  string
}

// SSHKeys lists the caller's active (non-expired) SSH keys via
// GET /sshkeys?person_uuid={uuid}.
func (c *Client) SSHKeys(ctx context.Context, personUUID string) ([]SSHKey, error) {
	var out struct {
		Results []struct {
			UUID          string `json:"uuid"`
			Comment       string `json:"comment"`
			Description   string `json:"description"`
			Fingerprint   string `json:"fingerprint"`
			FabricKeyType string `json:"fabric_key_type"`
			PublicKey     string `json:"public_key"`
			ExpiresOn     string `json:"expires_on"`
		} `json:"results"`
	}
	path := "/sshkeys?person_uuid=" + url.QueryEscape(personUUID)
	if err := c.get(ctx, path, &out); err != nil {
		return nil, err
	}
	keys := make([]SSHKey, 0, len(out.Results))
	for _, r := range out.Results {
		keys = append(keys, SSHKey{
			UUID: r.UUID, Comment: r.Comment, Description: r.Description,
			Fingerprint: r.Fingerprint, FabricKeyType: r.FabricKeyType,
			PublicKey: r.PublicKey, ExpiresOn: r.ExpiresOn,
		})
	}
	return keys, nil
}

// MySSHKeys resolves the caller's UUID (WhoAmI) and lists their active SSH keys.
func (c *Client) MySSHKeys(ctx context.Context) ([]SSHKey, error) {
	who, err := c.WhoAmI(ctx)
	if err != nil {
		return nil, err
	}
	return c.SSHKeys(ctx, who.UUID)
}

// CreateSSHKey generates a keypair server-side via POST /sshkeys, registering
// the public half against the caller's account under keyType ("bastion" or
// "sliver"). It returns the generated pair, whose private key must be persisted
// immediately — FABRIC does not expose it again.
//
// comment and description are validated by the server: comment 5-100 chars
// matching ^[\w\-.@()]+$, description 5-255 chars.
func (c *Client) CreateSSHKey(ctx context.Context, keyType, comment, description string) (SSHKeyPair, error) {
	reqBody := map[string]any{
		"keytype":      keyType,
		"comment":      comment,
		"description":  description,
		"store_pubkey": true,
	}
	var out struct {
		Results []struct {
			PrivateOpenSSH string `json:"private_openssh"`
			PublicOpenSSH  string `json:"public_openssh"`
		} `json:"results"`
	}
	if err := c.do(ctx, http.MethodPost, "/sshkeys", reqBody, &out); err != nil {
		return SSHKeyPair{}, err
	}
	if len(out.Results) == 0 {
		return SSHKeyPair{}, fmt.Errorf("create sshkey (%s): %w", keyType, ErrNoRecord)
	}
	r := out.Results[0]
	return SSHKeyPair{PrivateOpenSSH: r.PrivateOpenSSH, PublicOpenSSH: r.PublicOpenSSH}, nil
}

// DeleteSSHKey removes a registered SSH key by UUID via DELETE /sshkeys/{uuid}.
func (c *Client) DeleteSSHKey(ctx context.Context, uuid string) error {
	return c.do(ctx, http.MethodDelete, "/sshkeys/"+url.PathEscape(uuid), nil, nil)
}

// get performs an authenticated GET against path and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// do performs an authenticated request against path (relative to baseURL),
// encoding body as JSON when non-nil and decoding the JSON response into out.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	token, err := c.ts.IDToken(ctx)
	if err != nil {
		return fmt.Errorf("getting FABRIC id token: %w", err)
	}
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding core api request body: %w", err)
		}
		reqBody = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("building core api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("calling core api %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return httpError(resp.StatusCode, string(respBody))
	}
	if readErr != nil {
		return fmt.Errorf("reading core api response: %w", readErr)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("decoding core api response: %w", err)
	}
	return nil
}
