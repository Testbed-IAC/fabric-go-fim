package catalog

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/internal/graphml"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// Broker-model GraphML uses a CompositeNode per site, "has" edges to worker
// nodes, and worker "has" edges to components.
const (
	classCompositeNode = "CompositeNode"
	edgeHas            = "has"
	nodeTypeSwitch     = "Switch"
)

// DecodeBrokerModel decodes the FABRIC broker query model returned by the
// orchestrator /resources endpoint (all detail levels). Unlike the advertised
// topology handled by DecodeAdvertised, the broker model represents each site
// as a CompositeNode aggregate whose child worker nodes carry the components.
// Every CompositeNode becomes a Site with its aggregate capacity, maintenance
// state, per-model component availability summed across workers, and the
// compute workers as hosts.
func DecodeBrokerModel(model string) (*Advertised, error) {
	g, err := graphml.Read(strings.NewReader(model))
	if err != nil {
		return nil, fmt.Errorf("%w: decoding broker model: %w", ErrCatalogLoad, err)
	}
	var adv Advertised
	for _, node := range g.Nodes() {
		if node.Class != classCompositeNode {
			continue
		}
		site := siteFromNode(node)
		site.State = siteMaintenanceState(node)

		for _, worker := range g.Neighbors(node.ID, edgeHas, graph.Outgoing) {
			if worker.Class != sliver.ClassNetworkNode {
				continue
			}
			host := hostFromNode(worker)
			host.Site = site.Name
			for _, comp := range g.Neighbors(worker.ID, edgeHas, graph.Outgoing) {
				if comp.Class != sliver.ClassComponent {
					continue
				}
				key := componentKey(sliver.ComponentType(comp.Props[sliver.PropType]), comp.Props[sliver.PropModel])
				availability := componentAvailability(comp)
				addComponentAvailability(host.Components, key, availability)
				addComponentAvailability(site.Components, key, availability)
			}
			// Switches carry no tenant capacity; only compute workers are hosts.
			if worker.Props[sliver.PropType] != nodeTypeSwitch {
				site.Hosts = append(site.Hosts, host)
			}
		}
		adv.Sites = append(adv.Sites, site)
	}
	return &adv, nil
}

// siteMaintenanceState reads the site's operational state from the
// MaintenanceInfo property, which maps site name to a status object.
func siteMaintenanceState(node *graph.Node) string {
	raw := node.Props[sliver.PropMaintenanceInfo]
	if raw == "" {
		return ""
	}
	var info map[string]struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(raw), &info); err != nil {
		return ""
	}
	if entry, ok := info[node.Props[sliver.PropName]]; ok {
		return entry.State
	}
	for _, entry := range info {
		return entry.State
	}
	return ""
}
