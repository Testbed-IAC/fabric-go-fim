package diff

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/internal/graphml"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// GraphDiff describes semantic drift between an expected topology graph and
// an actual topology graph. Added values exist only in the actual graph;
// removed values exist only in the expected graph.
type GraphDiff struct {
	AddedNodes   []NodeDiff
	RemovedNodes []NodeDiff
	ChangedNodes []NodeChange
	AddedEdges   []EdgeDiff
	RemovedEdges []EdgeDiff
}

// NodeDiff describes a node that exists on only one side of a graph diff.
type NodeDiff struct {
	Name  string
	Class string
	Props map[string]string
}

// NodeChange describes semantic changes for a node matched by Name.
type NodeChange struct {
	Name            string
	Class           string
	ClassChanged    *ClassChange
	PropertyChanges []PropertyChange
}

// ClassChange describes a node Class mismatch.
type ClassChange struct {
	Expected string
	Actual   string
}

// PropertyChange describes a normalized property mismatch.
type PropertyChange struct {
	Name     string
	Expected string
	Actual   string
}

// EdgeDiff describes an edge that exists on only one side of a graph diff.
type EdgeDiff struct {
	SourceName string
	TargetName string
	Class      string
}

// Diagnostic is a field-oriented diff diagnostic suitable for Terraform
// provider conversion.
type Diagnostic interface {
	error
	Field() string
	Suggestion() string
}

// FieldCategory identifies whether a drift diagnostic represents user intent
// or expected FABRIC-computed state.
type FieldCategory string

const (
	// FieldCategoryUserIntent marks topology fields that come from configuration.
	FieldCategoryUserIntent FieldCategory = "user_intent"
	// FieldCategoryComputed marks topology fields assigned by FABRIC at runtime.
	FieldCategoryComputed FieldCategory = "computed"
)

// ClassifiedDiagnostic couples a drift diagnostic with its reconciliation
// category.
type ClassifiedDiagnostic struct {
	Diagnostic Diagnostic
	Category   FieldCategory
}

// DiffGraphs compares expected and actual graphs by semantic topology intent,
// ignoring generated IDs, GraphML-local IDs, and runtime-only FABRIC fields.
func DiffGraphs(expected, actual *graph.Graph) GraphDiff {
	if expected == nil && actual == nil {
		return GraphDiff{}
	}
	if expected == nil {
		expected = graph.New("")
	}
	if actual == nil {
		actual = graph.New("")
	}
	normalizedExpected, err := NormalizeGraph(expected)
	if err != nil {
		return GraphDiff{}
	}
	normalizedActual, err := NormalizeGraph(actual)
	if err != nil {
		return GraphDiff{}
	}
	return diffNormalizedGraphs(normalizedExpected, normalizedActual)
}

// DiffGraphML parses and compares two GraphML documents by semantic topology
// intent rather than raw XML bytes.
func DiffGraphML(expectedGraphML, actualGraphML []byte) (GraphDiff, error) {
	expected, err := graphml.Read(bytes.NewReader(expectedGraphML))
	if err != nil {
		return GraphDiff{}, fmt.Errorf("topology: diff graphml: parse expected: %w", err)
	}
	actual, err := graphml.Read(bytes.NewReader(actualGraphML))
	if err != nil {
		return GraphDiff{}, fmt.Errorf("topology: diff graphml: parse actual: %w", err)
	}
	return DiffGraphs(expected, actual), nil
}

// Empty reports whether the graph diff contains no semantic changes.
func (d GraphDiff) Empty() bool {
	return !d.HasChanges()
}

// HasChanges reports whether the graph diff contains any semantic changes.
func (d GraphDiff) HasChanges() bool {
	return len(d.AddedNodes) > 0 ||
		len(d.RemovedNodes) > 0 ||
		len(d.ChangedNodes) > 0 ||
		len(d.AddedEdges) > 0 ||
		len(d.RemovedEdges) > 0
}

