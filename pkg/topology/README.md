# pkg/topology

Package `topology` is the primary construction API for FABRIC ASM topologies.
Terraform provider code should start here.

This package owns:

- topology creation and loading
- builder methods for nodes, services, facilities, switches, links, and ports
- validation errors with field-level diagnostics
- deterministic graph and element IDs
- GraphML serialization
- topology-level diff wrappers

This package does not own generic graph storage, XML parsing details, raw
catalog data, or sliver property schemas.

## Basic Usage

```go
topo := topology.NewWithID(topology.DeriveGraphID("slice-a"))

vm, err := topo.AddNode(topology.NodeOpts{
	Name: "vm1",
	Site: "RENC",
})
if err != nil {
	return err
}

nic, err := vm.AddComponent(topology.ComponentOpts{
	Name: "nic1",
	Type: sliver.ComponentTypeSharedNIC,
	Model: "ConnectX-6",
})
if err != nil {
	return err
}

_, err = topo.AddNetworkService(topology.NetworkServiceOpts{
	Name:       "lan1",
	Type:       sliver.ServiceTypeL2Bridge,
	Interfaces: []*topology.Interface{nic.Interfaces()[0]},
})
```

## Construction Entry Points

Top-level functions:

- `New()` creates a topology with a random graph ID.
- `NewWithID(graphID)` creates a topology with a caller-provided graph ID.
- `DeriveGraphID(stableKey)` derives a stable graph ID from a caller key.
- `Load(r)` parses GraphML into a topology.

Topology methods:

- `AddNode(NodeOpts)` creates a top-level FABRIC `NetworkNode`.
- `AddNetworkService(NetworkServiceOpts)` creates a top-level network service.
- `AddFacility(FacilityOpts)` creates a facility node with internal VLAN service and ports.
- `AddSwitch(SwitchOpts)` creates a switch node with internal service and ports.
- `AddPortMirrorService(PortMirrorOpts)` creates a port mirror service.
- `AddLink(LinkOpts)` creates an explicit link node.
- `Node(name)` looks up a network node by name.
- `Serialize(w)` writes GraphML.
- `SerializeString()` returns GraphML as a string.
- `GraphID()` returns the topology graph ID.

## Facade Types

The facade types hide internal graph nodes:

- `Node`
- `Component`
- `NetworkService`
- `Interface`
- `Link`

Common methods expose stable IDs, names, typed slivers, and owned children.
Examples:

- `Node.ID`, `Node.Name`, `Node.Site`, `Node.Sliver`
- `Node.AddComponent`, `Node.AddNetworkService`
- `Node.Components`, `Node.NetworkServices`, `Node.InterfaceList`
- `Component.Sliver`, `Component.NetworkServices`, `Component.Interfaces`
- `NetworkService.Sliver`, `NetworkService.Interfaces`, `NetworkService.AddInterface`
- `NetworkService.ConnectInterface`, `NetworkService.DisconnectInterface`
- `Interface.Sliver`, `Interface.Type`, `Interface.AddChildInterface`
- `Link.Sliver`

Provider code should keep references to facade types while building a desired
topology. It should not inspect `internal/graph`.

## Option Structs

Option structs are the stable shape a provider should map schema data into:

- `NodeOpts`
- `ComponentOpts`
- `InterfaceOpts`
- `NetworkServiceOpts`
- `FacilityOpts`
- `FacilityInterfaceOpts`
- `SwitchOpts`
- `PortMirrorOpts`
- `LinkOpts`

Names are part of deterministic identity. Once a topology has been serialized
or stored in Terraform state, changing a name should be treated as replacing
that topology element.

## Diff Entry Points

The topology package re-exports semantic diffing so provider code does not need
to import graph internals:

- `DiffTopologies(expected, actual)`
- `DiffGraphML(expectedGraphML, actualGraphML)`
- `DiffTopologyGraphML(expectedGraphML, actualGraphML)`
- `DiffGraphs(expected, actual)`
- `NormalizeGraph(g)`

`DiffTopologies` is the preferred provider entrypoint. It returns
`TopologyDiff`, which wraps the graph-level `GraphDiff` and exposes:

- `Empty`
- `HasChanges`
- `Summary`
- `Diagnostics`

Diffing ignores generated IDs and runtime-only fields such as management IPs,
reservation metadata, sliver IDs, timestamps, lease values, and slice state.

## Errors

Builder methods return errors that can be matched with:

- `ErrDuplicateName`
- `ErrNotFound`
- `ErrInvalidOption`
- `ErrConstraintViolation`

Topology errors implement `Diagnostic` when there is field-level context:

```go
var diag topology.Diagnostic
if errors.As(err, &diag) {
	field := diag.Field()
	hint := diag.Suggestion()
}
```

This is intended to map cleanly to Terraform diagnostics.

## Invariants

- GraphML output must remain semantically compatible with Python FIM fixtures.
- Deterministic UUID behavior must not change without an intentional state migration.
- Public APIs should expose topology concepts, not internal graph structure.
- Diff output must stay stable across insertion order and JSON key ordering.
