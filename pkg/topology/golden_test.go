package topology

// golden_test.go — cross-validation tests between Python FIM fixtures and Go output.
//
// Each test:
//  1. Reads a Python FIM fixture from testdata/fixtures/<pattern>.graphml.
//  2. Builds the semantically equivalent topology using the Go API.
//  3. Serialises the Go topology to GraphML.
//  4. Parses that output back into a property graph.
//  5. Compares the two graphs semantically using topology.DiffGraphs:
//     - same node Names
//     - same Classes and Types
//     - same edge structure (has/connects by name)
//     - same Capacities, Labels, and Site assignments
//     - NodeID and GraphID are excluded from comparison (UUIDs will differ)
//
// Tests silently skip (t.Skip) when the fixture file does not exist.
// Run testdata/generate_fixtures.py first to generate the fixture files.

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// fixturesDir is resolved relative to the package directory at test time.
const fixturesDir = "../../testdata/fixtures"

// loadFixture reads a fixture file and returns a Topology loaded from it.
// If the file does not exist the calling test is skipped.
func loadFixture(t *testing.T, name string) *Topology {
	t.Helper()
	path := filepath.Join(fixturesDir, name+".graphml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skipf("fixture %s.graphml not found — run testdata/generate_fixtures.py first", name)
	}
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	topo, err := Load(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Load fixture %s: %v", name, err)
	}
	return topo
}

// compareFixtureToGo serialises the Go topology, parses it, and compares the
// resulting graph against the Python fixture graph using name-based equality.
func compareFixtureToGo(t *testing.T, fixtureName string, goTopo *Topology) {
	t.Helper()
	fixtureTopo := loadFixture(t, fixtureName)
	assertGraphDiffEmpty(t, fixtureTopo, goTopo)
}

func assertGraphDiffEmpty(t *testing.T, expected, actual *Topology) {
	t.Helper()
	diff := DiffGraphs(expected.g, actual.g)
	if diff.Empty() {
		return
	}
	t.Fatalf("semantic graph diff: %s\n%+v", diff.Summary(), diff)
}

// ---------------------------------------------------------------------------
// Pattern tests
// ---------------------------------------------------------------------------

// TestGolden_BareVM verifies a single VM node with no components.
func TestGolden_BareVM(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-bare_vm"))
	buildBareVM(t, topo)
	compareFixtureToGo(t, "bare_vm", topo)
}

// TestGolden_VM_SharedNIC verifies a VM with one SharedNIC component
// (ConnectX-6) producing a Component → OVS service → SharedPort sub-tree.
func TestGolden_VM_SharedNIC(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-vm_shared_nic"))
	buildSharedNIC(t, topo)
	compareFixtureToGo(t, "vm_shared_nic", topo)
}

// TestGolden_VM_SmartNIC verifies a VM with one SmartNIC component
// (ConnectX-6) producing a Component → OVS service → 2× DedicatedPort sub-tree.
func TestGolden_VM_SmartNIC(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-vm_smart_nic"))
	buildSmartNIC(t, topo)
	compareFixtureToGo(t, "vm_smart_nic", topo)
}

// TestGolden_VM_GPU verifies a VM with one GPU component (RTX6000) producing a
// leaf Component with no service and no ports.
func TestGolden_VM_GPU(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-vm_gpu"))
	buildGPU(t, topo)
	compareFixtureToGo(t, "vm_gpu", topo)
}

// TestGolden_VM_NVMe verifies a VM with one NVMe component (P4510).
func TestGolden_VM_NVMe(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-vm_nvme"))
	buildNVME(t, topo)
	compareFixtureToGo(t, "vm_nvme", topo)
}

// TestGolden_VM_SubInterface verifies a VLAN-tagged SubInterface child of a
// DedicatedPort.
func TestGolden_VM_SubInterface(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-vm_subinterface"))
	buildSubInterface(t, topo)
	compareFixtureToGo(t, "vm_subinterface", topo)
}

// TestGolden_LAN_L2Bridge verifies a 3-node L2Bridge spanning two VMs on the
// same site (RENC).
func TestGolden_LAN_L2Bridge(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-lan_l2bridge"))
	buildL2Bridge(t, topo)
	compareFixtureToGo(t, "lan_l2bridge", topo)
}

// TestGolden_LAN_L2STS verifies a cross-site L2STS between RENC and UKY.
func TestGolden_LAN_L2STS(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-lan_l2sts"))
	buildL2STS(t, topo)
	compareFixtureToGo(t, "lan_l2sts", topo)
}

