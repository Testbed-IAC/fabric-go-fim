package sliver

import (
	"errors"
	"testing"
)

func TestSliverPropsRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		props func() (map[string]string, error)
		read  func(map[string]string) error
	}{
		{
			name: "network node",
			props: func() (map[string]string, error) {
				return (&NodeSliver{BaseSliver: BaseSliver{NodeID: "node-id", GraphID: "graph-id", Name: "vm1", Capacities: &Capacities{Core: 2, RAM: 8, Disk: 10}}, Type: NodeTypeVM, Site: "RENC", ImageRef: "default_rocky_9", ImageType: "qcow2"}).ToProps()
			},
			read: func(props map[string]string) error {
				var got NodeSliver
				if err := got.FromProps(props); err != nil {
					return err
				}
				if got.Name != "vm1" || got.Type != NodeTypeVM || got.Capacities.Core != 2 {
					t.Fatalf("unexpected node sliver: %#v", got)
				}
				return nil
			},
		},
		{
			name: "network service",
			props: func() (map[string]string, error) {
				return (&NetworkServiceSliver{BaseSliver: BaseSliver{NodeID: "svc-id", GraphID: "graph-id", Name: "lan1"}, Type: ServiceTypeL2Bridge, Layer: LayerL2, Site: "RENC"}).ToProps()
			},
			read: func(props map[string]string) error {
				var got NetworkServiceSliver
				if err := got.FromProps(props); err != nil {
					return err
				}
				if got.Type != ServiceTypeL2Bridge || got.Layer != LayerL2 {
					t.Fatalf("unexpected service sliver: %#v", got)
				}
				return nil
			},
		},
		{
			name: "component",
			props: func() (map[string]string, error) {
				return (&ComponentSliver{BaseSliver: BaseSliver{NodeID: "comp-id", GraphID: "graph-id", Name: "vm1-gpu1", Capacities: &Capacities{Unit: 1}}, Type: ComponentTypeGPU, Model: "RTX6000"}).ToProps()
			},
			read: func(props map[string]string) error {
				var got ComponentSliver
				if err := got.FromProps(props); err != nil {
					return err
				}
				if got.Model != "RTX6000" || got.Type != ComponentTypeGPU {
					t.Fatalf("unexpected component sliver: %#v", got)
				}
				return nil
			},
		},
		{
			name: "interface",
			props: func() (map[string]string, error) {
				return (&InterfaceSliver{BaseSliver: BaseSliver{NodeID: "if-id", GraphID: "graph-id", Name: "vm1-snic-p1", Labels: &Labels{LocalName: "p1"}, Capacities: &Capacities{Unit: 1, BW: 100}}, Type: InterfaceTypeDedicatedPort}).ToProps()
			},
			read: func(props map[string]string) error {
				var got InterfaceSliver
				if err := got.FromProps(props); err != nil {
					return err
				}
				if got.Type != InterfaceTypeDedicatedPort || got.Labels.LocalName != "p1" {
					t.Fatalf("unexpected interface sliver: %#v", got)
				}
				return nil
			},
		},
		{
			name: "link",
			props: func() (map[string]string, error) {
				return (&LinkSliver{BaseSliver: BaseSliver{NodeID: "link-id", GraphID: "graph-id", Name: "link1"}, Type: LinkTypePatch, Layer: LayerL2}).ToProps()
			},
			read: func(props map[string]string) error {
				var got LinkSliver
				if err := got.FromProps(props); err != nil {
					return err
				}
				if got.Type != LinkTypePatch || got.Layer != LayerL2 {
					t.Fatalf("unexpected link sliver: %#v", got)
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props, err := tt.props()
			if err != nil {
				t.Fatalf("ToProps: %v", err)
			}
			if err := tt.read(props); err != nil {
				t.Fatalf("FromProps: %v", err)
			}
		})
	}
}

func TestValidation(t *testing.T) {
	if err := (Capacities{Core: -1}).Validate(); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("negative capacity error = %v, want ErrInvalidValue", err)
	}
	if err := (Labels{VLAN: "5000"}).Validate(); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("invalid VLAN error = %v, want ErrInvalidValue", err)
	}
	if err := (Tags{"ok", "not ok"}).Validate(); !errors.Is(err, ErrInvalidValue) {
		t.Fatalf("invalid tag error = %v, want ErrInvalidValue", err)
	}
}
