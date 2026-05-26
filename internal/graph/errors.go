package graph

import "errors"

// ErrDuplicateNode indicates a node with the same ID already exists.
var ErrDuplicateNode = errors.New("graph: duplicate node")

// ErrMissingNode indicates a referenced node does not exist.
var ErrMissingNode = errors.New("graph: missing node")

// ErrInvalidNode indicates a node is missing required graph-layer fields.
var ErrInvalidNode = errors.New("graph: invalid node")
