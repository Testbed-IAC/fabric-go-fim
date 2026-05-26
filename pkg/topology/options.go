package topology

import "github.com/CSC478-WCU/fabric-go-fim/pkg/sliver"

// NodeOpts configures a call to Topology.AddNode.
//
// Name and Site are required for VM, Server, and Container nodes.
// Type defaults to NodeTypeVM when unset.
// NodeID, if non-empty, overrides the deterministic UUID derivation and
// pins the node to a specific identifier (useful when loading known state).
type NodeOpts struct {
	Name          string
	Site          string
	Type          sliver.NodeType
	NodeID        string
	Capacities    *sliver.Capacities
	CapacityHints *sliver.CapacityHints
	ImageRef      string
	ImageType     string
	MgmtIP        string
	BootScript    string
	Labels        *sliver.Labels
	UserData      sliver.UserData
	Tags          sliver.Tags
}

// ComponentOpts configures a call to Node.AddComponent.
//
// Name identifies the component within its parent node (e.g., "nic1", "gpu1").
// Either (Type + Model) or FABlibName must be supplied; FABlibName takes
// precedence and resolves via catalog.ResolveFABlibModel.
//
// NodeID and NSNodeID, if non-empty, override the deterministic UUID derivation
// for the Component and its internal NetworkService respectively.
// InterfaceNodeIDs, if non-nil, overrides the UUID for individual ports, keyed
// by port label (e.g., "p1", "p2").
type ComponentOpts struct {
	Name             string
	Type             sliver.ComponentType
	Model            string
	FABlibName       string
	NodeID           string
	NSNodeID         string
	InterfaceNodeIDs map[string]string
}

// InterfaceOpts configures interface creation on a NetworkService or Component port.
//
// Name is required. Type defaults to InterfaceTypeServicePort when called via
// NetworkService.AddInterface.  NodeID, if non-empty, overrides UUID derivation.
type InterfaceOpts struct {
	Name       string
	Type       sliver.InterfaceType
	NodeID     string
	Labels     *sliver.Labels
	Capacities *sliver.Capacities
}

// NetworkServiceOpts configures a call to Topology.AddNetworkService or
// Node.AddNetworkService.
//
// Name and Type are required.  Interfaces lists the component ports to connect;
// ConnectInterface is called on each in order.
//
// For FABNetv4/FABNetv6 services, set Gateway to supply routing metadata.
// For PortMirror services, use AddPortMirrorService instead (which fills
// MirrorPort, MirrorVLAN, MirrorDirection automatically).
//
// Site is inferred from the first interface when Type has a single-site
// constraint and no explicit Site is provided.
type NetworkServiceOpts struct {
	Name            string
	Type            sliver.ServiceType
	NodeID          string
	Interfaces      []*Interface
	Labels          *sliver.Labels
	Capacities      *sliver.Capacities
	Gateway         *sliver.Gateway
	ERO             *sliver.ERO
	PathInfo        *sliver.PathInfo
	Technology      string
	Site            string
	MirrorPort      string
	MirrorVLAN      string
	MirrorDirection sliver.MirrorDirection
}

// FacilityInterfaceOpts configures one port on a Facility node.
// Each port becomes an InterfaceTypeFacilityPort ConnectionPoint connected to
// the facility's internal VLAN service.
type FacilityInterfaceOpts struct {
	Name       string
	Labels     *sliver.Labels
	Capacities *sliver.Capacities
	NodeID     string
}

// FacilityOpts configures a call to Topology.AddFacility.
//
// A Facility models an external network attachment point such as an ESnet
// link or campus peering port.  AddFacility creates a NodeTypeFacility
// NetworkNode, an internal VLAN service, and one FacilityPort interface
// (or one per entry in Interfaces).
//
// Labels are applied to both the interface and the internal VLAN service to
// match Python FIM's representation of facility VLAN assignments.
// Type defaults to ServiceTypeVLAN when unset.
type FacilityOpts struct {
	Name       string
	Site       string
	Type       sliver.ServiceType
	NodeID     string
	NSNodeID   string
	Labels     *sliver.Labels
	Capacities *sliver.Capacities
	Interfaces []FacilityInterfaceOpts
}

// SwitchOpts configures a call to Topology.AddSwitch.
//
// A Switch models a NodeTypeSwitch network node with an internal P4 service and
// NPorts DedicatedPort interfaces named "p1" through "pN".
// NPorts defaults to 8 when unset.
// PortLabels and PortCapacities, if non-nil, are applied to every port.
type SwitchOpts struct {
	Name           string
	Site           string
	Type           sliver.ServiceType
	NodeID         string
	NSNodeID       string
	NPorts         int
	PortLabels     *sliver.Labels
	PortCapacities *sliver.Capacities
}

// PortMirrorOpts configures a call to Topology.AddPortMirrorService.
//
// FromInterfaceName identifies the source interface to mirror (by Name).
// FromInterfaceVLAN optionally restricts mirroring to a specific VLAN.
// ToInterface is the destination interface that receives the mirrored traffic.
// Direction defaults to MirrorBoth when unset.
// Site is inferred from ToInterface when unset.
type PortMirrorOpts struct {
	Name              string
	NodeID            string
	FromInterfaceName string
	FromInterfaceVLAN string
	ToInterface       *Interface
	Direction         sliver.MirrorDirection
	Site              string
}

// LinkOpts configures a call to Topology.AddLink.
//
// Type defaults to LinkTypeL2Path when unset.
// For Patch and L1Path links, Interfaces must contain exactly two entries.
type LinkOpts struct {
	Name       string
	Type       sliver.LinkType
	NodeID     string
	Interfaces []*Interface
}
