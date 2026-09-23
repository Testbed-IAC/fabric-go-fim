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

const maxTransientRetries = 3

// WaitForSlice polls a FABRIC slice until it reaches a configured terminal state,
// retrying transient client errors up to maxTransientRetries times with
// exponential backoff.
func WaitForSlice(ctx context.Context, c SliceGetter, sliceID string, successStates, failureStates []string, timeout, interval time.Duration) (*client.Slice, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	success := stateSet(successStates)
	failure := stateSet(failureStates)
	retries := 0

	for {
		slice, err := c.GetSlice(ctx, sliceID)
		if errors.Is(err, client.ErrNotFound) && success["Dead"] {
			return &client.Slice{SliceID: sliceID, State: "Dead"}, nil
		}
		delay := interval
		switch {
		case err == nil:
			retries = 0
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
		case client.IsTransient(err) && retries < maxTransientRetries:
			retries++
			delay = interval << retries
		default:
			return nil, fmt.Errorf("polling slice %s: %w", sliceID, err)
		}
		if err := wait(ctx, deadline.C, timeout, delay); err != nil {
			return nil, fmt.Errorf("polling slice %s: %w", sliceID, err)
		}
	}
}

// WaitForPOA polls a FABRIC POA until it reaches Success or Failed, retrying
// transient client errors up to maxTransientRetries times with exponential
// backoff.
func WaitForPOA(ctx context.Context, c POAGetter, poaID string, timeout, interval time.Duration) (*client.POA, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	retries := 0

	for {
		poa, err := c.GetPOA(ctx, poaID)
		delay := interval
		switch {
		case err == nil:
			retries = 0
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
		case client.IsTransient(err) && retries < maxTransientRetries:
			retries++
			delay = interval << retries
		default:
			return nil, fmt.Errorf("polling poa %s: %w", poaID, err)
		}
		if err := wait(ctx, deadline.C, timeout, delay); err != nil {
			return nil, fmt.Errorf("polling poa %s: %w", poaID, err)
		}
	}
}

func wait(ctx context.Context, deadline <-chan time.Time, timeout, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-deadline:
		return fmt.Errorf("timeout after %s", timeout)
	case <-timer.C:
		return nil
	}
}

func stateSet(states []string) map[string]bool {
	out := make(map[string]bool, len(states))
	for _, state := range states {
		out[state] = true
	}
	return out
}
