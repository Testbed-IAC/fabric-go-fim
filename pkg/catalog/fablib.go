package catalog

import "github.com/CSC478-WCU/fabric-go-fim/pkg/sliver"

// FABlib component model name constants.
const (
	FABlibNICBasic                 = "NIC_Basic"
	FABlibNICOpenStack             = "NIC_OpenStack"
	FABlibNICConnectX5             = "NIC_ConnectX_5"
	FABlibNICConnectX6             = "NIC_ConnectX_6"
	FABlibNICConnectX7100          = "NIC_ConnectX_7_100"
	FABlibNICConnectX7400          = "NIC_ConnectX_7_400"
	FABlibNICBlueField2ConnectX6   = "NIC_BlueField2_ConnectX_6"
	FABlibNICBlueField2ConnectX6Py = "NIC_BlueField_2_ConnectX_6"
	FABlibNICP4                    = "NIC_P4"
	FABlibGPUTeslaT4               = "GPU_TeslaT4"
	FABlibGPURTX6000               = "GPU_RTX6000"
	FABlibGPUA30                   = "GPU_A30"
	FABlibGPUA40                   = "GPU_A40"
	FABlibNVMEP4510                = "NVME_P4510"
	FABlibFPGAXilinxU280           = "FPGA_Xilinx_U280"
	FABlibFPGAXilinxSN1022         = "FPGA_Xilinx_SN1022"
)

type fablibModel struct {
	componentType sliver.ComponentType
	model         string
}

var fablibModels = map[string]fablibModel{
	FABlibNICBasic:                 {sliver.ComponentTypeSharedNIC, "ConnectX-6"},
	FABlibNICOpenStack:             {sliver.ComponentTypeSharedNIC, "OpenStack-vNIC"},
	FABlibNICConnectX5:             {sliver.ComponentTypeSmartNIC, "ConnectX-5"},
	FABlibNICConnectX6:             {sliver.ComponentTypeSmartNIC, "ConnectX-6"},
	FABlibNICConnectX7100:          {sliver.ComponentTypeSmartNIC, "ConnectX-7-100"},
	FABlibNICConnectX7400:          {sliver.ComponentTypeSmartNIC, "ConnectX-7-400"},
	FABlibNICBlueField2ConnectX6:   {sliver.ComponentTypeSmartNIC, "BlueField-2-ConnectX-6"},
	FABlibNICBlueField2ConnectX6Py: {sliver.ComponentTypeSmartNIC, "BlueField-2-ConnectX-6"},
	FABlibNICP4:                    {sliver.ComponentTypeFPGA, "Xilinx-U280"},
	FABlibGPUTeslaT4:               {sliver.ComponentTypeGPU, "Tesla T4"},
	FABlibGPURTX6000:               {sliver.ComponentTypeGPU, "RTX6000"},
	FABlibGPUA30:                   {sliver.ComponentTypeGPU, "A30"},
	FABlibGPUA40:                   {sliver.ComponentTypeGPU, "A40"},
	FABlibNVMEP4510:                {sliver.ComponentTypeNVME, "P4510"},
	FABlibFPGAXilinxU280:           {sliver.ComponentTypeFPGA, "Xilinx-U280"},
	FABlibFPGAXilinxSN1022:         {sliver.ComponentTypeFPGA, "Xilinx-SN1022"},
}

// ResolveFABlibModel resolves a FABlib model name to a component type and catalog model.
func ResolveFABlibModel(fablibName string) (sliver.ComponentType, string, bool) {
	resolved, ok := fablibModels[fablibName]
	return resolved.componentType, resolved.model, ok
}
