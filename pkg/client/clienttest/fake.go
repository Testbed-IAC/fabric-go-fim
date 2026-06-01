// Package clienttest provides test doubles for pkg/client.
package clienttest

import (
	"context"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/client"
)

// Client is a function-backed fake FABRIC client.
type Client struct {
	CreateFn         func(context.Context, string, string, []string, client.CreateOpts) ([]client.Sliver, error)
	GetFn            func(context.Context, string) (*client.Slice, error)
	ListFn           func(context.Context, string, []string) ([]client.Slice, error)
	ModifyFn         func(context.Context, string, string) ([]client.Sliver, error)
	AcceptFn         func(context.Context, string) (*client.Slice, error)
	RenewFn          func(context.Context, string, string) error
	DeleteFn         func(context.Context, string) error
	SliversFn        func(context.Context, string) ([]client.Sliver, error)
	ResourceFn       func(context.Context, client.ResourcesQuery) (string, error)
	PortalResourceFn func(context.Context, client.ResourcesQuery) (string, error)
	CreatePOAFn      func(context.Context, string, client.POARequest) (*client.POA, error)
	GetPOAFn         func(context.Context, string) (*client.POA, error)
	MetricsFn        func(context.Context, client.MetricsQuery) (string, error)
	Calls            []string
}

// CreateSlice records and handles a CreateSlice call.
func (c *Client) CreateSlice(ctx context.Context, name, graphML string, sshKeys []string, opts client.CreateOpts) ([]client.Sliver, error) {
	c.Calls = append(c.Calls, "CreateSlice:"+name)
	if c.CreateFn == nil {
		return nil, nil
	}
	return c.CreateFn(ctx, name, graphML, sshKeys, opts)
}

// GetSlice records and handles a GetSlice call.
func (c *Client) GetSlice(ctx context.Context, sliceID string) (*client.Slice, error) {
	c.Calls = append(c.Calls, "GetSlice:"+sliceID)
	if c.GetFn == nil {
		return nil, nil
	}
	return c.GetFn(ctx, sliceID)
}

// ListSlices records and handles a ListSlices call.
func (c *Client) ListSlices(ctx context.Context, name string, states []string) ([]client.Slice, error) {
	c.Calls = append(c.Calls, "ListSlices:"+name)
	if c.ListFn == nil {
		return nil, nil
	}
	return c.ListFn(ctx, name, states)
}

// ModifySlice records and handles a ModifySlice call.
func (c *Client) ModifySlice(ctx context.Context, sliceID, graphML string) ([]client.Sliver, error) {
	c.Calls = append(c.Calls, "ModifySlice:"+sliceID)
	if c.ModifyFn == nil {
		return nil, nil
	}
	return c.ModifyFn(ctx, sliceID, graphML)
}

// AcceptModify records and handles an AcceptModify call.
func (c *Client) AcceptModify(ctx context.Context, sliceID string) (*client.Slice, error) {
	c.Calls = append(c.Calls, "AcceptModify:"+sliceID)
	if c.AcceptFn == nil {
		return nil, nil
	}
	return c.AcceptFn(ctx, sliceID)
}

// RenewSlice records and handles a RenewSlice call.
func (c *Client) RenewSlice(ctx context.Context, sliceID, leaseEndTime string) error {
	c.Calls = append(c.Calls, "RenewSlice:"+sliceID)
	if c.RenewFn == nil {
		return nil
	}
	return c.RenewFn(ctx, sliceID, leaseEndTime)
}

// DeleteSlice records and handles a DeleteSlice call.
func (c *Client) DeleteSlice(ctx context.Context, sliceID string) error {
	c.Calls = append(c.Calls, "DeleteSlice:"+sliceID)
	if c.DeleteFn == nil {
		return nil
	}
	return c.DeleteFn(ctx, sliceID)
}

// GetSlivers records and handles a GetSlivers call.
func (c *Client) GetSlivers(ctx context.Context, sliceID string) ([]client.Sliver, error) {
	c.Calls = append(c.Calls, "GetSlivers:"+sliceID)
	if c.SliversFn == nil {
		return nil, nil
	}
	return c.SliversFn(ctx, sliceID)
}

// GetResources records and handles a GetResources call.
func (c *Client) GetResources(ctx context.Context, query client.ResourcesQuery) (string, error) {
	c.Calls = append(c.Calls, "GetResources")
	if c.ResourceFn == nil {
		return "", nil
	}
	return c.ResourceFn(ctx, query)
}

// GetPortalResources records and handles a GetPortalResources call.
func (c *Client) GetPortalResources(ctx context.Context, query client.ResourcesQuery) (string, error) {
	c.Calls = append(c.Calls, "GetPortalResources")
	if c.PortalResourceFn == nil {
		return "", nil
	}
	return c.PortalResourceFn(ctx, query)
}

// CreatePOA records and handles a CreatePOA call.
func (c *Client) CreatePOA(ctx context.Context, sliverID string, request client.POARequest) (*client.POA, error) {
	c.Calls = append(c.Calls, "CreatePOA:"+sliverID)
	if c.CreatePOAFn == nil {
		return nil, nil
	}
	return c.CreatePOAFn(ctx, sliverID, request)
}

// GetPOA records and handles a GetPOA call.
func (c *Client) GetPOA(ctx context.Context, poaID string) (*client.POA, error) {
	c.Calls = append(c.Calls, "GetPOA:"+poaID)
	if c.GetPOAFn == nil {
		return nil, nil
	}
	return c.GetPOAFn(ctx, poaID)
}

// GetMetricsOverview records and handles a GetMetricsOverview call.
func (c *Client) GetMetricsOverview(ctx context.Context, query client.MetricsQuery) (string, error) {
	c.Calls = append(c.Calls, "GetMetricsOverview")
	if c.MetricsFn == nil {
		return "", nil
	}
	return c.MetricsFn(ctx, query)
}

var _ client.API = (*Client)(nil)