// TestGolden_L2PTP verifies a point-to-point L2PTP using DedicatedPorts.
func TestGolden_L2PTP(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-l2ptp"))
	buildL2PTP(t, topo)
	compareFixtureToGo(t, "l2ptp", topo)
}

// TestGolden_FABNetv4 verifies an L3 FABNetv4 service with a Gateway.
func TestGolden_FABNetv4(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-fabnetv4"))
	buildFABNetv4(t, topo)
	compareFixtureToGo(t, "fabnetv4", topo)
}

// TestGolden_FABNetv6 verifies an L3 FABNetv6 service (IPv6 gateway variant).
func TestGolden_FABNetv6(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-fabnetv6"))
	buildFABNetv6(t, topo)
	compareFixtureToGo(t, "fabnetv6", topo)
}

// TestGolden_FABNetv4Ext verifies an externally-stitched FABNetv4Ext L3 service.
func TestGolden_FABNetv4Ext(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-fabnetv4_ext"))
	buildFABNetv4Ext(t, topo)
	compareFixtureToGo(t, "fabnetv4_ext", topo)
}

// TestGolden_FABNetv6Ext verifies an externally-stitched FABNetv6Ext L3 service.
func TestGolden_FABNetv6Ext(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-fabnetv6_ext"))
	buildFABNetv6Ext(t, topo)
	compareFixtureToGo(t, "fabnetv6_ext", topo)
}

// TestGolden_L3VPN verifies a multi-site L3VPN service (RENC + UKY).
func TestGolden_L3VPN(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-l3vpn"))
	buildL3VPN(t, topo)
	compareFixtureToGo(t, "l3vpn", topo)
}

// TestGolden_L2Multisite verifies a multi-site L2Multisite service (RENC + UKY).
func TestGolden_L2Multisite(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-l2multisite"))
	buildL2Multisite(t, topo)
	compareFixtureToGo(t, "l2multisite", topo)
}

// TestGolden_MPLS verifies a single-site MPLS tunnel service.
func TestGolden_MPLS(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-mpls"))
	buildMPLS(t, topo)
	compareFixtureToGo(t, "mpls", topo)
}

// TestGolden_VLAN verifies a single-site VLAN-terminated service.
func TestGolden_VLAN(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-vlan"))
	buildVLANService(t, topo)
	compareFixtureToGo(t, "vlan", topo)
}

// TestGolden_FacilityPort verifies a Facility NetworkNode with a VLAN service
// and a FacilityPort interface.
func TestGolden_FacilityPort(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-facility_port"))
	buildFacility(t, topo)
	compareFixtureToGo(t, "facility_port", topo)
}

// TestGolden_SwitchNode verifies a Switch NetworkNode with a P4 service and
// four DedicatedPort interfaces.
func TestGolden_SwitchNode(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-switch_node"))
	buildSwitch(t, topo)
	compareFixtureToGo(t, "switch_node", topo)
}

// TestGolden_PortMirror verifies a PortMirror service that references a source
// interface by name and connects to a destination interface.
func TestGolden_PortMirror(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-port_mirror"))
	buildPortMirror(t, topo)
	compareFixtureToGo(t, "port_mirror", topo)
}

// TestGolden_ExplicitLink verifies an explicit L2Path Link connecting two
// DedicatedPort interfaces from different SmartNICs.
func TestGolden_ExplicitLink(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-explicit_link"))
	buildExplicitLink(t, topo)
	compareFixtureToGo(t, "explicit_link", topo)
}

// ---------------------------------------------------------------------------
// FABNetv6 builder (not in topology_test.go)
// ---------------------------------------------------------------------------

func buildFABNetv6(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "v6net",
		Type:       sliver.ServiceTypeFABNetv6,
		Interfaces: ifaces,
		Gateway:    &sliver.Gateway{IPv6: "2001:db8::1", IPv6Subnet: "2001:db8::/32"},
	}); err != nil {
		t.Fatalf("AddNetworkService FABNetv6: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Gap service-type builders (not in topology_test.go)
// ---------------------------------------------------------------------------

func buildFABNetv4Ext(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "v4ext",
		Type:       sliver.ServiceTypeFABNetv4Ext,
		Interfaces: ifaces,
		Gateway:    &sliver.Gateway{IPv4: "10.0.0.1", IPv4Subnet: "10.0.0.0/24"},
	}); err != nil {
		t.Fatalf("AddNetworkService FABNetv4Ext: %v", err)
	}
}

