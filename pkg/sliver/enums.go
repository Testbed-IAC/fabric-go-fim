package sliver

// NodeType is the wire-format Type string for a NetworkNode.
// It appears as the "Type" property in ASM GraphML.
type NodeType string

// Known NetworkNode type values.
const (
	// NodeTypeServer is a bare-metal server node.
	NodeTypeServer NodeType = "Server"
	// NodeTypeVM is a virtual machine node.
	NodeTypeVM NodeType = "VM"
	// NodeTypeContainer is a container node.
	NodeTypeContainer NodeType = "Container"
	// NodeTypeSwitch is an SDN switch (P4 or OVS).
	NodeTypeSwitch NodeType = "Switch"
	// NodeTypeNAS is a network-attached storage node.
	NodeTypeNAS NodeType = "NAS"
	// NodeTypeFacility is a FABRIC facility attachment (e.g., ESnet peering point).
	NodeTypeFacility NodeType = "Facility"
)

// ServiceType is the wire-format Type string for a NetworkService.
// It appears as the "Type" property in ASM GraphML.
type ServiceType string

// Known NetworkService type values.
const (
	// ServiceTypeP4 is a P4-programmable switch service, used internally by FPGA components.
	ServiceTypeP4 ServiceType = "P4"
	// ServiceTypeOVS is an Open vSwitch service, used internally by NIC components.
	ServiceTypeOVS ServiceType = "OVS"
	// ServiceTypeVLAN is a VLAN-terminated service for facility ports.
	ServiceTypeVLAN ServiceType = "VLAN"
	// ServiceTypeMPLS is an MPLS tunnel service.
	ServiceTypeMPLS ServiceType = "MPLS"
	// ServiceTypeL2Path is an explicit layer-2 path (same- or cross-site).
	ServiceTypeL2Path ServiceType = "L2Path"
	// ServiceTypeL2STS is a cross-site layer-2 site-to-site service.
	ServiceTypeL2STS ServiceType = "L2STS"
	// ServiceTypeL2PTP is a cross-site layer-2 point-to-point service.
	ServiceTypeL2PTP ServiceType = "L2PTP"
	// ServiceTypeL2Multisite is a multi-site layer-2 service.
	ServiceTypeL2Multisite ServiceType = "L2Multisite"
	// ServiceTypeL2Bridge is a same-site layer-2 Ethernet bridge.
	ServiceTypeL2Bridge ServiceType = "L2Bridge"
	// ServiceTypeFABNetv4 is a routed IPv4 FABRIC network with gateway.
	ServiceTypeFABNetv4 ServiceType = "FABNetv4"
	// ServiceTypeFABNetv6 is a routed IPv6 FABRIC network with gateway.
	ServiceTypeFABNetv6 ServiceType = "FABNetv6"
	// ServiceTypeFABNetv4Ext is a FABNetv4 with an external stitching endpoint.
	ServiceTypeFABNetv4Ext ServiceType = "FABNetv4Ext"
	// ServiceTypeFABNetv6Ext is a FABNetv6 with an external stitching endpoint.
	ServiceTypeFABNetv6Ext ServiceType = "FABNetv6Ext"
	// ServiceTypeL3VPN is a layer-3 VPN service.
	ServiceTypeL3VPN ServiceType = "L3VPN"
	// ServiceTypePortMirror is a traffic-mirroring service.
	ServiceTypePortMirror ServiceType = "PortMirror"
)

// ComponentType is the wire-format Type string for a Component.
// It appears as the "Type" property in ASM GraphML.
type ComponentType string

// Known Component type values.
const (
	// ComponentTypeGPU is a GPU accelerator (e.g., RTX6000, Tesla T4).
	ComponentTypeGPU ComponentType = "GPU"
	// ComponentTypeSmartNIC is a SmartNIC with dedicated ports (e.g., ConnectX-6).
	ComponentTypeSmartNIC ComponentType = "SmartNIC"
	// ComponentTypeSharedNIC is a shared NIC whose ports are split across VMs (e.g., ConnectX-6 in SR-IOV mode).
	ComponentTypeSharedNIC ComponentType = "SharedNIC"
	// ComponentTypeFPGA is an FPGA accelerator with P4-programmable network ports (e.g., Xilinx-U280).
	ComponentTypeFPGA ComponentType = "FPGA"
	// ComponentTypeNVME is an NVMe SSD storage device (e.g., P4510).
	ComponentTypeNVME ComponentType = "NVME"
	// ComponentTypeStorage is a generic storage component.
	ComponentTypeStorage ComponentType = "Storage"
)

// InterfaceType is the wire-format Type string for a ConnectionPoint.
// It appears as the "Type" property in ASM GraphML.
type InterfaceType string

