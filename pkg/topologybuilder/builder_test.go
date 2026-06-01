package topologybuilder

import (
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/topology"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/userdata"
)

func TestBuildBareVM(t *testing.T) {
	t.Parallel()
	topo, graphML, err := Build(SliceSpec{
		Name: "test-slice",
		Nodes: []NodeSpec{{
			Name:   "vm1",
			Site:   "RENC",
			Routes: []userdata.Route{{Subnet: "10.0.0.0/24", NextHop: "10.0.0.1"}},
		}},
	})
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if _, err := topology.Load(strings.NewReader(graphML)); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	node, ok := topo.Node("vm1")
	if !ok {
		t.Fatal("missing node vm1")
	}
	sl, err := node.Sliver()
	if err != nil {
		t.Fatalf("Sliver returned error: %v", err)
	}
	if sl.Capacities.Core != 2 || sl.Capacities.RAM != 8 || sl.Capacities.Disk != 10 {
		t.Fatalf("capacities = %+v, want defaults", sl.Capacities)
	}
	if len(sl.UserData) == 0 {
		t.Fatal("expected user-data from route")
	}
}

func TestNormalizeMirrorDirection(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   sliver.MirrorDirection
		want sliver.MirrorDirection
	}{
		{in: "", want: ""},
		{in: "both", want: sliver.MirrorBoth},
		{in: "Both", want: sliver.MirrorBoth},
		{in: "rx", want: sliver.MirrorRXOnly},
		{in: "RX_Only", want: sliver.MirrorRXOnly},
		{in: "tx", want: sliver.MirrorTXOnly},
		{in: "TX_Only", want: sliver.MirrorTXOnly},
	}
	for _, tc := range cases {
		if got := NormalizeMirrorDirection(tc.in); got != tc.want {
			t.Fatalf("NormalizeMirrorDirection(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
