// Package graphml reads and writes FABRIC ASM GraphML documents.
//
// # Overview
//
// This package is the serialization layer between an in-memory property graph
// (internal/graph) and the GraphML wire format used by both the Go and Python
// FIM implementations.
//
// # Reading
//
// Read parses a GraphML document into a *graph.Graph.  The reader is tolerant:
//   - It accepts both named key IDs (attr.name="Name") and opaque key IDs
//     (id="d3") as used by different Neo4j and Python FIM exporters.
//   - It accepts Neo4j-style "labels" and "label" node attributes.
//   - Unknown keys and properties are silently ignored.
//   - The PropXMLLabels ("labels") attribute is stripped from node properties
//     because the writer regenerates it from the node class; storing it would
//     cause a round-trip asymmetry.
//
// # Writing
//
// Write serializes a *graph.Graph to GraphML.  The writer is strict:
//   - Key elements are emitted in sorted, deterministic order.
//   - Nodes and edges are emitted in graph insertion order.
//   - Node XML IDs are sequential integers ("n0", "n1", …) independent of the
//     UUID-based NodeID property, avoiding large diffs on re-ordering.
//   - The "labels" XML attribute is synthesized from the node Class property.
//
// # Test equality
//
// GraphsEqual provides strict graph equality for GraphML round-trip tests. It
// is not the topology drift comparator; semantic topology comparison lives in
// pkg/diff and is re-exported by pkg/topology.
package graphml
