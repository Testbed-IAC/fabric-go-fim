package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeAdvertised(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "advertised_topology.graphml"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	advertised, err := DecodeAdvertised(string(body))
	if err != nil {
		t.Fatalf("DecodeAdvertised: %v", err)
	}
	if len(advertised.Sites) != 1 {
		t.Fatalf("sites = %d, want 1", len(advertised.Sites))
	}
	site := advertised.Sites[0]
	if site.Name != "RENC" || !site.PTP || !site.IPv4Management {
		t.Fatalf("site = %+v, want RENC with flags", site)
	}
	if site.Cores.Capacity != 100 || site.Cores.Allocated != 10 || site.Cores.Available != 90 {
		t.Fatalf("site cores = %+v, want 100/10/90", site.Cores)
	}
	if len(site.Hosts) != 1 || site.Hosts[0].Name != "renc-w1" {
		t.Fatalf("site hosts = %+v, want renc-w1", site.Hosts)
	}
	component := site.Components["GPU/A30"]
	if component.Capacity != 2 || component.Allocated != 1 || component.Available != 1 {
		t.Fatalf("component GPU/A30 = %+v, want 2/1/1", component)
	}
	if len(advertised.FacilityPorts) != 1 {
		t.Fatalf("facility ports = %d, want 1", len(advertised.FacilityPorts))
	}
	port := advertised.FacilityPorts[0]
	if port.Name != "RENC-ESnet" || port.Site != "RENC" || port.Switch != "renc-data-sw" || port.VLANRange != "100-200" || port.Bandwidth != 100 {
		t.Fatalf("facility port = %+v, want decoded fields", port)
	}
	if len(advertised.Links) != 1 || advertised.Links[0].Capacity != 100 || advertised.Links[0].Layer != "L2" {
		t.Fatalf("links = %+v, want one L2 link", advertised.Links)
	}
}

func TestDecodeAdvertisedInvalidGraphML(t *testing.T) {
	t.Parallel()
	if _, err := DecodeAdvertised("<not-graphml>"); !errors.Is(err, ErrCatalogLoad) {
		t.Fatalf("DecodeAdvertised error = %v, want ErrCatalogLoad", err)
	}
}
