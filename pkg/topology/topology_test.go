package topology

import (
	"bytes"
	"errors"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

func TestConstructionPatterns(t *testing.T) {
	tests := []struct {
		name  string
		build func(*testing.T, *Topology)
		nodes int
	}{
		{name: "bare VM", build: buildBareVM, nodes: 1},
		{name: "VM with shared NIC", build: buildSharedNIC, nodes: 4},
		{name: "VM with SmartNIC", build: buildSmartNIC, nodes: 5},
		{name: "VM with GPU", build: buildGPU, nodes: 2},
		{name: "VM with NVMe", build: buildNVME, nodes: 2},
		{name: "SubInterface", build: buildSubInterface, nodes: 6},
		{name: "L2Bridge", build: buildL2Bridge, nodes: 13},
		{name: "L2STS", build: buildL2STS, nodes: 13},
		{name: "L2PTP", build: buildL2PTP, nodes: 15},
		{name: "FABNetv4", build: buildFABNetv4, nodes: 13},
		{name: "Facility", build: buildFacility, nodes: 3},
		{name: "Switch", build: buildSwitch, nodes: 6},
		{name: "PortMirror", build: buildPortMirror, nodes: 14},
		{name: "Explicit Link", build: buildExplicitLink, nodes: 15},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topo := NewWithID(DeriveGraphID(tt.name))
			tt.build(t, topo)
			if got := len(topo.g.Nodes()); got != tt.nodes {
				t.Fatalf("node count = %d, want %d", got, tt.nodes)
			}
			var buffer bytes.Buffer
			if err := topo.Serialize(&buffer); err != nil {
				t.Fatalf("Serialize: %v", err)
			}
			if _, err := Load(bytes.NewReader(buffer.Bytes())); err != nil {
				t.Fatalf("Load serialized: %v", err)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	topo := NewWithID("graph-id")
	if _, err := topo.AddNode(NodeOpts{Name: "v", Type: sliver.NodeTypeVM}); !errors.Is(err, ErrInvalidOption) {
		t.Fatalf("short name error = %v, want ErrInvalidOption", err)
	}
	if _, err := topo.AddNode(NodeOpts{Name: "vm1", Type: sliver.NodeTypeVM}); !errors.Is(err, ErrConstraintViolation) {
		t.Fatalf("missing site error = %v, want ErrConstraintViolation", err)
	}
}

func buildBareVM(t *testing.T, topo *Topology) {
	t.Helper()
	addVM(t, topo, "vm1", "RENC")
}

func buildSharedNIC(t *testing.T, topo *Topology) {
	t.Helper()
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "nic1", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"})
}

func buildSmartNIC(t *testing.T, topo *Topology) {
	t.Helper()
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "snic", Type: sliver.ComponentTypeSmartNIC, Model: "ConnectX-6"})
}

func buildGPU(t *testing.T, topo *Topology) {
	t.Helper()
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "gpu1", Type: sliver.ComponentTypeGPU, Model: "RTX6000"})
}

func buildNVME(t *testing.T, topo *Topology) {
	t.Helper()
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "ssd", Type: sliver.ComponentTypeNVME, Model: "P4510"})
}

func buildSubInterface(t *testing.T, topo *Topology) {
	t.Helper()
	vm := addVM(t, topo, "vm1", "RENC")
	component := addComponent(t, vm, ComponentOpts{Name: "snic", Type: sliver.ComponentTypeSmartNIC, Model: "ConnectX-6"})
	iface := component.InterfaceList()[0]
	if _, err := iface.AddChildInterface(InterfaceOpts{Name: "sub100", Labels: &sliver.Labels{VLAN: "100"}}); err != nil {
		t.Fatalf("AddChildInterface: %v", err)
	}
}

func buildL2Bridge(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{Name: "lan1", Type: sliver.ServiceTypeL2Bridge, Interfaces: ifaces}); err != nil {
		t.Fatalf("AddNetworkService: %v", err)
	}
}

func buildL2STS(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "UKY")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{Name: "lan1", Type: sliver.ServiceTypeL2STS, Interfaces: ifaces}); err != nil {
		t.Fatalf("AddNetworkService: %v", err)
	}
}

