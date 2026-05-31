package fabtime

import (
	"errors"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
	}{
		{name: "FABRIC layout", input: "2026-05-30 19:04:54 +00:00"},
		{name: "legacy orchestrator layout", input: "2026-05-30 19:04:54 +0000"},
		{name: "RFC3339", input: "2026-05-30T19:04:54Z"},
		{name: "RFC3339Nano", input: "2026-05-30T19:04:54.123456789Z"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Parse(tc.input); err != nil {
				t.Fatalf("Parse(%q): %v", tc.input, err)
			}
		})
	}
}

func TestParseRejectsInvalid(t *testing.T) {
	t.Parallel()
	if _, err := Parse("not a time"); !errors.Is(err, ErrInvalidTime) {
		t.Fatalf("Parse returned %v, want ErrInvalidTime", err)
	}
}

func TestFormat(t *testing.T) {
	t.Parallel()
	tm := time.Date(2026, 5, 30, 15, 4, 54, 0, time.FixedZone("EDT", -4*60*60))
	if got, want := Format(tm), "2026-05-30 19:04:54 +00:00"; got != want {
		t.Fatalf("Format = %q, want %q", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	tm, err := Parse("2026-05-30T19:04:54Z")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	roundTrip, err := Parse(Format(tm))
	if err != nil {
		t.Fatalf("Parse(Format): %v", err)
	}
	if !roundTrip.Equal(tm) {
		t.Fatalf("round trip = %s, want %s", roundTrip, tm)
	}
}
