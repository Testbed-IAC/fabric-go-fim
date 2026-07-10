package auth

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestCLITokenURL(t *testing.T) {
	t.Parallel()
	got, err := CLITokenURL("cm.fabric-testbed.net", "http://localhost:8484/callback", "proj-1", "", 4)
	if err != nil {
		t.Fatalf("CLITokenURL: %v", err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse result %q: %v", got, err)
	}
	if u.Scheme != "https" || u.Host != "cm.fabric-testbed.net" {
		t.Errorf("scheme://host = %s://%s, want https://cm.fabric-testbed.net", u.Scheme, u.Host)
	}
	if u.Path != "/credmgr/tokens/create_cli" {
		t.Errorf("path = %q, want /credmgr/tokens/create_cli", u.Path)
	}
	q := u.Query()
	for key, want := range map[string]string{
		"redirect_uri": "http://localhost:8484/callback",
		"project_id":   "proj-1",
		"scope":        "all",
		"lifetime":     "4",
	} {
		if q.Get(key) != want {
			t.Errorf("%s = %q, want %q", key, q.Get(key), want)
		}
	}
}

func TestCLITokenURLOmitsOptionalParams(t *testing.T) {
	t.Parallel()
	got, err := CLITokenURL("", "http://localhost:9000/cb", "", "all", 0)
	if err != nil {
		t.Fatalf("CLITokenURL: %v", err)
	}
	u, _ := url.Parse(got)
	if u.Host != "cm.fabric-testbed.net" {
		t.Errorf("default host = %q, want cm.fabric-testbed.net", u.Host)
	}
	q := u.Query()
	if q.Has("project_id") {
		t.Errorf("project_id should be omitted, got %q", q.Get("project_id"))
	}
	if q.Has("lifetime") {
		t.Errorf("lifetime should be omitted, got %q", q.Get("lifetime"))
	}
}

func TestCLITokenURLRequiresRedirect(t *testing.T) {
	t.Parallel()
	if _, err := CLITokenURL("", "", "", "", 0); err == nil {
		t.Fatal("expected error for missing redirect_uri")
	}
}

func TestWriteTokenFileRoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "token.json")
	tf := TokenFile{IDToken: "a.b.c", RefreshToken: "refresh-1", State: "valid", TokenHash: "h", CreatedAt: "now", ExpiresAt: "later"}
	if err := WriteTokenFile(path, tf); err != nil {
		t.Fatalf("WriteTokenFile: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("perm = %o, want 600", perm)
	}
	got, err := ParseTokenFile(path)
	if err != nil {
		t.Fatalf("ParseTokenFile: %v", err)
	}
	if got != tf {
		t.Errorf("round trip mismatch:\n got %+v\nwant %+v", got, tf)
	}
}

func TestWriteTokenFileRejectsEmptyIDToken(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "token.json")
	if err := WriteTokenFile(path, TokenFile{RefreshToken: "r"}); err == nil {
		t.Fatal("expected error for missing id_token")
	}
}
