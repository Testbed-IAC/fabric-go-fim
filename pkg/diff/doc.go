// Package diff provides semantic normalization and drift comparison for FABRIC
// topology graphs.
//
// Diffing is based on topology intent. Generated IDs, GraphML-local ordering,
// JSON key ordering, and runtime-only FABRIC fields are ignored so callers do
// not get noisy drift reports for broker observation state.
package diff
