package graph

import (
	"errors"
	"testing"
)

func TestGraphOperations(t *testing.T) {
	g := New("graph-id")
	if err := g.AddNode("A", "n1", map[string]string{"Name": "one"}); err != nil {
		t.Fatalf("AddNode n1: %v", err)
	}
	if err := g.AddNode("B", "n2", map[string]string{"Name": "two"}); err != nil {
		t.Fatalf("AddNode n2: %v", err)
	}
	if err := g.AddEdge("connects", "n1", "n2"); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}

	tests := []struct {
		name string
		dir  Direction
		want string
	}{
		{name: "outgoing", dir: Outgoing, want: "n2"},
		{name: "incoming", dir: Incoming, want: ""},
		{name: "both", dir: Both, want: "n2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := g.Neighbors("n1", "connects", tt.dir)
			if tt.want == "" && len(got) != 0 {
				t.Fatalf("got %d neighbors, want none", len(got))
			}
			if tt.want != "" && (len(got) != 1 || got[0].ID != tt.want) {
				t.Fatalf("got %#v, want one neighbor %q", got, tt.want)
			}
		})
	}
}

func TestGraphErrorsAndDelete(t *testing.T) {
	g := New("graph-id")
	if err := g.AddNode("A", "n1", nil); err != nil {
		t.Fatalf("AddNode: %v", err)
	}
	if err := g.AddNode("A", "n1", nil); !errors.Is(err, ErrDuplicateNode) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateNode", err)
	}
	if err := g.AddEdge("has", "n1", "missing"); !errors.Is(err, ErrMissingNode) {
		t.Fatalf("missing target error = %v, want ErrMissingNode", err)
	}
	if err := g.AddNode("B", "n2", nil); err != nil {
		t.Fatalf("AddNode n2: %v", err)
	}
	if err := g.AddEdge("has", "n1", "n2"); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	g.DeleteNode("n2")
	if _, ok := g.Node("n2"); ok {
		t.Fatalf("deleted node is still present")
	}
	if got := g.Edges(); len(got) != 0 {
		t.Fatalf("incident edges = %d, want 0", len(got))
	}
}
