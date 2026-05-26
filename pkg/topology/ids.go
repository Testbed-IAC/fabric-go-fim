package topology

import (
	"strings"

	"github.com/google/uuid"
)

var rootNamespace = uuid.UUID{0x0d, 0x82, 0xbf, 0xc4, 0xb1, 0xf7, 0x42, 0xb6, 0x9b, 0x9c, 0x36, 0xc8, 0xec, 0xfe, 0x78, 0x1d}

var (
	namespaceTopology  = uuid.NewSHA1(rootNamespace, []byte("topology"))
	namespaceNode      = uuid.NewSHA1(rootNamespace, []byte("node"))
	namespaceComponent = uuid.NewSHA1(rootNamespace, []byte("component"))
	namespaceService   = uuid.NewSHA1(rootNamespace, []byte("service"))
	namespaceInterface = uuid.NewSHA1(rootNamespace, []byte("interface"))
	namespaceLink      = uuid.NewSHA1(rootNamespace, []byte("link"))
)

// DeriveGraphID returns a deterministic topology GraphID from a stable key.
func DeriveGraphID(stableKey string) string {
	return deriveID(namespaceTopology, stableKey)
}

func deriveID(namespace uuid.UUID, parts ...string) string {
	return uuid.NewSHA1(namespace, []byte(strings.Join(parts, "\x1f"))).String()
}

func chooseID(provided string, namespace uuid.UUID, parts ...string) string {
	if provided != "" {
		return provided
	}
	return deriveID(namespace, parts...)
}
