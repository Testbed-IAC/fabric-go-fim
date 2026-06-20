package topologybuilder

import (
	"fmt"
	"net"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/catalog"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/userdata"
)

// BuildModifyFromExisting builds a modify graph by loading the orchestrator's
// persisted slice model and removing the nodes and network services the desired
// spec drops. This preserves every persisted NodeID, reservation id, and
// structural element (service ports, links) the same way fablib does, which is
// what stock FABRIC's modify path expects.
//
// Additions (a node or network service not already in the persisted slice) and an
// empty existingModel are not reconciled in place yet and fall back to Build.
func BuildModifyFromExisting(spec SliceSpec, existingModel string) (*topology.Topology, string, error) {
	if strings.TrimSpace(existingModel) == "" {
		return Build(spec)
	}
	base, err := topology.Load(strings.NewReader(existingModel))
	if err != nil {
		return nil, "", fmt.Errorf("loading existing slice model: %w", err)
	}

	keepNodes := make(map[string]bool, len(spec.Nodes)+len(spec.Facilities))
	for _, n := range spec.Nodes {
		keepNodes[n.Name] = true
	}
	for _, f := range spec.Facilities {
		keepNodes[f.Name] = true
	}
	keepNets := make(map[string]bool, len(spec.Networks))
	for _, nw := range spec.Networks {
		keepNets[nw.Name] = true
	}

	// Additions are not reconciled against the existing graph yet, so fall back
	// to a full rebuild when the spec introduces a node or network service the
	// persisted slice does not already have.
	for _, n := range spec.Nodes {
		if _, ok := base.Node(n.Name); !ok {
			return Build(spec)
		}
	}
	for _, nw := range spec.Networks {
		if _, ok := base.NetworkService(nw.Name); !ok {
			return Build(spec)
		}
	}

	// Remove dropped nodes, detaching their network-service interfaces first so
	// no dangling service port or link is left behind on a kept service.
	for _, node := range base.Nodes() {
		if keepNodes[node.Name()] {
			continue
		}
		ifaces := node.InterfaceList()
		for _, svc := range base.NetworkServices() {
			for _, iface := range ifaces {
				_ = svc.DisconnectInterface(iface)
			}
		}
		if err := base.RemoveNode(node.Name()); err != nil {
			return nil, "", fmt.Errorf("removing node %q: %w", node.Name(), err)
		}
	}

	// Remove dropped top-level network services. Component-owned services (e.g.
	// per-NIC OVS realizations) are left to their node's removal.
	for _, svc := range base.SliceNetworkServices() {
		if keepNets[svc.Name()] {
			continue
		}
		if err := base.RemoveNetworkService(svc.Name()); err != nil {
			return nil, "", fmt.Errorf("removing network service %q: %w", svc.Name(), err)
		}
	}

	graphML, err := base.SerializeString()
	if err != nil {
		return nil, "", fmt.Errorf("serializing modify topology: %w", err)
	}
	return base, graphML, nil
}

