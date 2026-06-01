// Package sshkeys normalizes FABRIC SSH key inputs shared by API clients and
// higher-level tooling.
package sshkeys

import "errors"

var (
	// ErrMissingKeys indicates no SSH public keys were provided.
	ErrMissingKeys = errors.New("missing ssh keys")
	// ErrAmbiguousSource indicates both the legacy single key and key list were provided.
	ErrAmbiguousSource = errors.New("configure exactly one ssh key source")
)

// Select returns the effective SSH public key list from either a legacy single
// key or a multi-key list. Exactly one source must be configured.
func Select(legacyKey string, keys []string) ([]string, error) {
	hasLegacyKey := legacyKey != ""
	hasKeys := len(keys) > 0
	if hasLegacyKey == hasKeys {
		if hasLegacyKey {
			return nil, ErrAmbiguousSource
		}
		return nil, ErrMissingKeys
	}
	if hasLegacyKey {
		return []string{legacyKey}, nil
	}
	return append([]string(nil), keys...), nil
}
