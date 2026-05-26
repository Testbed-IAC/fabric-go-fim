package graphml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"sort"

	"github.com/Testbed-IAC/fabric-go-fim/internal/graph"
	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

const (
	graphMLNamespace = "http://graphml.graphdrawing.org/xmlns"
	xsiNamespace     = "http://www.w3.org/2001/XMLSchema-instance"
	schemaLocation   = "http://graphml.graphdrawing.org/xmlns http://graphml.graphdrawing.org/xmlns/1.0/graphml.xsd"
)

// Write serializes a graph to GraphML.
func Write(w io.Writer, g *graph.Graph) error {
	if g == nil {
		return fmt.Errorf("%w: graph is nil", ErrInvalidGraphML)
	}
	nodes := g.Nodes()
	edges := g.Edges()
	nodeKeys, edgeKeys := collectKeys(nodes, edges)
	doc := document{
		XMLNS:          graphMLNamespace,
		XSI:            xsiNamespace,
		SchemaLocation: schemaLocation,
		Graph:          graphXML{ID: "G", EdgeDefault: "directed"},
	}
	for _, name := range nodeKeys {
		doc.Keys = append(doc.Keys, keyXML{ID: name, For: "node", AttrName: name, AttrType: "string"})
	}
	for _, name := range edgeKeys {
		doc.Keys = append(doc.Keys, keyXML{ID: name, For: "edge", AttrName: name, AttrType: "string"})
	}
	xmlIDByNodeID := make(map[string]string, len(nodes))
	for index, node := range nodes {
		xmlID := fmt.Sprintf("n%d", index)
		xmlIDByNodeID[node.ID] = xmlID
		doc.Graph.Nodes = append(doc.Graph.Nodes, nodeToXML(node, xmlID, nodeKeys))
	}
	for index, edge := range edges {
		source, sourceOK := xmlIDByNodeID[edge.Source]
		target, targetOK := xmlIDByNodeID[edge.Target]
		if !sourceOK || !targetOK {
			return fmt.Errorf("%w: edge %d references missing endpoint %q -> %q", ErrInvalidGraphML, index, edge.Source, edge.Target)
		}
		doc.Graph.Edges = append(doc.Graph.Edges, edgeXML{
			ID:     fmt.Sprintf("e%d", index),
			Source: source,
			Target: target,
			Label:  edge.Class,
			Data: []dataXML{
				{Key: sliver.PropClass, Value: edge.Class},
				{Key: "label", Value: edge.Class},
			},
		})
	}
	return writeDocument(w, &doc)
}

func collectKeys(nodes []*graph.Node, edges []*graph.Edge) ([]string, []string) {
	nodeSet := map[string]struct{}{}
	for _, node := range nodes {
		nodeSet[sliver.PropXMLLabels] = struct{}{}
		for key := range node.Props {
			nodeSet[key] = struct{}{}
		}
	}
	edgeSet := map[string]struct{}{}
	if len(edges) > 0 {
		edgeSet[sliver.PropClass] = struct{}{}
		edgeSet["label"] = struct{}{}
	}
	return sortedKeys(nodeSet), sortedKeys(edgeSet)
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func nodeToXML(node *graph.Node, xmlID string, nodeKeys []string) nodeXML {
	label := ":GraphNode:" + node.Class
	dataByKey := make(map[string]string, len(node.Props)+1)
	dataByKey[sliver.PropXMLLabels] = label
	for key, value := range node.Props {
		if value != "" {
			dataByKey[key] = value
		}
	}
	out := nodeXML{ID: xmlID, Labels: label}
	for _, key := range nodeKeys {
		if value, ok := dataByKey[key]; ok {
			out.Data = append(out.Data, dataXML{Key: key, Value: value})
		}
	}
	return out
}

func writeDocument(w io.Writer, doc *document) error {
	var buffer bytes.Buffer
	buffer.WriteString(xml.Header)
	encoder := xml.NewEncoder(&buffer)
	encoder.Indent("", "  ")
	if err := encoder.Encode(doc); err != nil {
		return fmt.Errorf("graphml: encode: %w", err)
	}
	if err := encoder.Flush(); err != nil {
		return fmt.Errorf("graphml: flush: %w", err)
	}
	cleaned := bytes.ReplaceAll(buffer.Bytes(), []byte(` xmlns=""`), nil)
	if _, err := w.Write(cleaned); err != nil {
		return fmt.Errorf("graphml: write: %w", err)
	}
	return nil
}
