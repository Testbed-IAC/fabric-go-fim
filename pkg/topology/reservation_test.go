package topology

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// stampRI writes a ReservationInfo onto a node as the orchestrator would have
// persisted it, exercising the same PropReservationInfo encoding the graph uses.
func stampRI(t *testing.T, topo *Topology, nodeID, rid string) {
	t.Helper()
	n, ok := topo.g.Node(nodeID)
	if !ok {
		t.Fatalf("node %s not found", nodeID)
	}
	data, err := json.Marshal(sliver.ReservationInfo{ReservationID: rid, ReservationState: "Active"})
	if err != nil {
		t.Fatal(err)
	}
	n.Props[sliver.PropReservationInfo] = string(data)
	if err := topo.g.UpdateNode(nodeID, n.Props); err != nil {
		t.Fatal(err)
	}
}

func readRID(t *testing.T, topo *Topology, nodeID string) string {
	t.Helper()
	n, ok := topo.g.Node(nodeID)
	if !ok {
		t.Fatalf("node %s not found", nodeID)
	}
	raw := n.Props[sliver.PropReservationInfo]
	if raw == "" {
		return ""
	}
	var ri sliver.ReservationInfo
	if err := json.Unmarshal([]byte(raw), &ri); err != nil {
		t.Fatal(err)
	}
	return ri.ReservationID
}

// buildSliceA builds a vm1 + SharedNIC topology and returns the node and
// component NodeIDs. A second call with the same slice/element names must
// produce identical NodeIDs (they are deterministic) — that is what makes
// reservation-id stamping onto a rebuilt modify graph match exactly.
func buildSliceA(t *testing.T) (*Topology, string, string) {
	t.Helper()
	topo := NewWithID(DeriveGraphID("slice-a"))
	n, err := topo.AddNode(NodeOpts{Name: "vm1", Site: "RENC"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := n.AddComponent(ComponentOpts{Name: "nic1", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"})
	if err != nil {
		t.Fatal(err)
	}
	return topo, n.ID(), c.ID()
}

func TestCopyReservationInfoFrom(t *testing.T) {
	existing, nodeID, compID := buildSliceA(t)
	stampRI(t, existing, nodeID, "rid-node-1")
	stampRI(t, existing, compID, "rid-comp-1")

	desired, dNodeID, dCompID := buildSliceA(t)
	if dNodeID != nodeID || dCompID != compID {
		t.Fatalf("deterministic NodeIDs diverged: node %q/%q comp %q/%q", nodeID, dNodeID, compID, dCompID)
	}
	// Freshly built graph carries no reservation info.
	if got := readRID(t, desired, nodeID); got != "" {
		t.Fatalf("freshly built node already had reservation id %q", got)
	}

	if got := desired.CopyReservationInfoFrom(existing); got != 2 {
		t.Fatalf("CopyReservationInfoFrom stamped %d elements, want 2 (node + component)", got)
	}
	if got := readRID(t, desired, nodeID); got != "rid-node-1" {
		t.Fatalf("node reservation id = %q, want rid-node-1", got)
	}
	if got := readRID(t, desired, compID); got != "rid-comp-1" {
		t.Fatalf("component reservation id = %q, want rid-comp-1", got)
	}

	// Reservation info survives a serialize -> reload round-trip (so the modify
	// graph actually carries it to the orchestrator).
	model, err := desired.SerializeString()
	if err != nil {
		t.Fatal(err)
	}
	reloaded, err := Load(strings.NewReader(model))
	if err != nil {
		t.Fatal(err)
	}
	if got := readRID(t, reloaded, nodeID); got != "rid-node-1" {
		t.Fatalf("reservation id lost across serialize round-trip: got %q", got)
	}

	// Idempotent: re-copying changes nothing.
	if got := desired.CopyReservationInfoFrom(existing); got != 0 {
		t.Fatalf("second CopyReservationInfoFrom stamped %d, want 0", got)
	}
}

func TestCopyReservationInfoFrom_NoMatchesAndNilSource(t *testing.T) {
	desired, _, _ := buildSliceA(t)

	if got := desired.CopyReservationInfoFrom(nil); got != 0 {
		t.Fatalf("nil source stamped %d, want 0", got)
	}
	// Source with matching IDs but no reservation info stamps nothing.
	bare, _, _ := buildSliceA(t)
	if got := desired.CopyReservationInfoFrom(bare); got != 0 {
		t.Fatalf("reservation-free source stamped %d, want 0", got)
	}
	// Source from a different slice name has different NodeIDs -> no matches.
	other := NewWithID(DeriveGraphID("slice-b"))
	on, err := other.AddNode(NodeOpts{Name: "vm1", Site: "RENC"})
	if err != nil {
		t.Fatal(err)
	}
	stampRI(t, other, on.ID(), "rid-other")
	if got := desired.CopyReservationInfoFrom(other); got != 0 {
		t.Fatalf("cross-slice source stamped %d, want 0", got)
	}
}
