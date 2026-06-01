package client

import (
	"context"
	"encoding/json"
	"fmt"

	openapi "github.com/Testbed-IAC/fabric-orchestrator-go-client"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/auth"
)

// Client adapts the generated FABRIC orchestrator OpenAPI client to a stable,
// tool-friendly interface.
type Client struct {
	api *openapi.APIClient
	ts  auth.TokenSource
}

// New returns a high-level FABRIC orchestrator client.
func New(orchestratorURL string, ts auth.TokenSource) *Client {
	cfg := openapi.NewConfiguration()
	if orchestratorURL != "" {
		cfg.Servers = openapi.ServerConfigurations{{URL: orchestratorURL}}
	}
	cfg.HTTPClient = WithContentTypeFix(cfg.HTTPClient)
	return &Client{api: openapi.NewAPIClient(cfg), ts: ts}
}

func (c *Client) authCtx(ctx context.Context) (context.Context, error) {
	token, err := c.ts.IDToken(ctx)
	if err != nil {
		return ctx, fmt.Errorf("getting FABRIC id token: %w", err)
	}
	return context.WithValue(ctx, openapi.ContextAccessToken, token), nil
}

// CreateSlice creates a FABRIC slice.
func (c *Client) CreateSlice(ctx context.Context, name, graphML string, sshKeys []string, opts CreateOpts) ([]Sliver, error) {
	body := openapi.NewSlicesPost(graphML, sshKeys)
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	req := c.api.SlicesAPI.SlicesCreatesPost(authCtx).Name(name).SlicesPost(*body)
	if opts.LifetimeHours > 0 {
		req = req.Lifetime(opts.LifetimeHours)
	}
	if opts.LeaseStartTime != "" {
		req = req.LeaseStartTime(opts.LeaseStartTime)
	}
	if opts.LeaseEndTime != "" {
		req = req.LeaseEndTime(opts.LeaseEndTime)
	}
	resp, httpResp, err := req.Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	return convertSlivers(resp.Data), nil
}

// GetSlice returns one FABRIC slice by ID.
func (c *Client) GetSlice(ctx context.Context, sliceID string) (*Slice, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	resp, httpResp, err := c.api.SlicesAPI.SlicesSliceIdGet(authCtx, sliceID).GraphFormat("GRAPHML").Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, ErrNotFound
	}
	s := resp.Data[0]
	return &Slice{
		SliceID:        s.GetSliceId(),
		Name:           s.GetName(),
		State:          s.GetState(),
		GraphID:        s.GetGraphId(),
		Model:          s.GetModel(),
		LeaseStartTime: s.GetLeaseStartTime(),
		LeaseEndTime:   s.GetLeaseEndTime(),
	}, nil
}

// ListSlices returns slices filtered by name and state.
func (c *Client) ListSlices(ctx context.Context, name string, states []string) ([]Slice, error) {
	out := []Slice{}
	var offset int32
	for {
		authCtx, err := c.authCtx(ctx)
		if err != nil {
			return nil, err
		}
		req := c.api.SlicesAPI.SlicesGet(authCtx).Limit(200).Offset(offset)
		if name != "" {
			req = req.Name(name).ExactMatch(true)
		}
		if len(states) > 0 {
			req = req.States(states)
		}
		page, httpResp, err := req.Execute()
		if err != nil {
			return nil, mapHTTPErr(httpResp, err)
		}
		for _, s := range page.Data {
			out = append(out, Slice{SliceID: s.GetSliceId(), Name: s.GetName(), GraphID: s.GetGraphId(), State: s.GetState()})
		}
		if len(page.Data) < 200 {
			return out, nil
		}
		offset += 200
	}
}

// ModifySlice submits a slice graph modification.
func (c *Client) ModifySlice(ctx context.Context, sliceID, graphML string) ([]Sliver, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	resp, httpResp, err := c.api.SlicesAPI.SlicesModifySliceIdPut(authCtx, sliceID).Body(graphML).Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	return convertSlivers(resp.Data), nil
}

