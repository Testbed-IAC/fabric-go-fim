package topology

import (
	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/diff"
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

// FieldCategory identifies whether a drift diagnostic represents user intent
// or expected FABRIC-computed state.
type FieldCategory = diff.FieldCategory

const (
	// FieldCategoryUserIntent marks topology fields that come from configuration.
	FieldCategoryUserIntent = diff.FieldCategoryUserIntent
	// FieldCategoryComputed marks topology fields assigned by FABRIC at runtime.
	FieldCategoryComputed = diff.FieldCategoryComputed
)

// ClassifiedDiagnostic couples a drift diagnostic with its reconciliation
// category.
type ClassifiedDiagnostic = diff.ClassifiedDiagnostic

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

// HasUserIntentChanges reports whether the topology diff contains
// configuration-owned drift.
func (d TopologyDiff) HasUserIntentChanges() bool {
	return d.RawGraph.HasUserIntentChanges()
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

// ClassifiedDiagnostics returns topology drift diagnostics with field
// categories used for reconciliation decisions.
func (d TopologyDiff) ClassifiedDiagnostics() []ClassifiedDiagnostic {
	return d.RawGraph.ClassifiedDiagnostics()
}

// UserIntentDiagnostics returns only diagnostics for configuration-owned
// topology fields.
func (d TopologyDiff) UserIntentDiagnostics() []Diagnostic {
	diffDiagnostics := d.RawGraph.UserIntentDiagnostics()
	out := make([]Diagnostic, 0, len(diffDiagnostics))
	for _, diag := range diffDiagnostics {
		out = append(out, diag)
	}
	return out
}
