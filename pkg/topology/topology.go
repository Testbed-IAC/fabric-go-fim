package topology

import (
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/google/uuid"

	"github.com/CSC478-WCU/fabric-go-fim/internal/graph"
	"github.com/CSC478-WCU/fabric-go-fim/internal/graphml"
	"github.com/CSC478-WCU/fabric-go-fim/pkg/sliver"
)

var (
	nodeNamePattern      = regexp.MustCompile(`^[\w\-\.]{2,255}$`)
	serviceNamePattern   = regexp.MustCompile(`^[\w\-_\.]{2,255}$`)
	componentNamePattern = regexp.MustCompile(`^[\w\-_\.\ ]{2,255}$`)
	interfaceNamePattern = regexp.MustCompile(`^[\w\-+_/\.\ :]{1,255}$`)
	linkNamePattern      = regexp.MustCompile(`^[\w\-+_/\.\ :]{2,255}$`)
)

// Topology is a FABRIC ASM topology backed by a property graph.
type Topology struct {
	graphID string
	g       *graph.Graph
}

// New creates an empty topology with a fresh random GraphID.
func New() *Topology {
	return NewWithID(uuid.NewString())
}

// NewWithID creates an empty topology with the provided stable GraphID.
func NewWithID(graphID string) *Topology {
	return &Topology{graphID: graphID, g: graph.New(graphID)}
}

// Load parses GraphML into a topology.
func Load(r io.Reader) (*Topology, error) {
	g, err := graphml.Read(r)
	if err != nil {
		return nil, err
	}
	return &Topology{graphID: g.ID, g: g}, nil
}

// Serialize writes the topology as GraphML.
func (t *Topology) Serialize(w io.Writer) error {
	return graphml.Write(w, t.g)
}

