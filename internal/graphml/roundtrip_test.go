package graphml

// roundtrip_test.go — additional round-trip tests for the GraphML reader/writer.
// Covers fixture files (when present) and a set of hand-crafted edge cases.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CSC478-WCU/fabric-go-fim/internal/graph"
)

// ---------------------------------------------------------------------------
// Fixture-driven round-trip (requires testdata/fixtures/)
// ---------------------------------------------------------------------------

// TestRoundTrip_AllFixtures reads every .graphml file in testdata/fixtures/,
// writes it back with the Go writer, reads the output again, and asserts the
// two parsed graphs are semantically identical (same NodeIDs, Classes, and
// Props — the writer preserves everything stored in the property map).
// The test skips silently when the fixtures directory is absent or empty.
func TestRoundTrip_AllFixtures(t *testing.T) {
	fixturesDir := filepath.Join("..", "..", "testdata", "fixtures")
	entries, err := os.ReadDir(fixturesDir)
	if os.IsNotExist(err) || (err == nil && len(entries) == 0) {
		t.Skip("testdata/fixtures/ is absent or empty — run testdata/generate_fixtures.py first")
	}
	if err != nil {
		t.Fatalf("read fixtures dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".graphml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".graphml")
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(fixturesDir, entry.Name())
			input, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}
			roundTripAssertEqual(t, input)
		})
	}
}

// ---------------------------------------------------------------------------
// Edge-case round-trips (inline GraphML, no fixture file required)
// ---------------------------------------------------------------------------

// TestRoundTrip_OpaqueKeyIDs verifies that d0/d1/… style key IDs (as produced
// by NetworkX) are resolved to their attr.name before comparison.
func TestRoundTrip_OpaqueKeyIDs(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="d0" for="node" attr.name="Class"   attr.type="string"/>
  <key id="d1" for="node" attr.name="Name"    attr.type="string"/>
  <key id="d2" for="node" attr.name="Type"    attr.type="string"/>
  <key id="d3" for="node" attr.name="NodeID"  attr.type="string"/>
  <key id="d4" for="node" attr.name="GraphID" attr.type="string"/>
  <key id="d5" for="edge" attr.name="Class"   attr.type="string"/>
  <key id="d6" for="edge" attr.name="label"   attr.type="string"/>
  <graph id="G" edgedefault="directed">
    <node id="n0">
      <data key="d0">NetworkNode</data>
      <data key="d1">vm1</data>
      <data key="d2">VM</data>
      <data key="d3">aaaaaaaa-0000-0000-0000-000000000001</data>
      <data key="d4">aaaaaaaa-0000-0000-0000-000000000000</data>
    </node>
    <node id="n1">
      <data key="d0">Component</data>
      <data key="d1">vm1-nic1</data>
      <data key="d2">SharedNIC</data>
      <data key="d3">aaaaaaaa-0000-0000-0000-000000000002</data>
      <data key="d4">aaaaaaaa-0000-0000-0000-000000000000</data>
    </node>
    <edge id="e0" source="n0" target="n1">
      <data key="d5">has</data>
      <data key="d6">has</data>
    </edge>
  </graph>
</graphml>`)
	roundTripAssertEqual(t, input)
}

// TestRoundTrip_NamedKeyIDs verifies that named key IDs (attr.name == id) —
// as used by the FIM neo4j exporter — survive a round-trip unchanged.
func TestRoundTrip_NamedKeyIDs(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="Class"   for="node" attr.name="Class"   attr.type="string"/>
  <key id="Name"    for="node" attr.name="Name"    attr.type="string"/>
  <key id="Type"    for="node" attr.name="Type"    attr.type="string"/>
  <key id="NodeID"  for="node" attr.name="NodeID"  attr.type="string"/>
  <key id="GraphID" for="node" attr.name="GraphID" attr.type="string"/>
  <key id="Site"    for="node" attr.name="Site"    attr.type="string"/>
  <key id="Class"   for="edge" attr.name="Class"   attr.type="string"/>
  <key id="label"   for="edge" attr.name="label"   attr.type="string"/>
  <graph id="G" edgedefault="directed">
    <node id="n0">
      <data key="Class">NetworkNode</data>
      <data key="Name">vm1</data>
      <data key="Type">VM</data>
      <data key="NodeID">bbbbbbbb-0000-0000-0000-000000000001</data>
      <data key="GraphID">bbbbbbbb-0000-0000-0000-000000000000</data>
      <data key="Site">RENC</data>
    </node>
  </graph>
</graphml>`)
	g, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read named-key GraphML: %v", err)
	}
	if n := g.Nodes(); len(n) != 1 || n[0].Props["Site"] != "RENC" {
		t.Fatalf("unexpected nodes after named-key parse: %#v", g.Nodes())
	}
	roundTripAssertEqual(t, input)
}

