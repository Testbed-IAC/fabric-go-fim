package sliver

import (
	"encoding/json"
	"fmt"
)

// ToProps emits the property bag for a NetworkNode sliver.
func (s *NodeSliver) ToProps() (map[string]string, error) {
	props := baseProps(ClassNetworkNode, &s.BaseSliver)
	props[PropType] = string(s.Type)
	if s.Site != "" {
		props[PropSite] = s.Site
	}
	if s.ImageRef != "" && s.ImageType != "" {
		props[PropImageRef] = s.ImageRef + "," + s.ImageType
	}
	setString(props, PropMgmtIP, s.MgmtIP)
	setString(props, PropAllocationConstraints, s.AllocationConstraints)
	setString(props, PropServiceEndpoint, s.ServiceEndpoint)
	if err := setJSON(props, PropLocation, s.Location); err != nil {
		return nil, err
	}
	if err := setJSON(props, PropMaintenanceInfo, s.MaintenanceInfo); err != nil {
		return nil, err
	}
	return props, validateBase(&s.BaseSliver)
}

// FromProps populates a NetworkNode sliver from a property bag.
func (s *NodeSliver) FromProps(props map[string]string) error {
	if err := requireClass(props, ClassNetworkNode); err != nil {
		return err
	}
	s.Type = NodeType(props[PropType])
	s.Site = props[PropSite]
	if imageRef := props[PropImageRef]; imageRef != "" && imageRef != "None" {
		ref, imageType := splitImageRef(imageRef)
		s.ImageRef = ref
		s.ImageType = imageType
	}
	s.MgmtIP = props[PropMgmtIP]
	s.AllocationConstraints = props[PropAllocationConstraints]
	s.ServiceEndpoint = props[PropServiceEndpoint]
	if err := readBase(&s.BaseSliver, props); err != nil {
		return err
	}
	if err := readJSON(props, PropLocation, &s.Location); err != nil {
		return err
	}
	return readJSONValue(props, PropMaintenanceInfo, &s.MaintenanceInfo)
}

// ToProps emits the property bag for a NetworkService sliver.
func (s *NetworkServiceSliver) ToProps() (map[string]string, error) {
	props := baseProps(ClassNetworkService, &s.BaseSliver)
	props[PropType] = string(s.Type)
	if s.Layer != "" {
		props[PropLayer] = string(s.Layer)
	}
	setString(props, PropTechnology, s.Technology)
	setString(props, PropAllocationConstraints, s.AllocationConstraints)
	setString(props, PropControllerURL, s.ControllerURL)
	setString(props, PropSite, s.Site)
	setString(props, PropMirrorPort, s.MirrorPort)
	setString(props, PropMirrorVLAN, s.MirrorVLAN)
	if s.MirrorDirection != "" {
		props[PropMirrorDirection] = string(s.MirrorDirection)
	}
	if err := setJSON(props, PropGateway, s.Gateway); err != nil {
		return nil, err
	}
	if err := setJSON(props, PropERO, s.ERO); err != nil {
		return nil, err
	}
	if err := setJSON(props, PropPathInfo, s.PathInfo); err != nil {
		return nil, err
	}
	return props, validateBase(&s.BaseSliver)
}

// FromProps populates a NetworkService sliver from a property bag.
func (s *NetworkServiceSliver) FromProps(props map[string]string) error {
	if err := requireClass(props, ClassNetworkService); err != nil {
		return err
	}
	s.Type = ServiceType(props[PropType])
	s.Layer = NSLayer(props[PropLayer])
	s.Technology = props[PropTechnology]
	s.AllocationConstraints = props[PropAllocationConstraints]
	s.ControllerURL = props[PropControllerURL]
	s.Site = props[PropSite]
	s.MirrorPort = props[PropMirrorPort]
	s.MirrorVLAN = props[PropMirrorVLAN]
	s.MirrorDirection = MirrorDirection(props[PropMirrorDirection])
	if err := readBase(&s.BaseSliver, props); err != nil {
		return err
	}
	if err := readJSON(props, PropGateway, &s.Gateway); err != nil {
		return err
	}
	if err := readJSON(props, PropERO, &s.ERO); err != nil {
		return err
	}
	return readJSON(props, PropPathInfo, &s.PathInfo)
}