// SerializeString returns the GraphML serialization as a string.
func (t *Topology) SerializeString() (string, error) {
	var buffer bytes.Buffer
	if err := t.Serialize(&buffer); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

// GraphID returns the topology GraphID.
func (t *Topology) GraphID() string {
	return t.graphID
}

// AddNode adds a top-level NetworkNode.
func (t *Topology) AddNode(opts NodeOpts) (*Node, error) {
	if opts.Type == "" {
		opts.Type = sliver.NodeTypeVM
	}
	if err := t.validateNodeOpts(opts); err != nil {
		return nil, err
	}
	nodeID := chooseID(opts.NodeID, namespaceNode, t.graphID, opts.Name)
	node := sliver.NodeSliver{
		BaseSliver: sliver.BaseSliver{
			NodeID:        nodeID,
			GraphID:       t.graphID,
			Name:          opts.Name,
			Capacities:    opts.Capacities,
			CapacityHints: opts.CapacityHints,
			Labels:        opts.Labels,
			BootScript:    opts.BootScript,
			UserData:      opts.UserData,
			Tags:          opts.Tags,
		},
		Type:      opts.Type,
		Site:      opts.Site,
		ImageRef:  opts.ImageRef,
		ImageType: opts.ImageType,
		MgmtIP:    opts.MgmtIP,
	}
	props, err := node.ToProps()
	if err != nil {
		return nil, err
	}
	if err := t.g.AddNode(sliver.ClassNetworkNode, nodeID, props); err != nil {
		return nil, err
	}
	return &Node{t: t, id: nodeID}, nil
}

// AddNetworkService adds a top-level NetworkService and optionally connects interfaces.
func (t *Topology) AddNetworkService(opts NetworkServiceOpts) (*NetworkService, error) {
	return t.addNetworkService("", opts)
}

// AddFacility adds a Facility NetworkNode with an internal service and ports.
func (t *Topology) AddFacility(opts FacilityOpts) (*Node, error) {
	serviceType := opts.Type
	if serviceType == "" {
		serviceType = sliver.ServiceTypeVLAN
	}
	node, err := t.AddNode(NodeOpts{Name: opts.Name, Site: opts.Site, Type: sliver.NodeTypeFacility, NodeID: opts.NodeID})
	if err != nil {
		return nil, err
	}
	service, err := node.AddNetworkService(NetworkServiceOpts{Name: opts.Name + "-ns", Type: serviceType, NodeID: opts.NSNodeID, Site: opts.Site, Labels: opts.Labels})
	if err != nil {
		t.g.DeleteNode(node.id)
		return nil, err
	}
	if len(opts.Interfaces) == 0 {
		_, err = service.AddInterface(InterfaceOpts{Name: opts.Name + "-int", Type: sliver.InterfaceTypeFacilityPort, Labels: opts.Labels, Capacities: opts.Capacities})
		if err != nil {
			t.g.DeleteNode(node.id)
			return nil, err
		}
		return node, nil
	}
	for _, iface := range opts.Interfaces {
		_, err := service.AddInterface(InterfaceOpts{Name: iface.Name, Type: sliver.InterfaceTypeFacilityPort, NodeID: iface.NodeID, Labels: iface.Labels, Capacities: iface.Capacities})
		if err != nil {
			t.g.DeleteNode(node.id)
			return nil, err
		}
	}
	return node, nil
}

// AddSwitch adds a Switch NetworkNode with an internal service and ports.
func (t *Topology) AddSwitch(opts SwitchOpts) (*Node, error) {
	serviceType := opts.Type
	if serviceType == "" {
		serviceType = sliver.ServiceTypeP4
	}
	nPorts := opts.NPorts
	if nPorts == 0 {
		nPorts = 8
	}
	node, err := t.AddNode(NodeOpts{Name: opts.Name, Site: opts.Site, Type: sliver.NodeTypeSwitch, NodeID: opts.NodeID})
	if err != nil {
		return nil, err
	}
	service, err := node.AddNetworkService(NetworkServiceOpts{Name: opts.Name + "-ns", Type: serviceType, NodeID: opts.NSNodeID, Site: opts.Site})
	if err != nil {
		t.g.DeleteNode(node.id)
		return nil, err
	}
	for i := 1; i <= nPorts; i++ {
		portName := fmt.Sprintf("p%d", i)
		labels := &sliver.Labels{LocalName: portName}
		if opts.PortLabels != nil {
			copyLabels := *opts.PortLabels
			if copyLabels.LocalName == "" {
				copyLabels.LocalName = portName
			}
			labels = &copyLabels
		}
		capacities := &sliver.Capacities{BW: 100}
		if opts.PortCapacities != nil {
			capacityCopy := *opts.PortCapacities
			capacities = &capacityCopy
		}
		if _, err := service.AddInterface(InterfaceOpts{Name: portName, Type: sliver.InterfaceTypeDedicatedPort, Labels: labels, Capacities: capacities}); err != nil {
			t.g.DeleteNode(node.id)
			return nil, err
		}
	}
	return node, nil
}

// AddPortMirrorService adds a PortMirror service.
func (t *Topology) AddPortMirrorService(opts PortMirrorOpts) (*NetworkService, error) {
	direction := opts.Direction
	if direction == "" {
		direction = sliver.MirrorBoth
	}
	if opts.ToInterface == nil {
		return nil, diagnostic(fmt.Errorf("%w: ToInterface is required", ErrInvalidOption), "ToInterface", "")
	}
	site := opts.Site
	if site == "" {
		site = t.siteOfInterface(opts.ToInterface)
	}
	source, sourceOK := t.interfaceByName(opts.FromInterfaceName)
	if !sourceOK {
		return nil, diagnostic(fmt.Errorf("%w: mirrored source interface %q does not exist", ErrNotFound, opts.FromInterfaceName), "FromInterfaceName", "")
	}
	sourceSite := t.siteOfInterface(source)
	if sourceSite != "" && site != "" && sourceSite != site {
		return nil, diagnostic(fmt.Errorf("%w: PortMirror %q source site %q does not match destination site %q", ErrConstraintViolation, opts.Name, sourceSite, site), "Site", "")
	}
	return t.addNetworkService("", NetworkServiceOpts{
		Name:            opts.Name,
		Type:            sliver.ServiceTypePortMirror,
		NodeID:          opts.NodeID,
		Interfaces:      []*Interface{opts.ToInterface},
		Site:            site,
		MirrorPort:      opts.FromInterfaceName,
		MirrorVLAN:      opts.FromInterfaceVLAN,
		MirrorDirection: direction,
	})
}

// AddLink adds an explicit Link node connected to interfaces.
func (t *Topology) AddLink(opts LinkOpts) (*Link, error) {
	if opts.Type == "" {
		opts.Type = sliver.LinkTypeL2Path
	}
	if !linkNamePattern.MatchString(opts.Name) {
		return nil, diagnostic(fmt.Errorf("%w: link name %q must match %s", ErrInvalidOption, opts.Name, linkNamePattern.String()), "Name", "")
	}
	if t.topLevelNameExists(opts.Name) {
		return nil, diagnostic(fmt.Errorf("%w: top-level name %q already exists", ErrDuplicateName, opts.Name), "Name", "")
	}
	layer, ok := sliver.LayerForLinkType(opts.Type)
	if !ok {
		return nil, diagnostic(fmt.Errorf("%w: unknown link type %q", ErrInvalidOption, opts.Type), "Type", "")
	}
	if (opts.Type == sliver.LinkTypePatch || opts.Type == sliver.LinkTypeL1Path) && len(opts.Interfaces) != 2 {
		return nil, diagnostic(fmt.Errorf("%w: %s link %q requires exactly 2 interfaces, got %d", ErrConstraintViolation, opts.Type, opts.Name, len(opts.Interfaces)), "Interfaces", "")
	}
	linkID := chooseID(opts.NodeID, namespaceLink, t.graphID, opts.Name)
	link := sliver.LinkSliver{BaseSliver: sliver.BaseSliver{NodeID: linkID, GraphID: t.graphID, Name: opts.Name}, Type: opts.Type, Layer: layer}
	props, err := link.ToProps()
	if err != nil {
		return nil, err
	}
	if err := t.g.AddNode(sliver.ClassLink, linkID, props); err != nil {
		return nil, err
	}
	for _, iface := range opts.Interfaces {
		if err := t.g.AddEdge(sliver.EdgeConnects, linkID, iface.id); err != nil {
			t.g.DeleteNode(linkID)
			return nil, err
		}
	}
	return &Link{t: t, id: linkID}, nil
}

// Node returns a NetworkNode by name.
func (t *Topology) Node(name string) (*Node, bool) {
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassNetworkNode && node.Props[sliver.PropName] == name {
			return &Node{t: t, id: node.ID}, true
		}
	}
	return nil, false
}

