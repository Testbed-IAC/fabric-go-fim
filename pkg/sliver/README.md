# pkg/sliver

Package `sliver` owns typed ASM/FIM records, enum values, property names, and
conversion between typed values and graph property maps.

Terraform provider code may use these types while mapping schema values into
`pkg/topology` options. Provider code should usually not create graph nodes from
slivers directly.

## What This Package Owns

- FIM class, edge, and property constants
- enum values for node, service, component, interface, and link types
- value structs used in JSON-valued GraphML properties
- sliver structs for typed property conversion
- validation errors for malformed sliver values

It does not own graph storage, topology construction order, catalog lookup, or
GraphML XML encoding.

## Core Sliver Types

- `BaseSliver`
- `NodeSliver`
- `NetworkServiceSliver`
- `ComponentSliver`
- `InterfaceSliver`
- `LinkSliver`

Each sliver supports property-map conversion through its exported methods. The
topology package uses those conversions before inserting values into the graph.

## Value Types

Common provider-facing value types:

- `Capacities`
- `Labels`
- `Gateway`
- `CapacityHints`
- `ERO`
- `PathInfo`
- `Tags`
- `UserData`

Additional value types exist for runtime or compatibility fields:

- `Flags`
- `ReservationInfo`
- `StructuralInfo`
- `Location`
- `MaintenanceInfo`
- `LayoutData`
- `MeasurementData`

## Enum Families

- `NodeType`
- `ServiceType`
- `ComponentType`
- `InterfaceType`
- `LinkType`
- `NSLayer`
- `MirrorDirection`
- `MaintenanceState`

Helpers:

- `LayerForServiceType`
- `LayerForLinkType`

## Errors

- `ErrInvalidValue`
- `ErrInvalidJSON`
- `ErrMissingProperty`

## Invariants

- Enum string values must remain compatible with Python FIM.
- Property names must match the GraphML vocabulary used by Python FIM.
- JSON-valued properties should be encoded through the typed value structs.