// Build constructs a topology and serialized GraphML from spec.
func Build(spec SliceSpec) (*topology.Topology, string, error) {
	topo := topology.NewWithID(topology.DeriveGraphID(spec.Name))
	nodes := map[string]*topology.Node{}

	for _, node := range spec.Nodes {
		labels, err := nodeLabels(node)
		if err != nil {
			return nil, "", fmt.Errorf("building labels for node %s: %w", node.Name, err)
		}
		opts := topology.NodeOpts{
			Name:       node.Name,
			Site:       node.Site,
			Type:       sliver.NodeTypeVM,
			ImageRef:   defaultString(node.ImageRef, "default_rocky_9"),
			ImageType:  defaultString(node.ImageType, "qcow2"),
			BootScript: node.BootScript,
			Labels:     labels,
		}
		userData, err := assembleUserData(node)
		if err != nil {
			return nil, "", fmt.Errorf("building user-data for node %s: %w", node.Name, err)
		}
		if len(userData) > 0 {
			opts.UserData = userData
		}
		if node.InstanceType != "" {
			opts.CapacityHints = &sliver.CapacityHints{InstanceType: node.InstanceType}
			if caps, ok := explicitCapacities(node); ok {
				opts.Capacities = &caps
			}
		} else {
			caps := CapacitiesFromNode(node)
			opts.Capacities = &caps
		}
		built, err := topo.AddNode(opts)
		if err != nil {
			return nil, "", fmt.Errorf("adding node %s: %w", node.Name, err)
		}
		nodes[node.Name] = built
		for _, component := range node.Components {
			componentOpts := topology.ComponentOpts{
				Name:       component.Name,
				Type:       component.Type,
				Model:      component.Model,
				FABlibName: component.FABlibName,
				Labels:     component.Labels,
			}
			if _, err := built.AddComponent(componentOpts); err != nil {
				return nil, "", fmt.Errorf("adding component %s: %w", component.Name, err)
			}
		}
		for _, storage := range node.Storage {
			storageOpts := topology.ComponentOpts{
				Name:  storage.Name,
				Type:  sliver.ComponentTypeStorage,
				Model: defaultString(storage.Model, "NAS"),
			}
			if _, err := built.AddComponent(storageOpts); err != nil {
				return nil, "", fmt.Errorf("adding storage %s: %w", storage.Name, err)
			}
		}
	}

	facilities := map[string]*topology.Node{}
	for _, facility := range spec.Facilities {
		built, err := buildFacility(topo, facility)
		if err != nil {
			return nil, "", fmt.Errorf("adding facility %s: %w", facility.Name, err)
		}
		facilities[facility.Name] = built
	}

	for _, sw := range spec.Switches {
		if err := buildSwitch(topo, sw); err != nil {
			return nil, "", fmt.Errorf("adding switch %s: %w", sw.Name, err)
		}
	}

	for _, network := range spec.Networks {
		networkLabels := network.Labels
		if network.Type == "PortMirror" {
			firstIface := firstInterface(network)
			toInterface, err := resolveNetworkInterface(nodes, facilities, firstIface)
			if err != nil {
				return nil, "", fmt.Errorf("resolving mirror destination: %w", err)
			}
			if err := toInterface.SetLabels(firstIface.Labels); err != nil {
				return nil, "", fmt.Errorf("setting labels for port mirror %s interface: %w", network.Name, err)
			}
			_, err = topo.AddPortMirrorService(topology.PortMirrorOpts{
				Name:              network.Name,
				FromInterfaceName: network.MirrorFrom,
				ToInterface:       toInterface,
				Direction:         NormalizeMirrorDirection(network.MirrorDirection),
				Labels:            networkLabels,
			})
			if err != nil {
				return nil, "", fmt.Errorf("adding port mirror %s: %w", network.Name, err)
			}
			continue
		}
		ifaces := make([]*topology.Interface, 0, len(network.Interfaces))
		for _, ifaceSpec := range network.Interfaces {
			iface, err := resolveNetworkInterface(nodes, facilities, ifaceSpec)
			if err != nil {
				return nil, "", fmt.Errorf("resolving interface for network %s: %w", network.Name, err)
			}
			if err := iface.SetLabels(ifaceSpec.Labels); err != nil {
				return nil, "", fmt.Errorf("setting labels for network %s interface: %w", network.Name, err)
			}
			if err := addSubInterfaces(iface, ifaceSpec.SubInterfaces); err != nil {
				return nil, "", fmt.Errorf("adding sub-interfaces for network %s: %w", network.Name, err)
			}
			ifaces = append(ifaces, iface)
		}
		serviceType, err := resolveServiceType(topo, network, ifaces)
		if err != nil {
			return nil, "", fmt.Errorf("resolving type for network %s: %w", network.Name, err)
		}
		gateway, gatewayLabels, err := gatewayFromNetwork(network)
		if err != nil {
			return nil, "", fmt.Errorf("building gateway for network %s: %w", network.Name, err)
		}
		networkLabels = mergeNetworkLabels(networkLabels, gatewayLabels)
		opts := topology.NetworkServiceOpts{
			Name:       network.Name,
			Type:       serviceType,
			Interfaces: ifaces,
			Labels:     networkLabels,
			Site:       network.Site,
			Technology: network.Technology,
			Gateway:    gateway,
		}
		if network.Bandwidth > 0 {
			opts.Capacities = &sliver.Capacities{BW: int(network.Bandwidth)}
		}
		if _, err := topo.AddNetworkService(opts); err != nil {
			return nil, "", fmt.Errorf("adding network %s: %w", network.Name, err)
		}
	}

	graphML, err := topo.SerializeString()
	if err != nil {
		return nil, "", fmt.Errorf("serializing topology: %w", err)
	}
	return topo, graphML, nil
}

// ValidateCatalog validates instance and component selections against the embedded catalogs.
func ValidateCatalog(spec SliceSpec) error {
	instances, err := catalog.Instances()
	if err != nil {
		return fmt.Errorf("loading instance catalog: %w", err)
	}
	components, err := catalog.Components()
	if err != nil {
		return fmt.Errorf("loading component catalog: %w", err)
	}
	for _, node := range spec.Nodes {
		if node.InstanceType != "" {
			if _, ok := instances.Lookup(node.InstanceType); !ok {
				return fmt.Errorf("looking up instance type %s: %w", node.InstanceType, catalog.ErrNotFound)
			}
		}
		for _, component := range node.Components {
			componentType := component.Type
			componentModel := component.Model
			if component.FABlibName != "" {
				resolvedType, resolvedModel, ok := catalog.ResolveFABlibModel(component.FABlibName)
				if !ok {
					return fmt.Errorf("resolving FABlib model %s: %w", component.FABlibName, catalog.ErrNotFound)
				}
				componentType = resolvedType
				componentModel = resolvedModel
			}
			if _, ok := components.Lookup(componentType, componentModel); !ok {
				return fmt.Errorf("looking up component %s/%s: %w", componentType, componentModel, catalog.ErrNotFound)
			}
		}
	}
	return nil
}

