package catalog

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// Validate checks that the instance catalog contains usable flavor names and
// non-negative capacities.
func (c *InstanceCatalog) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: instance catalog is nil", ErrCatalogLoad)
	}
	if len(c.byName) == 0 {
		return fmt.Errorf("%w: instance catalog is empty", ErrCatalogLoad)
	}
	for _, name := range c.Names() {
		capacity := c.byName[name]
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("%w: instance catalog contains an empty name", ErrCatalogLoad)
		}
		if capacity.Core < 0 || capacity.RAM < 0 || capacity.Disk < 0 {
			return fmt.Errorf("%w: instance %q has negative capacity", ErrCatalogLoad, name)
		}
	}
	return nil
}

// Validate checks that the component catalog can be looked up deterministically
// and that entries contain parseable interface and capacity values.
func (c *ComponentCatalog) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: component catalog is nil", ErrCatalogLoad)
	}
	if len(c.entries) == 0 {
		return fmt.Errorf("%w: component catalog is empty", ErrCatalogLoad)
	}
	seen := make(map[string]struct{}, len(c.entries))
	for _, entry := range c.Entries() {
		key := componentKey(entry.Type, entry.Model)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: duplicate component entry %s", ErrCatalogLoad, key)
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(entry.Model) == "" {
			return fmt.Errorf("%w: component entry has empty model", ErrCatalogLoad)
		}
		if !knownComponentType(entry.Type) {
			return fmt.Errorf("%w: component %q has unknown type %q", ErrCatalogLoad, entry.Model, entry.Type)
		}
		if strings.TrimSpace(entry.Details) == "" {
			return fmt.Errorf("%w: component %s has empty details", ErrCatalogLoad, key)
		}
		for label, speed := range entry.Interfaces {
			if strings.TrimSpace(label) == "" {
				return fmt.Errorf("%w: component %s has empty interface label", ErrCatalogLoad, key)
			}
			value, err := strconv.Atoi(speed)
			if err != nil || value <= 0 {
				return fmt.Errorf("%w: component %s interface %s has invalid speed %q", ErrCatalogLoad, key, label, speed)
			}
		}
		if entry.Capacity != "" {
			value, err := strconv.Atoi(entry.Capacity)
			if err != nil || value <= 0 {
				return fmt.Errorf("%w: component %s has invalid capacity %q", ErrCatalogLoad, key, entry.Capacity)
			}
		}
	}
	return nil
}

func knownComponentType(componentType sliver.ComponentType) bool {
	switch componentType {
	case sliver.ComponentTypeGPU,
		sliver.ComponentTypeSmartNIC,
		sliver.ComponentTypeSharedNIC,
		sliver.ComponentTypeFPGA,
		sliver.ComponentTypeNVME,
		sliver.ComponentTypeStorage:
		return true
	default:
		return false
	}
}
