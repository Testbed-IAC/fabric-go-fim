package graphml

import (
	"reflect"

	"github.com/CSC478-WCU/fabric-go-fim/internal/graph"
)

// GraphsEqual reports whether two property graphs have the same nodes and edges.
func GraphsEqual(left, right *graph.Graph) bool {
	if left == nil || right == nil {
		return left == right
	}
	leftNodes := left.Nodes()
	rightNodes := right.Nodes()
	if len(leftNodes) != len(rightNodes) {
		return false
	}
	for index := range leftNodes {
		if leftNodes[index].ID != rightNodes[index].ID || leftNodes[index].Class != rightNodes[index].Class || !reflect.DeepEqual(leftNodes[index].Props, rightNodes[index].Props) {
			return false
		}
	}
	leftEdges := left.Edges()
	rightEdges := right.Edges()
	if len(leftEdges) != len(rightEdges) {
		return false
	}
	for index := range leftEdges {
		if *leftEdges[index] != *rightEdges[index] {
			return false
		}
	}
	return true
}
