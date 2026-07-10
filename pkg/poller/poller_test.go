package poller

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/client"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/client/clienttest"
)

func TestWaitForSliceStates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		states     []string
		notice     string
		success    []string
		failure    []string
		want       string
		wantErr    string
		wantSuffix string
	}{
		{name: "stable ok happy path", states: []string{"Nascent", "Configuring", "StableOK"}, notice: "notice", success: []string{"StableOK"}, failure: []string{"StableError"}, want: "StableOK"},
		{name: "stable error fails with notice", states: []string{"StableError"}, notice: "insufficient capacity", success: []string{"StableOK"}, failure: []string{"StableError"}, wantErr: "reached StableError: insufficient capacity"},
		{name: "stable error fails without notice", states: []string{"StableError"}, success: []string{"StableOK"}, failure: []string{"StableError"}, wantErr: "StableError", wantSuffix: "reached StableError"},
		{name: "allocated ok continues polling", states: []string{"AllocatedOK", "StableOK"}, notice: "notice", success: []string{"StableOK"}, failure: []string{"AllocatedError"}, want: "StableOK"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			i := 0
			c := &clienttest.Client{GetFn: func(context.Context, string) (*client.Slice, error) {
				state := tc.states[i]
				if i < len(tc.states)-1 {
					i++
				}
				return &client.Slice{SliceID: "slice-1", State: state, Notice: tc.notice}, nil
			}}
			got, err := WaitForSlice(context.Background(), c, "slice-1", tc.success, tc.failure, time.Second, time.Millisecond)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				if tc.wantSuffix != "" && !strings.HasSuffix(err.Error(), tc.wantSuffix) {
					t.Fatalf("expected error to end with %q (no empty notice suffix), got %q", tc.wantSuffix, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("WaitForSlice returned error: %v", err)
			}
			if got.State != tc.want {
				t.Fatalf("state = %q, want %q", got.State, tc.want)
			}
		})
	}
}

func TestWaitForPOA(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		states     []client.POA
		want       string
		wantErr    string
		wantSuffix string
	}{
		{name: "success", states: []client.POA{{POAID: "poa-1", State: "Running"}, {POAID: "poa-1", State: POASuccessState}}, want: POASuccessState},
		{name: "failure includes poa error", states: []client.POA{{POAID: "poa-1", State: POAFailedState, Error: "reboot failed"}}, wantErr: "reboot failed"},
		{name: "failure without poa error omits suffix", states: []client.POA{{POAID: "poa-1", State: POAFailedState}}, wantErr: POAFailedState, wantSuffix: "reached Failed"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			i := 0
			c := &clienttest.Client{GetPOAFn: func(context.Context, string) (*client.POA, error) {
				poa := tc.states[i]
				if i < len(tc.states)-1 {
					i++
				}
				return &poa, nil
			}}
			got, err := WaitForPOA(context.Background(), c, "poa-1", time.Second, time.Millisecond)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				if tc.wantSuffix != "" && !strings.HasSuffix(err.Error(), tc.wantSuffix) {
					t.Fatalf("expected error to end with %q (no empty detail suffix), got %q", tc.wantSuffix, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("WaitForPOA returned error: %v", err)
			}
			if got.State != tc.want {
				t.Fatalf("state = %q, want %q", got.State, tc.want)
			}
		})
	}
}
