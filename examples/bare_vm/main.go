package main

import (
	"fmt"
	"log"

	"github.com/CSC478-WCU/fabric-go-fim/pkg/sliver"
	"github.com/CSC478-WCU/fabric-go-fim/pkg/topology"
)

func main() {
	topo := topology.NewWithID(topology.DeriveGraphID("example-bare-vm"))

	_, err := topo.AddNode(topology.NodeOpts{
		Name:       "vm1",
		Site:       "RENC",
		Type:       sliver.NodeTypeVM,
		Capacities: &sliver.Capacities{Core: 2, RAM: 8, Disk: 10},
		ImageRef:   "default_rocky_9",
		ImageType:  "qcow2",
	})
	if err != nil {
		log.Fatal(err)
	}

	graphML, err := topo.SerializeString()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(graphML)
}