// Known ConnectionPoint type values.
const (
	// InterfaceTypeAccessPort is a layer-2 access port.
	InterfaceTypeAccessPort InterfaceType = "AccessPort"
	// InterfaceTypeTrunkPort is a layer-2 trunk port (multiple VLANs).
	InterfaceTypeTrunkPort InterfaceType = "TrunkPort"
	// InterfaceTypeServicePort is a service-attachment mirror of a component port,
	// created automatically by NetworkService.ConnectInterface.
	InterfaceTypeServicePort InterfaceType = "ServicePort"
	// InterfaceTypeDedicatedPort is a full, dedicated NIC port (SmartNIC, FPGA).
	InterfaceTypeDedicatedPort InterfaceType = "DedicatedPort"
	// InterfaceTypeSharedPort is a virtualized port on a shared NIC.
	InterfaceTypeSharedPort InterfaceType = "SharedPort"
	// InterfaceTypeVInt is a virtual internal interface.
	InterfaceTypeVInt InterfaceType = "vInt"
	// InterfaceTypeStitchPort is a stitching endpoint port.
	InterfaceTypeStitchPort InterfaceType = "StitchPort"
	// InterfaceTypeFacilityPort is a port on a Facility node (ESnet, etc.).
	InterfaceTypeFacilityPort InterfaceType = "FacilityPort"
	// InterfaceTypeSubInterface is a VLAN-tagged child of a DedicatedPort.
	InterfaceTypeSubInterface InterfaceType = "SubInterface"
)

// LinkType is the wire-format Type string for a Link.
// It appears as the "Type" property in ASM GraphML.
type LinkType string

// Known Link type values.
const (
	// LinkTypePatch is a short patch cable connecting a component port to its ServicePort.
	// It is created automatically by NetworkService.ConnectInterface.
	LinkTypePatch LinkType = "Patch"
	// LinkTypeL1Path is a physical layer-1 path.
	LinkTypeL1Path LinkType = "L1Path"
	// LinkTypeL2Path is a layer-2 explicit path.
	LinkTypeL2Path LinkType = "L2Path"
)

// NSLayer is the wire-format Layer string for a NetworkService or Link.
// It appears as the "Layer" property in ASM GraphML.
type NSLayer string

// Known network layer values.
const (
	LayerL0 NSLayer = "L0"
	LayerL1 NSLayer = "L1"
	LayerL2 NSLayer = "L2"
	LayerL3 NSLayer = "L3"
)

// MirrorDirection is the wire-format MirrorDirection string for a PortMirror service.
// It appears as the "MirrorDirection" property in ASM GraphML.
type MirrorDirection string

// Known PortMirror direction values.
const (
	// MirrorBoth mirrors traffic in both directions.
	MirrorBoth MirrorDirection = "Both"
	// MirrorRXOnly mirrors only received (ingress) traffic.
	MirrorRXOnly MirrorDirection = "RX_Only"
	// MirrorTXOnly mirrors only transmitted (egress) traffic.
	MirrorTXOnly MirrorDirection = "TX_Only"
)

// MaintenanceState is the wire-format state value within a MaintenanceInfo entry.
type MaintenanceState string

// Known MaintenanceInfo state values.
const (
	MaintenanceActive   MaintenanceState = "Active"
	MaintenancePreMaint MaintenanceState = "PreMaint"
	MaintenanceMaint    MaintenanceState = "Maint"
	MaintenanceUnknown  MaintenanceState = "Unknown"
)

// LayerForServiceType returns the NSLayer implied by the given ServiceType.
// It returns false if the service type is not recognised.
func LayerForServiceType(serviceType ServiceType) (NSLayer, bool) {
	switch serviceType {
	case ServiceTypeFABNetv4, ServiceTypeFABNetv6, ServiceTypeFABNetv4Ext, ServiceTypeFABNetv6Ext, ServiceTypeL3VPN:
		return LayerL3, true
	case ServiceTypeP4, ServiceTypeOVS, ServiceTypeVLAN, ServiceTypeMPLS, ServiceTypeL2Path, ServiceTypeL2STS, ServiceTypeL2PTP, ServiceTypeL2Multisite, ServiceTypeL2Bridge, ServiceTypePortMirror:
		return LayerL2, true
	default:
		return "", false
	}
}

// LayerForLinkType returns the NSLayer implied by the given LinkType.
// It returns false if the link type is not recognised.
func LayerForLinkType(linkType LinkType) (NSLayer, bool) {
	switch linkType {
	case LinkTypeL1Path:
		return LayerL1, true
	case LinkTypePatch, LinkTypeL2Path:
		return LayerL2, true
	default:
		return "", false
	}
}
