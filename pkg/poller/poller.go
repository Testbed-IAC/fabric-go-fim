// Package poller provides FABRIC slice and POA polling helpers.
package poller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/client"
)

const (
	// POASuccessState is the terminal success state returned by FABRIC POA.
	POASuccessState = "Success"
	// POAFailedState is the terminal failure state returned by FABRIC POA.
	POAFailedState = "Failed"
)

// SliceGetter is the client subset needed by WaitForSlice.
type SliceGetter interface {
	GetSlice(ctx context.Context, sliceID string) (*client.Slice, error)
}

// POAGetter is the client subset needed by WaitForPOA.
type POAGetter interface {
	GetPOA(ctx context.Context, poaID string) (*client.POA, error)
}

// WaitForSlice polls a FABRIC slice until it reaches a configured terminal state.
func WaitForSlice(ctx context.Context, c SliceGetter, sliceID string, successStates, failureStates []string, timeout, interval time.Duration) (*client.Slice, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	success := stateSet(successStates)
	failure := stateSet(failureStates)

	for {
		slice, err := c.GetSlice(ctx, sliceID)
		if errors.Is(err, client.ErrNotFound) && success["Dead"] {
			return &client.Slice{SliceID: sliceID, State: "Dead"}, nil
		}
		if err != nil {
			return nil, fmt.Errorf("polling slice %s: %w", sliceID, err)
		}
		if slice != nil {
			if success[slice.State] {
				return slice, nil
			}
			if failure[slice.State] {
				if slice.Notice != "" {
					return slice, fmt.Errorf("polling slice %s reached %s: %s", sliceID, slice.State, slice.Notice)
				}
				return slice, fmt.Errorf("polling slice %s reached %s", sliceID, slice.State)
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("polling slice %s: %w", sliceID, ctx.Err())
		case <-deadline.C:
			return nil, fmt.Errorf("polling slice %s: timeout after %s", sliceID, timeout)
		case <-ticker.C:
		}
	}
}

// WaitForPOA polls a FABRIC POA until it reaches Success or Failed.
func WaitForPOA(ctx context.Context, c POAGetter, poaID string, timeout, interval time.Duration) (*client.POA, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		poa, err := c.GetPOA(ctx, poaID)
		if err != nil {
			return nil, fmt.Errorf("polling poa %s: %w", poaID, err)
		}
		if poa != nil {
			switch poa.State {
			case POASuccessState:
				return poa, nil
			case POAFailedState:
				if poa.Error != "" {
					return poa, fmt.Errorf("polling poa %s reached %s: %s", poaID, poa.State, poa.Error)
				}
				return poa, fmt.Errorf("polling poa %s reached %s", poaID, poa.State)
			}
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("polling poa %s: %w", poaID, ctx.Err())
		case <-deadline.C:
			return nil, fmt.Errorf("polling poa %s: timeout after %s", poaID, timeout)
		case <-ticker.C:
		}
	}
}

func stateSet(states []string) map[string]bool {
	out := make(map[string]bool, len(states))
	for _, state := range states {
		out[state] = true
	}
	return out
}
