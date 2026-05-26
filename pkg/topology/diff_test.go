package topology

import (
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

func TestDiffTopologies(t *testing.T) {
	expected := NewWithID("expected")
	if _, err := expected.AddNode(NodeOpts{Name: "vm1", Site: "RENC", Type: sliver.NodeTypeVM}); err != nil {
		t.Fatalf("AddNode expected: %v", err)
	}
	actual := NewWithID("actual")
	if _, err := actual.AddNode(NodeOpts{Name: "vm1", Site: "UKY", Type: sliver.NodeTypeVM}); err != nil {
		t.Fatalf("AddNode actual: %v", err)
	}

	diff := DiffTopologies(expected, actual)

	if !diff.HasChanges() || !diff.RawGraph.HasChanges() {
		t.Fatalf("diff = %+v, want topology changes", diff)
	}
	if got := diff.Summary(); got != "0 added nodes, 0 removed nodes, 1 changed node, 0 added edges, 0 removed edges" {
		t.Fatalf("Summary() = %q", got)
	}
}

func TestDiffTopologyGraphML(t *testing.T) {
	expected := NewWithID("expected")
	if _, err := expected.AddNode(NodeOpts{Name: "vm1", Site: "RENC", Type: sliver.NodeTypeVM}); err != nil {
		t.Fatalf("AddNode expected: %v", err)
	}
	actual := NewWithID("actual")
	if _, err := actual.AddNode(NodeOpts{Name: "vm1", Site: "RENC", Type: sliver.NodeTypeVM}); err != nil {
		t.Fatalf("AddNode actual: %v", err)
	}
	expectedGraphML, err := expected.SerializeString()
	if err != nil {
		t.Fatalf("SerializeString expected: %v", err)
	}
	actualGraphML, err := actual.SerializeString()
	if err != nil {
		t.Fatalf("SerializeString actual: %v", err)
	}

	diff, err := DiffTopologyGraphML([]byte(expectedGraphML), []byte(actualGraphML))
	if err != nil {
		t.Fatalf("DiffTopologyGraphML: %v", err)
	}

	if !diff.Empty() {
		t.Fatalf("diff = %+v, want empty semantic topology diff", diff)
	}
}
