// Package permission models FABRIC project permission tags required by slice
// requests.
package permission

const (
	// TagVMNoLimitCPU allows VM CPU requests above the default limit.
	TagVMNoLimitCPU = "VM.NoLimitCPU"
	// TagVMNoLimitRAM allows VM RAM requests above the default limit.
	TagVMNoLimitRAM = "VM.NoLimitRAM"
	// TagVMNoLimitDisk allows VM disk requests above the default limit.
	TagVMNoLimitDisk = "VM.NoLimitDisk"
	// TagVMNoLimit allows aggregate VM requests above the default limits.
	TagVMNoLimit = "VM.NoLimit"
	// TagSliceNoLimitLifetime allows slice lifetimes longer than the default limit.
	TagSliceNoLimitLifetime = "Slice.NoLimitLifetime"
	// TagSliceMultisite allows slices spanning multiple FABRIC sites.
	TagSliceMultisite = "Slice.Multisite"
	// TagComponentGPU allows GPU components.
	TagComponentGPU = "Component.GPU"
	// TagComponentFPGA allows FPGA components.
	TagComponentFPGA = "Component.FPGA"
	// TagComponentSmartNICConnectX5 allows ConnectX-5 SmartNIC components.
	TagComponentSmartNICConnectX5 = "Component.SmartNIC_ConnectX_5"
	// TagComponentSmartNICConnectX6 allows ConnectX-6 SmartNIC components.
	TagComponentSmartNICConnectX6 = "Component.SmartNIC_ConnectX_6"
	// TagComponentSmartNICBlueField2ConnectX6 allows BlueField-2 ConnectX-6 SmartNIC components.
	TagComponentSmartNICBlueField2ConnectX6 = "Component.SmartNIC_BlueField2_ConnectX_6"
	// TagComponentSmartNICConnectX7100 allows ConnectX-7 100G SmartNIC components.
	TagComponentSmartNICConnectX7100 = "Component.SmartNIC_ConnectX_7_100"
	// TagComponentSmartNICConnectX7400 allows ConnectX-7 400G SmartNIC components.
	TagComponentSmartNICConnectX7400 = "Component.SmartNIC_ConnectX_7_400"
	// TagComponentStorage allows storage components.
	TagComponentStorage = "Component.Storage"
	// TagComponentNVME allows NVMe components.
	TagComponentNVME = "Component.NVME"
	// TagNetNoLimitBW allows network bandwidth above the default limit.
	TagNetNoLimitBW = "Net.NoLimitBW"
	// TagNetFABNetv4Ext allows external FABNetv4 services.
	TagNetFABNetv4Ext = "Net.FABNetv4Ext"
	// TagNetFABNetv6Ext allows external FABNetv6 services.
	TagNetFABNetv6Ext = "Net.FABNetv6Ext"
	// TagNetPortMirroring allows port mirror services.
	TagNetPortMirroring = "Net.PortMirroring"
)
