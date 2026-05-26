package sliver

// ASM graph node class names.
// These values appear as the Class property on every graph node and as the
// "labels" XML attribute in GraphML output (e.g., ":GraphNode:NetworkNode").
const (
	ClassNetworkNode     = "NetworkNode"
	ClassNetworkService  = "NetworkService"
	ClassComponent       = "Component"
	ClassConnectionPoint = "ConnectionPoint"
	ClassLink            = "Link"
)

// ASM graph edge class names.
// "has" expresses hierarchical ownership (e.g., NetworkNode → Component).
// "connects" expresses connectivity (e.g., NetworkService → ConnectionPoint).
const (
	EdgeHas      = "has"
	EdgeConnects = "connects"
)

// ASM node property keys.
//
// Scalar properties are stored as plain strings in the property map.
// Structured properties (those present in JSONPropertyNames) are stored as
// compact JSON strings and must be decoded before use.
//
// Property names are compatible with the Python FIM reference implementation.
const (
	PropAllocationConstraints = "AllocationConstraints"
	PropBootScript            = "BootScript"
	PropCapacityAllocations   = "CapacityAllocations"
	PropCapacityDelegations   = "CapacityDelegations"
	PropCapacityHints         = "CapacityHints"
	PropCapacities            = "Capacities"
	PropClass                 = "Class"
	PropControllerURL         = "ControllerURL"
	PropDetails               = "Details"
	PropERO                   = "ERO"
	PropFlags                 = "Flags"
	PropGateway               = "Gateway"
	PropGraphID               = "GraphID"
	PropImageRef              = "ImageRef"
	PropLabels                = "Labels"
	PropLabelAllocations      = "LabelAllocations"
	PropLabelDelegations      = "LabelDelegations"
	PropLayer                 = "Layer"
	PropLayoutData            = "LayoutData"
	PropLocation              = "Location"
	PropMaintenanceInfo       = "MaintenanceInfo"
	PropMeasurementData       = "MeasurementData"
	PropMgmtIP                = "MgmtIp"
	PropMirrorDirection       = "MirrorDirection"
	PropMirrorPort            = "MirrorPort"
	PropMirrorVLAN            = "MirrorVlan"
	PropModel                 = "Model"
	PropName                  = "Name"
	PropNodeID                = "NodeID"
	PropNodeMap               = "NodeMap"
	PropPathInfo              = "PathInfo"
	PropPeerLabels            = "PeerLabels"
	PropReservationInfo       = "ReservationInfo"
	PropServiceEndpoint       = "ServiceEndpoint"
	PropSite                  = "Site"
	PropStitchNode            = "StitchNode"
	PropStructuralInfo        = "StructuralInfo"
	PropTags                  = "Tags"
	PropTechnology            = "Technology"
	PropType                  = "Type"
	PropUserData              = "UserData"
	// PropXMLLabels is the GraphML "labels" attribute synthesised by the writer
	// from the node Class. It is never stored in the property map and is
	// stripped by the reader; callers should not set or inspect it directly.
	PropXMLLabels = "labels"
)

// JSONPropertyNames is the set of property keys whose values are stored as
// compact JSON strings. Callers that perform structural comparison use this set
// to detect and deserialize JSON values rather than comparing raw strings.
var JSONPropertyNames = map[string]struct{}{
	PropLabels:              {},
	PropCapacities:          {},
	PropLabelDelegations:    {},
	PropCapacityDelegations: {},
	PropLabelAllocations:    {},
	PropCapacityAllocations: {},
	PropReservationInfo:     {},
	PropStructuralInfo:      {},
	PropERO:                 {},
	PropPathInfo:            {},
	PropCapacityHints:       {},
	PropGateway:             {},
	PropTags:                {},
	PropFlags:               {},
	PropMeasurementData:     {},
	PropUserData:            {},
	PropLayoutData:          {},
	PropPeerLabels:          {},
	PropMaintenanceInfo:     {},
	PropLocation:            {},
	PropNodeMap:             {},
}