// AcceptModify accepts a completed slice modification.
func (c *Client) AcceptModify(ctx context.Context, sliceID string) (*Slice, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	resp, httpResp, err := c.api.SlicesAPI.SlicesModifySliceIdAcceptPost(authCtx, sliceID).Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return nil, ErrNotFound
	}
	s := resp.Data[0]
	return &Slice{SliceID: s.GetSliceId(), Name: s.GetName(), State: s.GetState(), GraphID: s.GetGraphId(), Model: s.GetModel()}, nil
}

// RenewSlice renews a slice lease.
func (c *Client) RenewSlice(ctx context.Context, sliceID, leaseEndTime string) error {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return err
	}
	_, httpResp, err := c.api.SlicesAPI.SlicesRenewSliceIdPost(authCtx, sliceID).LeaseEndTime(leaseEndTime).Execute()
	if err != nil {
		return mapHTTPErr(httpResp, err)
	}
	return nil
}

// DeleteSlice deletes a slice.
func (c *Client) DeleteSlice(ctx context.Context, sliceID string) error {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return err
	}
	_, httpResp, err := c.api.SlicesAPI.SlicesDeleteSliceIdDelete(authCtx, sliceID).Execute()
	if err != nil {
		return mapHTTPErr(httpResp, err)
	}
	return nil
}

// GetSlivers returns slivers for a slice.
func (c *Client) GetSlivers(ctx context.Context, sliceID string) ([]Sliver, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	resp, httpResp, err := c.api.SliversAPI.SliversGet(authCtx).SliceId(sliceID).Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	return convertSlivers(resp.Data), nil
}

// GetResources returns the orchestrator resources model.
func (c *Client) GetResources(ctx context.Context, query ResourcesQuery) (string, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return "", err
	}
	req := c.api.ResourcesAPI.ResourcesGet(authCtx).Level(query.Level).ForceRefresh(query.ForceRefresh)
	if query.StartDate != "" {
		req = req.StartDate(query.StartDate)
	}
	if query.EndDate != "" {
		req = req.EndDate(query.EndDate)
	}
	if query.Includes != "" {
		req = req.Includes(query.Includes)
	}
	if query.Excludes != "" {
		req = req.Excludes(query.Excludes)
	}
	resp, httpResp, err := req.Execute()
	if err != nil {
		return "", mapHTTPErr(httpResp, err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return "", nil
	}
	return resp.Data[0].GetModel(), nil
}

// GetPortalResources returns portal resources in a selected graph format.
func (c *Client) GetPortalResources(ctx context.Context, query ResourcesQuery) (string, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return "", err
	}
	req := c.api.ResourcesAPI.PortalresourcesGet(authCtx).GraphFormat(query.GraphFormat).Level(query.Level).ForceRefresh(query.ForceRefresh)
	if query.StartDate != "" {
		req = req.StartDate(query.StartDate)
	}
	if query.EndDate != "" {
		req = req.EndDate(query.EndDate)
	}
	if query.Includes != "" {
		req = req.Includes(query.Includes)
	}
	if query.Excludes != "" {
		req = req.Excludes(query.Excludes)
	}
	resp, httpResp, err := req.Execute()
	if err != nil {
		return "", mapHTTPErr(httpResp, err)
	}
	if resp == nil || len(resp.Data) == 0 {
		return "", nil
	}
	return resp.Data[0].GetModel(), nil
}

// CreatePOA creates a perform-operational-action request.
func (c *Client) CreatePOA(ctx context.Context, sliverID string, request POARequest) (*POA, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	resp, httpResp, err := c.api.PoasAPI.PoasCreateSliverIdPost(authCtx, sliverID).PoaPost(openapiPOARequest(request)).Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	poa, err := convertPOA(resp)
	if err != nil {
		return nil, fmt.Errorf("converting poa response: %w", err)
	}
	return poa, nil
}

// GetPOA returns one POA status.
func (c *Client) GetPOA(ctx context.Context, poaID string) (*POA, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return nil, err
	}
	resp, httpResp, err := c.api.PoasAPI.PoasPoaIdGet(authCtx, poaID).Execute()
	if err != nil {
		return nil, mapHTTPErr(httpResp, err)
	}
	poa, err := convertPOA(resp)
	if err != nil {
		return nil, fmt.Errorf("converting poa response: %w", err)
	}
	return poa, nil
}

