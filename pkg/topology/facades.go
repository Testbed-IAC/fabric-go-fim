package topology

import (
	"fmt"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/catalog"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// Node is a facade over a NetworkNode graph node.
type Node struct {
	t  *Topology
	id string
}

// Component is a facade over a Component graph node.
type Component struct {
	t  *Topology
	id string
}

// NetworkService is a facade over a NetworkService graph node.
type NetworkService struct {
	t  *Topology
	id string
}

// Interface is a facade over a ConnectionPoint graph node.
type Interface struct {
	t  *Topology
	id string
}

// Link is a facade over a Link graph node.
type Link struct {
	t  *Topology
	id string
}

// ID returns the stable NodeID for the node.
func (n *Node) ID() string { return n.id }

// Name returns the NetworkNode name.
func (n *Node) Name() string { return n.prop(sliver.PropName) }

// Site returns the NetworkNode site.
func (n *Node) Site() string { return n.prop(sliver.PropSite) }

// Sliver returns the typed NetworkNode sliver.
func (n *Node) Sliver() (*sliver.NodeSliver, error) {
	props := n.props()
	var out sliver.NodeSliver
	if err := out.FromProps(props); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddComponent adds a catalog component under this node.
func (n *Node) AddComponent(opts ComponentOpts) (*Component, error) {
	componentType := opts.Type
	model := opts.Model
	if opts.FABlibName != "" {
		resolvedType, resolvedModel, ok := catalog.ResolveFABlibModel(opts.FABlibName)
		if !ok {
			return nil, diagnostic(fmt.Errorf("%w: FABlib model %q is unknown", catalog.ErrNotFound, opts.FABlibName), "FABlibName", "")
		}
		componentType = resolvedType
		model = resolvedModel
	}
	if !componentNamePattern.MatchString(opts.Name) {
		return nil, diagnostic(fmt.Errorf("%w: component name %q must match %s", ErrInvalidOption, opts.Name, componentNamePattern.String()), "Name", "")
	}
	finalName := opts.Name
	for _, component := range n.Components() {
		if component.Name() == finalName {
			return nil, diagnostic(fmt.Errorf("%w: component %q already exists on node %q", ErrDuplicateName, finalName, n.Name()), "Name", "")
		}
	}
	components, err := catalog.Components()
	if err != nil {
		return nil, err
	}
	generated, err := components.Generate(catalog.GenerateOpts{
		ParentNodeName:   n.Name(),
		ChildName:        opts.Name,
		Type:             componentType,
		Model:            model,
		ComponentNodeID:  chooseID(opts.NodeID, namespaceComponent, n.t.graphID, n.Name(), opts.Name),
		NSNodeID:         chooseID(opts.NSNodeID, namespaceService, n.t.graphID, n.Name(), opts.Name, "nsl2"),
		InterfaceNodeIDs: opts.InterfaceNodeIDs,
		GraphID:          n.t.graphID,
	})
	if err != nil {
		return nil, err
	}
	componentProps, err := generated.Component.ToProps()
	if err != nil {
		return nil, err
	}
	if err := n.t.g.AddNode(sliver.ClassComponent, generated.Component.NodeID, componentProps); err != nil {
		return nil, err
	}
	if err := n.t.g.AddEdge(sliver.EdgeHas, n.id, generated.Component.NodeID); err != nil {
		n.t.g.DeleteNode(generated.Component.NodeID)
		return nil, err
	}
	if generated.Service == nil {
		return &Component{t: n.t, id: generated.Component.NodeID}, nil
	}
	serviceProps, err := generated.Service.ToProps()
	if err != nil {
		n.t.g.DeleteNode(generated.Component.NodeID)
		return nil, err
	}
	if err := n.t.g.AddNode(sliver.ClassNetworkService, generated.Service.NodeID, serviceProps); err != nil {
		n.t.g.DeleteNode(generated.Component.NodeID)
		return nil, err
	}
	if err := n.t.g.AddEdge(sliver.EdgeHas, generated.Component.NodeID, generated.Service.NodeID); err != nil {
		n.t.g.DeleteNode(generated.Component.NodeID)
		return nil, err
	}
	for _, iface := range generated.Interfaces {
		if iface.NodeID == "" {
			iface.NodeID = deriveID(namespaceInterface, n.t.graphID, n.Name(), opts.Name, iface.Labels.LocalName)
		}
		props, err := iface.ToProps()
		if err != nil {
			n.t.g.DeleteNode(generated.Component.NodeID)
			return nil, err
		}
		if err := n.t.g.AddNode(sliver.ClassConnectionPoint, iface.NodeID, props); err != nil {
			n.t.g.DeleteNode(generated.Component.NodeID)
			return nil, err
		}
		if err := n.t.g.AddEdge(sliver.EdgeConnects, generated.Service.NodeID, iface.NodeID); err != nil {
			n.t.g.DeleteNode(generated.Component.NodeID)
			return nil, err
		}
	}
	return &Component{t: n.t, id: generated.Component.NodeID}, nil
}

// AddNetworkService adds a NetworkService owned by this node.
func (n *Node) AddNetworkService(opts NetworkServiceOpts) (*NetworkService, error) {
	return n.t.addNetworkService(n.id, opts)
}

// ValidateNetworkService validates a NetworkService owned by this node without
// mutating the topology. It applies the same FIM construction rules as
// AddNetworkService.
func (n *Node) ValidateNetworkService(opts NetworkServiceOpts) error {
	return n.t.validateNetworkServiceOpts(n.id, opts)
}

// Components returns components directly owned by this node.
func (n *Node) Components() []*Component {
	var out []*Component
	for _, child := range n.t.g.Neighbors(n.id, sliver.EdgeHas, graph.Outgoing) {
		if child.Class == sliver.ClassComponent {
			out = append(out, &Component{t: n.t, id: child.ID})
		}
	}
	return out
}

// NetworkServices returns services directly owned by this node.
func (n *Node) NetworkServices() []*NetworkService {
	var out []*NetworkService
	for _, child := range n.t.g.Neighbors(n.id, sliver.EdgeHas, graph.Outgoing) {
		if child.Class == sliver.ClassNetworkService {
			out = append(out, &NetworkService{t: n.t, id: child.ID})
		}
	}
	return out
}

// InterfaceList returns all interfaces reachable through this node's components and services.
func (n *Node) InterfaceList() []*Interface {
	var out []*Interface
	for _, service := range n.NetworkServices() {
		out = append(out, service.Interfaces()...)
	}
	for _, component := range n.Components() {
		out = append(out, component.InterfaceList()...)
	}
	return out
}

// RemoveComponent removes a component by final or child name.
func (n *Node) RemoveComponent(name string) error {
	for _, component := range n.Components() {
		if component.Name() == name || component.Name() == n.Name()+"-"+name {
			n.t.deleteDescendants(component.id)
			n.t.g.DeleteNode(component.id)
			return nil
		}
	}
	return fmt.Errorf("%w: component %q", ErrNotFound, name)
}

// ID returns the stable NodeID for the component.
func (c *Component) ID() string { return c.id }

// Name returns the Component name.
func (c *Component) Name() string { return c.prop(sliver.PropName) }

// Sliver returns the typed Component sliver.
func (c *Component) Sliver() (*sliver.ComponentSliver, error) {
	var out sliver.ComponentSliver
	if err := out.FromProps(c.props()); err != nil {
		return nil, err
	}
	return &out, nil
}

// NetworkServices returns services owned by this component.
func (c *Component) NetworkServices() []*NetworkService {
	var out []*NetworkService
	for _, child := range c.t.g.Neighbors(c.id, sliver.EdgeHas, graph.Outgoing) {
		if child.Class == sliver.ClassNetworkService {
			out = append(out, &NetworkService{t: c.t, id: child.ID})
		}
	}
	return out
}

// InterfaceList returns interfaces exposed by this component.
func (c *Component) InterfaceList() []*Interface {
	var out []*Interface
	for _, service := range c.NetworkServices() {
		out = append(out, service.Interfaces()...)
	}
	return out
}

// Interfaces returns interfaces exposed by this component.
func (c *Component) Interfaces() []*Interface {
	return c.InterfaceList()
}

// ID returns the stable NodeID for the service.
func (s *NetworkService) ID() string { return s.id }

// Name returns the NetworkService name.
func (s *NetworkService) Name() string { return s.prop(sliver.PropName) }

// Sliver returns the typed NetworkService sliver.
func (s *NetworkService) Sliver() (*sliver.NetworkServiceSliver, error) {
	var out sliver.NetworkServiceSliver
	if err := out.FromProps(s.props()); err != nil {
		return nil, err
	}
	return &out, nil
}

// Interfaces returns ConnectionPoints directly connected to this service.
func (s *NetworkService) Interfaces() []*Interface {
	var out []*Interface
	for _, child := range s.t.g.Neighbors(s.id, sliver.EdgeConnects, graph.Outgoing) {
		if child.Class == sliver.ClassConnectionPoint {
			out = append(out, &Interface{t: s.t, id: child.ID})
		}
	}
	return out
}

// AddInterface adds a ConnectionPoint directly to this service.
func (s *NetworkService) AddInterface(opts InterfaceOpts) (*Interface, error) {
	if opts.Type == "" {
		opts.Type = sliver.InterfaceTypeServicePort
	}
	if !interfaceNamePattern.MatchString(opts.Name) {
		return nil, diagnostic(fmt.Errorf("%w: interface name %q must match %s", ErrInvalidOption, opts.Name, interfaceNamePattern.String()), "Name", "")
	}
	interfaceID := chooseID(opts.NodeID, namespaceInterface, s.t.graphID, s.Name(), opts.Name)
	iface := sliver.InterfaceSliver{
		BaseSliver: sliver.BaseSliver{
			NodeID:     interfaceID,
			GraphID:    s.t.graphID,
			Name:       opts.Name,
			Labels:     opts.Labels,
			Capacities: opts.Capacities,
		},
		Type: opts.Type,
	}
	props, err := iface.ToProps()
	if err != nil {
		return nil, err
	}
	if err := s.t.g.AddNode(sliver.ClassConnectionPoint, interfaceID, props); err != nil {
		return nil, err
	}
	if err := s.t.g.AddEdge(sliver.EdgeConnects, s.id, interfaceID); err != nil {
		s.t.g.DeleteNode(interfaceID)
		return nil, err
	}
	return &Interface{t: s.t, id: interfaceID}, nil
}

// ConnectInterface connects an existing interface to this service with a ServicePort and Link.
func (s *NetworkService) ConnectInterface(iface *Interface) error {
	if iface == nil {
		return diagnostic(fmt.Errorf("%w: interface is nil", ErrInvalidOption), "Interfaces", "")
	}
	// Python FIM names service-attachment ports as "{parent_node}-{interface}"
	// (e.g. "vm1-nic1-p1") rather than "{service}-{interface}". Derive the
	// node prefix by traversing the interface's component subtree.
	nodePrefix := iface.parentNodeName()
	var peerName string
	if nodePrefix != "" {
		peerName = nodePrefix + "-" + iface.Name()
	} else {
		peerName = s.Name() + "-" + iface.Name()
	}
	// Use iface.id (unique per interface) so two interfaces with the same Name
	// on different VMs don't hash to the same service-port UUID.
	peerID := deriveID(namespaceInterface, s.t.graphID, s.Name(), iface.id)
	peer := sliver.InterfaceSliver{BaseSliver: sliver.BaseSliver{NodeID: peerID, GraphID: s.t.graphID, Name: peerName}, Type: sliver.InterfaceTypeServicePort}
	peerProps, err := peer.ToProps()
	if err != nil {
		return err
	}
	linkType := sliver.LinkTypePatch
	if iface.Type() == sliver.InterfaceTypeSharedPort {
		linkType = sliver.LinkTypeL2Path
	}
	layer, _ := sliver.LayerForLinkType(linkType)
	linkID := deriveID(namespaceLink, s.t.graphID, s.Name(), iface.id, "link")
	link := sliver.LinkSliver{BaseSliver: sliver.BaseSliver{NodeID: linkID, GraphID: s.t.graphID, Name: peerName + "-link"}, Type: linkType, Layer: layer}
	linkProps, err := link.ToProps()
	if err != nil {
		return err
	}
	if err := s.t.g.AddNode(sliver.ClassConnectionPoint, peerID, peerProps); err != nil {
		return err
	}
	if err := s.t.g.AddNode(sliver.ClassLink, linkID, linkProps); err != nil {
		s.t.g.DeleteNode(peerID)
		return err
	}
	// Python FIM edge model: service→service_port, interface→link, service_port→link.
	if err := s.t.g.AddEdge(sliver.EdgeConnects, s.id, peerID); err != nil {
		s.t.g.DeleteNode(peerID)
		s.t.g.DeleteNode(linkID)
		return err
	}
	if err := s.t.g.AddEdge(sliver.EdgeConnects, iface.id, linkID); err != nil {
		s.t.g.DeleteNode(peerID)
		s.t.g.DeleteNode(linkID)
		return err
	}
	if err := s.t.g.AddEdge(sliver.EdgeConnects, peerID, linkID); err != nil {
		s.t.g.DeleteNode(peerID)
		s.t.g.DeleteNode(linkID)
		return err
	}
	return nil
}

// DisconnectInterface removes this service's ServicePort and Link for an interface.
func (s *NetworkService) DisconnectInterface(iface *Interface) error {
	for _, peer := range iface.GetPeers(sliver.InterfaceTypeServicePort) {
		parent := peer.parentService()
		if parent != nil && parent.id == s.id {
			s.t.deleteLinksIncidentTo(peer.id)
			s.t.g.DeleteNode(peer.id)
			return nil
		}
	}
	return fmt.Errorf("%w: interface %q is not connected to service %q", ErrNotFound, iface.Name(), s.Name())
}

// Peer connects two NetworkServices by connecting their first interfaces.
func (s *NetworkService) Peer(other *NetworkService) error {
	if other == nil || len(other.Interfaces()) == 0 {
		return diagnostic(fmt.Errorf("%w: other service has no interfaces", ErrInvalidOption), "other", "")
	}
	return s.ConnectInterface(other.Interfaces()[0])
}

// ID returns the stable NodeID for the interface.
func (i *Interface) ID() string { return i.id }

// Name returns the interface name.
func (i *Interface) Name() string { return i.prop(sliver.PropName) }

// Type returns the interface type.
func (i *Interface) Type() sliver.InterfaceType { return sliver.InterfaceType(i.prop(sliver.PropType)) }

// Sliver returns the typed Interface sliver.
func (i *Interface) Sliver() (*sliver.InterfaceSliver, error) {
	var out sliver.InterfaceSliver
	if err := out.FromProps(i.props()); err != nil {
		return nil, err
	}
	return &out, nil
}

// AddChildInterface adds a VLAN-tagged SubInterface under a DedicatedPort.
func (i *Interface) AddChildInterface(opts InterfaceOpts) (*Interface, error) {
	if i.Type() != sliver.InterfaceTypeDedicatedPort {
		return nil, diagnostic(fmt.Errorf("%w: parent interface %q must be DedicatedPort, got %s", ErrConstraintViolation, i.Name(), i.Type()), "Type", "")
	}
	if opts.Labels == nil || opts.Labels.VLAN == "" {
		return nil, diagnostic(fmt.Errorf("%w: SubInterface %q requires Labels.VLAN", ErrConstraintViolation, opts.Name), "Labels.VLAN", "")
	}
	for _, child := range i.ChildInterfaces() {
		sl, err := child.Sliver()
		if err == nil && sl.Labels != nil && sl.Labels.VLAN == opts.Labels.VLAN {
			return nil, diagnostic(fmt.Errorf("%w: VLAN %s already exists under interface %q", ErrDuplicateName, opts.Labels.VLAN, i.Name()), "Labels.VLAN", "")
		}
	}
	parentSliver, err := i.Sliver()
	if err != nil {
		return nil, err
	}
	labels := *opts.Labels
	if parentSliver.Labels != nil {
		labels.LocalName = parentSliver.Labels.LocalName
	}
	childName := opts.Name
	childID := chooseID(opts.NodeID, namespaceInterface, i.t.graphID, i.Name(), opts.Name)
	child := sliver.InterfaceSliver{BaseSliver: sliver.BaseSliver{NodeID: childID, GraphID: i.t.graphID, Name: childName, Labels: &labels, Capacities: opts.Capacities}, Type: sliver.InterfaceTypeSubInterface}
	props, err := child.ToProps()
	if err != nil {
		return nil, err
	}
	if err := i.t.g.AddNode(sliver.ClassConnectionPoint, childID, props); err != nil {
		return nil, err
	}
	if err := i.t.g.AddEdge(sliver.EdgeConnects, i.id, childID); err != nil {
		i.t.g.DeleteNode(childID)
		return nil, err
	}
	return &Interface{t: i.t, id: childID}, nil
}

// ChildInterfaces returns SubInterfaces directly under this interface.
func (i *Interface) ChildInterfaces() []*Interface {
	var out []*Interface
	for _, child := range i.t.g.Neighbors(i.id, sliver.EdgeConnects, graph.Outgoing) {
		if child.Class == sliver.ClassConnectionPoint && child.Props[sliver.PropType] == string(sliver.InterfaceTypeSubInterface) {
			out = append(out, &Interface{t: i.t, id: child.ID})
		}
	}
	return out
}

// GetPeers returns interface peers reached through Link nodes.
// Python FIM model: interface→link and service_port→link, so the link is an
// Outgoing neighbour of the interface, and peers are Incoming neighbours of the link.
func (i *Interface) GetPeers(interfaceType sliver.InterfaceType) []*Interface {
	var out []*Interface
	for _, link := range i.t.g.Neighbors(i.id, sliver.EdgeConnects, graph.Outgoing) {
		if link.Class != sliver.ClassLink {
			continue
		}
		for _, peer := range i.t.g.Neighbors(link.ID, sliver.EdgeConnects, graph.Incoming) {
			if peer.ID != i.id && peer.Class == sliver.ClassConnectionPoint && (interfaceType == "" || peer.Props[sliver.PropType] == string(interfaceType)) {
				out = append(out, &Interface{t: i.t, id: peer.ID})
			}
		}
	}
	return out
}

// ID returns the stable NodeID for the link.
func (l *Link) ID() string { return l.id }

// Name returns the Link name.
func (l *Link) Name() string { return l.prop(sliver.PropName) }

// Sliver returns the typed Link sliver.
func (l *Link) Sliver() (*sliver.LinkSliver, error) {
	var out sliver.LinkSliver
	if err := out.FromProps(l.props()); err != nil {
		return nil, err
	}
	return &out, nil
}

func (n *Node) props() map[string]string           { return facadeProps(n.t, n.id) }
func (c *Component) props() map[string]string      { return facadeProps(c.t, c.id) }
func (s *NetworkService) props() map[string]string { return facadeProps(s.t, s.id) }
func (i *Interface) props() map[string]string      { return facadeProps(i.t, i.id) }
func (l *Link) props() map[string]string           { return facadeProps(l.t, l.id) }

func (n *Node) prop(key string) string           { return n.props()[key] }
func (c *Component) prop(key string) string      { return c.props()[key] }
func (s *NetworkService) prop(key string) string { return s.props()[key] }
func (i *Interface) prop(key string) string      { return i.props()[key] }
func (l *Link) prop(key string) string           { return l.props()[key] }

func facadeProps(t *Topology, id string) map[string]string {
	node, ok := t.g.Node(id)
	if !ok {
		return map[string]string{}
	}
	return node.Props
}

func (i *Interface) parentService() *NetworkService {
	for _, parent := range i.t.g.Neighbors(i.id, sliver.EdgeConnects, graph.Incoming) {
		if parent.Class == sliver.ClassNetworkService {
			return &NetworkService{t: i.t, id: parent.ID}
		}
	}
	return nil
}

// parentNodeName traverses interface → internal service → component → NetworkNode
// and returns the NetworkNode's Name property. Returns "" if the interface is not
// attached to a component subtree (e.g. a top-level FacilityPort).
func (i *Interface) parentNodeName() string {
	for _, svcNode := range i.t.g.Neighbors(i.id, sliver.EdgeConnects, graph.Incoming) {
		if svcNode.Class != sliver.ClassNetworkService {
			continue
		}
		for _, compNode := range i.t.g.Neighbors(svcNode.ID, sliver.EdgeHas, graph.Incoming) {
			if compNode.Class != sliver.ClassComponent {
				continue
			}
			for _, vmNode := range i.t.g.Neighbors(compNode.ID, sliver.EdgeHas, graph.Incoming) {
				if vmNode.Class == sliver.ClassNetworkNode {
					return vmNode.Props[sliver.PropName]
				}
			}
		}
	}
	return ""
}
