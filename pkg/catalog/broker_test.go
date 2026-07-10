package catalog

import (
	"errors"
	"testing"
)

// brokerModelFixture mirrors the orchestrator /resources broker model: generated
// d-key definitions, a CompositeNode site, a worker NetworkNode with components,
// and a Switch worker that must not count as a host.
const brokerModelFixture = `<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="d0" for="node" attr.name="GraphID" attr.type="string"/>
  <key id="d1" for="node" attr.name="Class" attr.type="string"/>
  <key id="d2" for="node" attr.name="NodeID" attr.type="string"/>
  <key id="d3" for="node" attr.name="Name" attr.type="string"/>
  <key id="d4" for="node" attr.name="Type" attr.type="string"/>
  <key id="d5" for="node" attr.name="Capacities" attr.type="string"/>
  <key id="d6" for="node" attr.name="CapacityAllocations" attr.type="string"/>
  <key id="d8" for="node" attr.name="Flags" attr.type="string"/>
  <key id="d9" for="node" attr.name="Site" attr.type="string"/>
  <key id="d11" for="node" attr.name="MaintenanceInfo" attr.type="string"/>
  <key id="d12" for="node" attr.name="Model" attr.type="string"/>
  <key id="d15" for="edge" attr.name="Class" attr.type="string"/>
  <graph edgedefault="undirected">
    <node id="1" labels=":GraphNode:CompositeNode">
      <data key="d1">CompositeNode</data>
      <data key="d2">nid-1</data>
      <data key="d0">g1</data>
      <data key="d3">STAR</data>
      <data key="d4">Server</data>
      <data key="d5">{"core": 768, "ram": 2820, "disk": 100}</data>
      <data key="d6">{"core": 646, "ram": 1592, "disk": 30}</data>
      <data key="d8">{"ptp": true, "ipv4_management": false}</data>
      <data key="d9">STAR</data>
      <data key="d11">{"STAR": {"state": "Active"}}</data>
    </node>
    <node id="2" labels=":GraphNode:NetworkNode">
      <data key="d1">NetworkNode</data>
      <data key="d2">nid-2</data>
      <data key="d0">g1</data>
      <data key="d3">star-w1</data>
      <data key="d4">Server</data>
      <data key="d5">{"core": 128, "ram": 478, "disk": 2233}</data>
      <data key="d6">{"core": 100, "ram": 200, "disk": 500}</data>
      <data key="d9">STAR</data>
    </node>
    <node id="3" labels=":GraphNode:Component">
      <data key="d1">Component</data>
      <data key="d2">nid-3</data>
      <data key="d0">g1</data>
      <data key="d3">GPU-RTX6000</data>
      <data key="d4">GPU</data>
      <data key="d12">RTX6000</data>
      <data key="d5">{"unit": 3}</data>
      <data key="d6">{"unit": 1}</data>
    </node>
    <node id="4" labels=":GraphNode:NetworkNode">
      <data key="d1">NetworkNode</data>
      <data key="d2">nid-4</data>
      <data key="d0">g1</data>
      <data key="d3">star-p4-sw</data>
      <data key="d4">Switch</data>
      <data key="d5">{"unit": 1}</data>
    </node>
    <node id="5" labels=":GraphNode:CompositeNode">
      <data key="d1">CompositeNode</data>
      <data key="d2">nid-5</data>
      <data key="d0">g1</data>
      <data key="d3">MAX</data>
      <data key="d4">Server</data>
      <data key="d5">{"core": 640, "ram": 2390, "disk": 100}</data>
      <data key="d6">{"core": 530, "ram": 952, "disk": 10}</data>
      <data key="d11">{"MAX": {"state": "Maint"}}</data>
    </node>
    <edge source="1" target="2" label="has"><data key="d15">has</data></edge>
    <edge source="1" target="4" label="has"><data key="d15">has</data></edge>
    <edge source="2" target="3" label="has"><data key="d15">has</data></edge>
  </graph>
</graphml>`

func TestDecodeBrokerModel(t *testing.T) {
	t.Parallel()
	adv, err := DecodeBrokerModel(brokerModelFixture)
	if err != nil {
		t.Fatalf("DecodeBrokerModel: %v", err)
	}
	if len(adv.Sites) != 2 {
		t.Fatalf("sites = %d, want 2", len(adv.Sites))
	}

	sites := map[string]Site{}
	for _, s := range adv.Sites {
		sites[s.Name] = s
	}

	star, ok := sites["STAR"]
	if !ok {
		t.Fatalf("STAR site missing")
	}
	if star.State != "Active" {
		t.Errorf("STAR state = %q, want Active", star.State)
	}
	if star.Cores.Available != 122 || star.Cores.Capacity != 768 {
		t.Errorf("STAR cores = %+v, want 122/768 available/capacity", star.Cores)
	}
	if !star.PTP || star.IPv4Management {
		t.Errorf("STAR flags = ptp %v ipv4 %v, want ptp true ipv4 false", star.PTP, star.IPv4Management)
	}
	if len(star.Hosts) != 1 || star.Hosts[0].Name != "star-w1" {
		t.Errorf("STAR hosts = %+v, want only star-w1 (switch excluded)", star.Hosts)
	}
	gpu := star.Components["GPU/RTX6000"]
	if gpu.Capacity != 3 || gpu.Allocated != 1 || gpu.Available != 2 {
		t.Errorf("STAR GPU/RTX6000 = %+v, want 3/1/2", gpu)
	}

	if max := sites["MAX"]; max.State != "Maint" {
		t.Errorf("MAX state = %q, want Maint", max.State)
	}
}

func TestDecodeBrokerModelInvalid(t *testing.T) {
	t.Parallel()
	if _, err := DecodeBrokerModel("<nope/>"); !errors.Is(err, ErrCatalogLoad) {
		t.Fatalf("error = %v, want ErrCatalogLoad", err)
	}
}
