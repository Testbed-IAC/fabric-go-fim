package topology

import (
	"fmt"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// NetworkServiceConstraint describes the FIM construction limits for a
// NetworkService type.
//
// Zero MaxInterfaces or MaxSites means no explicit upper bound. Zero ExactSites
// means the service does not require an exact site count.
type NetworkServiceConstraint struct {
	MinInterfaces          int
	MaxInterfaces          int
	MaxSites               int
	ExactSites             int
	RequiredInterfaceTypes []sliver.InterfaceType
}

var networkServiceConstraints = map[sliver.ServiceType]NetworkServiceConstraint{
	sliver.ServiceTypeP4:          {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeOVS:         {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeVLAN:        {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeMPLS:        {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeL2Path:      {MinInterfaces: 1, MaxInterfaces: 2, MaxSites: 2},
	sliver.ServiceTypeL2STS:       {MinInterfaces: 2, MaxSites: 2, ExactSites: 2},
	sliver.ServiceTypeL2PTP:       {MinInterfaces: 2, MaxInterfaces: 2, MaxSites: 2, ExactSites: 2, RequiredInterfaceTypes: []sliver.InterfaceType{sliver.InterfaceTypeDedicatedPort, sliver.InterfaceTypeFacilityPort, sliver.InterfaceTypeSubInterface}},
	sliver.ServiceTypeL2Multisite: {MinInterfaces: 1},
	sliver.ServiceTypeL2Bridge:    {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeFABNetv4:    {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeFABNetv6:    {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeFABNetv4Ext: {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeFABNetv6Ext: {MinInterfaces: 1, MaxSites: 1},
	sliver.ServiceTypeL3VPN:       {MinInterfaces: 1},
	sliver.ServiceTypePortMirror:  {MinInterfaces: 1, MaxInterfaces: 1, MaxSites: 1},
}

// NetworkServiceConstraints returns a copy of the known FIM NetworkService
// construction constraints keyed by service type.
func NetworkServiceConstraints() map[sliver.ServiceType]NetworkServiceConstraint {
	out := make(map[sliver.ServiceType]NetworkServiceConstraint, len(networkServiceConstraints))
	for serviceType, constraint := range networkServiceConstraints {
		constraint.RequiredInterfaceTypes = append([]sliver.InterfaceType(nil), constraint.RequiredInterfaceTypes...)
		out[serviceType] = constraint
	}
	return out
}

// NetworkServiceConstraintFor returns the construction constraint for a known
// NetworkService type.
func NetworkServiceConstraintFor(serviceType sliver.ServiceType) (NetworkServiceConstraint, bool) {
	constraint, ok := networkServiceConstraints[serviceType]
	constraint.RequiredInterfaceTypes = append([]sliver.InterfaceType(nil), constraint.RequiredInterfaceTypes...)
	return constraint, ok
}

func (c NetworkServiceConstraint) allowsInterfaceType(interfaceType sliver.InterfaceType) bool {
	if len(c.RequiredInterfaceTypes) == 0 {
		return true
	}
	for _, required := range c.RequiredInterfaceTypes {
		if interfaceType == required {
			return true
		}
	}
	return false
}

func validateServiceInterfaces(t *Topology, parentID string, opts NetworkServiceOpts) error {
	constraint := networkServiceConstraints[opts.Type]
	if parentID == "" && len(opts.Interfaces) < constraint.MinInterfaces {
		return diagnostic(fmt.Errorf("%w: service %q of type %s requires at least %d interfaces, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.MinInterfaces, len(opts.Interfaces)), "Interfaces", "")
	}
	if constraint.MaxInterfaces > 0 && len(opts.Interfaces) > constraint.MaxInterfaces {
		return diagnostic(fmt.Errorf("%w: service %q of type %s allows at most %d interfaces, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.MaxInterfaces, len(opts.Interfaces)), "Interfaces", "")
	}
	sites := make(map[string]struct{})
	for _, iface := range opts.Interfaces {
		if iface == nil {
			return diagnostic(fmt.Errorf("%w: service %q has nil interface", ErrInvalidOption, opts.Name), "Interfaces", "")
		}
		if !constraint.allowsInterfaceType(iface.Type()) {
			return diagnostic(fmt.Errorf("%w: service %q of type %s cannot connect interface %q of type %s", ErrConstraintViolation, opts.Name, opts.Type, iface.Name(), iface.Type()), "Interfaces", "")
		}
		site := t.siteOfInterface(iface)
		if site != "" {
			sites[site] = struct{}{}
		}
	}
	if opts.Site != "" {
		sites[opts.Site] = struct{}{}
	}
	if constraint.MaxSites > 0 && len(sites) > constraint.MaxSites {
		return diagnostic(fmt.Errorf("%w: service %q of type %s allows at most %d sites, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.MaxSites, len(sites)), "Interfaces", "")
	}
	if constraint.ExactSites > 0 && len(sites) != constraint.ExactSites && len(opts.Interfaces) > 0 {
		return diagnostic(fmt.Errorf("%w: service %q of type %s requires exactly %d sites, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.ExactSites, len(sites)), "Interfaces", "")
	}
	if opts.Gateway != nil {
		if err := opts.Gateway.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func serviceMaxSites(serviceType sliver.ServiceType) int {
	return networkServiceConstraints[serviceType].MaxSites
}

func (t *Topology) siteOfInterface(iface *Interface) string {
	if iface == nil {
		return ""
	}
	if iface.Type() == sliver.InterfaceTypeSubInterface {
		for _, parent := range t.g.Neighbors(iface.id, sliver.EdgeConnects, graph.Incoming) {
			if parent.Class == sliver.ClassConnectionPoint {
				return t.siteOfInterface(&Interface{t: t, id: parent.ID})
			}
		}
	}
	for _, service := range t.g.Neighbors(iface.id, sliver.EdgeConnects, graph.Incoming) {
		if service.Class != sliver.ClassNetworkService {
			continue
		}
		if site := service.Props[sliver.PropSite]; site != "" {
			return site
		}
		for _, owner := range t.g.Neighbors(service.ID, sliver.EdgeHas, graph.Incoming) {
			switch owner.Class {
			case sliver.ClassNetworkNode:
				return owner.Props[sliver.PropSite]
			case sliver.ClassComponent:
				for _, node := range t.g.Neighbors(owner.ID, sliver.EdgeHas, graph.Incoming) {
					if node.Class == sliver.ClassNetworkNode {
						return node.Props[sliver.PropSite]
					}
				}
			}
		}
	}
	return ""
}
