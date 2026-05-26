package diff

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/internal/graphml"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

func TestDiffGraphsEmptyForSameGraph(t *testing.T) {
	g := testGraph(t, "graph-a", testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)})

	diff := DiffGraphs(g, g)

	if !diff.Empty() {
		t.Fatalf("diff = %+v, want empty", diff)
	}
	if got, want := diff.Summary(), "0 added nodes, 0 removed nodes, 0 changed nodes, 0 added edges, 0 removed edges"; got != want {
		t.Fatalf("Summary() = %q, want %q", got, want)
	}
}

func TestDiffGraphsEmptyForSemanticallyEquivalentFixture(t *testing.T) {
	expected := testGraph(t, "fixture-equivalent", testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)})
	expectedGraphML := writeTestGraphML(t, expected)
	actual, err := graphml.Read(strings.NewReader(string(expectedGraphML)))
	if err != nil {
		t.Fatalf("Read expected GraphML: %v", err)
	}

	diff := DiffGraphs(expected, actual)

	if !diff.Empty() {
		t.Fatalf("diff = %+v, want empty", diff)
	}
}

func TestDiffGraphsAddedNode(t *testing.T) {
	expected := testGraph(t, "expected", testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)})
	actual := testGraph(t, "actual",
		testNode{id: "node-a2", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b", name: "vm2", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "UKY"}},
	)

	diff := DiffGraphs(expected, actual)

	if len(diff.AddedNodes) != 1 {
		t.Fatalf("AddedNodes = %+v, want one added node", diff.AddedNodes)
	}
	if got := diff.AddedNodes[0]; got.Name != "vm2" || got.Class != sliver.ClassNetworkNode || got.Props[sliver.PropSite] != "UKY" {
		t.Fatalf("AddedNodes[0] = %+v, want vm2 NetworkNode with Site", got)
	}
}

func TestDiffGraphsRemovedNode(t *testing.T) {
	expected := testGraph(t, "expected",
		testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b", name: "vm2", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
	)
	actual := testGraph(t, "actual", testNode{id: "node-a2", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)})

	diff := DiffGraphs(expected, actual)

	if len(diff.RemovedNodes) != 1 {
		t.Fatalf("RemovedNodes = %+v, want one removed node", diff.RemovedNodes)
	}
	if got := diff.RemovedNodes[0]; got.Name != "vm2" || got.Class != sliver.ClassNetworkNode {
		t.Fatalf("RemovedNodes[0] = %+v, want vm2 NetworkNode", got)
	}
}

func TestDiffGraphsChangedClass(t *testing.T) {
	expected := testGraph(t, "expected", testNode{id: "node-a", name: "thing1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)})
	actual := testGraph(t, "actual", testNode{id: "node-a2", name: "thing1", class: sliver.ClassComponent, typ: string(sliver.ComponentTypeGPU)})

	diff := DiffGraphs(expected, actual)

	if len(diff.ChangedNodes) != 1 || diff.ChangedNodes[0].ClassChanged == nil {
		t.Fatalf("ChangedNodes = %+v, want class change", diff.ChangedNodes)
	}
	change := diff.ChangedNodes[0].ClassChanged
	if change.Expected != sliver.ClassNetworkNode || change.Actual != sliver.ClassComponent {
		t.Fatalf("ClassChanged = %+v, want NetworkNode -> Component", change)
	}
}

func TestDiffGraphsChangedMeaningfulProperty(t *testing.T) {
	expected := testGraph(t, "expected", testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "RENC"}})
	actual := testGraph(t, "actual", testNode{id: "node-a2", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "UKY"}})

	diff := DiffGraphs(expected, actual)

	if len(diff.ChangedNodes) != 1 || len(diff.ChangedNodes[0].PropertyChanges) != 1 {
		t.Fatalf("ChangedNodes = %+v, want one property change", diff.ChangedNodes)
	}
	change := diff.ChangedNodes[0].PropertyChanges[0]
	if change.Name != sliver.PropSite || change.Expected != "RENC" || change.Actual != "UKY" {
		t.Fatalf("PropertyChanges[0] = %+v, want Site RENC -> UKY", change)
	}
}

func TestDiffGraphsIgnoresUUIDAndRuntimeDifferences(t *testing.T) {
	expected := testGraph(t, "expected", testNode{
		id:    "expected-node-id",
		name:  "vm1",
		class: sliver.ClassNetworkNode,
		typ:   string(sliver.NodeTypeVM),
		props: map[string]string{
			sliver.PropSite:            "RENC",
			sliver.PropMgmtIP:          "192.0.2.10",
			sliver.PropReservationInfo: `{"reservation_id":"expected"}`,
		},
	})
	actual := testGraph(t, "actual", testNode{
		id:    "actual-node-id",
		name:  "vm1",
		class: sliver.ClassNetworkNode,
		typ:   string(sliver.NodeTypeVM),
		props: map[string]string{
			sliver.PropSite:            "RENC",
			sliver.PropMgmtIP:          "192.0.2.200",
			sliver.PropReservationInfo: `{"reservation_id":"actual"}`,
		},
	})

	diff := DiffGraphs(expected, actual)

	if !diff.Empty() {
		t.Fatalf("diff = %+v, want runtime and UUID differences ignored", diff)
	}
}