// GetMetricsOverview returns metrics overview results encoded as JSON.
func (c *Client) GetMetricsOverview(ctx context.Context, query MetricsQuery) (string, error) {
	authCtx, err := c.authCtx(ctx)
	if err != nil {
		return "", err
	}
	req := c.api.MetricsAPI.MetricsOverviewGet(authCtx)
	if len(query.ExcludedProjects) > 0 {
		req = req.ExcludedProjects(query.ExcludedProjects)
	}
	resp, httpResp, err := req.Execute()
	if err != nil {
		return "", mapHTTPErr(httpResp, err)
	}
	results, err := metricsResultsJSON(resp)
	if err != nil {
		return "", fmt.Errorf("converting metrics overview: %w", err)
	}
	return results, nil
}

func convertSlivers(in []openapi.Sliver) []Sliver {
	out := make([]Sliver, 0, len(in))
	for _, s := range in {
		out = append(out, Sliver{
			SliceID:      s.GetSliceId(),
			SliverID:     s.GetSliverId(),
			GraphNodeID:  s.GetGraphNodeId(),
			SliverType:   s.GetSliverType(),
			State:        s.GetState(),
			PendingState: s.GetPendingState(),
			JoinState:    s.GetJoinState(),
			ManagementIP: managementIPFromSliverPayload(s.GetSliver()),
			Notice:       s.GetNotice(),
		})
	}
	return out
}

func openapiPOARequest(request POARequest) openapi.PoaPost {
	out := openapi.NewPoaPost(request.Operation)
	data := openapi.PoaPostData{}
	for _, mapping := range request.VCPUCPUMap {
		data.VcpuCpuMap = append(data.VcpuCpuMap, *openapi.NewPoaPostDataVcpuCpuMap(mapping.VCPU, mapping.CPU))
	}
	data.NodeSet = append([]string(nil), request.NodeSet...)
	data.Bdf = append([]string(nil), request.BDF...)
	for _, key := range request.Keys {
		data.Keys = append(data.Keys, *openapi.NewPoaPostDataKeys(key.Key, key.Comment))
	}
	if len(data.VcpuCpuMap) > 0 || len(data.NodeSet) > 0 || len(data.Bdf) > 0 || len(data.Keys) > 0 {
		out.SetData(data)
	}
	return *out
}

func convertPOA(in *openapi.Poa) (*POA, error) {
	if in == nil || len(in.GetData()) == 0 {
		return &POA{}, nil
	}
	data := in.GetData()[0]
	info, err := infoJSON(data.GetInfo())
	if err != nil {
		return nil, fmt.Errorf("encoding poa info: %w", err)
	}
	return &POA{
		POAID:     data.GetPoaId(),
		Operation: data.GetOperation(),
		State:     data.GetState(),
		SliverID:  data.GetSliverId(),
		SliceID:   data.GetSliceId(),
		Error:     data.GetError(),
		InfoJSON:  info,
	}, nil
}

func infoJSON(info map[string]interface{}) (string, error) {
	if len(info) == 0 {
		return "", nil
	}
	body, err := json.Marshal(info)
	if err != nil {
		return "", fmt.Errorf("marshal info: %w", err)
	}
	return string(body), nil
}

func metricsResultsJSON(metrics *openapi.Metrics) (string, error) {
	results := []map[string]interface{}{}
	if metrics != nil {
		results = metrics.GetResults()
	}
	body, err := json.Marshal(results)
	if err != nil {
		return "", fmt.Errorf("marshal metrics results: %w", err)
	}
	return string(body), nil
}

func managementIPFromSliverPayload(payload map[string]interface{}) string {
	for _, key := range []string{"management_ip", "managementIP", "mgmt_ip", "mgmtIP"} {
		value, ok := payload[key].(string)
		if ok {
			return value
		}
	}
	return ""
}

var _ API = (*Client)(nil)
