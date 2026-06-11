# fabric-go-fim

`fabric-go-fim` is a Go implementation of the FABRIC Information Model (FIM) Abstract Slice Model. It builds FABRIC experiment topologies in Go, serializes them to and loads them from FABRIC-compatible GraphML, validates requested components against embedded catalogs, derives the permission tags a request needs, compares topology intent for drift, and provides a high-level orchestrator client with token handling and polling.

This module is the backend for [terraform-provider-fabric](../terraform-provider-fabric), which depends on it through a workspace `replace` directive. It is a standalone library with no Terraform dependency and is usable on its own.

## Packages

| Import path | Provides | Stability |
|---|---|---|
| `pkg/topology` | Primary topology construction and GraphML API: `New`, `NewWithID`, `AddNode`, `AddNetworkService`, `AddFacility`, `AddSwitch`, `AddLink`, `AddPortMirrorService`, `Serialize`. | Public |
| `pkg/topologybuilder` | Builds a topology and GraphML from a plain Go `SliceSpec`: `Build`, `ValidateCatalog`, `ValidateResourcesSummary`, `PermissionRequest`, `CapacitiesFromNode`. | Public |
| `pkg/catalog` | Embedded FABRIC instance and component catalogs, model lookup, and `DecodeAdvertised` for advertised-resource GraphML. | Public |
| `pkg/sliver` | Typed ASM/FIM records, enum values (node types, service types, mirror directions), and property-key encoding. | Public |
| `pkg/permission` | Derives the project permission tags a slice request requires: `RequiredTags`, `Missing`, `TagForSmartNICModel`. | Public |
| `pkg/client` | High-level FABRIC orchestrator client: the `API` interface and `New`. | Public |
| `pkg/auth` | Token sources (`StaticToken`, `FileToken`) and unverified JWT claim parsing (`ParseJWT`, `Claims`). | Public |
| `pkg/poller` | Slice and POA polling helpers: `WaitForSlice`, `WaitForPOA`. | Public |
| `pkg/fabtime` | Parses and formats FABRIC orchestrator lease timestamps. | Public |
| `pkg/sshkeys` | Normalizes SSH key inputs shared by API clients and topology builders. | Public |
| `pkg/userdata` | Typed FABlib-compatible user-data envelope (boot scripts, post-boot commands, routes, uploads). | Public |
| `pkg/diff` | Semantic graph normalization and drift comparison: `NormalizeGraph`, `DiffTopologies`, `DiffTopologyGraphML`. | Public |
| `pkg/fim` | Top-level module metadata. | Public |
| `internal/graph` | Graph data structure. | Internal — do not import |
| `internal/graphml` | GraphML XML reader/writer. | Internal — do not import |

Topology comparison ignores runtime-only FABRIC metadata (UUIDs, XML ordering, JSON key ordering), so two graphs representing the same intent compare equal.

## Requirements

| Requirement | Version / Condition |
|---|---|
| Go | 1.24 |
| `fabric-orchestrator-go-client` | Sibling module, wired by the `replace` directive in `go.mod`. Required only for `pkg/client`. |

Topology construction, catalog lookup, permission derivation, and GraphML serialization have no orchestrator dependency.

## Build a topology

This produces FABRIC-compatible GraphML for a single VM. It matches [`examples/bare_vm`](examples/bare_vm).

```go
package main

import (
	"fmt"
	"log"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
)

func main() {
	// DeriveGraphID makes the graph ID deterministic for repeatable output.
	topo := topology.NewWithID(topology.DeriveGraphID("example-bare-vm"))

	if _, err := topo.AddNode(topology.NodeOpts{
		Name:       "vm1",
		Site:       "UTAH",
		Type:       sliver.NodeTypeVM,
		Capacities: &sliver.Capacities{Core: 2, RAM: 8, Disk: 10},
		ImageRef:   "default_rocky_9",
		ImageType:  "qcow2",
	}); err != nil {
		log.Fatal(err)
	}

	graphML, err := topo.SerializeString()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(graphML)
}
```

`pkg/topologybuilder.Build` is the higher-level entry point: it takes a declarative `SliceSpec`, validates components against the catalog, and returns a `*topology.Topology` plus its GraphML string. `PermissionRequest(spec)` returns the `permission.Request` for the same spec, which `permission.Missing` checks against a project's tags before submission.

## Catalog

Component and instance data is embedded from `pkg/catalog/data/`. Component models, grouped by type:

| Type | Models |
|---|---|
| `GPU` | `RTX6000`, `Tesla T4`, `A40`, `A30` |
| `SmartNIC` | `ConnectX-5`, `ConnectX-6`, `BlueField-2-ConnectX-6`, `ConnectX-7-100`, `ConnectX-7-400` |
| `SharedNIC` | `ConnectX-6`, `OpenStack-vNIC` |
| `NVME` | `P4510` (1 TB) |
| `Storage` | `NAS` (site-local NAS share) |
| `FPGA` | `Xilinx-U280`, `Xilinx-SN1022` |

Each `SmartNIC` model maps to a distinct permission tag through `permission.TagForSmartNICModel`; an unrecognized model falls back to the `ConnectX-6` tag.

Instance types follow the pattern `fabric.c<cores>.m<ram>.d<disk>`, with `cores` in {1, 2, 4, 6, … 64}, `ram` in {2, 4, 8, 16, 32, 64, 128, 256} GB, and `disk` in {10, 50, 100, 500, 1000} GB — 869 entries in `instances.json`. Look them up through the catalog API rather than hardcoding.

Common image references include `default_rocky_9` and `default_ubuntu_22`. Image availability is site-dependent; decode advertised resources with `catalog.DecodeAdvertised` to confirm what a site offers.

To update catalog data, replace the JSON files and run the catalog tests, which validate every model's type, details, and interface map:

```sh
go test ./pkg/catalog
```

## Examples

Runnable examples under `examples/`:

- `bare_vm` — one VM, serialized to GraphML.
- `l2bridge` — two VMs joined by an L2Bridge network service.
- `serialize_load` — round-trip a topology through GraphML.
- `drift_detection` — compare two topologies for semantic drift.

## Relationship to Python FIM

Python FIM is the behavioral reference for topology semantics. This module does not call Python at runtime; its tests compare Go-generated topologies against GraphML fixtures produced by Python FIM. The comparison is semantic, not byte-for-byte — UUIDs and XML/JSON ordering may differ while representing the same topology.

## Development

```sh
gofmt -w .
go test ./...
```

Regenerate fixtures only when topology fixture behavior changes:

```sh
python testdata/generate_fixtures.py
go test ./...
```
