package sshkeys

import (
	"errors"
	"testing"
)

func TestSelect(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		legacyKey string
		keys      []string
		want      []string
		wantErr   error
	}{
		{name: "legacy", legacyKey: "ssh-ed25519 AAAA", want: []string{"ssh-ed25519 AAAA"}},
		{name: "list", keys: []string{"ssh-ed25519 AAAA", "ssh-ed25519 BBBB"}, want: []string{"ssh-ed25519 AAAA", "ssh-ed25519 BBBB"}},
		{name: "missing", wantErr: ErrMissingKeys},
		{name: "ambiguous", legacyKey: "ssh-ed25519 AAAA", keys: []string{"ssh-ed25519 BBBB"}, wantErr: ErrAmbiguousSource},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := Select(tc.legacyKey, tc.keys)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Select error = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("Select length = %d, want %d", len(got), len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("Select[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}
