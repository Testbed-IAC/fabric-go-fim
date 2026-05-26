package graphml

import "errors"

// ErrInvalidGraphML indicates an input GraphML document is structurally invalid.
var ErrInvalidGraphML = errors.New("graphml: invalid graphml")
