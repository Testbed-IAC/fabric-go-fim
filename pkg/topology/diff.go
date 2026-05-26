package topology

import (
	"github.com/CSC478-WCU/fabric-go-fim/internal/graph"
	"github.com/CSC478-WCU/fabric-go-fim/pkg/diff"
)

// GraphDiff describes semantic drift between an expected topology graph and
// an actual topology graph.
type GraphDiff = diff.GraphDiff

// NodeDiff describes a node that exists on only one side of a graph diff.
type NodeDiff = diff.NodeDiff

// NodeChange describes semantic changes for a node matched by name.
type NodeChange = diff.NodeChange

// ClassChange describes a node Class mismatch.
type ClassChange = diff.ClassChange

// PropertyChange describes a normalized property mismatch.
type PropertyChange = diff.PropertyChange

// EdgeDiff describes an edge that exists on only one side of a graph diff.
type EdgeDiff = diff.EdgeDiff

// TopologyDiff wraps graph drift in a topology-facing type that can later grow
// Terraform-specific grouping without changing graph-level diff behavior.
type TopologyDiff struct {
	RawGraph GraphDiff
}

// NormalizeGraph returns a semantic, deterministic copy of g for topology
// drift comparison.
func NormalizeGraph(g *graph.Graph) (*graph.Graph, error) {
	return diff.NormalizeGraph(g)
}

// DiffGraphs compares expected and actual graphs by semantic topology intent.
func DiffGraphs(expected, actual *graph.Graph) GraphDiff {
	return diff.DiffGraphs(expected, actual)
}

// DiffTopologies compares two topology objects without exposing graph internals
// to callers such as Terraform providers.
func DiffTopologies(expected, actual *Topology) TopologyDiff {
	var expectedGraph, actualGraph *graph.Graph
	if expected != nil {
		expectedGraph = expected.g
	}
	if actual != nil {
		actualGraph = actual.g
	}
	return TopologyDiff{RawGraph: diff.DiffGraphs(expectedGraph, actualGraph)}
}

// DiffGraphML parses and compares two GraphML documents by semantic topology
// intent rather than raw XML bytes.
func DiffGraphML(expectedGraphML, actualGraphML []byte) (GraphDiff, error) {
	return diff.DiffGraphML(expectedGraphML, actualGraphML)
}

// DiffTopologyGraphML parses and compares two GraphML documents, returning the
// topology-facing diff wrapper intended for provider integration.
func DiffTopologyGraphML(expectedGraphML, actualGraphML []byte) (TopologyDiff, error) {
	graphDiff, err := diff.DiffGraphML(expectedGraphML, actualGraphML)
	if err != nil {
		return TopologyDiff{}, err
	}
	return TopologyDiff{RawGraph: graphDiff}, nil
}

// Empty reports whether the topology diff contains no semantic changes.
func (d TopologyDiff) Empty() bool {
	return d.RawGraph.Empty()
}

// HasChanges reports whether the topology diff contains any semantic changes.
func (d TopologyDiff) HasChanges() bool {
	return d.RawGraph.HasChanges()
}

// Summary returns the graph-level summary for this topology diff.
func (d TopologyDiff) Summary() string {
	return d.RawGraph.Summary()
}

// Diagnostics returns graph-level diagnostics for this topology diff.
func (d TopologyDiff) Diagnostics() []Diagnostic {
	diffDiagnostics := d.RawGraph.Diagnostics()
	out := make([]Diagnostic, 0, len(diffDiagnostics))
	for _, diag := range diffDiagnostics {
		out = append(out, diag)
	}
	return out
}
