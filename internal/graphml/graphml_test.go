package graphml

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureRoundTripSemantic(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "fixtures", "bare_vm.graphml")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	g, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	var out bytes.Buffer
	if err := Write(&out, g); err != nil {
		t.Fatalf("Write: %v", err)
	}
	got, err := Read(bytes.NewReader(out.Bytes()))
	if err != nil {
		t.Fatalf("Read generated: %v", err)
	}
	if !GraphsEqual(g, got) {
		t.Fatalf("round-trip graph differs")
	}
}

func TestReadResolvesOpaqueKeyIDs(t *testing.T) {
	input := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<graphml xmlns="http://graphml.graphdrawing.org/xmlns">
  <key id="d0" for="node" attr.name="Class" attr.type="string"/>
  <key id="d1" for="node" attr.name="Name" attr.type="string"/>
  <key id="d2" for="node" attr.name="Type" attr.type="string"/>
  <key id="d3" for="node" attr.name="NodeID" attr.type="string"/>
  <key id="d4" for="node" attr.name="GraphID" attr.type="string"/>
  <graph id="G" edgedefault="directed">
    <node id="1"><data key="d0">NetworkNode</data><data key="d1">vm1</data><data key="d2">VM</data><data key="d3">node-id</data><data key="d4">graph-id</data></node>
  </graph>
</graphml>`)
	g, err := Read(bytes.NewReader(input))
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	nodes := g.Nodes()
	if len(nodes) != 1 || nodes[0].Class != "NetworkNode" || nodes[0].Props["Name"] != "vm1" {
		t.Fatalf("unexpected nodes: %#v", nodes)
	}
}
