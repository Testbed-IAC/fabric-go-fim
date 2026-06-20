package topologybuilder

import (
	"testing"
)

func modifySpec() SliceSpec {
	return SliceSpec{
		Name: "modify-slice",
		Nodes: []NodeSpec{
			{Name: "vm1", Site: "RENC", Cores: 2, RAM: 8, Disk: 10},
			{Name: "vm2", Site: "RENC", Cores: 2, RAM: 8, Disk: 10},
		},
	}
}

// With no existing model, BuildForModify is identical to Build.
func TestBuildForModify_EmptyModelMatchesBuild(t *testing.T) {
	spec := modifySpec()
	_, want, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := BuildForModify(spec, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("BuildForModify with empty existing model changed the GraphML")
	}
	if _, _, err := BuildForModify(spec, "   "); err != nil {
		t.Fatalf("BuildForModify with blank existing model: %v", err)
	}
}

// An existing model that carries no reservation info leaves the build unchanged.
func TestBuildForModify_NoReservationInfoSourceUnchanged(t *testing.T) {
	spec := modifySpec()
	_, existingModel, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, want, err := Build(spec)
	if err != nil {
		t.Fatal(err)
	}
	_, got, err := BuildForModify(spec, existingModel)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("BuildForModify changed GraphML even though the source had no reservation info")
	}
}

func TestBuildForModify_InvalidModel(t *testing.T) {
	if _, _, err := BuildForModify(modifySpec(), "<graphml>not-closed"); err == nil {
		t.Fatal("expected an error loading a malformed existing model")
	}
}
