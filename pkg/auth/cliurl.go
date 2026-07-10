package auth

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// CLITokenURL builds the FABRIC credential manager browser-login URL for the
// CLI token flow (credmgr/tokens/create_cli). redirectURI is the local callback
// the credential manager redirects back to with the token in its query
// parameters. An empty credmgrURL defaults to cm.fabric-testbed.net; an empty
// scope defaults to "all"; an empty projectID selects the account's first
// active project; a non-positive lifetimeHours omits the lifetime parameter.
func CLITokenURL(credmgrURL, redirectURI, projectID, scope string, lifetimeHours int) (string, error) {
	if redirectURI == "" {
		return "", fmt.Errorf("cli token url: missing redirect_uri")
	}
	if scope == "" {
		scope = "all"
	}
	params := url.Values{}
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scope)
	if projectID != "" {
		params.Set("project_id", projectID)
	}
	if lifetimeHours > 0 {
		params.Set("lifetime", strconv.Itoa(lifetimeHours))
	}
	return credmgrEndpoint(credmgrURL, "/credmgr/tokens/create_cli", params.Encode())
}

// credmgrEndpoint normalizes a credential manager host or URL and appends path
// and an encoded query. An empty rawURL selects the FABRIC production
// credential manager; a bare host gains an https scheme.
func credmgrEndpoint(rawURL, path, rawQuery string) (string, error) {
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
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	parsed.RawQuery = rawQuery
	return parsed.String(), nil
}