func buildL2PTP(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoDedicatedInterfaces(t, topo, "RENC", "UKY")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{Name: "ptp1", Type: sliver.ServiceTypeL2PTP, Interfaces: ifaces}); err != nil {
		t.Fatalf("AddNetworkService: %v", err)
	}
}

func buildFABNetv4(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{Name: "v4net", Type: sliver.ServiceTypeFABNetv4, Interfaces: ifaces, Gateway: &sliver.Gateway{IPv4: "10.0.0.1", IPv4Subnet: "10.0.0.0/24"}}); err != nil {
		t.Fatalf("AddNetworkService: %v", err)
	}
}

func buildFacility(t *testing.T, topo *Topology) {
	t.Helper()
	if _, err := topo.AddFacility(FacilityOpts{Name: "ESnet-DTN", Site: "RENC", Labels: &sliver.Labels{VLAN: "100"}, Capacities: &sliver.Capacities{BW: 10}}); err != nil {
		t.Fatalf("AddFacility: %v", err)
	}
}

func buildSwitch(t *testing.T, topo *Topology) {
	t.Helper()
	if _, err := topo.AddSwitch(SwitchOpts{Name: "sw1", Site: "RENC", NPorts: 4}); err != nil {
		t.Fatalf("AddSwitch: %v", err)
	}
}

func buildPortMirror(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{Name: "lan1", Type: sliver.ServiceTypeL2Bridge, Interfaces: []*Interface{ifaces[0]}}); err != nil {
		t.Fatalf("source service: %v", err)
	}
	if _, err := topo.AddPortMirrorService(PortMirrorOpts{Name: "pm1", FromInterfaceName: ifaces[0].Name(), FromInterfaceVLAN: "100", ToInterface: ifaces[1], Direction: sliver.MirrorBoth}); err != nil {
		t.Fatalf("AddPortMirrorService: %v", err)
	}
}

func buildExplicitLink(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoDedicatedInterfaces(t, topo, "RENC", "RENC")
	// Python FIM: add_network_service(nstype=L2Path) creates a NetworkService + service ports + patch links.
	if _, err := topo.AddNetworkService(NetworkServiceOpts{Name: "phys-link", Type: sliver.ServiceTypeL2Path, Interfaces: ifaces}); err != nil {
		t.Fatalf("AddNetworkService L2Path: %v", err)
	}
}

func addVM(t *testing.T, topo *Topology, name, site string) *Node {
	t.Helper()
	vm, err := topo.AddNode(NodeOpts{Name: name, Site: site, Type: sliver.NodeTypeVM, Capacities: &sliver.Capacities{Core: 2, RAM: 8, Disk: 10}, ImageRef: "default_rocky_9", ImageType: "qcow2"})
	if err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	return vm
}

func addComponent(t *testing.T, node *Node, opts ComponentOpts) *Component {
	t.Helper()
	component, err := node.AddComponent(opts)
	if err != nil {
		t.Fatalf("AddComponent: %v", err)
	}
	return component
}

func twoSharedInterfaces(t *testing.T, topo *Topology, siteA, siteB string) []*Interface {
	t.Helper()
	vm1 := addVM(t, topo, "vm1", siteA)
	vm2 := addVM(t, topo, "vm2", siteB)
	c1 := addComponent(t, vm1, ComponentOpts{Name: "nic1", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"})
	c2 := addComponent(t, vm2, ComponentOpts{Name: "nic1", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"})
	return []*Interface{c1.InterfaceList()[0], c2.InterfaceList()[0]}
}

func twoDedicatedInterfaces(t *testing.T, topo *Topology, siteA, siteB string) []*Interface {
	t.Helper()
	vm1 := addVM(t, topo, "vm1", siteA)
	vm2 := addVM(t, topo, "vm2", siteB)
	c1 := addComponent(t, vm1, ComponentOpts{Name: "snic1", Type: sliver.ComponentTypeSmartNIC, Model: "ConnectX-6"})
	c2 := addComponent(t, vm2, ComponentOpts{Name: "snic1", Type: sliver.ComponentTypeSmartNIC, Model: "ConnectX-6"})
	return []*Interface{c1.InterfaceList()[0], c2.InterfaceList()[0]}
}
