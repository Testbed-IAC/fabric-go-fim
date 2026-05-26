package graph

import (
	"fmt"
	"iter"
	"sync"
)

// Direction filters neighbor traversal by edge direction.
type Direction int

// Direction values for neighbor traversal.
const (
	Outgoing Direction = iota
	Incoming
	Both
)

// Graph is a thread-safe directed graph keyed by stable node UUIDs.
type Graph struct {
	mu    sync.RWMutex
	ID    string
	nodes map[string]*Node
	order []string
	edges []*Edge
}

// Node is a graph node with string-valued properties.
type Node struct {
	ID    string
	Class string
	Props map[string]string
}

// Edge is a directed graph edge with a class string.
type Edge struct {
	Class  string
	Source string
	Target string
}

// New creates an empty graph with the provided GraphID.
func New(graphID string) *Graph {
	return &Graph{ID: graphID, nodes: make(map[string]*Node)}
}

// AddNode adds a node and copies its property map.
func (g *Graph) AddNode(class, nodeID string, props map[string]string) error {
	if nodeID == "" {
		return fmt.Errorf("%w: node ID is required", ErrInvalidNode)
	}
	if class == "" {
		return fmt.Errorf("%w: class is required for node %q", ErrInvalidNode, nodeID)
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.nodes[nodeID]; exists {
		return fmt.Errorf("%w: %q", ErrDuplicateNode, nodeID)
	}
	g.nodes[nodeID] = &Node{ID: nodeID, Class: class, Props: cloneMap(props)}
	g.order = append(g.order, nodeID)
	return nil
}

// AddEdge adds a directed edge between existing nodes.
func (g *Graph) AddEdge(class, fromNodeID, toNodeID string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, exists := g.nodes[fromNodeID]; !exists {
		return fmt.Errorf("%w: edge source %q", ErrMissingNode, fromNodeID)
	}
	if _, exists := g.nodes[toNodeID]; !exists {
		return fmt.Errorf("%w: edge target %q", ErrMissingNode, toNodeID)
	}
	g.edges = append(g.edges, &Edge{Class: class, Source: fromNodeID, Target: toNodeID})
	return nil
}

// Node returns a copy of the node with the given stable ID.
func (g *Graph) Node(nodeID string) (*Node, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	node, exists := g.nodes[nodeID]
	if !exists {
		return nil, false
	}
	return cloneNode(node), true
}

// UpdateNode replaces an existing node property map.
func (g *Graph) UpdateNode(nodeID string, props map[string]string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	node, exists := g.nodes[nodeID]
	if !exists {
		return fmt.Errorf("%w: %q", ErrMissingNode, nodeID)
	}
	node.Props = cloneMap(props)
	if class := props["Class"]; class != "" {
		node.Class = class
	}
	return nil
}

// DeleteNode removes a node and all incident edges.
func (g *Graph) DeleteNode(nodeID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.nodes, nodeID)
	for index, id := range g.order {
		if id == nodeID {
			g.order = append(g.order[:index], g.order[index+1:]...)
			break
		}
	}
	edges := g.edges[:0]
	for _, edge := range g.edges {
		if edge.Source != nodeID && edge.Target != nodeID {
			edges = append(edges, edge)
		}
	}
	g.edges = edges
}

// DeleteEdge removes the first edge matching class and endpoints.
func (g *Graph) DeleteEdge(class, fromNodeID, toNodeID string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for index, edge := range g.edges {
		if edge.Class == class && edge.Source == fromNodeID && edge.Target == toNodeID {
			g.edges = append(g.edges[:index], g.edges[index+1:]...)
			return true
		}
	}
	return false
}

// Neighbors returns copied neighboring nodes that match the edge class and direction.
func (g *Graph) Neighbors(nodeID, edgeClass string, dir Direction) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var out []*Node
	for _, edge := range g.edges {
		if edgeClass != "" && edge.Class != edgeClass {
			continue
		}
		if (dir == Outgoing || dir == Both) && edge.Source == nodeID {
			if node := g.nodes[edge.Target]; node != nil {
				out = append(out, cloneNode(node))
			}
		}
		if (dir == Incoming || dir == Both) && edge.Target == nodeID {
			if node := g.nodes[edge.Source]; node != nil {
				out = append(out, cloneNode(node))
			}
		}
	}
	return out
}

// Nodes returns graph nodes in insertion order.
func (g *Graph) Nodes() []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Node, 0, len(g.order))
	for _, id := range g.order {
		if node := g.nodes[id]; node != nil {
			out = append(out, cloneNode(node))
		}
	}
	return out
}

// AllNodes returns an iterator over graph nodes in insertion order.
func (g *Graph) AllNodes() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		for _, node := range g.Nodes() {
			if !yield(node) {
				return
			}
		}
	}
}

// Edges returns graph edges in insertion order.
func (g *Graph) Edges() []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Edge, 0, len(g.edges))
	for _, edge := range g.edges {
		out = append(out, cloneEdge(edge))
	}
	return out
}

// AllEdges returns an iterator over graph edges in insertion order.
func (g *Graph) AllEdges() iter.Seq[*Edge] {
	return func(yield func(*Edge) bool) {
		for _, edge := range g.Edges() {
			if !yield(edge) {
				return
			}
		}
	}
}

func cloneNode(node *Node) *Node {
	return &Node{ID: node.ID, Class: node.Class, Props: cloneMap(node.Props)}
}

func cloneEdge(edge *Edge) *Edge {
	return &Edge{Class: edge.Class, Source: edge.Source, Target: edge.Target}
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
