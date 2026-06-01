package sliver

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
)

var (
	bdfPattern       = regexp.MustCompile(`^[0-9a-fA-F]{1,4}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]+$`)
	macPattern       = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)
	tagPattern       = regexp.MustCompile(`^[\w-]{1,255}$`)
	bgpKeyPattern    = regexp.MustCompile(`^[\w\-+_/\.:]{6,150}$`)
	accountIDPattern = regexp.MustCompile(`^[\w\-/\.]{3,100}$`)
	regionPattern    = regexp.MustCompile(`^[\w\-\.]{3,100}$`)
	usbIDPattern     = regexp.MustCompile(`^[0-9a-f]{4}:[0-9a-f]{4}$`)
)

// Capacities captures numeric resource capacities for ASM nodes.
type Capacities struct {
	CPU       int `json:"cpu,omitempty"`
	Core      int `json:"core,omitempty"`
	RAM       int `json:"ram,omitempty"`
	Disk      int `json:"disk,omitempty"`
	BW        int `json:"bw,omitempty"`
	BurstSize int `json:"burst_size,omitempty"`
	Unit      int `json:"unit,omitempty"`
	MTU       int `json:"mtu,omitempty"`
}

// Empty reports whether every capacity field is unset.
func (c Capacities) Empty() bool {
	return c == Capacities{}
}

// Validate returns an error if any capacity field is negative.
func (c Capacities) Validate() error {
	values := map[string]int{"cpu": c.CPU, "core": c.Core, "ram": c.RAM, "disk": c.Disk, "bw": c.BW, "burst_size": c.BurstSize, "unit": c.Unit, "mtu": c.MTU}
	for field, value := range values {
		if value < 0 {
			return fmt.Errorf("%w: capacities.%s must be non-negative, got %d", ErrInvalidValue, field, value)
		}
	}
	return nil
}

// Add returns an element-wise capacity sum.
func (c Capacities) Add(other Capacities) Capacities {
	return Capacities{CPU: c.CPU + other.CPU, Core: c.Core + other.Core, RAM: c.RAM + other.RAM, Disk: c.Disk + other.Disk, BW: c.BW + other.BW, BurstSize: c.BurstSize + other.BurstSize, Unit: c.Unit + other.Unit, MTU: c.MTU + other.MTU}
}

// Sub returns an element-wise capacity difference.
func (c Capacities) Sub(other Capacities) Capacities {
	return Capacities{CPU: c.CPU - other.CPU, Core: c.Core - other.Core, RAM: c.RAM - other.RAM, Disk: c.Disk - other.Disk, BW: c.BW - other.BW, BurstSize: c.BurstSize - other.BurstSize, Unit: c.Unit - other.Unit, MTU: c.MTU - other.MTU}
}

// GreaterOrEqual reports whether every capacity field is greater than or equal to other.
func (c Capacities) GreaterOrEqual(other Capacities) bool {
	return c.CPU >= other.CPU && c.Core >= other.Core && c.RAM >= other.RAM && c.Disk >= other.Disk && c.BW >= other.BW && c.BurstSize >= other.BurstSize && c.Unit >= other.Unit && c.MTU >= other.MTU
}

