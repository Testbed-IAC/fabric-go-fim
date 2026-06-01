package topologybuilder

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/catalog"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

const (
	// SeverityError indicates a validation failure.
	SeverityError = "error"
	// SeverityWarning indicates a non-fatal validation concern.
	SeverityWarning = "warning"
)

// Finding describes one resource-summary validation result.
type Finding struct {
	Severity string
	Subject  string
	Index    int
	Field    string
	Summary  string
	Detail   string
}

// ValidateResourcesSummary validates requested sites, capacities, and component
// counts against a portal resources summary.
func ValidateResourcesSummary(spec SliceSpec, summary *catalog.ResourcesSummary) []Finding {
	data, ok := summary.First()
	if !ok || len(data.Sites) == 0 {
		return nil
	}
	var findings []Finding
	siteCaps := map[string]sliver.Capacities{}
	siteComponents := map[string]map[string]int{}
	for i, node := range spec.Nodes {
		if node.Site == "" {
			continue
		}
		site, found := summary.Site(node.Site)
		if !found {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Subject:  "node",
				Index:    i,
				Field:    "site",
				Summary:  "Unknown FABRIC site",
				Detail:   fmt.Sprintf("The FABRIC portal resources summary does not include site %q. Known active and inactive sites: %s.", node.Site, knownSiteNames(data.Sites)),
			})
			continue
		}
		if site.State != "" && site.State != "Active" {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Subject:  "node",
				Index:    i,
				Field:    "site",
				Summary:  "FABRIC site is not active",
				Detail:   fmt.Sprintf("Site %q is currently %q in the FABRIC portal resources summary. Choose an active site or retry when the site becomes active.", node.Site, site.State),
			})
		}
		siteCaps[node.Site] = siteCaps[node.Site].Add(CapacitiesFromNode(node))
		for _, component := range node.Components {
			key, ok := resourcesComponentKey(component)
			if !ok {
				continue
			}
			if siteComponents[node.Site] == nil {
				siteComponents[node.Site] = map[string]int{}
			}
			siteComponents[node.Site][key]++
			if site.Components != nil {
				if availability, found := site.Components[key]; !found || availability.Capacity == 0 {
					findings = append(findings, Finding{
						Severity: SeverityError,
						Subject:  "node_component",
						Index:    i,
						Field:    "component",
						Summary:  "FABRIC component unavailable at site",
						Detail:   fmt.Sprintf("Site %q does not advertise component %q in the FABRIC portal resources summary.", node.Site, key),
					})
				}
			}
		}
	}
	for siteName, requested := range siteCaps {
		site, ok := summary.Site(siteName)
		if !ok {
			continue
		}
		if requested.Core > site.CoresCapacity || requested.RAM > site.RAMCapacity || requested.Disk > site.DiskCapacity {
			findings = append(findings, Finding{
				Severity: SeverityError,
				Subject:  "slice",
				Summary:  "FABRIC site capacity exceeded",
				Detail:   fmt.Sprintf("Requested aggregate capacity at site %q is core=%d ram=%d disk=%d, but advertised site capacity is core=%d ram=%d disk=%d.", siteName, requested.Core, requested.RAM, requested.Disk, site.CoresCapacity, site.RAMCapacity, site.DiskCapacity),
			})
			continue
		}
		if requested.Core > site.CoresAvailable || requested.RAM > site.RAMAvailable || requested.Disk > site.DiskAvailable {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				Subject:  "slice",
				Summary:  "FABRIC site availability may be insufficient",
				Detail:   fmt.Sprintf("Requested aggregate capacity at site %q is core=%d ram=%d disk=%d, while currently advertised availability is core=%d ram=%d disk=%d. Availability can change before apply, but the orchestrator may reject this request.", siteName, requested.Core, requested.RAM, requested.Disk, site.CoresAvailable, site.RAMAvailable, site.DiskAvailable),
			})
		}
		for componentKey, requestedCount := range siteComponents[siteName] {
			availability, found := site.Components[componentKey]
			if !found {
				continue
			}
			if requestedCount > availability.Capacity {
				findings = append(findings, Finding{
					Severity: SeverityError,
					Subject:  "slice",
					Summary:  "FABRIC component capacity exceeded",
					Detail:   fmt.Sprintf("Requested %d of component %q at site %q, but advertised site capacity is %d.", requestedCount, componentKey, siteName, availability.Capacity),
				})
				continue
			}
			if requestedCount > availability.Available {
				findings = append(findings, Finding{
					Severity: SeverityWarning,
					Subject:  "slice",
					Summary:  "FABRIC component availability may be insufficient",
					Detail:   fmt.Sprintf("Requested %d of component %q at site %q, while currently advertised availability is %d. Availability can change before apply, but the orchestrator may reject this request.", requestedCount, componentKey, siteName, availability.Available),
				})
			}
		}
	}
	return findings
}

func resourcesComponentKey(component ComponentSpec) (string, bool) {
	componentType := component.Type
	componentModel := component.Model
	if component.FABlibName != "" {
		resolvedType, resolvedModel, ok := catalog.ResolveFABlibModel(component.FABlibName)
		if !ok {
			return "", false
		}
		componentType = resolvedType
		componentModel = resolvedModel
	}
	if componentType == "" || componentModel == "" {
		return "", false
	}
	return string(componentType) + "-" + componentModel, true
}

func knownSiteNames(sites []catalog.SiteSummary) string {
	names := make([]string, 0, len(sites))
	for _, site := range sites {
		if site.Name != "" {
			names = append(names, site.Name)
		}
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
