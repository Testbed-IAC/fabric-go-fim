package graphml

import (
	"encoding/xml"
	"fmt"
	"io"

	"github.com/CSC478-WCU/fabric-go-fim/internal/graph"
	"github.com/CSC478-WCU/fabric-go-fim/pkg/sliver"
)

// Read parses a GraphML document into a property graph.
func Read(r io.Reader) (*graph.Graph, error) {
	var doc document
	if err := xml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("graphml: parse: %w", err)
	}
	keyNames := make(map[string]map[string]string)
	for _, key := range doc.Keys {
		if keyNames[key.For] == nil {
			keyNames[key.For] = make(map[string]string)
		}
		name := key.AttrName
		if name == "" {
			name = key.ID
		}
		keyNames[key.For][key.ID] = name
	}
	xmlIDToNodeID := make(map[string]string, len(doc.Graph.Nodes))
	var g *graph.Graph
	for _, node := range doc.Graph.Nodes {
		props := resolveData(keyNames["node"], node.Data)
		delete(props, sliver.PropXMLLabels)
		class := props[sliver.PropClass]
		if class == "" {
			return nil, fmt.Errorf("%w: node %q has no Class property", ErrInvalidGraphML, node.ID)
		}
		nodeID := props[sliver.PropNodeID]
		if nodeID == "" {
			return nil, fmt.Errorf("%w: node %q has no NodeID property", ErrInvalidGraphML, node.ID)
		}
		if props[sliver.PropName] == "" {
			return nil, fmt.Errorf("%w: node %q has no Name property", ErrInvalidGraphML, node.ID)
		}
		if props[sliver.PropType] == "" {
			return nil, fmt.Errorf("%w: node %q has no Type property", ErrInvalidGraphML, node.ID)
		}
		graphID := props[sliver.PropGraphID]
		if graphID == "" {
			return nil, fmt.Errorf("%w: node %q has no GraphID property", ErrInvalidGraphML, node.ID)
		}
		if g == nil {
			g = graph.New(graphID)
		}
		if err := g.AddNode(class, nodeID, props); err != nil {
			return nil, fmt.Errorf("graphml: node %q: %w", node.ID, err)
		}
		xmlIDToNodeID[node.ID] = nodeID
	}
	if g == nil {
		g = graph.New("")
	}
	for _, edge := range doc.Graph.Edges {
		props := resolveData(keyNames["edge"], edge.Data)
		class := props[sliver.PropClass]
		if class == "" {
			class = edge.Label
		}
		if class == "" {
			class = props["label"]
		}
		if class == "" {
			return nil, fmt.Errorf("%w: edge %q has no Class or label", ErrInvalidGraphML, edge.ID)
		}
		source := xmlIDToNodeID[edge.Source]
		target := xmlIDToNodeID[edge.Target]
		if source == "" || target == "" {
			return nil, fmt.Errorf("%w: edge %q references missing endpoint %q -> %q", ErrInvalidGraphML, edge.ID, edge.Source, edge.Target)
		}
		if err := g.AddEdge(class, source, target); err != nil {
			return nil, fmt.Errorf("graphml: edge %q: %w", edge.ID, err)
		}
	}
	return g, nil
}

func resolveData(keys map[string]string, values []dataXML) map[string]string {
	props := make(map[string]string, len(values))
	for _, value := range values {
		name := value.Key
		if resolved := keys[value.Key]; resolved != "" {
			name = resolved
		}
		if value.Value != "None" {
			props[name] = value.Value
		}
	}
	return props
}