// Labels captures optional addressing, VLAN, and device labels for ASM nodes.
type Labels struct {
	BDF            string `json:"bdf,omitempty"`
	MAC            string `json:"mac,omitempty"`
	IPv4           string `json:"ipv4,omitempty"`
	IPv4Range      string `json:"ipv4_range,omitempty"`
	IPv4Subnet     string `json:"ipv4_subnet,omitempty"`
	IPv6           string `json:"ipv6,omitempty"`
	IPv6Range      string `json:"ipv6_range,omitempty"`
	IPv6Subnet     string `json:"ipv6_subnet,omitempty"`
	VLAN           string `json:"vlan,omitempty"`
	VLANRange      string `json:"vlan_range,omitempty"`
	InnerVLAN      string `json:"inner_vlan,omitempty"`
	ASN            string `json:"asn,omitempty"`
	Instance       string `json:"instance,omitempty"`
	InstanceParent string `json:"instance_parent,omitempty"`
	LocalName      string `json:"local_name,omitempty"`
	LocalType      string `json:"local_type,omitempty"`
	DeviceName     string `json:"device_name,omitempty"`
	BGPKey         string `json:"bgp_key,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
	Region         string `json:"region,omitempty"`
	USBID          string `json:"usb_id,omitempty"`
	NUMA           *int   `json:"numa,omitempty"`
}

// Empty reports whether every label field is unset.
func (l Labels) Empty() bool {
	return l == Labels{}
}

// Validate returns an error if any set label field violates the FIM format rules.
func (l Labels) Validate() error {
	checkPattern := func(field, value string, pattern *regexp.Regexp) error {
		if value != "" && !pattern.MatchString(value) {
			return fmt.Errorf("%w: labels.%s has invalid value %q", ErrInvalidValue, field, value)
		}
		return nil
	}
	if err := checkPattern("bdf", l.BDF, bdfPattern); err != nil {
		return err
	}
	if err := checkPattern("mac", l.MAC, macPattern); err != nil {
		return err
	}
	if err := checkIP("ipv4", l.IPv4, false); err != nil {
		return err
	}
	if err := checkIP("ipv6", l.IPv6, true); err != nil {
		return err
	}
	if err := checkCIDR("ipv4_subnet", l.IPv4Subnet, false); err != nil {
		return err
	}
	if err := checkCIDR("ipv6_subnet", l.IPv6Subnet, true); err != nil {
		return err
	}
	if err := checkRange("ipv4_range", l.IPv4Range, false); err != nil {
		return err
	}
	if err := checkRange("ipv6_range", l.IPv6Range, true); err != nil {
		return err
	}
	if err := checkVLAN("vlan", l.VLAN); err != nil {
		return err
	}
	if err := checkVLAN("inner_vlan", l.InnerVLAN); err != nil {
		return err
	}
	if err := checkVLANRange("vlan_range", l.VLANRange); err != nil {
		return err
	}
	if l.ASN != "" {
		asn, err := strconv.ParseUint(l.ASN, 10, 32)
		if err != nil || asn == 0 {
			return fmt.Errorf("%w: labels.asn must be an integer in 1..4294967295, got %q", ErrInvalidValue, l.ASN)
		}
	}
	if err := checkPattern("bgp_key", l.BGPKey, bgpKeyPattern); err != nil {
		return err
	}
	if l.BGPKey != "" && l.ASN == "" {
		return fmt.Errorf("%w: %w", ErrInvalidValue, ErrBGPKeyRequiresASN)
	}
	if err := checkPattern("account_id", l.AccountID, accountIDPattern); err != nil {
		return err
	}
	if err := checkPattern("region", l.Region, regionPattern); err != nil {
		return err
	}
	if err := checkPattern("usb_id", l.USBID, usbIDPattern); err != nil {
		return err
	}
	if l.NUMA != nil && (*l.NUMA < -1 || *l.NUMA > 7) {
		return fmt.Errorf("%w: labels.numa must be in -1..7, got %d", ErrInvalidValue, *l.NUMA)
	}
	return nil
}

// Gateway captures a routed service gateway address.
type Gateway struct {
	IPv4       string `json:"ipv4,omitempty"`
	IPv4Subnet string `json:"ipv4_subnet,omitempty"`
	IPv6       string `json:"ipv6,omitempty"`
	IPv6Subnet string `json:"ipv6_subnet,omitempty"`
	MAC        string `json:"mac,omitempty"`
}

// Empty reports whether every gateway field is unset.
func (g Gateway) Empty() bool {
	return g == Gateway{}
}

// Validate returns an error if the gateway fields are not a valid IPv4 or IPv6 gateway pair.
func (g Gateway) Validate() error {
	if g.Empty() {
		return nil
	}
	hasIPv4 := g.IPv4 != "" || g.IPv4Subnet != ""
	hasIPv6 := g.IPv6 != "" || g.IPv6Subnet != ""
	if hasIPv4 == hasIPv6 {
		return fmt.Errorf("%w: gateway must set either ipv4+ipv4_subnet or ipv6+ipv6_subnet", ErrInvalidValue)
	}
	if hasIPv4 {
		if g.IPv4 == "" || g.IPv4Subnet == "" {
			return fmt.Errorf("%w: gateway ipv4 and ipv4_subnet must both be set", ErrInvalidValue)
		}
		if err := checkIP("gateway.ipv4", g.IPv4, false); err != nil {
			return err
		}
		return checkCIDR("gateway.ipv4_subnet", g.IPv4Subnet, false)
	}
	if g.IPv6 == "" || g.IPv6Subnet == "" {
		return fmt.Errorf("%w: gateway ipv6 and ipv6_subnet must both be set", ErrInvalidValue)
	}
	if err := checkIP("gateway.ipv6", g.IPv6, true); err != nil {
		return err
	}
	return checkCIDR("gateway.ipv6_subnet", g.IPv6Subnet, true)
}

// Flags captures FIM boolean behavior flags.
type Flags struct {
	AutoConfig     bool `json:"auto_config"`
	AutoMount      bool `json:"auto_mount"`
	IPv4Management bool `json:"ipv4_management"`
	PTP            bool `json:"ptp"`
}

// ReservationInfo captures orchestrator reservation state.
type ReservationInfo struct {
	ReservationID    string `json:"reservation_id,omitempty"`
	ReservationState string `json:"reservation_state,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
}