// ToProps emits the property bag for a Component sliver.
func (s *ComponentSliver) ToProps() (map[string]string, error) {
	props := baseProps(ClassComponent, &s.BaseSliver)
	props[PropType] = string(s.Type)
	setString(props, PropModel, s.Model)
	return props, validateBase(&s.BaseSliver)
}

// FromProps populates a Component sliver from a property bag.
func (s *ComponentSliver) FromProps(props map[string]string) error {
	if err := requireClass(props, ClassComponent); err != nil {
		return err
	}
	s.Type = ComponentType(props[PropType])
	s.Model = props[PropModel]
	return readBase(&s.BaseSliver, props)
}

// ToProps emits the property bag for an Interface sliver.
func (s *InterfaceSliver) ToProps() (map[string]string, error) {
	props := baseProps(ClassConnectionPoint, &s.BaseSliver)
	props[PropType] = string(s.Type)
	if err := setJSON(props, PropPeerLabels, s.PeerLabels); err != nil {
		return nil, err
	}
	return props, validateBase(&s.BaseSliver)
}

// FromProps populates an Interface sliver from a property bag.
func (s *InterfaceSliver) FromProps(props map[string]string) error {
	if err := requireClass(props, ClassConnectionPoint); err != nil {
		return err
	}
	s.Type = InterfaceType(props[PropType])
	if err := readBase(&s.BaseSliver, props); err != nil {
		return err
	}
	return readJSON(props, PropPeerLabels, &s.PeerLabels)
}

// ToProps emits the property bag for a Link sliver.
func (s *LinkSliver) ToProps() (map[string]string, error) {
	props := baseProps(ClassLink, &s.BaseSliver)
	props[PropType] = string(s.Type)
	if s.Layer != "" {
		props[PropLayer] = string(s.Layer)
	}
	setString(props, PropTechnology, s.Technology)
	return props, validateBase(&s.BaseSliver)
}

// FromProps populates a Link sliver from a property bag.
func (s *LinkSliver) FromProps(props map[string]string) error {
	if err := requireClass(props, ClassLink); err != nil {
		return err
	}
	s.Type = LinkType(props[PropType])
	s.Layer = NSLayer(props[PropLayer])
	s.Technology = props[PropTechnology]
	return readBase(&s.BaseSliver, props)
}

func baseProps(class string, s *BaseSliver) map[string]string {
	props := make(map[string]string, 24)
	for key, value := range s.Opaque {
		if value != "" && value != "None" {
			props[key] = value
		}
	}
	props[PropClass] = class
	props[PropName] = s.Name
	props[PropNodeID] = s.NodeID
	props[PropGraphID] = s.GraphID

	if s.StitchNode {
		props[PropStitchNode] = "true"
	} else {
		props[PropStitchNode] = "false"
	}
	setString(props, PropDetails, s.Details)
	setString(props, PropBootScript, s.BootScript)
	mustSetJSON(props, PropCapacities, s.Capacities)
	mustSetJSON(props, PropCapacityHints, s.CapacityHints)
	mustSetJSON(props, PropLabels, s.Labels)
	mustSetJSON(props, PropCapacityAllocations, s.CapacityAllocations)
	mustSetJSON(props, PropLabelAllocations, s.LabelAllocations)
	mustSetJSON(props, PropReservationInfo, s.ReservationInfo)
	mustSetJSON(props, PropStructuralInfo, s.StructuralInfo)
	mustSetJSON(props, PropTags, s.Tags)
	if s.Flags != nil {
		encoded, _ := json.Marshal(s.Flags)
		props[PropFlags] = string(encoded)
	}
	mustSetBlob(props, PropUserData, []byte(s.UserData))
	mustSetBlob(props, PropLayoutData, []byte(s.LayoutData))
	mustSetBlob(props, PropMeasurementData, []byte(s.MeasurementData))
	if s.NodeMap != [2]string{} {
		mustSetJSON(props, PropNodeMap, s.NodeMap)
	}
	return props
}

