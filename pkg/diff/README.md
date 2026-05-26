# pkg/diff

Package `diff` owns semantic normalization and drift comparison for topology
graphs.

Most provider code should call the wrappers in `pkg/topology`. This package is
public for callers that already have graph values and need lower-level control.

## What This Package Owns

- canonical graph normalization for comparison
- semantic graph diffing
- GraphML-to-GraphML semantic comparison
- deterministic diagnostics for drift reporting

It does not own topology construction, catalog lookup, or GraphML serialization
policy.

## API

Functions:

- `NormalizeGraph(g) (*graph.Graph, error)`
- `DiffGraphs(expected, actual) GraphDiff`
- `DiffGraphML(expectedGraphML, actualGraphML) (GraphDiff, error)`

Diff result types:

- `GraphDiff`
- `NodeDiff`
- `NodeChange`
- `ClassChange`
- `PropertyChange`
- `EdgeDiff`
- `Diagnostic`

`GraphDiff` exposes:

- `Empty() bool`
- `HasChanges() bool`
- `Summary() string`
- `Diagnostics() []Diagnostic`

## Comparison Rules

The diff is based on desired topology intent:

- node matching is name-based
- edge matching is endpoint-name and edge-class based
- JSON-valued properties are canonicalized before comparison
- output ordering is deterministic
- generated IDs are ignored
- runtime-only FABRIC fields are ignored

Ignored runtime fields include management IPs, reservation metadata, sliver IDs,
timestamps, lease values, allocation fields, maintenance data, measurement
data, layout data, and slice state.

## Provider Guidance

Use `topology.DiffTopologies` when comparing desired Terraform configuration
against current topology state. Use `diff.DiffGraphML` only when the provider
has GraphML bytes and does not have topology objects.
