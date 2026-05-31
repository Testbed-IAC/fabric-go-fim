package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/internal/graphml"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// Advertised contains verified fields decoded from advertised-topology GraphML.
type Advertised struct {
	Sites         []Site         `json:"sites,omitempty"`
	FacilityPorts []FacilityPort `json:"facility_ports,omitempty"`
	Links         []Link         `json:"links,omitempty"`
}

// CapacityAvailability captures capacity, allocated, and available counts.
type CapacityAvailability struct {
	Capacity  int `json:"capacity,omitempty"`
	Allocated int `json:"allocated,omitempty"`
	Available int `json:"available,omitempty"`
}

// ComponentAvail captures component availability counts.
type ComponentAvail struct {
	Capacity  int `json:"capacity,omitempty"`
	Allocated int `json:"allocated,omitempty"`
	Available int `json:"available,omitempty"`
}

// Site contains site-level advertised resource details.
type Site struct {
	Name           string                    `json:"name,omitempty"`
	Cores          CapacityAvailability      `json:"cores,omitempty"`
	RAM            CapacityAvailability      `json:"ram,omitempty"`
	Disk           CapacityAvailability      `json:"disk,omitempty"`
	Hosts          []Host                    `json:"hosts,omitempty"`
	Components     map[string]ComponentAvail `json:"components,omitempty"`
	PTP            bool                      `json:"ptp,omitempty"`
	IPv4Management bool                      `json:"ipv4_management,omitempty"`
}

// Host contains host-level advertised resource details.
type Host struct {
	Name       string                    `json:"name,omitempty"`
	Site       string                    `json:"site,omitempty"`
	Cores      CapacityAvailability      `json:"cores,omitempty"`
	RAM        CapacityAvailability      `json:"ram,omitempty"`
	Disk       CapacityAvailability      `json:"disk,omitempty"`
	Components map[string]ComponentAvail `json:"components,omitempty"`
}

// FacilityPort contains advertised facility-port details.
type FacilityPort struct {
	Name      string `json:"name,omitempty"`
	Site      string `json:"site,omitempty"`
	Switch    string `json:"switch,omitempty"`
	VLANRange string `json:"vlan_range,omitempty"`
	Bandwidth int    `json:"bandwidth,omitempty"`
}

// Link contains advertised network link details.
type Link struct {
	Name     string   `json:"name,omitempty"`
	Layer    string   `json:"layer,omitempty"`
	Sites    []string `json:"sites,omitempty"`
	Capacity int      `json:"capacity,omitempty"`
}

// DecodeAdvertised parses verified fields from advertised-topology GraphML.
func DecodeAdvertised(model string) (*Advertised, error) {
	g, err := graphml.Read(strings.NewReader(model))
	if err != nil {
		return nil, fmt.Errorf("%w: decoding advertised topology: %w", ErrCatalogLoad, err)
	}
	decoder := advertisedDecoder{graph: g, sites: map[string]*Site{}, hosts: map[string]*Host{}}
	decoder.decodeSitesAndHosts()
	decoder.decodeComponents()
	decoder.decodeFacilityPorts()
	decoder.decodeLinks()
	return decoder.result(), nil
}

type advertisedDecoder struct {
	graph *graph.Graph
	sites map[string]*Site
	hosts map[string]*Host
	ports []FacilityPort
	links []Link
}

func (d *advertisedDecoder) decodeSitesAndHosts() {
	for _, node := range d.graph.Nodes() {
		switch node.Props[sliver.PropType] {
		case "Site":
			site := siteFromNode(node)
			d.sites[site.Name] = &site
		case "Server", "Host", "Worker":
			host := hostFromNode(node)
			if host.Site == "" {
				host.Site = d.parentSite(node.ID)
			}
			d.hosts[host.Name] = &host
		}
	}
}

func (d *advertisedDecoder) decodeComponents() {
	for _, node := range d.graph.Nodes() {
		if node.Class != sliver.ClassComponent {
			continue
		}
		key := componentKey(sliver.ComponentType(node.Props[sliver.PropType]), node.Props[sliver.PropModel])
		if key == "-" {
			continue
		}
		availability := componentAvailability(node)
		parent := d.parentNode(node.ID)
		if parent == nil {
			continue
		}
		if host := d.hosts[parent.Props[sliver.PropName]]; host != nil {
			addComponentAvailability(host.Components, key, availability)
			if site := d.sites[host.Site]; site != nil {
				addComponentAvailability(site.Components, key, availability)
			}
			continue
		}
		if site := d.sites[parent.Props[sliver.PropName]]; site != nil {
			addComponentAvailability(site.Components, key, availability)
		}
	}
}

func (d *advertisedDecoder) decodeFacilityPorts() {
	for _, node := range d.graph.Nodes() {
		if node.Props[sliver.PropType] != string(sliver.InterfaceTypeFacilityPort) {
			continue
		}
		labels := labelsFromProps(node.Props)
		capacities := capacitiesFromProps(node.Props, sliver.PropCapacities)
		port := FacilityPort{
			Name:      node.Props[sliver.PropName],
			Site:      firstNonEmpty(node.Props[sliver.PropSite], d.parentSite(node.ID)),
			Switch:    node.Props["Switch"],
			VLANRange: firstNonEmpty(labels.VLANRange, labels.VLAN),
			Bandwidth: capacities.BW,
		}
		d.addFacilityPort(port)
	}
}

