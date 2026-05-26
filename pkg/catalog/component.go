package catalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

var (
	componentOnce sync.Once
	componentData *ComponentCatalog
	componentErr  error
)

// ComponentCatalog contains component models keyed by type and model.
type ComponentCatalog struct {
	entries []ComponentEntry
	byKey   map[string]*ComponentEntry
	byAlias map[string]*ComponentEntry
}

// ComponentEntry is one embedded component catalog entry.
type ComponentEntry struct {
	Model      string               `json:"Model"`
	AlsoModels []string             `json:"AlsoModels,omitempty"`
	Type       sliver.ComponentType `json:"Type"`
	Details    string               `json:"Details"`
	Interfaces map[string]string    `json:"Interfaces,omitempty"`
	Capacity   string               `json:"Capacity,omitempty"`
}

// GenerateOpts captures inputs for generating a component subtree.
type GenerateOpts struct {
	ParentNodeName   string
	ChildName        string
	Type             sliver.ComponentType
	Model            string
	ComponentNodeID  string
	NSNodeID         string
	InterfaceNodeIDs map[string]string
	GraphID          string
}

// GeneratedComponent contains component, optional internal service, and interfaces.
type GeneratedComponent struct {
	Component  sliver.ComponentSliver
	Service    *sliver.NetworkServiceSliver
	Interfaces []sliver.InterfaceSliver
}

// Components returns the embedded component catalog.
func Components() (*ComponentCatalog, error) {
	componentOnce.Do(func() {
		var entries []ComponentEntry
		if err := json.Unmarshal(componentCatalogJSON, &entries); err != nil {
			componentErr = fmt.Errorf("%w: components.json: %w", ErrCatalogLoad, err)
			return
		}
		componentData = newComponentCatalog(entries)
	})
	return componentData, componentErr
}

// Entries returns a sorted copy of catalog entries.
func (c *ComponentCatalog) Entries() []ComponentEntry {
	if c == nil {
		return nil
	}
	out := append([]ComponentEntry(nil), c.entries...)
	sort.Slice(out, func(i, j int) bool {
		return string(out[i].Type)+"/"+out[i].Model < string(out[j].Type)+"/"+out[j].Model
	})
	return out
}

// Lookup returns the component entry for a type and model or alias.
func (c *ComponentCatalog) Lookup(componentType sliver.ComponentType, model string) (ComponentEntry, bool) {
	entry := c.resolve(componentType, model)
	if entry == nil {
		return ComponentEntry{}, false
	}
	return *entry, true
}

// Generate returns slivers for a catalog component subtree.
func (c *ComponentCatalog) Generate(opts GenerateOpts) (GeneratedComponent, error) {
	entry := c.resolve(opts.Type, opts.Model)
	if entry == nil {
		return GeneratedComponent{}, fmt.Errorf("%w: no component catalog entry for %s/%s", ErrNotFound, opts.Type, opts.Model)
	}
	// Python FIM names components by their child name only (e.g. "nic1"), while
	// the internal service and service-attachment nodes use the full path
	// "{parent}-{child}" (e.g. "vm1-nic1-l2ovs") to remain globally unique.
	componentName := opts.ChildName
	servicePath := opts.ParentNodeName + "-" + opts.ChildName
	component := sliver.ComponentSliver{
		BaseSliver: sliver.BaseSliver{
			NodeID:     ensureID(opts.ComponentNodeID),
			GraphID:    opts.GraphID,
			Name:       componentName,
			Details:    entry.Details,
			Capacities: &sliver.Capacities{Unit: 1},
		},
		Type:  entry.Type,
		Model: entry.Model,
	}
	if entry.Capacity != "" {
		disk, err := strconv.Atoi(entry.Capacity)
		if err != nil {
			return GeneratedComponent{}, fmt.Errorf("%w: component %s capacity %q is not an integer: %w", ErrCatalogLoad, entry.Model, entry.Capacity, err)
		}
		component.Capacities.Disk = disk
	}
	out := GeneratedComponent{Component: component}
	if len(entry.Interfaces) == 0 {
		return out, nil
	}
	serviceType := sliver.ServiceTypeOVS
	if entry.Type == sliver.ComponentTypeFPGA {
		serviceType = sliver.ServiceTypeP4
	}
	portType := sliver.InterfaceTypeDedicatedPort
	if entry.Type == sliver.ComponentTypeSharedNIC {
		portType = sliver.InterfaceTypeSharedPort
	}
	service := sliver.NetworkServiceSliver{
		BaseSliver: sliver.BaseSliver{
			NodeID:  ensureID(opts.NSNodeID),
			GraphID: opts.GraphID,
			Name:    servicePath + "-l2" + strings.ToLower(string(serviceType)),
		},
		Type:  serviceType,
		Layer: sliver.LayerL2,
	}
	out.Service = &service
	labels := make([]string, 0, len(entry.Interfaces))
	for label := range entry.Interfaces {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		bw, err := strconv.Atoi(entry.Interfaces[label])
		if err != nil {
			return GeneratedComponent{}, fmt.Errorf("%w: component %s interface %s speed %q is not an integer: %w", ErrCatalogLoad, entry.Model, label, entry.Interfaces[label], err)
		}
		nodeID := ""
		if opts.InterfaceNodeIDs != nil {
			nodeID = opts.InterfaceNodeIDs[label]
		}
		// Python FIM does not populate BW on SharedPort ConnectionPoints.
		bwVal := bw
		if portType == sliver.InterfaceTypeSharedPort {
			bwVal = 0
		}
		out.Interfaces = append(out.Interfaces, sliver.InterfaceSliver{
			BaseSliver: sliver.BaseSliver{
				NodeID:     ensureID(nodeID),
				GraphID:    opts.GraphID,
				Name:       opts.ChildName + "-" + label,
				Labels:     &sliver.Labels{LocalName: label},
				Capacities: &sliver.Capacities{Unit: 1, BW: bwVal},
			},
			Type: portType,
		})
	}
	return out, nil
}

func newComponentCatalog(entries []ComponentEntry) *ComponentCatalog {
	catalog := &ComponentCatalog{entries: entries, byKey: make(map[string]*ComponentEntry), byAlias: make(map[string]*ComponentEntry)}
	for index := range catalog.entries {
		entry := &catalog.entries[index]
		catalog.byKey[componentKey(entry.Type, entry.Model)] = entry
		for _, alias := range entry.AlsoModels {
			catalog.byAlias[componentKey(entry.Type, alias)] = entry
		}
	}
	return catalog
}

func (c *ComponentCatalog) resolve(componentType sliver.ComponentType, model string) *ComponentEntry {
	if c == nil {
		return nil
	}
	if entry := c.byKey[componentKey(componentType, model)]; entry != nil {
		return entry
	}
	return c.byAlias[componentKey(componentType, model)]
}

func componentKey(componentType sliver.ComponentType, model string) string {
	return string(componentType) + "/" + model
}

func ensureID(provided string) string {
	if provided != "" {
		return provided
	}
	return uuid.NewString()
}