// Nodes returns top-level NetworkNodes in insertion order.
func (t *Topology) Nodes() []*Node {
	var out []*Node
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassNetworkNode {
			out = append(out, &Node{t: t, id: node.ID})
		}
	}
	return out
}

// NetworkService returns a NetworkService by name.
func (t *Topology) NetworkService(name string) (*NetworkService, bool) {
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassNetworkService && node.Props[sliver.PropName] == name {
			return &NetworkService{t: t, id: node.ID}, true
		}
	}
	return nil, false
}

// NetworkServices returns all NetworkServices in insertion order.
func (t *Topology) NetworkServices() []*NetworkService {
	var out []*NetworkService
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassNetworkService {
			out = append(out, &NetworkService{t: t, id: node.ID})
		}
	}
	return out
}

func (t *Topology) interfaceByName(name string) (*Interface, bool) {
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassConnectionPoint && node.Props[sliver.PropName] == name {
			return &Interface{t: t, id: node.ID}, true
		}
	}
	return nil, false
}

// Facilities returns all Facility NetworkNodes.
func (t *Topology) Facilities() []*Node {
	return t.nodesByType(sliver.NodeTypeFacility)
}

// Switches returns all Switch NetworkNodes.
func (t *Topology) Switches() []*Node {
	return t.nodesByType(sliver.NodeTypeSwitch)
}

