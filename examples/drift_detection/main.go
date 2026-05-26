package main

import (
	"fmt"
	"log"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
)

func main() {
	desired := topology.NewWithID(topology.DeriveGraphID("example-drift"))
	if _, err := desired.AddNode(topology.NodeOpts{Name: "vm1", Site: "RENC"}); err != nil {
		log.Fatal(err)
	}

	actual := topology.NewWithID(topology.DeriveGraphID("example-drift"))
	if _, err := actual.AddNode(topology.NodeOpts{Name: "vm1", Site: "UKY"}); err != nil {
		log.Fatal(err)
	}

	diff := topology.DiffTopologies(desired, actual)
	fmt.Println(diff.Summary())
}