func TestDiffGraphsEdgeDrift(t *testing.T) {
	expected := testGraph(t, "expected",
		testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b", name: "nic1", class: sliver.ClassComponent, typ: string(sliver.ComponentTypeSharedNIC)},
	)
	actual := testGraph(t, "actual",
		testNode{id: "node-a2", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b2", name: "nic1", class: sliver.ClassComponent, typ: string(sliver.ComponentTypeSharedNIC)},
	)
	if err := actual.AddEdge(sliver.EdgeHas, "node-a2", "node-b2"); err != nil {
		t.Fatalf("AddEdge actual: %v", err)
	}

	diff := DiffGraphs(expected, actual)

	if len(diff.AddedEdges) != 1 || len(diff.RemovedEdges) != 0 {
		t.Fatalf("edge diff = added %+v removed %+v, want one added edge", diff.AddedEdges, diff.RemovedEdges)
	}
	if got := diff.AddedEdges[0]; got.SourceName != "vm1" || got.TargetName != "nic1" || got.Class != sliver.EdgeHas {
		t.Fatalf("AddedEdges[0] = %+v, want vm1 has nic1", got)
	}
}

func TestDiffGraphsJSONNormalization(t *testing.T) {
	expected := testGraph(t, "expected", testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropLabels: `{"b":"2","a":"1"}`}})
	actual := testGraph(t, "actual", testNode{id: "node-a2", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropLabels: `{"a":"1","b":"2"}`}})

	diff := DiffGraphs(expected, actual)

	if !diff.Empty() {
		t.Fatalf("diff = %+v, want equivalent JSON ignored", diff)
	}
}