// Links returns all Link nodes.
func (t *Topology) Links() []*Link {
	var out []*Link
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassLink {
			out = append(out, &Link{t: t, id: node.ID})
		}
	}
	return out
}

// RemoveNode removes a NetworkNode and its descendants.
func (t *Topology) RemoveNode(name string) error {
	node, ok := t.Node(name)
	if !ok {
		return fmt.Errorf("%w: node %q", ErrNotFound, name)
	}
	t.deleteDescendants(node.id)
	t.g.DeleteNode(node.id)
	return nil
}

// RemoveNetworkService removes a NetworkService and its service ports and links.
func (t *Topology) RemoveNetworkService(name string) error {
	service, ok := t.NetworkService(name)
	if !ok {
		return fmt.Errorf("%w: network service %q", ErrNotFound, name)
	}
	for _, iface := range service.Interfaces() {
		t.deleteLinksIncidentTo(iface.id)
		t.g.DeleteNode(iface.id)
	}
	t.g.DeleteNode(service.id)
	return nil
}

// RemoveLink removes a Link by name.
func (t *Topology) RemoveLink(name string) error {
	for _, link := range t.Links() {
		if link.Name() == name {
			t.g.DeleteNode(link.id)
			return nil
		}
	}
	return fmt.Errorf("%w: link %q", ErrNotFound, name)
}

func (t *Topology) addNetworkService(parentID string, opts NetworkServiceOpts) (*NetworkService, error) {
	if err := t.validateNetworkServiceOpts(parentID, opts); err != nil {
		return nil, err
	}
	layer, _ := sliver.LayerForServiceType(opts.Type)
	site := opts.Site
	if site == "" && serviceMaxSites(opts.Type) == 1 && len(opts.Interfaces) > 0 {
		site = t.siteOfInterface(opts.Interfaces[0])
	}
	serviceID := chooseID(opts.NodeID, namespaceService, t.graphID, opts.Name)
	service := sliver.NetworkServiceSliver{
		BaseSliver: sliver.BaseSliver{
			NodeID:     serviceID,
			GraphID:    t.graphID,
			Name:       opts.Name,
			Labels:     opts.Labels,
			Capacities: opts.Capacities,
		},
		Type:            opts.Type,
		Layer:           layer,
		Technology:      opts.Technology,
		Site:            site,
		Gateway:         opts.Gateway,
		ERO:             opts.ERO,
		PathInfo:        opts.PathInfo,
		MirrorPort:      opts.MirrorPort,
		MirrorVLAN:      opts.MirrorVLAN,
		MirrorDirection: opts.MirrorDirection,
	}
	props, err := service.ToProps()
	if err != nil {
		return nil, err
	}
	if err := t.g.AddNode(sliver.ClassNetworkService, serviceID, props); err != nil {
		return nil, err
	}
	if parentID != "" {
		if err := t.g.AddEdge(sliver.EdgeHas, parentID, serviceID); err != nil {
			t.g.DeleteNode(serviceID)
			return nil, err
		}
	}
	facade := &NetworkService{t: t, id: serviceID}
	for _, iface := range opts.Interfaces {
		if err := facade.ConnectInterface(iface); err != nil {
			t.g.DeleteNode(serviceID)
			return nil, err
		}
	}
	return facade, nil
}