func nodeLabels(node NodeSpec) (*sliver.Labels, error) {
	labels := node.Labels
	host := node.Host
	if host == "" {
		return labels, nil
	}
	if labels == nil {
		return &sliver.Labels{InstanceParent: host}, nil
	}
	if labels.InstanceParent != "" && labels.InstanceParent != host {
		return nil, fmt.Errorf("node host %q conflicts with labels.instance_parent %q", host, labels.InstanceParent)
	}
	labelCopy := *labels
	labelCopy.InstanceParent = host
	return &labelCopy, nil
}

func explicitCapacities(node NodeSpec) (sliver.Capacities, bool) {
	caps := sliver.Capacities{Core: int(node.Cores), RAM: int(node.RAM), Disk: int(node.Disk)}
	return caps, !caps.Empty()
}

// CapacitiesFromNode returns explicit/default VM capacities for node.
func CapacitiesFromNode(node NodeSpec) sliver.Capacities {
	core := node.Cores
	ram := node.RAM
	disk := node.Disk
	if core == 0 {
		core = 2
	}
	if ram == 0 {
		ram = 8
	}
	if disk == 0 {
		disk = 10
	}
	return sliver.Capacities{Core: int(core), RAM: int(ram), Disk: int(disk)}
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func resolveNetworkInterface(nodes, facilities map[string]*topology.Node, spec InterfaceRef) (*topology.Interface, error) {
	if spec.Facility != "" {
		facility, ok := facilities[spec.Facility]
		if !ok {
			return nil, fmt.Errorf("facility %q: %w", spec.Facility, topology.ErrNotFound)
		}
		return resolveNodeInterface(facility, spec.Facility, spec)
	}
	if spec.Node == "" {
		return nil, fmt.Errorf("%w: interface must set node or facility", topology.ErrInvalidOption)
	}
	node, ok := nodes[spec.Node]
	if !ok {
		return nil, fmt.Errorf("node %q: %w", spec.Node, topology.ErrNotFound)
	}
	return resolveNodeInterface(node, spec.Node, spec)
}

func resolveNodeInterface(node *topology.Node, nodeName string, spec InterfaceRef) (*topology.Interface, error) {
	port := int(spec.Port)
	if spec.Component != "" {
		for _, component := range node.Components() {
			if component.Name() != spec.Component && component.Name() != node.Name()+"-"+spec.Component {
				continue
			}
			interfaces := component.Interfaces()
			if port < 0 || port >= len(interfaces) {
				return nil, fmt.Errorf("component %q port %d: %w", spec.Component, port, topology.ErrNotFound)
			}
			return interfaces[port], nil
		}
		return nil, fmt.Errorf("component %q: %w", spec.Component, topology.ErrNotFound)
	}
	if spec.Name != "" {
		for _, iface := range node.InterfaceList() {
			if iface.Name() == spec.Name {
				return iface, nil
			}
		}
		return nil, fmt.Errorf("interface %q: %w", spec.Name, topology.ErrNotFound)
	}
	interfaces := node.InterfaceList()
	if port < 0 || port >= len(interfaces) {
		return nil, fmt.Errorf("node %q port %d: %w", nodeName, port, topology.ErrNotFound)
	}
	return interfaces[port], nil
}

func firstInterface(network NetworkSpec) InterfaceRef {
	if len(network.Interfaces) == 0 {
		return InterfaceRef{}
	}
	return network.Interfaces[0]
}

func assembleUserData(node NodeSpec) (sliver.UserData, error) {
	data := userdata.NodeData{}
	data.Routes = append(data.Routes, node.Routes...)
	for _, command := range node.PostBootExecute {
		data.PostBootTasks = append(data.PostBootTasks, userdata.PostBootTask{Type: userdata.TaskExecute, Args: []string{command}})
	}
	for _, upload := range node.PostBootUploads {
		data.PostBootTasks = append(data.PostBootTasks, userdata.PostBootTask{
			Type: userdata.TaskUploadFile,
			Args: []string{upload.LocalPath, upload.RemotePath},
		})
	}
	data.PostUpdate = append(data.PostUpdate, node.PostUpdate...)
	for _, storage := range node.Storage {
		if storage.AutoMount {
			data.Storage = true
		}
	}
	if data.IsEmpty() {
		return nil, nil
	}
	encoded, err := data.Encode()
	if err != nil {
		return nil, fmt.Errorf("encoding user-data: %w", err)
	}
	return sliver.UserData(encoded), nil
}

func buildFacility(topo *topology.Topology, spec FacilitySpec) (*topology.Node, error) {
	labels := applyVLAN(spec.Labels, spec.VLAN)
	var capacities *sliver.Capacities
	if spec.Bandwidth > 0 || spec.MTU > 0 {
		capacities = &sliver.Capacities{BW: int(spec.Bandwidth), MTU: int(spec.MTU)}
	}
	opts := topology.FacilityOpts{Name: spec.Name, Site: spec.Site, Labels: labels, Capacities: capacities}
	for _, iface := range spec.Interfaces {
		ifaceLabels := applyVLAN(iface.Labels, iface.VLAN)
		opts.Interfaces = append(opts.Interfaces, topology.FacilityInterfaceOpts{Name: iface.Name, Labels: ifaceLabels})
	}
	return topo.AddFacility(opts)
}

func buildSwitch(topo *topology.Topology, spec SwitchSpec) error {
	_, err := topo.AddSwitch(topology.SwitchOpts{
		Name:       spec.Name,
		Site:       spec.Site,
		NPorts:     int(spec.NPorts),
		PortLabels: spec.PortLabels,
	})
	return err
}

func addSubInterfaces(parent *topology.Interface, subs []SubInterfaceSpec) error {
	for _, sub := range subs {
		labels := applyVLAN(sub.Labels, sub.VLAN)
		var capacities *sliver.Capacities
		if sub.Bandwidth > 0 {
			capacities = &sliver.Capacities{BW: int(sub.Bandwidth)}
		}
		if _, err := parent.AddChildInterface(topology.InterfaceOpts{Name: sub.Name, Labels: labels, Capacities: capacities}); err != nil {
			return fmt.Errorf("adding sub-interface %s: %w", sub.Name, err)
		}
	}
	return nil
}

func resolveServiceType(topo *topology.Topology, network NetworkSpec, ifaces []*topology.Interface) (sliver.ServiceType, error) {
	if network.Type != "" {
		return network.Type, nil
	}
	return topo.InferServiceType(ifaces, false)
}

func gatewayFromNetwork(network NetworkSpec) (*sliver.Gateway, *sliver.Labels, error) {
	subnet := network.Subnet
	if network.Gateway == nil && subnet == "" {
		return nil, nil, nil
	}
	gateway := sliver.Gateway{}
	if network.Gateway != nil {
		gateway = *network.Gateway
	}
	if subnet != "" {
		if isIPv6CIDR(subnet) {
			if gateway.IPv6Subnet == "" {
				gateway.IPv6Subnet = subnet
			}
		} else if gateway.IPv4Subnet == "" {
			gateway.IPv4Subnet = subnet
		}
	}
	if gateway.IPv4 == "" && gateway.IPv6 == "" {
		return nil, &sliver.Labels{IPv4Subnet: gateway.IPv4Subnet, IPv6Subnet: gateway.IPv6Subnet}, nil
	}
	if err := gateway.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validating gateway: %w", err)
	}
	return &gateway, nil, nil
}

