package topology

import (
	"fmt"

	"github.com/CSC478-WCU/fabric-go-fim/internal/graph"
	"github.com/CSC478-WCU/fabric-go-fim/pkg/sliver"
)

type serviceConstraint struct {
	minInterfaces int
	maxInterfaces int
	maxSites      int
}

var serviceConstraints = map[sliver.ServiceType]serviceConstraint{
	sliver.ServiceTypeP4:          {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeOVS:         {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeVLAN:        {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeMPLS:        {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeL2Path:      {minInterfaces: 1, maxInterfaces: 2, maxSites: 2},
	sliver.ServiceTypeL2STS:       {minInterfaces: 2, maxSites: 2},
	sliver.ServiceTypeL2PTP:       {minInterfaces: 2, maxInterfaces: 2, maxSites: 2},
	sliver.ServiceTypeL2Multisite: {minInterfaces: 1},
	sliver.ServiceTypeL2Bridge:    {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeFABNetv4:    {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeFABNetv6:    {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeFABNetv4Ext: {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeFABNetv6Ext: {minInterfaces: 1, maxSites: 1},
	sliver.ServiceTypeL3VPN:       {minInterfaces: 1},
	sliver.ServiceTypePortMirror:  {minInterfaces: 1, maxInterfaces: 1, maxSites: 1},
}

func validateServiceInterfaces(t *Topology, parentID string, opts NetworkServiceOpts) error {
	constraint := serviceConstraints[opts.Type]
	if parentID == "" && len(opts.Interfaces) < constraint.minInterfaces {
		return diagnostic(fmt.Errorf("%w: service %q of type %s requires at least %d interfaces, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.minInterfaces, len(opts.Interfaces)), "Interfaces", "")
	}
	if constraint.maxInterfaces > 0 && len(opts.Interfaces) > constraint.maxInterfaces {
		return diagnostic(fmt.Errorf("%w: service %q of type %s allows at most %d interfaces, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.maxInterfaces, len(opts.Interfaces)), "Interfaces", "")
	}
	sites := make(map[string]struct{})
	for _, iface := range opts.Interfaces {
		if iface == nil {
			return diagnostic(fmt.Errorf("%w: service %q has nil interface", ErrInvalidOption, opts.Name), "Interfaces", "")
		}
		if opts.Type == sliver.ServiceTypeL2PTP {
			switch iface.Type() {
			case sliver.InterfaceTypeDedicatedPort, sliver.InterfaceTypeFacilityPort, sliver.InterfaceTypeSubInterface:
			default:
				return diagnostic(fmt.Errorf("%w: L2PTP service %q cannot connect interface %q of type %s", ErrConstraintViolation, opts.Name, iface.Name(), iface.Type()), "Interfaces", "")
			}
		}
		site := t.siteOfInterface(iface)
		if site != "" {
			sites[site] = struct{}{}
		}
	}
	if opts.Site != "" {
		sites[opts.Site] = struct{}{}
	}
	if constraint.maxSites > 0 && len(sites) > constraint.maxSites {
		return diagnostic(fmt.Errorf("%w: service %q of type %s allows at most %d sites, got %d", ErrConstraintViolation, opts.Name, opts.Type, constraint.maxSites, len(sites)), "Interfaces", "")
	}
	// L2STS and L2PTP are cross-site services that require exactly 2 sites.
	// L2Path allows same-site or cross-site (no minimum site count enforced).
	if (opts.Type == sliver.ServiceTypeL2STS || opts.Type == sliver.ServiceTypeL2PTP) && len(sites) != 2 && len(opts.Interfaces) > 0 {
		return diagnostic(fmt.Errorf("%w: service %q of type %s requires exactly 2 sites, got %d", ErrConstraintViolation, opts.Name, opts.Type, len(sites)), "Interfaces", "")
	}
	if opts.Gateway != nil {
		if err := opts.Gateway.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func serviceMaxSites(serviceType sliver.ServiceType) int {
	return serviceConstraints[serviceType].maxSites
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
