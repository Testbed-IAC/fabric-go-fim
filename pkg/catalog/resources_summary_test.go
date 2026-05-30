package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const resourcesSummaryFixture = `{
  "data": [
    {
      "sites": [
        {
          "name": "RENC",
          "state": "Active",
          "cores_capacity": 384,
          "cores_available": 192,
          "ram_capacity": 1434,
          "ram_available": 666,
          "disk_capacity": 64068,
          "disk_available": 57358,
          "components": {
            "SharedNIC-ConnectX-6": {"capacity": 381, "allocated": 39, "available": 342}
          }
        }
      ],
      "hosts": [
        {
          "name": "renc-w1",
          "site": "RENC",
          "state": "Active",
          "cores_available": 64,
          "ram_available": 256,
          "disk_available": 1000
        }
      ],
      "facility_ports": [
        {"name": "RENCI-FP", "port": "RENCI-FP-int", "site": "RENC", "switch": "port+renc-data-sw:Eth0", "vlans": "['100-110']"}
      ]
    }
  ],
  "size": 1,
  "status": 200,
  "type": "resources"
}`

func TestDecodeResourcesSummary(t *testing.T) {
	summary, err := DecodeResourcesSummary(strings.NewReader(resourcesSummaryFixture))
	if err != nil {
		t.Fatalf("DecodeResourcesSummary: %v", err)
	}
	site, ok := summary.Site("RENC")
	if !ok {
		t.Fatalf("Site(RENC) missing")
	}
	if site.State != "Active" || site.CoresAvailable != 192 || site.RAMAvailable != 666 || site.DiskAvailable != 57358 {
		t.Fatalf("site = %+v", site)
	}
	if got := site.Components["SharedNIC-ConnectX-6"].Available; got != 342 {
		t.Fatalf("component available = %d, want 342", got)
	}
	data, ok := summary.First()
	if !ok {
		t.Fatalf("First missing")
	}
	if len(data.Hosts) != 1 || data.Hosts[0].Name != "renc-w1" {
		t.Fatalf("hosts = %+v", data.Hosts)
	}
	if len(data.FacilityPorts) != 1 || data.FacilityPorts[0].Name != "RENCI-FP" {
		t.Fatalf("facility ports = %+v", data.FacilityPorts)
	}
}

func TestResourcesClientGetResourcesSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("level"); got != "1" {
			t.Fatalf("level query = %q, want 1", got)
		}
		if got := r.URL.Query().Get("type"); got != "sites,facility_ports" {
			t.Fatalf("type query = %q, want sites,facility_ports", got)
		}
		if got := r.URL.Query().Get("force_refresh"); got != "true" {
			t.Fatalf("force_refresh query = %q, want true", got)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(resourcesSummaryFixture))
	}))
	defer server.Close()

	client := NewResourcesClient(server.URL, server.Client())
	summary, err := client.GetResourcesSummary(context.Background(), ResourcesOptions{
		Level:        1,
		Types:        []string{"sites", "facility_ports"},
		ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("GetResourcesSummary: %v", err)
	}
	if _, ok := summary.Site("RENC"); !ok {
		t.Fatalf("Site(RENC) missing")
	}
}
