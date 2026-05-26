package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
)

func main() {
	desired := topology.NewWithID(topology.DeriveGraphID("example-serialize-load"))
	if _, err := desired.AddNode(topology.NodeOpts{Name: "vm1", Site: "RENC"}); err != nil {
		log.Fatal(err)
	}

	graphML, err := desired.SerializeString()
	if err != nil {
		log.Fatal(err)
	}

	loaded, err := topology.Load(strings.NewReader(graphML))
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(loaded.GraphID())
}