func (d *advertisedDecoder) decodeLinks() {
	for _, node := range d.graph.Nodes() {
		if node.Class != sliver.ClassLink {
			continue
		}
		capacities := capacitiesFromProps(node.Props, sliver.PropCapacities)
		link := Link{
			Name:     node.Props[sliver.PropName],
			Layer:    node.Props[sliver.PropLayer],
			Sites:    compactStrings(node.Props["Site1"], node.Props["Site2"]),
			Capacity: capacities.BW,
		}
		d.addLink(link)
	}
}

func (d *advertisedDecoder) result() *Advertised {
	out := &Advertised{}
	siteNames := make([]string, 0, len(d.sites))
	for name := range d.sites {
		siteNames = append(siteNames, name)
	}
	sort.Strings(siteNames)
	for _, name := range siteNames {
		site := *d.sites[name]
		hostNames := make([]string, 0, len(d.hosts))
		for _, host := range d.hosts {
			if host.Site == site.Name {
				hostNames = append(hostNames, host.Name)
			}
		}
		sort.Strings(hostNames)
		for _, hostName := range hostNames {
			site.Hosts = append(site.Hosts, *d.hosts[hostName])
		}
		out.Sites = append(out.Sites, site)
	}
	out.FacilityPorts = d.sortedFacilityPorts()
	out.Links = d.sortedLinks()
	return out
}

func siteFromNode(node *graph.Node) Site {
	caps := capacitiesFromProps(node.Props, sliver.PropCapacities)
	allocations := capacitiesFromProps(node.Props, sliver.PropCapacityAllocations)
	flags := flagsFromProps(node.Props)
	return Site{
		Name:           node.Props[sliver.PropName],
		Cores:          availability(caps.Core, allocations.Core),
		RAM:            availability(caps.RAM, allocations.RAM),
		Disk:           availability(caps.Disk, allocations.Disk),
		Components:     map[string]ComponentAvail{},
		PTP:            flags.PTP,
		IPv4Management: flags.IPv4Management,
	}
}

func hostFromNode(node *graph.Node) Host {
	caps := capacitiesFromProps(node.Props, sliver.PropCapacities)
	allocations := capacitiesFromProps(node.Props, sliver.PropCapacityAllocations)
	return Host{
		Name:       node.Props[sliver.PropName],
		Site:       node.Props[sliver.PropSite],
		Cores:      availability(caps.Core, allocations.Core),
		RAM:        availability(caps.RAM, allocations.RAM),
		Disk:       availability(caps.Disk, allocations.Disk),
		Components: map[string]ComponentAvail{},
	}
}

func capacitiesFromProps(props map[string]string, key string) sliver.Capacities {
	var capacities sliver.Capacities
	if raw := props[key]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &capacities); err == nil {
			return capacities
		}
	}
	return capacities
}

func labelsFromProps(props map[string]string) sliver.Labels {
	var labels sliver.Labels
	if raw := props[sliver.PropLabels]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &labels); err == nil {
			return labels
		}
	}
	return labels
}

func flagsFromProps(props map[string]string) sliver.Flags {
	var flags sliver.Flags
	if raw := props[sliver.PropFlags]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &flags); err == nil {
			return flags
		}
	}
	return flags
}

func availability(capacity, allocated int) CapacityAvailability {
	return CapacityAvailability{Capacity: capacity, Allocated: allocated, Available: capacity - allocated}
}

func componentAvailability(node *graph.Node) ComponentAvail {
	caps := capacitiesFromProps(node.Props, sliver.PropCapacities)
	allocations := capacitiesFromProps(node.Props, sliver.PropCapacityAllocations)
	return ComponentAvail{Capacity: caps.Unit, Allocated: allocations.Unit, Available: caps.Unit - allocations.Unit}
}

func addComponentAvailability(target map[string]ComponentAvail, key string, value ComponentAvail) {
	current := target[key]
	current.Capacity += value.Capacity
	current.Allocated += value.Allocated
	current.Available += value.Available
	target[key] = current
}

func (d *advertisedDecoder) parentNode(nodeID string) *graph.Node {
	for _, parent := range d.graph.Neighbors(nodeID, sliver.EdgeHas, graph.Incoming) {
		return parent
	}
	return nil
}

func (d *advertisedDecoder) parentSite(nodeID string) string {
	seen := map[string]bool{}
	for _, current := range d.graph.Neighbors(nodeID, "", graph.Incoming) {
		if site := d.siteNameForNode(current, seen); site != "" {
			return site
		}
	}
	return ""
}

func (d *advertisedDecoder) siteNameForNode(node *graph.Node, seen map[string]bool) string {
	if node == nil || seen[node.ID] {
		return ""
	}
	seen[node.ID] = true
	if node.Props[sliver.PropType] == "Site" {
		return node.Props[sliver.PropName]
	}
	if site := node.Props[sliver.PropSite]; site != "" {
		return site
	}
	for _, parent := range d.graph.Neighbors(node.ID, "", graph.Incoming) {
		if site := d.siteNameForNode(parent, seen); site != "" {
			return site
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (d *advertisedDecoder) addFacilityPort(port FacilityPort) {
	d.ports = append(d.ports, port)
}

func (d *advertisedDecoder) sortedFacilityPorts() []FacilityPort {
	out := append([]FacilityPort(nil), d.ports...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (d *advertisedDecoder) addLink(link Link) {
	d.links = append(d.links, link)
}

func (d *advertisedDecoder) sortedLinks() []Link {
	out := append([]Link(nil), d.links...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func compactStrings(values ...string) []string {
	out := []string{}
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}
