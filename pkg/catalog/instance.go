package catalog

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

var (
	instanceOnce sync.Once
	instanceData *InstanceCatalog
	instanceErr  error
)

// InstanceCatalog contains VM instance flavors keyed by FABRIC instance name.
type InstanceCatalog struct {
	byName map[string]sliver.Capacities
}

// Instances returns the embedded instance catalog.
func Instances() (*InstanceCatalog, error) {
	instanceOnce.Do(func() {
		var raw map[string]sliver.Capacities
		if err := json.Unmarshal(instanceSizesJSON, &raw); err != nil {
			instanceErr = fmt.Errorf("%w: instances.json: %w", ErrCatalogLoad, err)
			return
		}
		instanceData = &InstanceCatalog{byName: raw}
	})
	return instanceData, instanceErr
}

// Lookup returns capacities for an instance name.
func (c *InstanceCatalog) Lookup(name string) (sliver.Capacities, bool) {
	if c == nil {
		return sliver.Capacities{}, false
	}
	capacity, ok := c.byName[name]
	return capacity, ok
}

// Names returns sorted instance names.
func (c *InstanceCatalog) Names() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.byName))
	for name := range c.byName {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// MapCapacitiesToInstance returns the smallest fitting instance flavor.
func (c *InstanceCatalog) MapCapacitiesToInstance(want sliver.Capacities) string {
	if c == nil || len(c.byName) == 0 {
		return ""
	}
	bestName := ""
	bestScore := math.MaxInt
	for _, name := range c.Names() {
		have := c.byName[name]
		if !have.GreaterOrEqual(want) {
			continue
		}
		score := (have.Core - want.Core) + (have.RAM - want.RAM) + (have.Disk - want.Disk)
		if score < bestScore {
			bestScore = score
			bestName = name
		}
	}
	if bestName != "" {
		return bestName
	}
	largestName := ""
	largestScore := -1
	for _, name := range c.Names() {
		have := c.byName[name]
		score := have.Core + have.RAM + have.Disk
		if score > largestScore {
			largestScore = score
			largestName = name
		}
	}
	return largestName
}
