package topology

import (
	"fmt"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// InferServiceType determines the L2 NetworkService type implied by a set of
// interfaces, mirroring FABlib's NetworkService.__calculate_l2_nstype so that a
// topology built by this package selects the same service type FABlib would.
//
// The rules, by distinct interface site count:
//   - 0 or 1 site → L2Bridge (a same-site Ethernet bridge).
//   - 2 sites → L2PTP when the link involves a single facility port or ERO is
//     requested and no basic (shared) NIC is present; otherwise L2STS.
//   - more than 2 sites → ErrConstraintViolation: FABRIC L2 networks span at
//     most two sites.
//
// "Basic NIC" maps to a SharedPort interface (FABlib's NIC_Basic); a facility
// port maps to a FacilityPort interface. eroEnabled reflects whether the caller
// requested an explicit route (ERO), which forces point-to-point.
func (t *Topology) InferServiceType(interfaces []*Interface, eroEnabled bool) (sliver.ServiceType, error) {
	sites := make(map[string]struct{}, len(interfaces))
	includesFacilityPort := false
	facilityPortInterfaces := 0
	basicNICCount := 0
	for _, iface := range interfaces {
		if iface == nil {
			return "", diagnostic(fmt.Errorf("%w: nil interface in service-type inference", ErrInvalidOption), "Interfaces", "")
		}
		sites[t.siteOfInterface(iface)] = struct{}{}
		switch iface.Type() {
		case sliver.InterfaceTypeFacilityPort:
			includesFacilityPort = true
			facilityPortInterfaces++
		case sliver.InterfaceTypeSharedPort:
			basicNICCount++
		}
	}
	return inferL2ServiceType(len(sites), includesFacilityPort, facilityPortInterfaces, basicNICCount, len(interfaces), eroEnabled)
}

// inferL2ServiceType is the pure decision core of InferServiceType, separated
// from topology traversal so the branch logic stays a direct transcription of
// FABlib's algorithm.
func inferL2ServiceType(siteCount int, includesFacilityPort bool, facilityPortInterfaces, basicNICCount, interfaceCount int, eroEnabled bool) (sliver.ServiceType, error) {
	switch {
	case siteCount <= 1:
		return sliver.ServiceTypeL2Bridge, nil
	case siteCount == 2:
		// FABlib forces L2PTP for a lone facility port or an explicit route,
		// but only when no basic NIC is present (basic NICs require STS).
		if ((includesFacilityPort && facilityPortInterfaces < 2) || eroEnabled) && basicNICCount == 0 {
			return sliver.ServiceTypeL2PTP, nil
		}
		if interfaceCount >= 2 {
			return sliver.ServiceTypeL2STS, nil
		}
		return "", diagnostic(fmt.Errorf("%w: cannot infer a two-site service type from %d interface(s)", ErrConstraintViolation, interfaceCount), "Type", "Specify the network type explicitly.")
	default:
		return "", diagnostic(fmt.Errorf("%w: networks are limited to 2 unique sites, got %d", ErrConstraintViolation, siteCount), "Type", "Split the network into per-site services or use an L3 service such as FABNetv4.")
	}
}
