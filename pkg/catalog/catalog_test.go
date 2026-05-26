package catalog

import (
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

func TestInstanceCatalog(t *testing.T) {
	instances, err := Instances()
	if err != nil {
		t.Fatalf("Instances: %v", err)
	}
	if err := instances.Validate(); err != nil {
		t.Fatalf("Validate instances: %v", err)
	}
	if _, ok := instances.Lookup("fabric.c2.m8.d10"); !ok {
		t.Fatalf("expected fabric.c2.m8.d10 in instance catalog")
	}
	if got := instances.MapCapacitiesToInstance(sliver.Capacities{Core: 2, RAM: 8, Disk: 10}); got == "" {
		t.Fatalf("MapCapacitiesToInstance returned empty")
	}
}

func TestComponentCatalogGenerate(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	if err := components.Validate(); err != nil {
		t.Fatalf("Validate components: %v", err)
	}
	tests := []struct {
		name       string
		ctype      sliver.ComponentType
		model      string
		wantIfaces int
		wantSvc    bool
	}{
		{name: "shared nic", ctype: sliver.ComponentTypeSharedNIC, model: "ConnectX-6", wantIfaces: 1, wantSvc: true},
		{name: "smart nic", ctype: sliver.ComponentTypeSmartNIC, model: "ConnectX-6", wantIfaces: 2, wantSvc: true},
		{name: "gpu", ctype: sliver.ComponentTypeGPU, model: "RTX6000", wantIfaces: 0, wantSvc: false},
		{name: "nvme", ctype: sliver.ComponentTypeNVME, model: "P4510", wantIfaces: 0, wantSvc: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := components.Generate(GenerateOpts{ParentNodeName: "vm1", ChildName: "dev", Type: tt.ctype, Model: tt.model, GraphID: "graph-id"})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if len(got.Interfaces) != tt.wantIfaces {
				t.Fatalf("interfaces = %d, want %d", len(got.Interfaces), tt.wantIfaces)
			}
			if (got.Service != nil) != tt.wantSvc {
				t.Fatalf("service presence = %v, want %v", got.Service != nil, tt.wantSvc)
			}
		})
	}
}

func TestResolveFABlibModel(t *testing.T) {
	tests := []string{FABlibNICBasic, FABlibNICConnectX6, FABlibGPUTeslaT4, FABlibNVMEP4510, FABlibFPGAXilinxU280}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, ok := ResolveFABlibModel(name); !ok {
				t.Fatalf("ResolveFABlibModel(%q) failed", name)
			}
		})
	}
}