func readBase(s *BaseSliver, props map[string]string) error {
	s.Name = props[PropName]
	s.NodeID = props[PropNodeID]
	s.GraphID = props[PropGraphID]
	s.Details = props[PropDetails]
	s.BootScript = props[PropBootScript]
	s.StitchNode = props[PropStitchNode] == "true"
	if err := readJSON(props, PropCapacities, &s.Capacities); err != nil {
		return err
	}
	if err := readJSON(props, PropCapacityHints, &s.CapacityHints); err != nil {
		return err
	}
	if err := readJSON(props, PropLabels, &s.Labels); err != nil {
		return err
	}
	if err := readJSON(props, PropCapacityAllocations, &s.CapacityAllocations); err != nil {
		return err
	}
	if err := readJSON(props, PropLabelAllocations, &s.LabelAllocations); err != nil {
		return err
	}
	if err := readJSON(props, PropReservationInfo, &s.ReservationInfo); err != nil {
		return err
	}
	if err := readJSON(props, PropStructuralInfo, &s.StructuralInfo); err != nil {
		return err
	}
	if err := readJSONValue(props, PropTags, &s.Tags); err != nil {
		return err
	}
	if err := readJSON(props, PropFlags, &s.Flags); err != nil {
		return err
	}
	if err := readJSONValue(props, PropNodeMap, &s.NodeMap); err != nil {
		return err
	}
	if err := readBlob(props, PropUserData, &s.UserData); err != nil {
		return err
	}
	if err := readBlob(props, PropLayoutData, &s.LayoutData); err != nil {
		return err
	}
	if err := readBlob(props, PropMeasurementData, &s.MeasurementData); err != nil {
		return err
	}
	s.Opaque = make(map[string]string)
	for key, value := range props {
		if _, known := knownProps[key]; !known && value != "" && value != "None" {
			s.Opaque[key] = value
		}
	}
	return validateRequiredBase(s)
}

func validateRequiredBase(s *BaseSliver) error {
	if s.Name == "" {
		return fmt.Errorf("%w: %s is required", ErrMissingProperty, PropName)
	}
	if s.NodeID == "" {
		return fmt.Errorf("%w: %s is required on %q", ErrMissingProperty, PropNodeID, s.Name)
	}
	if s.GraphID == "" {
		return fmt.Errorf("%w: %s is required on %q", ErrMissingProperty, PropGraphID, s.Name)
	}
	return nil
}

func validateBase(s *BaseSliver) error {
	if err := validateRequiredBase(s); err != nil {
		return err
	}
	if s.Capacities != nil {
		if err := s.Capacities.Validate(); err != nil {
			return err
		}
	}
	if s.CapacityAllocations != nil {
		if err := s.CapacityAllocations.Validate(); err != nil {
			return err
		}
	}
	if s.Labels != nil {
		if err := s.Labels.Validate(); err != nil {
			return err
		}
	}
	if s.LabelAllocations != nil {
		if err := s.LabelAllocations.Validate(); err != nil {
			return err
		}
	}
	if err := s.Tags.Validate(); err != nil {
		return err
	}
	if len(s.BootScript) >= 1024 {
		return fmt.Errorf("%w: BootScript on %q must be shorter than 1024 bytes", ErrInvalidValue, s.Name)
	}
	if err := validateBlobSize("UserData", []byte(s.UserData), 2048); err != nil {
		return err
	}
	if err := validateBlobSize("LayoutData", []byte(s.LayoutData), 1024); err != nil {
		return err
	}
	return validateBlobSize("MeasurementData", []byte(s.MeasurementData), 4096)
}

