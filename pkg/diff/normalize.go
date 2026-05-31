package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

var ignoredDiffProps = map[string]struct{}{
	sliver.PropClass:               {},
	sliver.PropGraphID:             {},
	sliver.PropNodeID:              {},
	sliver.PropXMLLabels:           {},
	sliver.PropMgmtIP:              {},
	sliver.PropReservationInfo:     {},
	sliver.PropCapacityAllocations: {},
	sliver.PropLabelAllocations:    {},
	sliver.PropFlags:               {},
	sliver.PropMaintenanceInfo:     {},
	sliver.PropMeasurementData:     {},
	sliver.PropLayoutData:          {},
}

var computedDiffProps = map[string]struct{}{
	sliver.PropMgmtIP:              {},
	sliver.PropReservationInfo:     {},
	sliver.PropCapacityAllocations: {},
	sliver.PropLabelAllocations:    {},
	sliver.PropMaintenanceInfo:     {},
	sliver.PropMeasurementData:     {},
	sliver.PropLayoutData:          {},
}

// NormalizeGraph returns a semantic, deterministic copy of g for topology
// drift comparison. Generated IDs and runtime-only properties are removed from
// comparison state, JSON-valued properties are canonicalized, and nodes and
// edges are ordered by stable names instead of insertion or XML ID order.
func NormalizeGraph(g *graph.Graph) (*graph.Graph, error) {
	if g == nil {
		return nil, fmt.Errorf("topology: normalize graph: graph is nil")
	}

	nodes := g.Nodes()
	sort.SliceStable(nodes, func(i, j int) bool {
		return compareNodes(nodes[i], nodes[j]) < 0
	})

	out := graph.New("normalized")
	oldIDToNewID := make(map[string]string, len(nodes))
	usedIDs := make(map[string]int, len(nodes))
	for _, node := range nodes {
		name := nodeName(node)
		newID := normalizedNodeID(name, node.Class, usedIDs)
		props := normalizeProps(node.Props, name, node.Class)
		props[sliver.PropNodeID] = newID
		props[sliver.PropGraphID] = out.ID
		props[sliver.PropClass] = node.Class
		if err := out.AddNode(node.Class, newID, props); err != nil {
			return nil, fmt.Errorf("topology: normalize graph: add node %q: %w", name, err)
		}
		oldIDToNewID[node.ID] = newID
	}

	edges := g.Edges()
	sort.SliceStable(edges, func(i, j int) bool {
		return compareEdges(edges[i], edges[j], g) < 0
	})
	for _, edge := range edges {
		source := oldIDToNewID[edge.Source]
		target := oldIDToNewID[edge.Target]
		if source == "" || target == "" {
			return nil, fmt.Errorf("topology: normalize graph: edge %s references missing endpoint %q -> %q", edge.Class, edge.Source, edge.Target)
		}
		if err := out.AddEdge(edge.Class, source, target); err != nil {
			return nil, fmt.Errorf("topology: normalize graph: add edge %s: %w", edge.Class, err)
		}
	}
	return out, nil
}

func normalizeProps(props map[string]string, name, class string) map[string]string {
	out := make(map[string]string, len(props))
	for key, value := range props {
		if ignoreDiffProp(key) {
			continue
		}
		if _, isJSON := sliver.JSONPropertyNames[key]; isJSON {
			value = normalizeJSON(value)
		}
		if value != "" && value != "None" {
			out[key] = value
		}
	}
	if name != "" {
		out[sliver.PropName] = name
	}
	out[sliver.PropClass] = class
	return out
}

func normalizeJSON(value string) string {
	if value == "" {
		return value
	}
	var decoded any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return value
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return value
	}
	return string(encoded)
}

func ignoreDiffProp(key string) bool {
	if _, ignored := ignoredDiffProps[key]; ignored {
		return true
	}
	lower := strings.ToLower(key)
	return strings.Contains(lower, "timestamp") ||
		strings.Contains(lower, "lease") ||
		strings.Contains(lower, "reservationid") ||
		strings.Contains(lower, "reservation_id") ||
		strings.Contains(lower, "sliverid") ||
		strings.Contains(lower, "sliver_id") ||
		strings.Contains(lower, "slicestate") ||
		strings.Contains(lower, "slice_state")
}

func categoryForField(field string) FieldCategory {
	for key := range computedDiffProps {
		if strings.HasSuffix(field, "."+key) {
			return FieldCategoryComputed
		}
	}
	lower := strings.ToLower(field)
	if strings.Contains(lower, "allocation") ||
		strings.Contains(lower, "reservation") ||
		strings.Contains(lower, "management_ip") ||
		strings.Contains(lower, "mgmtip") ||
		strings.Contains(lower, "sliver") ||
		strings.Contains(lower, "maintenance") ||
		strings.Contains(lower, "measurement") {
		return FieldCategoryComputed
	}
	return FieldCategoryUserIntent
}

func normalizedNodeID(name, class string, used map[string]int) string {
	base := name
	if base == "" {
		base = "unnamed"
	}
	base = strings.NewReplacer(" ", "_", "/", "_", ":", "_").Replace(base)
	base = class + ":" + base
	if count := used[base]; count > 0 {
		used[base] = count + 1
		return fmt.Sprintf("%s:%d", base, count+1)
	}
	used[base] = 1
	return base
}

func compareNodes(left, right *graph.Node) int {
	if cmp := strings.Compare(nodeName(left), nodeName(right)); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(left.Class, right.Class); cmp != 0 {
		return cmp
	}
	return strings.Compare(left.ID, right.ID)
}

func compareEdges(left, right *graph.Edge, g *graph.Graph) int {
	leftKey := edgeSortKey(left, g)
	rightKey := edgeSortKey(right, g)
	return strings.Compare(leftKey, rightKey)
}

func edgeSortKey(edge *graph.Edge, g *graph.Graph) string {
	sourceName := nodeNameByID(g, edge.Source)
	targetName := nodeNameByID(g, edge.Target)
	return edge.Class + "\x00" + sourceName + "\x00" + targetName
}

func nodeNameByID(g *graph.Graph, id string) string {
	node, ok := g.Node(id)
	if !ok {
		return ""
	}
	return nodeName(node)
}

func nodeName(node *graph.Node) string {
	if node == nil {
		return ""
	}
	if name := node.Props[sliver.PropName]; name != "" {
		return name
	}
	return node.ID
}