// Summary returns a deterministic human-readable one-line summary.
func (d GraphDiff) Summary() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s",
		plural(len(d.AddedNodes), "added node", "added nodes"),
		plural(len(d.RemovedNodes), "removed node", "removed nodes"),
		plural(len(d.ChangedNodes), "changed node", "changed nodes"),
		plural(len(d.AddedEdges), "added edge", "added edges"),
		plural(len(d.RemovedEdges), "removed edge", "removed edges"),
	)
}

// Diagnostics returns deterministic field-oriented diagnostics for future
// Terraform provider conversion.
func (d GraphDiff) Diagnostics() []Diagnostic {
	var out []Diagnostic
	for _, node := range d.AddedNodes {
		out = append(out, diffDiagnostic{
			field:      "graph.nodes." + node.Name,
			err:        fmt.Errorf("topology drift: added %s node %q", node.Class, node.Name),
			suggestion: "Remove the node from the current topology or add it to desired configuration.",
		})
	}
	for _, node := range d.RemovedNodes {
		out = append(out, diffDiagnostic{
			field:      "graph.nodes." + node.Name,
			err:        fmt.Errorf("topology drift: removed %s node %q", node.Class, node.Name),
			suggestion: "Recreate the node in the current topology or remove it from desired configuration.",
		})
	}
	for _, node := range d.ChangedNodes {
		if node.ClassChanged != nil {
			out = append(out, diffDiagnostic{
				field:      "graph.nodes." + node.Name + ".Class",
				err:        fmt.Errorf("topology drift: node %q class changed from %q to %q", node.Name, node.ClassChanged.Expected, node.ClassChanged.Actual),
				suggestion: "Update desired configuration or replace the current topology element.",
			})
		}
		for _, prop := range node.PropertyChanges {
			out = append(out, diffDiagnostic{
				field:      "graph.nodes." + node.Name + "." + prop.Name,
				err:        fmt.Errorf("topology drift: node %q property %q changed from %q to %q", node.Name, prop.Name, prop.Expected, prop.Actual),
				suggestion: "Update desired configuration or reconcile the current topology property.",
			})
		}
	}
	for _, edge := range d.AddedEdges {
		out = append(out, diffDiagnostic{
			field:      fmt.Sprintf("graph.edges.%s.%s.%s", edge.SourceName, edge.Class, edge.TargetName),
			err:        fmt.Errorf("topology drift: added edge (%s)-[%s]->(%s)", edge.SourceName, edge.Class, edge.TargetName),
			suggestion: "Remove the current topology connection or add it to desired configuration.",
		})
	}
	for _, edge := range d.RemovedEdges {
		out = append(out, diffDiagnostic{
			field:      fmt.Sprintf("graph.edges.%s.%s.%s", edge.SourceName, edge.Class, edge.TargetName),
			err:        fmt.Errorf("topology drift: removed edge (%s)-[%s]->(%s)", edge.SourceName, edge.Class, edge.TargetName),
			suggestion: "Recreate the current topology connection or remove it from desired configuration.",
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Field() != out[j].Field() {
			return out[i].Field() < out[j].Field()
		}
		return out[i].Error() < out[j].Error()
	})
	return out
}

// ClassifiedDiagnostics returns diagnostics with field categories used for
// reconciliation decisions.
func (d GraphDiff) ClassifiedDiagnostics() []ClassifiedDiagnostic {
	diagnostics := d.Diagnostics()
	out := make([]ClassifiedDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		out = append(out, ClassifiedDiagnostic{Diagnostic: diagnostic, Category: categoryForField(diagnostic.Field())})
	}
	return out
}

// UserIntentDiagnostics returns only diagnostics for configuration-owned
// topology fields.
func (d GraphDiff) UserIntentDiagnostics() []Diagnostic {
	classified := d.ClassifiedDiagnostics()
	out := make([]Diagnostic, 0, len(classified))
	for _, diagnostic := range classified {
		if diagnostic.Category == FieldCategoryUserIntent {
			out = append(out, diagnostic.Diagnostic)
		}
	}
	return out
}

// HasUserIntentChanges reports whether the graph diff contains configuration
// drift that should be shown to Terraform users.
func (d GraphDiff) HasUserIntentChanges() bool {
	return len(d.UserIntentDiagnostics()) > 0
}

func diffNormalizedGraphs(expected, actual *graph.Graph) GraphDiff {
	expectedByName := nodesByName(expected)
	actualByName := nodesByName(actual)
	diff := GraphDiff{}

	for _, name := range sortedNodeNames(actualByName) {
		if _, found := expectedByName[name]; !found {
			diff.AddedNodes = append(diff.AddedNodes, makeNodeDiff(actualByName[name]))
		}
	}
	for _, name := range sortedNodeNames(expectedByName) {
		if _, found := actualByName[name]; !found {
			diff.RemovedNodes = append(diff.RemovedNodes, makeNodeDiff(expectedByName[name]))
		}
	}
	for _, name := range sortedNodeNames(expectedByName) {
		expectedNode := expectedByName[name]
		actualNode, found := actualByName[name]
		if !found {
			continue
		}
		change := NodeChange{Name: name, Class: expectedNode.Class}
		if expectedNode.Class != actualNode.Class {
			change.ClassChanged = &ClassChange{Expected: expectedNode.Class, Actual: actualNode.Class}
		}
		change.PropertyChanges = diffProps(expectedNode.Props, actualNode.Props)
		if change.ClassChanged != nil || len(change.PropertyChanges) > 0 {
			diff.ChangedNodes = append(diff.ChangedNodes, change)
		}
	}

	expectedEdges := edgeSet(expected)
	actualEdges := edgeSet(actual)
	for _, key := range sortedEdgeKeys(actualEdges) {
		if _, found := expectedEdges[key]; !found {
			diff.AddedEdges = append(diff.AddedEdges, actualEdges[key])
		}
	}
	for _, key := range sortedEdgeKeys(expectedEdges) {
		if _, found := actualEdges[key]; !found {
			diff.RemovedEdges = append(diff.RemovedEdges, expectedEdges[key])
		}
	}
	return diff
}

func diffProps(expected, actual map[string]string) []PropertyChange {
	keys := make([]string, 0, len(expected))
	for key := range expected {
		if !ignoreDiffProp(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	changes := make([]PropertyChange, 0)
	for _, key := range keys {
		expectedValue := expected[key]
		actualValue := actual[key]
		if _, isJSON := sliver.JSONPropertyNames[key]; isJSON {
			expectedValue = normalizeJSON(expectedValue)
			actualValue = normalizeJSON(actualValue)
		}
		if expectedValue != actualValue {
			changes = append(changes, PropertyChange{Name: key, Expected: expectedValue, Actual: actualValue})
		}
	}
	return changes
}

func makeNodeDiff(node *graph.Node) NodeDiff {
	return NodeDiff{
		Name:  nodeName(node),
		Class: node.Class,
		Props: cloneStringMap(node.Props),
	}
}

func nodesByName(g *graph.Graph) map[string]*graph.Node {
	out := make(map[string]*graph.Node)
	for _, node := range g.Nodes() {
		out[nodeName(node)] = node
	}
	return out
}

func sortedNodeNames(nodes map[string]*graph.Node) []string {
	names := make([]string, 0, len(nodes))
	for name := range nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func edgeSet(g *graph.Graph) map[string]EdgeDiff {
	out := make(map[string]EdgeDiff)
	for _, edge := range g.Edges() {
		item := EdgeDiff{
			SourceName: nodeNameByID(g, edge.Source),
			TargetName: nodeNameByID(g, edge.Target),
			Class:      edge.Class,
		}
		out[edgeKey(item)] = item
	}
	return out
}

func edgeKey(edge EdgeDiff) string {
	return strings.Join([]string{edge.SourceName, edge.Class, edge.TargetName}, "\x00")
}

func sortedEdgeKeys(edges map[string]EdgeDiff) []string {
	keys := make([]string, 0, len(edges))
	for key := range edges {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("1 %s", singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

type diffDiagnostic struct {
	err        error
	field      string
	suggestion string
}

func (d diffDiagnostic) Error() string      { return d.err.Error() }
func (d diffDiagnostic) Unwrap() error      { return d.err }
func (d diffDiagnostic) Field() string      { return d.field }
func (d diffDiagnostic) Suggestion() string { return d.suggestion }