func mergeNetworkLabels(base, extra *sliver.Labels) *sliver.Labels {
	if extra == nil {
		return base
	}
	if base == nil {
		base = &sliver.Labels{}
	}
	if base.IPv4Subnet == "" {
		base.IPv4Subnet = extra.IPv4Subnet
	}
	if base.IPv6Subnet == "" {
		base.IPv6Subnet = extra.IPv6Subnet
	}
	return base
}

// NormalizeMirrorDirection maps common FABlib aliases to canonical FIM values.
func NormalizeMirrorDirection(value sliver.MirrorDirection) sliver.MirrorDirection {
	switch value {
	case "", "Both", "both":
		if value == "" {
			return ""
		}
		return sliver.MirrorBoth
	case "RX_Only", "rx":
		return sliver.MirrorRXOnly
	case "TX_Only", "tx":
		return sliver.MirrorTXOnly
	default:
		return value
	}
}

func applyVLAN(labels *sliver.Labels, vlan string) *sliver.Labels {
	if vlan == "" {
		return labels
	}
	if labels == nil {
		return &sliver.Labels{VLAN: vlan}
	}
	labelCopy := *labels
	if labelCopy.VLAN == "" {
		labelCopy.VLAN = vlan
	}
	return &labelCopy
}

func isIPv6CIDR(cidr string) bool {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ip.To4() == nil
}
