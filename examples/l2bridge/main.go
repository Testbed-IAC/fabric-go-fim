package main

import (
	"fmt"
	"log"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/catalog"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
)

func main() {
	topo := topology.NewWithID(topology.DeriveGraphID("example-l2bridge"))

	ifaces := make([]*topology.Interface, 0, 2)
	for _, name := range []string{"vm1", "vm2"} {
		vm, err := topo.AddNode(topology.NodeOpts{Name: name, Site: "RENC"})
		if err != nil {
			log.Fatal(err)
		}
		nic, err := vm.AddComponent(topology.ComponentOpts{Name: "nic1", FABlibName: catalog.FABlibNICBasic})
		if err != nil {
			log.Fatal(err)
		}
		ifaces = append(ifaces, nic.Interfaces()[0])
	}

	_, err := topo.AddNetworkService(topology.NetworkServiceOpts{
		Name:       "lan1",
		Type:       sliver.ServiceTypeL2Bridge,
		Interfaces: ifaces,
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
