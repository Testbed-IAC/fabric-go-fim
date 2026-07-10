package coreapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/auth"
)

// stubToken is a minimal auth.TokenSource for tests.
type stubToken struct {
	token string
	err   error
}

func (s stubToken) IDToken(context.Context) (string, error) { return s.token, s.err }
func (s stubToken) ProjectID() string                       { return "" }
func (s stubToken) Claims() *auth.Claims                    { return nil }

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL, stubToken{token: "test-token"}, srv.Client())
}

func TestWhoAmI(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/whoami" {
			t.Errorf("path = %q, want /whoami", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want Bearer test-token", got)
		}
		_, _ = w.Write([]byte(`{"results":[{"uuid":"u-123","email":"a@b.net"}]}`))
	})

	who, err := c.WhoAmI(context.Background())
	if err != nil {
		t.Fatalf("WhoAmI: %v", err)
	}
	if who.UUID != "u-123" || who.Email != "a@b.net" {
		t.Fatalf("WhoAmI = %+v, want {u-123 a@b.net}", who)
	}
}

func TestWhoAmIEmptyResults(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	if _, err := c.WhoAmI(context.Background()); !errors.Is(err, ErrNoRecord) {
		t.Fatalf("WhoAmI error = %v, want ErrNoRecord", err)
	}
}

func TestPersonAsSelf(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/people/u-123" {
			t.Errorf("path = %q, want /people/u-123", r.URL.Path)
		}
		if r.URL.Query().Get("as_self") != "true" {
			t.Errorf("as_self = %q, want true", r.URL.Query().Get("as_self"))
		}
		_, _ = w.Write([]byte(`{"results":[{"uuid":"u-123","name":"Ada","bastion_login":"ada_0001"}]}`))
	})

	p, err := c.Person(context.Background(), "u-123")
	if err != nil {
		t.Fatalf("Person: %v", err)
	}
	if p.BastionLogin != "ada_0001" || p.Name != "Ada" {
		t.Fatalf("Person = %+v, want bastion_login ada_0001, name Ada", p)
	}
}

func TestResolveIdentity(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/whoami":
			_, _ = w.Write([]byte(`{"results":[{"uuid":"u-9","email":"a@b.net"}]}`))
		case "/people/u-9":
			_, _ = w.Write([]byte(`{"results":[{"uuid":"u-9","bastion_login":"ada_0009"}]}`))
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	id, err := c.ResolveIdentity(context.Background())
	if err != nil {
		t.Fatalf("ResolveIdentity: %v", err)
	}
	if id.UUID != "u-9" || id.BastionLogin != "ada_0009" || id.Email != "a@b.net" {
		t.Fatalf("ResolveIdentity = %+v", id)
	}
}

func TestResolveIdentityNoBastionLogin(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/whoami":
			_, _ = w.Write([]byte(`{"results":[{"uuid":"u-9"}]}`))
		case "/people/u-9":
			_, _ = w.Write([]byte(`{"results":[{"uuid":"u-9","bastion_login":""}]}`))
		}
	})
	if _, err := c.ResolveIdentity(context.Background()); !errors.Is(err, ErrNoBastionLogin) {
		t.Fatalf("error = %v, want ErrNoBastionLogin", err)
	}
}

func TestSSHKeys(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sshkeys" {
			t.Errorf("path = %q, want /sshkeys", r.URL.Path)
		}
		if got := r.URL.Query().Get("person_uuid"); got != "u-1" {
			t.Errorf("person_uuid = %q, want u-1", got)
		}
		_, _ = w.Write([]byte(`{"results":[
			{"uuid":"k1","fingerprint":"MD5:aa","fabric_key_type":"bastion","public_key":"ssh-ed25519 AAAA","expires_on":"2030-01-01T00:00:00+00:00"},
			{"uuid":"k2","fingerprint":"MD5:bb","fabric_key_type":"sliver","public_key":"ssh-ed25519 BBBB"}
		]}`))
	})

	keys, err := c.SSHKeys(context.Background(), "u-1")
	if err != nil {
		t.Fatalf("SSHKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("len = %d, want 2", len(keys))
	}
	if keys[0].FabricKeyType != "bastion" || keys[0].Fingerprint != "MD5:aa" {
		t.Errorf("key[0] = %+v", keys[0])
	}
}

func TestCreateSSHKey(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sshkeys" {
			t.Errorf("%s %s, want POST /sshkeys", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["keytype"] != "bastion" || body["comment"] != "fabric-cli" || body["store_pubkey"] != true {
			t.Errorf("body = %+v", body)
		}
		_, _ = w.Write([]byte(`{"results":[{"private_openssh":"-----BEGIN-----\nAAA\n-----END-----","public_openssh":"ssh-ed25519 AAAA cmt"}]}`))
	})

	pair, err := c.CreateSSHKey(context.Background(), KeyTypeBastion, "fabric-cli", "bastion key via fabric-cli")
	if err != nil {
		t.Fatalf("CreateSSHKey: %v", err)
	}
	if !strings.Contains(pair.PrivateOpenSSH, "BEGIN") || !strings.HasPrefix(pair.PublicOpenSSH, "ssh-ed25519") {
		t.Fatalf("pair = %+v", pair)
	}
}

func TestCreateSSHKeyEmptyResults(t *testing.T) {
	t.Parallel()
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"results":[]}`))
	})
	if _, err := c.CreateSSHKey(context.Background(), KeyTypeSliver, "c", "d"); !errors.Is(err, ErrNoRecord) {
		t.Fatalf("error = %v, want ErrNoRecord", err)
	}
}

func TestDeleteSSHKey(t *testing.T) {
	t.Parallel()
	var gotMethod, gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":200}`))
	})
	if err := c.DeleteSSHKey(context.Background(), "k-1"); err != nil {
		t.Fatalf("DeleteSSHKey: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/sshkeys/k-1" {
		t.Fatalf("%s %s, want DELETE /sshkeys/k-1", gotMethod, gotPath)
	}
}

func TestHTTPErrorMapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
	}
	for _, tc := range cases {
		c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"error":"nope"}`))
		})
		if _, err := c.WhoAmI(context.Background()); !errors.Is(err, tc.want) {
			t.Errorf("status %d: error = %v, want %v", tc.status, err, tc.want)
		}
	}
}
