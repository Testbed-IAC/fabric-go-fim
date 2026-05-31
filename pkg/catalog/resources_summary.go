package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const DefaultPortalResourcesURL = "https://portal.fabric-testbed.net/api/resources"

// ResourcesOptions controls portal resources summary requests.
type ResourcesOptions struct {
	Level        int
	Types        []string
	ForceRefresh bool
}

// ResourcesClient fetches FABRIC portal resources summary JSON.
type ResourcesClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewResourcesClient returns a portal resources summary client.
func NewResourcesClient(baseURL string, httpClient *http.Client) *ResourcesClient {
	if baseURL == "" {
		baseURL = DefaultPortalResourcesURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &ResourcesClient{BaseURL: baseURL, HTTPClient: httpClient}
}

// GetResourcesSummary fetches and decodes the portal resources summary.
func (c *ResourcesClient) GetResourcesSummary(ctx context.Context, opts ResourcesOptions) (*ResourcesSummary, error) {
	if c == nil {
		c = NewResourcesClient("", nil)
	}
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DefaultPortalResourcesURL
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid resources URL %q: %w", ErrCatalogLoad, baseURL, err)
	}
	query := parsed.Query()
	if opts.Level > 0 {
		query.Set("level", strconv.Itoa(opts.Level))
	}
	if len(opts.Types) > 0 {
		query.Set("type", strings.Join(opts.Types, ","))
	}
	if opts.ForceRefresh {
		query.Set("force_refresh", "true")
	}
	parsed.RawQuery = query.Encode()

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: building resources request: %w", ErrCatalogLoad, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: fetching resources summary: %w", ErrCatalogLoad, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if err := resp.Body.Close(); err != nil {
			return nil, fmt.Errorf("%w: closing resources summary response: %w", ErrCatalogLoad, err)
		}
		return nil, fmt.Errorf("%w: fetching resources summary returned %s", ErrCatalogLoad, resp.Status)
	}
	summary, err := DecodeResourcesSummary(resp.Body)
	closeErr := resp.Body.Close()
	if err != nil {
		if closeErr != nil {
			return nil, fmt.Errorf("decoding resources summary and closing response: %w", errors.Join(err, closeErr))
		}
		return nil, err
	}
	if closeErr != nil {
		return nil, fmt.Errorf("%w: closing resources summary response: %w", ErrCatalogLoad, closeErr)
	}
	return summary, nil
}

// DecodeResourcesSummary decodes portal resources summary JSON.
func DecodeResourcesSummary(r io.Reader) (*ResourcesSummary, error) {
	var summary ResourcesSummary
	if err := json.NewDecoder(r).Decode(&summary); err != nil {
		return nil, fmt.Errorf("%w: decoding resources summary: %w", ErrCatalogLoad, err)
	}
	return &summary, nil
}

// ResourcesSummary is the FABRIC portal /api/resources response.
type ResourcesSummary struct {
	Size   int                    `json:"size,omitempty"`
	Status int                    `json:"status,omitempty"`
	Type   string                 `json:"type,omitempty"`
	Data   []ResourcesSummaryData `json:"data,omitempty"`
}

// First returns the first resources payload, if present.
func (s *ResourcesSummary) First() (ResourcesSummaryData, bool) {
	if s == nil || len(s.Data) == 0 {
		return ResourcesSummaryData{}, false
	}
	return s.Data[0], true
}

// Site returns a site summary by FABRIC site name.
func (s *ResourcesSummary) Site(name string) (SiteSummary, bool) {
	data, ok := s.First()
	if !ok {
		return SiteSummary{}, false
	}
	return data.Site(name)
}

// ResourcesSummaryData contains resource categories returned by the portal.
type ResourcesSummaryData struct {
	Sites         []SiteSummary         `json:"sites,omitempty"`
	Hosts         []HostSummary         `json:"hosts,omitempty"`
	Links         []LinkSummary         `json:"links,omitempty"`
	FacilityPorts []FacilityPortSummary `json:"facility_ports,omitempty"`
}

// Site returns a site summary by FABRIC site name.
func (d ResourcesSummaryData) Site(name string) (SiteSummary, bool) {
	for _, site := range d.Sites {
		if site.Name == name {
			return site, true
		}
	}
	return SiteSummary{}, false
}

// HostSummary contains host-level resource capacity details.
type HostSummary struct {
	Name           string                           `json:"name,omitempty"`
	Site           string                           `json:"site,omitempty"`
	State          string                           `json:"state,omitempty"`
	CoresCapacity  int                              `json:"cores_capacity,omitempty"`
	CoresAllocated int                              `json:"cores_allocated,omitempty"`
	CoresAvailable int                              `json:"cores_available,omitempty"`
	RAMCapacity    int                              `json:"ram_capacity,omitempty"`
	RAMAllocated   int                              `json:"ram_allocated,omitempty"`
	RAMAvailable   int                              `json:"ram_available,omitempty"`
	DiskCapacity   int                              `json:"disk_capacity,omitempty"`
	DiskAllocated  int                              `json:"disk_allocated,omitempty"`
	DiskAvailable  int                              `json:"disk_available,omitempty"`
	Components     map[string]ComponentAvailability `json:"components,omitempty"`
}

// SiteSummary contains site-level resource capacity details.
type SiteSummary struct {
	Name           string                           `json:"name,omitempty"`
	State          string                           `json:"state,omitempty"`
	Address        string                           `json:"address,omitempty"`
	Location       []float64                        `json:"location,omitempty"`
	HostsCount     int                              `json:"hosts_count,omitempty"`
	IPv4Management bool                             `json:"ipv4_management,omitempty"`
	PTPCapable     bool                             `json:"ptp_capable,omitempty"`
	CoresCapacity  int                              `json:"cores_capacity,omitempty"`
	CoresAllocated int                              `json:"cores_allocated,omitempty"`
	CoresAvailable int                              `json:"cores_available,omitempty"`
	RAMCapacity    int                              `json:"ram_capacity,omitempty"`
	RAMAllocated   int                              `json:"ram_allocated,omitempty"`
	RAMAvailable   int                              `json:"ram_available,omitempty"`
	DiskCapacity   int                              `json:"disk_capacity,omitempty"`
	DiskAllocated  int                              `json:"disk_allocated,omitempty"`
	DiskAvailable  int                              `json:"disk_available,omitempty"`
	Components     map[string]ComponentAvailability `json:"components,omitempty"`
}

// ComponentAvailability contains capacity counts for a component model at a site or host.
type ComponentAvailability struct {
	Capacity  int `json:"capacity,omitempty"`
	Allocated int `json:"allocated,omitempty"`
	Available int `json:"available,omitempty"`
}

// FacilityPortSummary contains a FABRIC facility port summary.
type FacilityPortSummary struct {
	Name   string `json:"name,omitempty"`
	Port   string `json:"port,omitempty"`
	Site   string `json:"site,omitempty"`
	Switch string `json:"switch,omitempty"`
	VLANs  string `json:"vlans,omitempty"`
}

// LinkSummary contains a FABRIC link summary.
type LinkSummary struct {
	Name     string `json:"name,omitempty"`
	Layer    string `json:"layer,omitempty"`
	NodeID1  string `json:"node_id1,omitempty"`
	NodeID2  string `json:"node_id2,omitempty"`
	Site1    string `json:"site1,omitempty"`
	Site2    string `json:"site2,omitempty"`
	Capacity int    `json:"capacity,omitempty"`
}
