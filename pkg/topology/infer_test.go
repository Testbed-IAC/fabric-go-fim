package topology

import (
	"errors"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// TestInferServiceType drives InferServiceType from real topologies so the
// inferred type reflects genuine interface sites and types, matching FABlib's
// __calculate_l2_nstype across every branch.
func TestInferServiceType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		interfaces func(*testing.T, *Topology) []*Interface
		ero        bool
		want       sliver.ServiceType
		wantErr    error
	}{
		{
			name:       "single site shared NICs -> L2Bridge",
			interfaces: func(t *testing.T, topo *Topology) []*Interface { return twoSharedInterfaces(t, topo, "RENC", "RENC") },
			want:       sliver.ServiceTypeL2Bridge,
		},
		{
			name: "single dedicated port one site -> L2Bridge",
			interfaces: func(t *testing.T, topo *Topology) []*Interface {
				return []*Interface{dedicatedInterface(t, topo, "vm1", "RENC")}
			},
			want: sliver.ServiceTypeL2Bridge,
		},
		{
			name:       "two-site basic NICs -> L2STS",
			interfaces: func(t *testing.T, topo *Topology) []*Interface { return twoSharedInterfaces(t, topo, "RENC", "UKY") },
			want:       sliver.ServiceTypeL2STS,
		},
		{
			name:       "two-site dedicated, no ERO -> L2STS",
			interfaces: func(t *testing.T, topo *Topology) []*Interface { return twoDedicatedInterfaces(t, topo, "RENC", "UKY") },
			want:       sliver.ServiceTypeL2STS,
		},
		{
			name:       "two-site dedicated, ERO enabled -> L2PTP",
			interfaces: func(t *testing.T, topo *Topology) []*Interface { return twoDedicatedInterfaces(t, topo, "RENC", "UKY") },
			ero:        true,
			want:       sliver.ServiceTypeL2PTP,
		},
		{
			name: "facility port + dedicated, two sites -> L2PTP",
			interfaces: func(t *testing.T, topo *Topology) []*Interface {
				return []*Interface{
					facilityPort(t, topo, "ESnet-DTN", "RENC"),
					dedicatedInterface(t, topo, "vm1", "UKY"),
				}
			},
			want: sliver.ServiceTypeL2PTP,
		},
		{
			name: "two facility ports, two sites -> L2STS",
			interfaces: func(t *testing.T, topo *Topology) []*Interface {
				return []*Interface{
					facilityPort(t, topo, "ESnet-A", "RENC"),
					facilityPort(t, topo, "ESnet-B", "UKY"),
				}
			},
			want: sliver.ServiceTypeL2STS,
		},
		{
			name: "facility port + basic NIC, two sites -> L2STS",
			interfaces: func(t *testing.T, topo *Topology) []*Interface {
				return []*Interface{
					facilityPort(t, topo, "ESnet-DTN", "RENC"),
					sharedInterface(t, topo, "vm1", "UKY"),
				}
			},
			want: sliver.ServiceTypeL2STS,
		},
		{
			name: "three sites -> error",
			interfaces: func(t *testing.T, topo *Topology) []*Interface {
				return []*Interface{
					sharedInterface(t, topo, "vm1", "RENC"),
					sharedInterface(t, topo, "vm2", "UKY"),
					sharedInterface(t, topo, "vm3", "STAR"),
				}
			},
			wantErr: ErrConstraintViolation,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			topo := NewWithID(DeriveGraphID(tt.name))
			ifaces := tt.interfaces(t, topo)
			got, err := topo.InferServiceType(ifaces, tt.ero)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("InferServiceType error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("InferServiceType returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("InferServiceType = %s, want %s", got, tt.want)
			}
		})
	}
}

func dedicatedInterface(t *testing.T, topo *Topology, vmName, site string) *Interface {
	t.Helper()
	vm := addVM(t, topo, vmName, site)
	c := addComponent(t, vm, ComponentOpts{Name: "snic1", Type: sliver.ComponentTypeSmartNIC, Model: "ConnectX-6"})
	return c.InterfaceList()[0]
}

func sharedInterface(t *testing.T, topo *Topology, vmName, site string) *Interface {
	t.Helper()
	vm := addVM(t, topo, vmName, site)
	c := addComponent(t, vm, ComponentOpts{Name: "nic1", Type: sliver.ComponentTypeSharedNIC, Model: "ConnectX-6"})
	return c.InterfaceList()[0]
}

func facilityPort(t *testing.T, topo *Topology, name, site string) *Interface {
	t.Helper()
	node, err := topo.AddFacility(FacilityOpts{Name: name, Site: site, Labels: &sliver.Labels{VLAN: "100"}, Capacities: &sliver.Capacities{BW: 10}})
	if err != nil {
		t.Fatalf("AddFacility: %v", err)
	}
	ifaces := node.InterfaceList()
	if len(ifaces) == 0 {
		t.Fatalf("facility %q has no interfaces", name)
	}
	return ifaces[0]
}