func (t *Topology) validateNodeOpts(opts NodeOpts) error {
	if !nodeNamePattern.MatchString(opts.Name) {
		return diagnostic(fmt.Errorf("%w: network node name %q must match %s", ErrInvalidOption, opts.Name, nodeNamePattern.String()), "Name", "")
	}
	if t.topLevelNameExists(opts.Name) {
		return diagnostic(fmt.Errorf("%w: top-level name %q already exists", ErrDuplicateName, opts.Name), "Name", "")
	}
	if (opts.Type == sliver.NodeTypeServer || opts.Type == sliver.NodeTypeVM || opts.Type == sliver.NodeTypeContainer) && opts.Site == "" {
		return diagnostic(fmt.Errorf("%w: Site is required for %s node %q", ErrConstraintViolation, opts.Type, opts.Name), "Site", "Set a FABRIC site code such as RENC or UKY.")
	}
	if opts.Type == sliver.NodeTypeSwitch || opts.Type == sliver.NodeTypeNAS || opts.Type == sliver.NodeTypeFacility {
		if opts.ImageRef != "" || opts.ImageType != "" {
			return diagnostic(fmt.Errorf("%w: %s node %q cannot set ImageRef or ImageType", ErrConstraintViolation, opts.Type, opts.Name), "ImageRef", "")
		}
	}
	if opts.Type == sliver.NodeTypeFacility && opts.MgmtIP != "" {
		return diagnostic(fmt.Errorf("%w: Facility node %q cannot set MgmtIP", ErrConstraintViolation, opts.Name), "MgmtIP", "")
	}
	return nil
}

func (t *Topology) validateNetworkServiceOpts(parentID string, opts NetworkServiceOpts) error {
	if !serviceNamePattern.MatchString(opts.Name) {
		return diagnostic(fmt.Errorf("%w: network service name %q must match %s", ErrInvalidOption, opts.Name, serviceNamePattern.String()), "Name", "")
	}
	if parentID == "" && t.topLevelNameExists(opts.Name) {
		return diagnostic(fmt.Errorf("%w: top-level name %q already exists", ErrDuplicateName, opts.Name), "Name", "")
	}
	layer, ok := sliver.LayerForServiceType(opts.Type)
	if !ok || layer == "" {
		return diagnostic(fmt.Errorf("%w: unknown service type %q", ErrInvalidOption, opts.Type), "Type", "")
	}
	if opts.Type == sliver.ServiceTypePortMirror {
		if opts.MirrorPort == "" || opts.MirrorDirection == "" || opts.Site == "" {
			return diagnostic(fmt.Errorf("%w: PortMirror %q requires MirrorPort, MirrorDirection, and Site", ErrConstraintViolation, opts.Name), "MirrorPort", "")
		}
	}
	return validateServiceInterfaces(t, parentID, opts)
}

func (t *Topology) topLevelNameExists(name string) bool {
	for _, node := range t.g.Nodes() {
		if node.Props[sliver.PropName] != name {
			continue
		}
		if node.Class == sliver.ClassNetworkNode || node.Class == sliver.ClassNetworkService || node.Class == sliver.ClassLink {
			if len(t.g.Neighbors(node.ID, sliver.EdgeHas, graph.Incoming)) == 0 {
				return true
			}
		}
	}
	return false
}

func (t *Topology) nodesByType(nodeType sliver.NodeType) []*Node {
	var out []*Node
	for _, node := range t.g.Nodes() {
		if node.Class == sliver.ClassNetworkNode && node.Props[sliver.PropType] == string(nodeType) {
			out = append(out, &Node{t: t, id: node.ID})
		}
	}
	return out
}

func (t *Topology) deleteDescendants(nodeID string) {
	for _, child := range t.g.Neighbors(nodeID, sliver.EdgeHas, graph.Outgoing) {
		t.deleteDescendants(child.ID)
		t.g.DeleteNode(child.ID)
	}
}

func (t *Topology) deleteLinksIncidentTo(nodeID string) {
	// service_port→link: the link is an Outgoing neighbour of the service port.
	for _, link := range t.g.Neighbors(nodeID, sliver.EdgeConnects, graph.Outgoing) {
		if link.Class == sliver.ClassLink {
			t.g.DeleteNode(link.ID)
		}
	}
}
