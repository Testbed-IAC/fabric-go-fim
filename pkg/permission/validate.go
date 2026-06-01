package permission

import "strings"

// Request summarizes the FABRIC resources requested by a slice.
type Request struct {
	LifetimeHours int64
	Nodes         []Node
	Networks      []Network
}

// Node summarizes one requested compute node for permission evaluation.
type Node struct {
	Name       string
	Site       string
	Cores      int64
	RAM        int64
	Disk       int64
	Components []Component
}

// Component summarizes one requested node component for permission evaluation.
type Component struct {
	Type  string
	Model string
}

// Network summarizes one requested network service for permission evaluation.
type Network struct {
	Type      string
	Bandwidth int64
}

// Requirement describes one project permission tag required by a request.
type Requirement struct {
	Tag      string
	Subject  string
	Index    int
	SubIndex int
	Field    string
}

// RequiredTags returns all project tag requirements implied by req.
func RequiredTags(req Request) []Requirement {
	var out []Requirement
	if req.LifetimeHours > 24 {
		out = append(out, Requirement{Tag: TagSliceNoLimitLifetime, Subject: "lifetime", Field: "lifetime_hours"})
	}
	sites := map[string]bool{}
	for i, node := range req.Nodes {
		if node.Site != "" {
			sites[node.Site] = true
		}
		if node.Cores > 64 || node.RAM > 384 || node.Disk > 1000 {
			out = append(out, Requirement{Tag: TagVMNoLimit, Subject: "node", Index: i})
		}
		if node.Cores > 2 {
			out = append(out, Requirement{Tag: TagVMNoLimitCPU, Subject: "node", Index: i, Field: "cores"})
		}
		if node.RAM > 8 {
			out = append(out, Requirement{Tag: TagVMNoLimitRAM, Subject: "node", Index: i, Field: "ram"})
		}
		if node.Disk > 10 {
			out = append(out, Requirement{Tag: TagVMNoLimitDisk, Subject: "node", Index: i, Field: "disk"})
		}
		for j, component := range node.Components {
			switch component.Type {
			case "GPU":
				out = append(out, Requirement{Tag: TagComponentGPU, Subject: "component", Index: i, SubIndex: j, Field: "type"})
			case "FPGA":
				out = append(out, Requirement{Tag: TagComponentFPGA, Subject: "component", Index: i, SubIndex: j, Field: "type"})
			case "NVME":
				out = append(out, Requirement{Tag: TagComponentNVME, Subject: "component", Index: i, SubIndex: j, Field: "type"})
			case "Storage":
				out = append(out, Requirement{Tag: TagComponentStorage, Subject: "component", Index: i, SubIndex: j, Field: "type"})
			case "SmartNIC":
				out = append(out, Requirement{Tag: TagForSmartNICModel(component.Model), Subject: "component", Index: i, SubIndex: j, Field: "model"})
			}
		}
	}
	if len(sites) > 1 {
		out = append(out, Requirement{Tag: TagSliceMultisite, Subject: "node"})
	}
	for i, network := range req.Networks {
		if network.Bandwidth > 10000 {
			out = append(out, Requirement{Tag: TagNetNoLimitBW, Subject: "network", Index: i, Field: "bandwidth"})
		}
		switch network.Type {
		case "FABNetv4Ext":
			out = append(out, Requirement{Tag: TagNetFABNetv4Ext, Subject: "network", Index: i, Field: "type"})
		case "FABNetv6Ext":
			out = append(out, Requirement{Tag: TagNetFABNetv6Ext, Subject: "network", Index: i, Field: "type"})
		case "PortMirror":
			out = append(out, Requirement{Tag: TagNetPortMirroring, Subject: "network", Index: i, Field: "type"})
		}
	}
	return out
}

// Missing returns the requirements whose tags are absent from projectTags.
func Missing(req Request, projectTags []string) []Requirement {
	have := make(map[string]bool, len(projectTags))
	for _, tag := range projectTags {
		have[tag] = true
	}
	var out []Requirement
	for _, requirement := range RequiredTags(req) {
		if requirement.Tag != "" && !have[requirement.Tag] {
			out = append(out, requirement)
		}
	}
	return out
}

// TagForSmartNICModel returns the permission tag required by a SmartNIC model.
func TagForSmartNICModel(model string) string {
	normalized := strings.ReplaceAll(model, "-", "_")
	switch normalized {
	case "ConnectX_5":
		return TagComponentSmartNICConnectX5
	case "ConnectX_6":
		return TagComponentSmartNICConnectX6
	case "BlueField_2_ConnectX_6":
		return TagComponentSmartNICBlueField2ConnectX6
	case "ConnectX_7_100":
		return TagComponentSmartNICConnectX7100
	case "ConnectX_7_400":
		return TagComponentSmartNICConnectX7400
	default:
		return TagComponentSmartNICConnectX6
	}
}
