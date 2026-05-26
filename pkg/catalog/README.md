# pkg/catalog

Package `catalog` exposes embedded FABRIC instance and component catalogs.

The JSON mirrors upstream Python FIM catalog data from
`InformationModel/fim/slivers/data` and is embedded at build time from:

- `data/instances.json`
- `data/components.json`

Maintainers should be able to replace those files and run `go test ./pkg/catalog`
without changing Go code.

## What This Package Owns

- loading embedded catalog JSON
- deterministic instance and component lookup
- validation of catalog shape and values
- FABlib model-name aliases
- generation of component sliver subtrees for `topology.Node.AddComponent`

It does not own topology graph mutation or Terraform schema mapping.

## Instance Catalog API

Entry point:

- `Instances() (*InstanceCatalog, error)`

Methods:

- `Lookup(name) (sliver.Capacities, bool)`
- `Names() []string`
- `MapCapacitiesToInstance(want sliver.Capacities) string`
- `Validate() error`

`MapCapacitiesToInstance` returns the smallest known instance that satisfies
the requested capacities. If no instance fits, it returns the largest known
instance name.

## Component Catalog API

Entry point:

- `Components() (*ComponentCatalog, error)`

Types:

- `ComponentCatalog`
- `ComponentEntry`
- `GenerateOpts`
- `GeneratedComponent`

Methods:

- `Lookup(componentType, model) (ComponentEntry, bool)`
- `Entries() []ComponentEntry`
- `Generate(opts) (GeneratedComponent, error)`
- `Validate() error`

`Generate` returns typed slivers for the component, optional internal service,
and interfaces. Topology code inserts those slivers into the graph.

## FABlib Aliases

`ResolveFABlibModel(name)` maps FABlib model names such as `NIC_ConnectX_6` to
the component type and catalog model used by this package.

Alias constants are exported with the `FABlib...` prefix, including common NIC,
GPU, NVMe, and FPGA model names.

## Maintenance Workflow

1. Replace `data/instances.json` or `data/components.json`.
2. Run `go test ./pkg/catalog`.
3. If generated topology fixtures change, run `python testdata/generate_fixtures.py`.
4. Run `go test ./...`.

Validation fails on duplicate component keys, unknown component types, empty
details, invalid port speeds, invalid capacities, and malformed instance data.