func buildFABNetv6Ext(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "v6ext",
		Type:       sliver.ServiceTypeFABNetv6Ext,
		Interfaces: ifaces,
		Gateway:    &sliver.Gateway{IPv6: "2001:db8::1", IPv6Subnet: "2001:db8::/32"},
	}); err != nil {
		t.Fatalf("AddNetworkService FABNetv6Ext: %v", err)
	}
}

func buildL3VPN(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "UKY")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "l3vpn1",
		Type:       sliver.ServiceTypeL3VPN,
		Interfaces: ifaces,
	}); err != nil {
		t.Fatalf("AddNetworkService L3VPN: %v", err)
	}
}

func buildL2Multisite(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "UKY")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "l2ms1",
		Type:       sliver.ServiceTypeL2Multisite,
		Interfaces: ifaces,
	}); err != nil {
		t.Fatalf("AddNetworkService L2Multisite: %v", err)
	}
}

func buildMPLS(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "mpls1",
		Type:       sliver.ServiceTypeMPLS,
		Interfaces: ifaces,
	}); err != nil {
		t.Fatalf("AddNetworkService MPLS: %v", err)
	}
}

func buildVLANService(t *testing.T, topo *Topology) {
	t.Helper()
	ifaces := twoSharedInterfaces(t, topo, "RENC", "RENC")
	if _, err := topo.AddNetworkService(NetworkServiceOpts{
		Name:       "vlan1",
		Type:       sliver.ServiceTypeVLAN,
		Interfaces: ifaces,
	}); err != nil {
		t.Fatalf("AddNetworkService VLAN: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Golden tests for per-catalog-model fixtures
// ---------------------------------------------------------------------------

// TestGolden_Catalog_SharedNIC_ConnectX6 cross-validates the SharedNIC ConnectX-6
// component against the Python fixture.
func TestGolden_Catalog_SharedNIC_ConnectX6(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-shared_nic_connectx6"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"})
	compareFixtureToGo(t, "catalog_shared_nic_connectx6", topo)
}

// TestGolden_Catalog_SmartNIC_ConnectX6 cross-validates the SmartNIC ConnectX-6
// component against the Python fixture.
func TestGolden_Catalog_SmartNIC_ConnectX6(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-smart_nic_connectx6"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeSmartNIC, Model: "ConnectX-6"})
	compareFixtureToGo(t, "catalog_smart_nic_connectx6", topo)
}

// TestGolden_Catalog_GPU_RTX6000 cross-validates the GPU RTX6000 component.
func TestGolden_Catalog_GPU_RTX6000(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-gpu_rtx6000"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeGPU, Model: "RTX6000"})
	compareFixtureToGo(t, "catalog_gpu_rtx6000", topo)
}

// TestGolden_Catalog_GPU_TeslaT4 cross-validates the GPU Tesla T4 component.
func TestGolden_Catalog_GPU_TeslaT4(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-gpu_teslaT4"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeGPU, Model: "Tesla T4"})
	compareFixtureToGo(t, "catalog_gpu_teslaT4", topo)
}

// TestGolden_Catalog_NVME_P4510 cross-validates the NVMe P4510 component.
func TestGolden_Catalog_NVME_P4510(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-nvme_p4510"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeNVME, Model: "P4510"})
	compareFixtureToGo(t, "catalog_nvme_p4510", topo)
}

// TestGolden_Catalog_Storage_NAS cross-validates the Storage NAS component
// (a port-less storage device).
func TestGolden_Catalog_Storage_NAS(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-storage_nas"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeStorage, Model: "NAS"})
	compareFixtureToGo(t, "catalog_storage_nas", topo)
}

// TestGolden_Catalog_FPGA_U280 cross-validates the FPGA Xilinx-U280 component.
func TestGolden_Catalog_FPGA_U280(t *testing.T) {
	topo := NewWithID(DeriveGraphID("golden-catalog-fpga_u280"))
	vm := addVM(t, topo, "vm1", "RENC")
	addComponent(t, vm, ComponentOpts{Name: "dev", Type: sliver.ComponentTypeFPGA, Model: "Xilinx-U280"})
	compareFixtureToGo(t, "catalog_fpga_u280", topo)
}

// ---------------------------------------------------------------------------
// Structural sanity tests (no fixture file required)
// ---------------------------------------------------------------------------

// TestGolden_Structural_BareVM_NodeCount verifies the node count of a bare VM
// topology (sanity-check that builders match the spec counts).
func TestGolden_Structural_BareVM_NodeCount(t *testing.T) {
	topo := New()
	buildBareVM(t, topo)
	if n := len(topo.g.Nodes()); n != 1 {
		t.Fatalf("bare VM node count = %d, want 1", n)
	}
}

// TestGolden_Structural_SharedNIC_NodeCount verifies the node count of a VM
// with a SharedNIC: VM + Component + OVS Service + SharedPort = 4 nodes.
func TestGolden_Structural_SharedNIC_NodeCount(t *testing.T) {
	topo := New()
	buildSharedNIC(t, topo)
	if n := len(topo.g.Nodes()); n != 4 {
		t.Fatalf("VM+SharedNIC node count = %d, want 4", n)
	}
}

// TestGolden_Structural_L2Bridge_NodeCount verifies the node count of a 2-VM
// L2Bridge topology: 2×(VM+Component+OVS+Port) + Service + 2×(ServicePort+Link) = 13.
func TestGolden_Structural_L2Bridge_NodeCount(t *testing.T) {
	topo := New()
	buildL2Bridge(t, topo)
	if n := len(topo.g.Nodes()); n != 13 {
		t.Fatalf("L2Bridge node count = %d, want 13", n)
	}
}

// TestGolden_Structural_FABNetv4_HasGateway verifies that the FABNetv4 service
// node carries a non-empty Gateway property after serialisation.
func TestGolden_Structural_FABNetv4_HasGateway(t *testing.T) {
	topo := New()
	buildFABNetv4(t, topo)
	svc, ok := topo.NetworkService("v4net")
	if !ok {
		t.Fatal("v4net service not found")
	}
	sl, err := svc.Sliver()
	if err != nil {
		t.Fatalf("Sliver: %v", err)
	}
	if sl.Gateway == nil || sl.Gateway.IPv4 == "" {
		t.Errorf("FABNetv4 service must have IPv4 Gateway, got %+v", sl.Gateway)
	}
}

// TestGolden_Structural_PortMirror_MirrorProps verifies that the PortMirror
// service carries MirrorPort and MirrorDirection properties.
func TestGolden_Structural_PortMirror_MirrorProps(t *testing.T) {
	topo := New()
	buildPortMirror(t, topo)
	svc, ok := topo.NetworkService("pm1")
	if !ok {
		t.Fatal("pm1 service not found")
	}
	sl, err := svc.Sliver()
	if err != nil {
		t.Fatalf("Sliver: %v", err)
	}
	if sl.MirrorPort == "" {
		t.Errorf("PortMirror service must have MirrorPort set")
	}
	if sl.MirrorDirection == "" {
		t.Errorf("PortMirror service must have MirrorDirection set")
	}
}

// TestGolden_Structural_SubInterface_VLAN verifies that the SubInterface
// carries the VLAN label from its parent DedicatedPort.
// SubInterfaces are connected to DedicatedPorts via EdgeConnects, not to the
// component's service, so we traverse ChildInterfaces() on each port.
func TestGolden_Structural_SubInterface_VLAN(t *testing.T) {
	topo := New()
	buildSubInterface(t, topo)
	vm, ok := topo.Node("vm1")
	if !ok {
		t.Fatal("vm1 not found")
	}
	found := false
	for _, comp := range vm.Components() {
		for _, port := range comp.InterfaceList() {
			for _, child := range port.ChildInterfaces() {
				found = true
				sl, err := child.Sliver()
				if err != nil {
					t.Fatalf("Sliver: %v", err)
				}
				if sl.Labels == nil || sl.Labels.VLAN == "" {
					t.Errorf("SubInterface %q must carry VLAN label", child.Name())
				}
			}
		}
	}
	if !found {
		t.Error("no SubInterface found under vm1 components")
	}
}

// TestGolden_Structural_ExplicitLink_ConnectsTwo verifies that the explicit
// L2Path NetworkService connects exactly two ServicePort interfaces.
// Python FIM creates an L2Path service (not a raw Link node) via add_network_service.
func TestGolden_Structural_ExplicitLink_ConnectsTwo(t *testing.T) {
	topo := New()
	buildExplicitLink(t, topo)
	svc, ok := topo.NetworkService("phys-link")
	if !ok {
		t.Fatal("phys-link NetworkService not found")
	}
	sl, err := svc.Sliver()
	if err != nil {
		t.Fatalf("Sliver: %v", err)
	}
	if sl.Type != sliver.ServiceTypeL2Path {
		t.Errorf("phys-link type = %q, want L2Path", sl.Type)
	}
	if ifaces := svc.Interfaces(); len(ifaces) != 2 {
		t.Errorf("phys-link interface count = %d, want 2", len(ifaces))
	}
}

// TestGolden_Structural_Switch_NPorts verifies that a switch with NPorts=4
// gets exactly 4 DedicatedPort interfaces.
func TestGolden_Structural_Switch_NPorts(t *testing.T) {
	topo := New()
	if _, err := topo.AddSwitch(SwitchOpts{Name: "sw1", Site: "RENC", NPorts: 4}); err != nil {
		t.Fatalf("AddSwitch: %v", err)
	}
	sw, ok := topo.Node("sw1")
	if !ok {
		t.Fatal("sw1 not found")
	}
	ifaces := sw.InterfaceList()
	if len(ifaces) != 4 {
		t.Errorf("switch interface count = %d, want 4", len(ifaces))
	}
	for _, iface := range ifaces {
		if iface.Type() != sliver.InterfaceTypeDedicatedPort {
			t.Errorf("switch interface %q type = %q, want DedicatedPort", iface.Name(), iface.Type())
		}
	}
}

// TestGolden_Structural_Facility_FacilityPort verifies that the Facility node
// has a FacilityPort interface with the expected VLAN label.
func TestGolden_Structural_Facility_FacilityPort(t *testing.T) {
	topo := New()
	buildFacility(t, topo)
	node, ok := topo.Node("ESnet-DTN")
	if !ok {
		t.Fatal("ESnet-DTN not found")
	}
	ifaces := node.InterfaceList()
	if len(ifaces) == 0 {
		t.Fatal("facility has no interfaces")
	}
	for _, iface := range ifaces {
		if iface.Type() == sliver.InterfaceTypeFacilityPort {
			sl, err := iface.Sliver()
			if err != nil {
				t.Fatalf("iface Sliver: %v", err)
			}
			if sl.Labels == nil || sl.Labels.VLAN == "" {
				t.Errorf("FacilityPort must carry VLAN label")
			}
			return
		}
	}
	t.Error("no FacilityPort found on ESnet-DTN")
}

// TestGolden_Structural_RoundTrip_AllPatterns verifies that every pattern
// topology survives a serialize → load round-trip and produces an identical
// graph (same NodeIDs since these are internally generated, not cross-impl).
func TestGolden_Structural_RoundTrip_AllPatterns(t *testing.T) {
	patterns := []struct {
		name  string
		build func(*testing.T, *Topology)
	}{
		{"bare_vm", buildBareVM},
		{"vm_shared_nic", buildSharedNIC},
		{"vm_smart_nic", buildSmartNIC},
		{"vm_gpu", buildGPU},
		{"vm_nvme", buildNVME},
		{"vm_subinterface", buildSubInterface},
		{"lan_l2bridge", buildL2Bridge},
		{"lan_l2sts", buildL2STS},
		{"l2ptp", buildL2PTP},
		{"fabnetv4", buildFABNetv4},
		{"fabnetv6", buildFABNetv6},
		{"fabnetv4_ext", buildFABNetv4Ext},
		{"fabnetv6_ext", buildFABNetv6Ext},
		{"l3vpn", buildL3VPN},
		{"l2multisite", buildL2Multisite},
		{"mpls", buildMPLS},
		{"vlan", buildVLANService},
		{"facility_port", buildFacility},
		{"switch_node", buildSwitch},
		{"port_mirror", buildPortMirror},
		{"explicit_link", buildExplicitLink},
	}
	for _, tt := range patterns {
		t.Run(tt.name, func(t *testing.T) {
			topo := NewWithID(DeriveGraphID("rt-" + tt.name))
			tt.build(t, topo)

			// Serialize.
			var buf bytes.Buffer
			if err := topo.Serialize(&buf); err != nil {
				t.Fatalf("Serialize: %v", err)
			}

			// Reload.
			loaded, err := Load(bytes.NewReader(buf.Bytes()))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			// The two graphs share the same deterministic UUIDs; the semantic
			// diff also verifies name-based topology equality.
			assertGraphDiffEmpty(t, topo, loaded)
		})
	}
}
