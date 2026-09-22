package topologybuilder

import (
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
)

// l2BridgeSpec builds an L2Bridge slice connecting each named VM's SharedNIC.
func l2BridgeSpec(nodeNames ...string) SliceSpec {
	spec := SliceSpec{Name: "rt-slice"}
	var refs []InterfaceRef
	for _, name := range nodeNames {
		spec.Nodes = append(spec.Nodes, NodeSpec{
			Name: name, Site: "RENC", Cores: 2, RAM: 8, Disk: 10,
			Components: []ComponentSpec{{Name: "nic1", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"}},
		})
		refs = append(refs, InterfaceRef{Node: name, Component: "nic1"})
	}
	spec.Networks = []NetworkSpec{{Name: "l2-bridge", Type: sliver.ServiceTypeL2Bridge, Interfaces: refs}}
	return spec
}

func nodeNames(t *topology.Topology) map[string]bool {
	out := map[string]bool{}
	for _, n := range t.Nodes() {
		out[n.Name()] = true
	}
	return out
}

// Removing a node from an L2Bridge slice must drop the node and its service
// attachment while preserving every kept element's NodeID (true round-trip),
// leaving no dangling service port for the removed node.
func TestBuildModifyFromExisting_RemovesNodeAndServicePort(t *testing.T) {
	existing, existingModel, err := Build(l2BridgeSpec("vm-1", "vm-2", "vm-3", "vm-4"))
	if err != nil {
		t.Fatal(err)
	}
	// NodeID of a kept VM and the bridge in the persisted graph.
	vm1, _ := existing.Node("vm-1")
	bridge, _ := existing.NetworkService("l2-bridge")
	wantVM1ID, wantBridgeID := vm1.ID(), bridge.ID()
	bridgeIfacesBefore := len(bridge.Interfaces())

	modTopo, modGraphML, err := BuildModifyFromExisting(l2BridgeSpec("vm-1", "vm-2", "vm-3"), existingModel)
	if err != nil {
		t.Fatal(err)
	}

	names := nodeNames(modTopo)
	if names["vm-4"] {
		t.Fatal("vm-4 still present after modify")
	}
	for _, keep := range []string{"vm-1", "vm-2", "vm-3"} {
		if !names[keep] {
			t.Fatalf("kept node %s missing after modify", keep)
		}
	}

	// Kept NodeIDs are preserved (this is what stock FABRIC diffs on).
	gotVM1, _ := modTopo.Node("vm-1")
	gotBridge, _ := modTopo.NetworkService("l2-bridge")
	if gotVM1.ID() != wantVM1ID {
		t.Fatalf("vm-1 NodeID changed: %q -> %q", wantVM1ID, gotVM1.ID())
	}
	if gotBridge.ID() != wantBridgeID {
		t.Fatalf("l2-bridge NodeID changed: %q -> %q", wantBridgeID, gotBridge.ID())
	}

	// The bridge lost exactly vm-4's service port, no dangling port remains.
	if got := len(gotBridge.Interfaces()); got != bridgeIfacesBefore-1 {
		t.Fatalf("bridge interface count = %d, want %d", got, bridgeIfacesBefore-1)
	}
	for _, iface := range gotBridge.Interfaces() {
		if strings.Contains(iface.Name(), "vm-4") {
			t.Fatalf("dangling service port for removed node: %q", iface.Name())
		}
		// The orchestrator rejects a modify unless every ServicePort has exactly
		// one peer. Removing vm-4 must not disturb the kept ports' peering.
		if iface.Type() == sliver.InterfaceTypeServicePort {
			if got := len(iface.GetPeers("")); got != 1 {
				t.Fatalf("ServicePort %q has %d peers after modify, want 1", iface.Name(), got)
			}
		}
	}

	// Each kept VM keeps its per-NIC OVS service (a component-owned service must
	// not be swept up by the top-level network-service removal).
	for _, vm := range []string{"vm-1", "vm-2", "vm-3"} {
		if _, ok := modTopo.NetworkService(vm + "-nic1-l2ovs"); !ok {
			t.Fatalf("kept node %s lost its OVS network service", vm)
		}
	}

	// No vm-4 element survives anywhere in the serialized graph.
	if strings.Contains(modGraphML, "vm-4") {
		t.Fatal("serialized modify graph still references vm-4")
	}
}

func TestBuildModifyFromExisting_AddsNodeInPlace(t *testing.T) {
	existing, existingModel, err := Build(l2BridgeSpec("vm-1", "vm-2"))
	if err != nil {
		t.Fatal(err)
	}
	vm1, _ := existing.Node("vm-1")
	bridge, _ := existing.NetworkService("l2-bridge")
	wantVM1ID, wantBridgeID := vm1.ID(), bridge.ID()
	bridgeIfacesBefore := len(bridge.Interfaces())

	modTopo, modGraphML, err := BuildModifyFromExisting(l2BridgeSpec("vm-1", "vm-2", "vm-3"), existingModel)
	if err != nil {
		t.Fatal(err)
	}
	if !nodeNames(modTopo)["vm-3"] {
		t.Fatal("added node vm-3 missing after modify")
	}
	gotVM1, _ := modTopo.Node("vm-1")
	gotBridge, _ := modTopo.NetworkService("l2-bridge")
	if gotVM1.ID() != wantVM1ID {
		t.Fatalf("vm-1 NodeID changed: %q -> %q", wantVM1ID, gotVM1.ID())
	}
	if gotBridge.ID() != wantBridgeID {
		t.Fatalf("l2-bridge NodeID changed: %q -> %q", wantBridgeID, gotBridge.ID())
	}
	if got := len(gotBridge.Interfaces()); got != bridgeIfacesBefore+1 {
		t.Fatalf("bridge interface count = %d, want %d", got, bridgeIfacesBefore+1)
	}
	for _, iface := range gotBridge.Interfaces() {
		if iface.Type() == sliver.InterfaceTypeServicePort {
			if got := len(iface.GetPeers("")); got != 1 {
				t.Fatalf("ServicePort %q has %d peers after modify, want 1", iface.Name(), got)
			}
		}
	}
	if _, ok := modTopo.NetworkService("vm-3-nic1-l2ovs"); !ok {
		t.Fatal("added node vm-3 has no OVS network service")
	}
	if !strings.Contains(modGraphML, "vm-3") {
		t.Fatal("serialized modify graph does not reference vm-3")
	}

	again, _, err := BuildModifyFromExisting(l2BridgeSpec("vm-1", "vm-2", "vm-3"), modGraphML)
	if err != nil {
		t.Fatal(err)
	}
	againBridge, _ := again.NetworkService("l2-bridge")
	if got := len(againBridge.Interfaces()); got != bridgeIfacesBefore+1 {
		t.Fatalf("unchanged modify altered bridge interface count: %d, want %d", got, bridgeIfacesBefore+1)
	}
}

func TestBuildModifyFromExisting_AddsNetworkInPlace(t *testing.T) {
	spec := l2BridgeSpec("vm-1", "vm-2")
	spec.Networks = nil
	existing, existingModel, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}
	vm1, _ := existing.Node("vm-1")
	wantVM1ID := vm1.ID()

	modTopo, _, err := BuildModifyFromExisting(l2BridgeSpec("vm-1", "vm-2"), existingModel)
	if err != nil {
		t.Fatal(err)
	}
	gotVM1, _ := modTopo.Node("vm-1")
	if gotVM1.ID() != wantVM1ID {
		t.Fatalf("vm-1 NodeID changed: %q -> %q", wantVM1ID, gotVM1.ID())
	}
	bridge, ok := modTopo.NetworkService("l2-bridge")
	if !ok {
		t.Fatal("added network l2-bridge missing after modify")
	}
	if got := len(bridge.Interfaces()); got != 2 {
		t.Fatalf("bridge interface count = %d, want 2", got)
	}
}