func TestDiffGraphsDeterministicOrdering(t *testing.T) {
	expectedA := testGraph(t, "expected",
		testNode{id: "node-b", name: "vm-b", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "RENC"}},
		testNode{id: "node-a", name: "vm-a", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-c", name: "vm-c", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
	)
	actualA := testGraph(t, "actual",
		testNode{id: "node-c2", name: "vm-c", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b2", name: "vm-b", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "UKY"}},
		testNode{id: "node-d", name: "vm-d", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
	)
	if err := expectedA.AddEdge(sliver.EdgeHas, "node-a", "node-b"); err != nil {
		t.Fatalf("AddEdge expectedA: %v", err)
	}
	if err := actualA.AddEdge(sliver.EdgeHas, "node-c2", "node-d"); err != nil {
		t.Fatalf("AddEdge actualA: %v", err)
	}

	expectedB := testGraph(t, "expected",
		testNode{id: "node-c", name: "vm-c", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-a", name: "vm-a", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b", name: "vm-b", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "RENC"}},
	)
	actualB := testGraph(t, "actual",
		testNode{id: "node-d", name: "vm-d", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b2", name: "vm-b", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropSite: "UKY"}},
		testNode{id: "node-c2", name: "vm-c", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
	)
	if err := expectedB.AddEdge(sliver.EdgeHas, "node-a", "node-b"); err != nil {
		t.Fatalf("AddEdge expectedB: %v", err)
	}
	if err := actualB.AddEdge(sliver.EdgeHas, "node-c2", "node-d"); err != nil {
		t.Fatalf("AddEdge actualB: %v", err)
	}

	diffA := DiffGraphs(expectedA, actualA)
	diffB := DiffGraphs(expectedB, actualB)

	if diffA.Summary() != diffB.Summary() {
		t.Fatalf("Summary order changed: %q != %q", diffA.Summary(), diffB.Summary())
	}
	if renderDiagnostics(diffA.Diagnostics()) != renderDiagnostics(diffB.Diagnostics()) {
		t.Fatalf("Diagnostics changed:\nA: %q\nB: %q", renderDiagnostics(diffA.Diagnostics()), renderDiagnostics(diffB.Diagnostics()))
	}
	if renderGraphDiff(diffA) != renderGraphDiff(diffB) {
		t.Fatalf("diff ordering changed:\nA: %s\nB: %s", renderGraphDiff(diffA), renderGraphDiff(diffB))
	}
}

func TestDiffGraphMLMatchesDiffGraphs(t *testing.T) {
	expected := testGraph(t, "expected",
		testNode{id: "node-a", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
		testNode{id: "node-b", name: "vm2", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
	)
	actual := testGraph(t, "actual", testNode{id: "node-a2", name: "vm1", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)})
	expectedBytes := writeTestGraphML(t, expected)
	actualBytes := writeTestGraphML(t, actual)

	fromGraphML, err := DiffGraphML(expectedBytes, actualBytes)
	if err != nil {
		t.Fatalf("DiffGraphML: %v", err)
	}
	fromGraphs := DiffGraphs(expected, actual)

	if renderGraphDiff(fromGraphML) != renderGraphDiff(fromGraphs) {
		t.Fatalf("DiffGraphML diff = %s, want %s", renderGraphDiff(fromGraphML), renderGraphDiff(fromGraphs))
	}
}

func TestNormalizeGraphReturnsDeterministicCopy(t *testing.T) {
	original := testGraph(t, "original",
		testNode{id: "node-b", name: "vm-b", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM), props: map[string]string{sliver.PropLabels: `{"z":"1","a":"2"}`}},
		testNode{id: "node-a", name: "vm-a", class: sliver.ClassNetworkNode, typ: string(sliver.NodeTypeVM)},
	)
	normalized, err := NormalizeGraph(original)
	if err != nil {
		t.Fatalf("NormalizeGraph: %v", err)
	}

	nodes := normalized.Nodes()
	if len(nodes) != 2 || nodes[0].Props[sliver.PropName] != "vm-a" || nodes[1].Props[sliver.PropName] != "vm-b" {
		t.Fatalf("normalized node order = %+v, want vm-a then vm-b", nodes)
	}
	if original.Nodes()[0].Props[sliver.PropName] != "vm-b" {
		t.Fatalf("NormalizeGraph mutated input node order")
	}
	if got := nodes[1].Props[sliver.PropLabels]; got != `{"a":"2","z":"1"}` {
		t.Fatalf("normalized Labels = %q, want canonical JSON", got)
	}
}

type testNode struct {
	id    string
	name  string
	class string
	typ   string
	props map[string]string
}

func testGraph(t *testing.T, graphID string, nodes ...testNode) *graph.Graph {
	t.Helper()
	g := graph.New(graphID)
	for _, node := range nodes {
		props := map[string]string{
			sliver.PropClass:   node.class,
			sliver.PropName:    node.name,
			sliver.PropNodeID:  node.id,
			sliver.PropGraphID: graphID,
			sliver.PropType:    node.typ,
		}
		for key, value := range node.props {
			props[key] = value
		}
		if err := g.AddNode(node.class, node.id, props); err != nil {
			t.Fatalf("AddNode %s: %v", node.name, err)
		}
	}
	return g
}

func writeTestGraphML(t *testing.T, g *graph.Graph) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if err := graphml.Write(&buffer, g); err != nil {
		t.Fatalf("graphml.Write: %v", err)
	}
	return buffer.Bytes()
}

func renderDiagnostics(diags []Diagnostic) string {
	var b strings.Builder
	for _, diag := range diags {
		b.WriteString(diag.Field())
		b.WriteString("=")
		b.WriteString(diag.Error())
		b.WriteString("|")
		b.WriteString(diag.Suggestion())
		b.WriteString("\n")
	}
	return b.String()
}

func renderGraphDiff(diff GraphDiff) string {
	var b strings.Builder
	b.WriteString(diff.Summary())
	b.WriteString("\nadded nodes:")
	for _, item := range diff.AddedNodes {
		b.WriteString(item.Name)
		b.WriteString("/")
		b.WriteString(item.Class)
		b.WriteString(";")
	}
	b.WriteString("\nremoved nodes:")
	for _, item := range diff.RemovedNodes {
		b.WriteString(item.Name)
		b.WriteString("/")
		b.WriteString(item.Class)
		b.WriteString(";")
	}
	b.WriteString("\nchanged nodes:")
	for _, item := range diff.ChangedNodes {
		b.WriteString(item.Name)
		b.WriteString("/")
		b.WriteString(item.Class)
		b.WriteString(";")
		for _, prop := range item.PropertyChanges {
			b.WriteString(prop.Name)
			b.WriteString("=")
			b.WriteString(prop.Expected)
			b.WriteString("->")
			b.WriteString(prop.Actual)
			b.WriteString(";")
		}
	}
	b.WriteString("\nadded edges:")
	for _, item := range diff.AddedEdges {
		b.WriteString(item.SourceName)
		b.WriteString("-")
		b.WriteString(item.Class)
		b.WriteString("->")
		b.WriteString(item.TargetName)
		b.WriteString(";")
	}
	b.WriteString("\nremoved edges:")
	for _, item := range diff.RemovedEdges {
		b.WriteString(item.SourceName)
		b.WriteString("-")
		b.WriteString(item.Class)
		b.WriteString("->")
		b.WriteString(item.TargetName)
		b.WriteString(";")
	}
	return b.String()
}
