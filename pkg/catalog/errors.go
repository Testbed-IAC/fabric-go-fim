package catalog

import "errors"

// ErrCatalogLoad indicates an embedded catalog could not be decoded.
var ErrCatalogLoad = errors.New("catalog: load failed")

// ErrNotFound indicates a catalog lookup did not match any entry.
var ErrNotFound = errors.New("catalog: entry not found")