// TestRoundTrip_CDATAValues verifies that property values wrapped in CDATA
// sections are decoded transparently and survive a round-trip.
func TestRoundTrip_CDATAValues(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="Class"      for="node" attr.name="Class"      attr.type="string"/>
  <key id="Name"       for="node" attr.name="Name"       attr.type="string"/>
  <key id="Type"       for="node" attr.name="Type"       attr.type="string"/>
  <key id="NodeID"     for="node" attr.name="NodeID"     attr.type="string"/>
  <key id="GraphID"    for="node" attr.name="GraphID"    attr.type="string"/>
  <key id="Capacities" for="node" attr.name="Capacities" attr.type="string"/>
  <graph id="G" edgedefault="directed">
    <node id="n0">
      <data key="Class">NetworkNode</data>
      <data key="Name">vm1</data>
      <data key="Type">VM</data>
      <data key="NodeID">cccccccc-0000-0000-0000-000000000001</data>
      <data key="GraphID">cccccccc-0000-0000-0000-000000000000</data>
      <data key="Capacities"><![CDATA[{"core":4,"ram":16,"disk":50}]]></data>
    </node>
  </graph>
</graphml>`)
	g, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read CDATA GraphML: %v", err)
	}
	nodes := g.Nodes()
	if len(nodes) != 1 {
		t.Fatalf("want 1 node, got %d", len(nodes))
	}
	if got := nodes[0].Props["Capacities"]; got != `{"core":4,"ram":16,"disk":50}` {
		t.Fatalf("Capacities = %q, want JSON string without CDATA wrapper", got)
	}
	roundTripAssertEqual(t, input)
}

// TestRoundTrip_EmptyTopology verifies that a GraphML document with no nodes
// is parsed into an empty graph and can be written back without error.
func TestRoundTrip_EmptyTopology(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <graph id="G" edgedefault="directed">
  </graph>
</graphml>`)
	g, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read empty topology: %v", err)
	}
	if n := len(g.Nodes()); n != 0 {
		t.Fatalf("want 0 nodes, got %d", n)
	}
	var out bytes.Buffer
	if err := Write(&out, g); err != nil {
		t.Fatalf("Write empty topology: %v", err)
	}
	got, err := Read(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("Read written empty topology: %v", err)
	}
	if !GraphsEqual(g, got) {
		t.Fatalf("empty topology round-trip differs")
	}
}

