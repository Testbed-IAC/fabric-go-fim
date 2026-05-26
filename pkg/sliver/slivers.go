package sliver

// BaseSliver contains the properties shared by every ASM node class.
// It is embedded by all concrete sliver types; callers should access it
// through the embedding type rather than directly.
//
// Identifiers:
//   - NodeID is a UUID that uniquely identifies the node within its graph.
//     pkg/topology derives NodeIDs deterministically from parent context.
//   - GraphID is the UUID of the topology graph that owns this node.
//
// Optional structured values (Capacities, Labels, …) are nil when unset;
// callers should check for nil before dereferencing.
type BaseSliver struct {
	NodeID              string
	GraphID             string
	Name                string
	Capacities          *Capacities
	CapacityHints       *CapacityHints
	Labels              *Labels
	CapacityAllocations *Capacities
	LabelAllocations    *Labels
	ReservationInfo     *ReservationInfo
	StructuralInfo      *StructuralInfo
	Details             string
	NodeMap             [2]string
	StitchNode          bool
	Tags                Tags
	Flags               *Flags
	UserData            UserData
	LayoutData          LayoutData
	MeasurementData     MeasurementData
	BootScript          string
	Opaque              map[string]string
}

// NodeSliver represents a NetworkNode — a compute endpoint such as a VM,
// bare-metal server, switch, NAS, or facility attachment point.
//
// The Type field narrows the role: NodeTypeVM and NodeTypeServer are compute
// nodes, NodeTypeSwitch is an SDN switch, NodeTypeFacility is a FABRIC
// facility (e.g., an ESnet link) and NodeTypeNAS is a storage node.
//
// Site identifies the FABRIC testbed site code (e.g., "RENC", "UKY") where
// the node is to be provisioned.
type NodeSliver struct {
	BaseSliver
	Type                  NodeType
	Site                  string
	ImageType             string
	ImageRef              string
	MgmtIP                string
	AllocationConstraints string
	ServiceEndpoint       string
	Location              *Location
	MaintenanceInfo       MaintenanceInfo
}

// NetworkServiceSliver represents a NetworkService — a logical network
// connectivity construct that connects two or more ConnectionPoints.
//
// Common service types include:
//   - ServiceTypeL2Bridge: same-site Ethernet bridge
//   - ServiceTypeL2STS / ServiceTypeL2PTP: cross-site layer-2 paths
//   - ServiceTypeL2Path: explicit layer-2 path (same- or cross-site)
//   - ServiceTypeFABNetv4 / ServiceTypeFABNetv6: routed IP networks
//   - ServiceTypeOVS / ServiceTypeP4: component-internal switching services
//   - ServiceTypePortMirror: traffic mirroring service
//   - ServiceTypeVLAN / ServiceTypeMPLS: VLAN/MPLS tunnel services
//
// Layer is normally derived from Type via LayerForServiceType; callers should
// rely on that function rather than setting it explicitly.
//
// Gateway, ERO, and PathInfo carry service-type-specific routing metadata.
// MirrorPort, MirrorVLAN, and MirrorDirection are only meaningful for
// ServiceTypePortMirror nodes.
type NetworkServiceSliver struct {
	BaseSliver
	Type                  ServiceType
	Layer                 NSLayer
	Technology            string
	AllocationConstraints string
	ControllerURL         string
	Site                  string
	Gateway               *Gateway
	ERO                   *ERO
	PathInfo              *PathInfo
	MirrorPort            string
	MirrorVLAN            string
	MirrorDirection       MirrorDirection
}

// ComponentSliver represents a hardware Component attached to a NetworkNode,
// such as a GPU, NIC, FPGA, or NVMe device.
//
// Type identifies the component category; Model identifies the specific
// hardware variant (e.g., "ConnectX-6", "RTX6000").  Both values must match
// an entry in the component catalog (pkg/catalog) for the component to be
// meaningful.
type ComponentSliver struct {
	BaseSliver
	Type  ComponentType
	Model string
}

// InterfaceSliver represents a ConnectionPoint — a logical port attached to a
// NetworkService or Component subtree.
//
// Type identifies the port role:
//   - InterfaceTypeDedicatedPort: a full NIC port (SmartNIC, FPGA)
//   - InterfaceTypeSharedPort: a port on a shared NIC (SharedNIC/ConnectX-6)
//   - InterfaceTypeServicePort: a service-attachment mirror of a component port
//   - InterfaceTypeSubInterface: a VLAN-tagged child of a DedicatedPort
//   - InterfaceTypeFacilityPort: a port on a Facility node
//
// Labels typically carries LocalName (the hardware port label, e.g. "p1")
// and optionally VLAN tags for SubInterface nodes.
//
// PeerLabels carries the peer-side addressing information negotiated at
// stitching time and is normally nil during topology construction.
type InterfaceSliver struct {
	BaseSliver
	Type       InterfaceType
	PeerLabels *Labels
}

// LinkSliver represents a Link — a physical or logical cable connecting two
// ConnectionPoints.
//
// In the Go FIM edge model, a Link node is created automatically by
// NetworkService.ConnectInterface between a component interface and its
// corresponding ServicePort. Direct Link construction is available via
// Topology.AddLink for explicit path modelling.
//
// Type is normally LinkTypePatch for component-to-service connections and
// LinkTypeL2Path for explicit cross-component paths.
// Layer is derived from Type via LayerForLinkType.
type LinkSliver struct {
	BaseSliver
	Type       LinkType
	Layer      NSLayer
	Technology string
}
