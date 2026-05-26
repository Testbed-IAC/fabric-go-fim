package catalog

import _ "embed"

//go:embed data/instances.json
var instanceSizesJSON []byte

//go:embed data/components.json
var componentCatalogJSON []byte