// TestRoundTrip_SingleNodeNoEdges verifies that a single node with no edges
// round-trips cleanly.
func TestRoundTrip_SingleNodeNoEdges(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="Class"   for="node" attr.name="Class"   attr.type="string"/>
  <key id="Name"    for="node" attr.name="Name"    attr.type="string"/>
  <key id="Type"    for="node" attr.name="Type"    attr.type="string"/>
  <key id="NodeID"  for="node" attr.name="NodeID"  attr.type="string"/>
  <key id="GraphID" for="node" attr.name="GraphID" attr.type="string"/>
  <graph id="G" edgedefault="directed">
    <node id="n0">
      <data key="Class">NetworkNode</data>
      <data key="Name">standalone</data>
      <data key="Type">VM</data>
      <data key="NodeID">dddddddd-0000-0000-0000-000000000001</data>
      <data key="GraphID">dddddddd-0000-0000-0000-000000000000</data>
    </node>
  </graph>
</graphml>`)
	roundTripAssertEqual(t, input)
}

// TestRoundTrip_MultipleEdgeClasses verifies that both "has" and "connects"
// edge classes survive a round-trip without conflation.
func TestRoundTrip_MultipleEdgeClasses(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="Class"   for="node" attr.name="Class"   attr.type="string"/>
  <key id="Name"    for="node" attr.name="Name"    attr.type="string"/>
  <key id="Type"    for="node" attr.name="Type"    attr.type="string"/>
  <key id="NodeID"  for="node" attr.name="NodeID"  attr.type="string"/>
  <key id="GraphID" for="node" attr.name="GraphID" attr.type="string"/>
  <key id="Class"   for="edge" attr.name="Class"   attr.type="string"/>
  <key id="label"   for="edge" attr.name="label"   attr.type="string"/>
  <graph id="G" edgedefault="directed">
    <node id="n0">
      <data key="Class">NetworkNode</data>
      <data key="Name">vm1</data>
      <data key="Type">VM</data>
      <data key="NodeID">eeeeeeee-0000-0000-0000-000000000001</data>
      <data key="GraphID">eeeeeeee-0000-0000-0000-000000000000</data>
    </node>
    <node id="n1">
      <data key="Class">Component</data>
      <data key="Name">vm1-nic1</data>
      <data key="Type">SharedNIC</data>
      <data key="NodeID">eeeeeeee-0000-0000-0000-000000000002</data>
      <data key="GraphID">eeeeeeee-0000-0000-0000-000000000000</data>
    </node>
    <node id="n2">
      <data key="Class">NetworkService</data>
      <data key="Name">vm1-nic1-l2ovs</data>
      <data key="Type">OVS</data>
      <data key="NodeID">eeeeeeee-0000-0000-0000-000000000003</data>
      <data key="GraphID">eeeeeeee-0000-0000-0000-000000000000</data>
    </node>
    <node id="n3">
      <data key="Class">ConnectionPoint</data>
      <data key="Name">vm1-nic1-p1</data>
      <data key="Type">SharedPort</data>
      <data key="NodeID">eeeeeeee-0000-0000-0000-000000000004</data>
      <data key="GraphID">eeeeeeee-0000-0000-0000-000000000000</data>
    </node>
    <edge id="e0" source="n0" target="n1" label="has">
      <data key="Class">has</data>
      <data key="label">has</data>
    </edge>
    <edge id="e1" source="n1" target="n2" label="has">
      <data key="Class">has</data>
      <data key="label">has</data>
    </edge>
    <edge id="e2" source="n2" target="n3" label="connects">
      <data key="Class">connects</data>
      <data key="label">connects</data>
    </edge>
  </graph>
</graphml>`)
	g, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	edges := g.Edges()
	if len(edges) != 3 {
		t.Fatalf("want 3 edges, got %d", len(edges))
	}
	classCounts := map[string]int{}
	for _, e := range edges {
		classCounts[e.Class]++
	}
	if classCounts["has"] != 2 || classCounts["connects"] != 1 {
		t.Fatalf("edge class counts = %v, want has:2 connects:1", classCounts)
	}
	roundTripAssertEqual(t, input)
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// roundTripAssertEqual reads input, writes it back, reads again, and asserts
// the two parsed graphs are identical using GraphsEqual (UUID-based equality,
// which is correct here because the writer preserves NodeIDs stored in Props).
func roundTripAssertEqual(t *testing.T, input []byte) {
	t.Helper()
	g1, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read first pass: %v", err)
	}
	var buf bytes.Buffer
	if err := Write(&buf, g1); err != nil {
		t.Fatalf("Write: %v", err)
	}
	g2, err := Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Read second pass: %v", err)
	}
	if !GraphsEqual(g1, g2) {
		t.Errorf("round-trip graph differs")
		logGraphDiff(t, g1, g2)
	}
}

// logGraphDiff logs a human-readable diff between two graphs when they differ.
func logGraphDiff(t *testing.T, expected, actual *graph.Graph) {
	t.Helper()
	expNodes := expected.Nodes()
	actNodes := actual.Nodes()
	t.Logf("node counts: expected=%d actual=%d", len(expNodes), len(actNodes))
	n := len(expNodes)
	if len(actNodes) < n {
		n = len(actNodes)
	}
	for i := 0; i < n; i++ {
		e, a := expNodes[i], actNodes[i]
		if e.ID != a.ID || e.Class != a.Class {
			t.Logf("  node[%d]: expected {ID:%s Class:%s Name:%s} actual {ID:%s Class:%s Name:%s}",
				i, e.ID, e.Class, e.Props["Name"], a.ID, a.Class, a.Props["Name"])
		}
	}
	expEdges := expected.Edges()
	actEdges := actual.Edges()
	t.Logf("edge counts: expected=%d actual=%d", len(expEdges), len(actEdges))
}