// StructuralInfo captures orchestrator structural graph references.
type StructuralInfo struct {
	SubGraphID    string   `json:"sub_graph_id,omitempty"`
	ParentGraphID string   `json:"parent_graph_id,omitempty"`
	ADMGraphIDs   []string `json:"adm_graph_ids,omitempty"`
}

// Location captures optional postal and geographic node location metadata.
type Location struct {
	Postal string  `json:"postal,omitempty"`
	Lat    float64 `json:"lat,omitempty"`
	Lon    float64 `json:"lon,omitempty"`
}

// CapacityHints captures the instance flavor hint for a NetworkNode.
type CapacityHints struct {
	InstanceType string `json:"instance_type,omitempty"`
}

// PathInfo captures path or graph route information.
type PathInfo struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// PathPayload captures bidirectional explicit path node lists.
type PathPayload struct {
	A2Z []string `json:"a2z"`
	Z2A []string `json:"z2a"`
}

// ERO captures explicit route object information.
type ERO struct {
	Type    string `json:"type"`
	Strict  string `json:"strict"`
	Payload any    `json:"payload"`
}

// MaintenanceEntry captures maintenance status for one element.
type MaintenanceEntry struct {
	State       MaintenanceState `json:"state"`
	Deadline    string           `json:"deadline,omitempty"`
	ExpectedEnd string           `json:"expected_end,omitempty"`
}

// MaintenanceInfo maps element names to maintenance entries.
type MaintenanceInfo map[string]MaintenanceEntry

// Tags captures node tags.
type Tags []string

// Validate returns an error if any tag violates the ASM tag format.
func (t Tags) Validate() error {
	for index, tag := range t {
		if !tagPattern.MatchString(tag) {
			return fmt.Errorf("%w: tags[%d] must match ^[\\w-]{1,255}$, got %q", ErrInvalidValue, index, tag)
		}
	}
	return nil
}

// UserData is an arbitrary JSON document stored on a sliver.
type UserData []byte

// LayoutData is an arbitrary JSON document stored on a sliver.
type LayoutData []byte

// MeasurementData is an arbitrary JSON document stored on a sliver.
type MeasurementData []byte

func checkIP(field, value string, wantIPv6 bool) error {
	if value == "" {
		return nil
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("%w: labels.%s must be an IP literal, got %q", ErrInvalidValue, field, value)
	}
	if wantIPv6 && ip.To4() != nil {
		return fmt.Errorf("%w: labels.%s must be IPv6, got %q", ErrInvalidValue, field, value)
	}
	if !wantIPv6 && ip.To4() == nil {
		return fmt.Errorf("%w: labels.%s must be IPv4, got %q", ErrInvalidValue, field, value)
	}
	return nil
}

func checkCIDR(field, value string, wantIPv6 bool) error {
	if value == "" {
		return nil
	}
	ip, _, err := net.ParseCIDR(value)
	if err != nil {
		return fmt.Errorf("%w: labels.%s must be CIDR, got %q", ErrInvalidValue, field, value)
	}
	if wantIPv6 && ip.To4() != nil {
		return fmt.Errorf("%w: labels.%s must be IPv6 CIDR, got %q", ErrInvalidValue, field, value)
	}
	if !wantIPv6 && ip.To4() == nil {
		return fmt.Errorf("%w: labels.%s must be IPv4 CIDR, got %q", ErrInvalidValue, field, value)
	}
	return nil
}

func checkRange(field, value string, wantIPv6 bool) error {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return fmt.Errorf("%w: labels.%s must be <ip>-<ip>, got %q", ErrInvalidValue, field, value)
	}
	if err := checkIP(field, parts[0], wantIPv6); err != nil {
		return err
	}
	return checkIP(field, parts[1], wantIPv6)
}

func checkVLAN(field, value string) error {
	if value == "" {
		return nil
	}
	vlan, err := strconv.Atoi(value)
	if err != nil || vlan < 0 || vlan > 4096 {
		return fmt.Errorf("%w: labels.%s must be an integer in 0..4096, got %q", ErrInvalidValue, field, value)
	}
	return nil
}

func checkVLANRange(field, value string) error {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return fmt.Errorf("%w: labels.%s must be <lo>-<hi>, got %q", ErrInvalidValue, field, value)
	}
	lo, loErr := strconv.Atoi(parts[0])
	hi, hiErr := strconv.Atoi(parts[1])
	if loErr != nil || hiErr != nil || lo < 0 || hi > 4096 || lo > hi {
		return fmt.Errorf("%w: labels.%s must be a VLAN range within 0..4096, got %q", ErrInvalidValue, field, value)
	}
	return nil
}

func jsonBlobBytes(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, nil
	}
	if !json.Valid(value) {
		return nil, fmt.Errorf("%w: blob must contain valid JSON", ErrInvalidJSON)
	}
	return value, nil
}
