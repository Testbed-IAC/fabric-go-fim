// Package topologybuilder builds FABRIC topology GraphML from plain Go request
// structs.
package topologybuilder

import (
	"github.com/Testbed-IAC/fabric-go-fim/pkg/permission"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/userdata"
)

// SliceSpec describes a FABRIC slice topology request.
type SliceSpec struct {
	Name          string
	LifetimeHours int64
	Nodes         []NodeSpec
	Networks      []NetworkSpec
	Facilities    []FacilitySpec
	Switches      []SwitchSpec
}

// NodeSpec describes one compute node.
type NodeSpec struct {
	Name            string
	Site            string
	Host            string
	InstanceType    string
	ImageRef        string
	ImageType       string
	Cores           int64
	RAM             int64
	Disk            int64
	BootScript      string
	PostBootExecute []string
	PostUpdate      []string
	Labels          *sliver.Labels
	Components      []ComponentSpec
	Storage         []StorageSpec
	Routes          []userdata.Route
	PostBootUploads []PostBootUploadSpec
}

// ComponentSpec describes one component attached to a node.
type ComponentSpec struct {
	Name       string
	Type       sliver.ComponentType
	Model      string
	FABlibName string
	Labels     *sliver.Labels
}

// StorageSpec describes one storage component attached to a node.
type StorageSpec struct {
	Name      string
	Model     string
	AutoMount bool
}

// PostBootUploadSpec describes one post-boot upload task.
type PostBootUploadSpec struct {
	LocalPath  string
	RemotePath string
}

// NetworkSpec describes one top-level network service.
type NetworkSpec struct {
	Name            string
	Type            sliver.ServiceType
	Bandwidth       int64
	Site            string
	Technology      string
	Subnet          string
	Interfaces      []InterfaceRef
	Gateway         *sliver.Gateway
	MirrorFrom      string
	MirrorDirection sliver.MirrorDirection
	Labels          *sliver.Labels
}

// InterfaceRef identifies a node or facility interface to connect.
type InterfaceRef struct {
	Node          string
	Component     string
	Facility      string
	Port          int64
	Name          string
	Labels        *sliver.Labels
	SubInterfaces []SubInterfaceSpec
}

// SubInterfaceSpec describes one VLAN sub-interface.
type SubInterfaceSpec struct {
	Name      string
	VLAN      string
	Bandwidth int64
	Labels    *sliver.Labels
}

// FacilitySpec describes one facility port node.
type FacilitySpec struct {
	Name       string
	Site       string
	VLAN       string
	Bandwidth  int64
	MTU        int64
	Labels     *sliver.Labels
	Interfaces []FacilityInterfaceSpec
}

// FacilityInterfaceSpec describes one facility interface.
type FacilityInterfaceSpec struct {
	Name   string
	VLAN   string
	Labels *sliver.Labels
}

// SwitchSpec describes one switch node.
type SwitchSpec struct {
	Name       string
	Site       string
	NPorts     int64
	PortLabels *sliver.Labels
}

// PermissionRequest returns the permission evaluation request for spec.
func PermissionRequest(spec SliceSpec) permission.Request {
	req := permission.Request{LifetimeHours: spec.LifetimeHours}
	for _, node := range spec.Nodes {
		n := permission.Node{
			Name:  node.Name,
			Site:  node.Site,
			Cores: node.Cores,
			RAM:   node.RAM,
			Disk:  node.Disk,
		}
		for _, component := range node.Components {
			n.Components = append(n.Components, permission.Component{
				Type:  string(component.Type),
				Model: component.Model,
			})
		}
		req.Nodes = append(req.Nodes, n)
	}
	for _, network := range spec.Networks {
		req.Networks = append(req.Networks, permission.Network{
			Type:      string(network.Type),
			Bandwidth: network.Bandwidth,
		})
	}
	return req
}
