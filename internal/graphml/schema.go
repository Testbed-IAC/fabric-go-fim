package graphml

import "encoding/xml"

type document struct {
	XMLName        xml.Name `xml:"graphml"`
	XMLNS          string   `xml:"xmlns,attr,omitempty"`
	XSI            string   `xml:"xmlns:xsi,attr,omitempty"`
	SchemaLocation string   `xml:"xsi:schemaLocation,attr,omitempty"`
	Keys           []keyXML `xml:"key"`
	Graph          graphXML `xml:"graph"`
}

type keyXML struct {
	ID       string `xml:"id,attr"`
	For      string `xml:"for,attr"`
	AttrName string `xml:"attr.name,attr"`
	AttrType string `xml:"attr.type,attr"`
}

type graphXML struct {
	ID          string    `xml:"id,attr,omitempty"`
	EdgeDefault string    `xml:"edgedefault,attr"`
	Nodes       []nodeXML `xml:"node"`
	Edges       []edgeXML `xml:"edge"`
}

type nodeXML struct {
	ID     string    `xml:"id,attr"`
	Labels string    `xml:"labels,attr,omitempty"`
	Data   []dataXML `xml:"data"`
}

type edgeXML struct {
	ID     string    `xml:"id,attr,omitempty"`
	Source string    `xml:"source,attr"`
	Target string    `xml:"target,attr"`
	Label  string    `xml:"label,attr,omitempty"`
	Data   []dataXML `xml:"data"`
}

type dataXML struct {
	Key   string `xml:"key,attr"`
	Value string `xml:",chardata"`
}