func setString(props map[string]string, key, value string) {
	if value != "" && value != "None" {
		props[key] = value
	}
}

func setJSON(props map[string]string, key string, value any) error {
	encoded, ok, err := marshalJSONProp(value)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalidJSON, key, err)
	}
	if ok {
		props[key] = encoded
	}
	return nil
}

func mustSetJSON(props map[string]string, key string, value any) {
	encoded, ok, err := marshalJSONProp(value)
	if err == nil && ok {
		props[key] = encoded
	}
}

func marshalJSONProp(value any) (string, bool, error) {
	if value == nil {
		return "", false, nil
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", false, err
	}
	text := string(bytes)
	if text == "null" || text == "{}" || text == "[]" || text == `""` {
		return "", false, nil
	}
	return text, true, nil
}

func readJSON[T any](props map[string]string, key string, target **T) error {
	value := props[key]
	if value == "" || value == "None" {
		return nil
	}
	var decoded T
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalidJSON, key, err)
	}
	*target = &decoded
	return nil
}

func readJSONValue[T any](props map[string]string, key string, target *T) error {
	value := props[key]
	if value == "" || value == "None" {
		return nil
	}
	if err := json.Unmarshal([]byte(value), target); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrInvalidJSON, key, err)
	}
	return nil
}

func readBlob[T ~[]byte](props map[string]string, key string, target *T) error {
	value := props[key]
	if value == "" || value == "None" {
		return nil
	}
	if !json.Valid([]byte(value)) {
		return fmt.Errorf("%w: %s contains invalid JSON", ErrInvalidJSON, key)
	}
	*target = T(value)
	return nil
}

func mustSetBlob(props map[string]string, key string, value []byte) {
	if len(value) > 0 {
		props[key] = string(value)
	}
}

func validateBlobSize(name string, value []byte, max int) error {
	if len(value) == 0 {
		return nil
	}
	bytes, err := jsonBlobBytes(value)
	if err != nil {
		return fmt.Errorf("%w: %s", err, name)
	}
	if len(bytes) > max {
		return fmt.Errorf("%w: %s must be at most %d bytes, got %d", ErrInvalidValue, name, max, len(bytes))
	}
	return nil
}

func requireClass(props map[string]string, want string) error {
	if props[PropClass] != want {
		return fmt.Errorf("%w: %s must be %q, got %q", ErrInvalidValue, PropClass, want, props[PropClass])
	}
	return nil
}

func splitImageRef(value string) (string, string) {
	for index, r := range value {
		if r == ',' {
			return value[:index], value[index+1:]
		}
	}
	return value, ""
}

var knownProps = map[string]struct{}{
	PropAllocationConstraints: {},
	PropBootScript:            {},
	PropCapacityAllocations:   {},
	PropCapacityDelegations:   {},
	PropCapacityHints:         {},
	PropCapacities:            {},
	PropClass:                 {},
	PropControllerURL:         {},
	PropDetails:               {},
	PropERO:                   {},
	PropFlags:                 {},
	PropGateway:               {},
	PropGraphID:               {},
	PropImageRef:              {},
	PropLabels:                {},
	PropLabelAllocations:      {},
	PropLabelDelegations:      {},
	PropLayer:                 {},
	PropLayoutData:            {},
	PropLocation:              {},
	PropMaintenanceInfo:       {},
	PropMeasurementData:       {},
	PropMgmtIP:                {},
	PropMirrorDirection:       {},
	PropMirrorPort:            {},
	PropMirrorVLAN:            {},
	PropModel:                 {},
	PropName:                  {},
	PropNodeID:                {},
	PropNodeMap:               {},
	PropPathInfo:              {},
	PropPeerLabels:            {},
	PropReservationInfo:       {},
	PropServiceEndpoint:       {},
	PropSite:                  {},
	PropStitchNode:            {},
	PropStructuralInfo:        {},
	PropTags:                  {},
	PropTechnology:            {},
	PropType:                  {},
	PropUserData:              {},
	PropXMLLabels:             {},
}
