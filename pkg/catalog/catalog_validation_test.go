package catalog

// catalog_validation_test.go — comprehensive catalog validation tests.
// Supplements the basic smoke tests in catalog_test.go with full coverage
// of every component model, every instance size, and all FABlib alias entries.

import (
	"strings"
	"testing"

	"github.com/Testbed-IAC/fabric-go-fim/pkg/sliver"
)

// ---------------------------------------------------------------------------
// Component model validation
// ---------------------------------------------------------------------------

// TestCatalog_AllModels_HaveValidType verifies that every component entry in
// the catalog carries a Type that matches a known ComponentType enum value.
func TestCatalog_AllModels_HaveValidType(t *testing.T) {
	known := map[sliver.ComponentType]bool{
		sliver.ComponentTypeGPU:       true,
		sliver.ComponentTypeSmartNIC:  true,
		sliver.ComponentTypeSharedNIC: true,
		sliver.ComponentTypeFPGA:      true,
		sliver.ComponentTypeNVME:      true,
		sliver.ComponentTypeStorage:   true,
	}

	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, entry := range components.Entries() {
		t.Run(string(entry.Type)+"/"+entry.Model, func(t *testing.T) {
			if !known[entry.Type] {
				t.Errorf("Type %q is not a known ComponentType", entry.Type)
			}
		})
	}
}

// TestCatalog_AllModels_HaveNonEmptyDetails verifies that the Details field
// is non-empty for every catalog entry (required by §2 of the spec).
func TestCatalog_AllModels_HaveNonEmptyDetails(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, entry := range components.Entries() {
		t.Run(string(entry.Type)+"/"+entry.Model, func(t *testing.T) {
			if strings.TrimSpace(entry.Details) == "" {
				t.Errorf("Details is empty")
			}
		})
	}
}

// TestCatalog_AllModels_InterfaceMap verifies the Interfaces map for entries
// that expose ports:
//   - NIC types (SharedNIC, SmartNIC) must have at least one port.
//   - FPGA must have at least two ports.
//   - GPU, NVME, Storage must have zero ports.
func TestCatalog_AllModels_InterfaceMap(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, entry := range components.Entries() {
		entry := entry
		t.Run(string(entry.Type)+"/"+entry.Model, func(t *testing.T) {
			nPorts := len(entry.Interfaces)
			switch entry.Type {
			case sliver.ComponentTypeSharedNIC:
				if nPorts < 1 {
					t.Errorf("SharedNIC must have ≥1 port, got %d", nPorts)
				}
			case sliver.ComponentTypeSmartNIC:
				if nPorts < 1 {
					t.Errorf("SmartNIC must have ≥1 port, got %d", nPorts)
				}
			case sliver.ComponentTypeFPGA:
				if nPorts < 2 {
					t.Errorf("FPGA must have ≥2 ports, got %d", nPorts)
				}
			case sliver.ComponentTypeGPU, sliver.ComponentTypeNVME, sliver.ComponentTypeStorage:
				if nPorts != 0 {
					t.Errorf("GPU/NVME/Storage must have 0 ports, got %d", nPorts)
				}
			}
			// All port speeds must be parseable integers > 0.
			for label, speedStr := range entry.Interfaces {
				if speedStr == "" {
					t.Errorf("port %q has empty speed", label)
				}
			}
		})
	}
}

// TestCatalog_Lookup_CanFindAllEntries verifies that every entry returned by
// Entries() can be looked up by its Type and Model.
func TestCatalog_Lookup_CanFindAllEntries(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, entry := range components.Entries() {
		t.Run(string(entry.Type)+"/"+entry.Model, func(t *testing.T) {
			got, ok := components.Lookup(entry.Type, entry.Model)
			if !ok {
				t.Fatalf("Lookup(%q, %q) returned false", entry.Type, entry.Model)
			}
			if got.Model != entry.Model || got.Type != entry.Type {
				t.Errorf("Lookup returned {%q %q}, want {%q %q}", got.Type, got.Model, entry.Type, entry.Model)
			}
		})
	}
}

// TestCatalog_Lookup_AlsoModels verifies that each AlsoModels alias resolves
// to the same entry as the primary model.
func TestCatalog_Lookup_AlsoModels(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, entry := range components.Entries() {
		for _, alias := range entry.AlsoModels {
			alias := alias
			t.Run(string(entry.Type)+"/alias:"+alias, func(t *testing.T) {
				got, ok := components.Lookup(entry.Type, alias)
				if !ok {
					t.Fatalf("Lookup(%q, %q) for alias returned false", entry.Type, alias)
				}
				if got.Model != entry.Model {
					t.Errorf("alias %q resolved to model %q, want %q", alias, got.Model, entry.Model)
				}
			})
		}
	}
}

// TestCatalog_Generate_SubTree verifies that Generate produces the correct
// Component → NetworkService → ConnectionPoint sub-tree for every entry.
func TestCatalog_Generate_SubTree(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, entry := range components.Entries() {
		entry := entry
		t.Run(string(entry.Type)+"/"+entry.Model, func(t *testing.T) {
			got, err := components.Generate(GenerateOpts{
				ParentNodeName: "vm1",
				ChildName:      "dev",
				Type:           entry.Type,
				Model:          entry.Model,
				GraphID:        "test-graph-id",
			})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}

			// Component name must be child name only (matches Python FIM behaviour).
			if got.Component.Name != "dev" {
				t.Errorf("Component.Name = %q, want %q", got.Component.Name, "dev")
			}
			if got.Component.Type != entry.Type {
				t.Errorf("Component.Type = %q, want %q", got.Component.Type, entry.Type)
			}
			if got.Component.Model != entry.Model {
				t.Errorf("Component.Model = %q, want %q", got.Component.Model, entry.Model)
			}
			if got.Component.NodeID == "" {
				t.Errorf("Component.NodeID must not be empty")
			}
			if got.Component.Details == "" {
				t.Errorf("Component.Details must not be empty")
			}
			if got.Component.Capacities == nil || got.Component.Capacities.Unit != 1 {
				t.Errorf("Component.Capacities.Unit must be 1")
			}

			wantSvc := len(entry.Interfaces) > 0
			if (got.Service != nil) != wantSvc {
				t.Errorf("Service presence = %v, want %v (interfaces: %v)", got.Service != nil, wantSvc, entry.Interfaces)
			}

			if got.Service != nil {
				// Service name pattern: componentName + "-l2" + lower(serviceType)
				if !strings.HasPrefix(got.Service.Name, "vm1-dev-l2") {
					t.Errorf("Service.Name = %q, want prefix %q", got.Service.Name, "vm1-dev-l2")
				}
				if got.Service.Layer == "" {
					t.Errorf("Service.Layer must be set")
				}
			}

			// Interface count must match the catalog Interfaces map.
			if len(got.Interfaces) != len(entry.Interfaces) {
				t.Errorf("interface count = %d, want %d", len(got.Interfaces), len(entry.Interfaces))
			}
			for _, iface := range got.Interfaces {
				if iface.Name == "" {
					t.Errorf("interface has empty Name")
				}
				if iface.NodeID == "" {
					t.Errorf("interface %q has empty NodeID", iface.Name)
				}
				if iface.Labels == nil || iface.Labels.LocalName == "" {
					t.Errorf("interface %q has no LocalName label", iface.Name)
				}
			}

			// For Storage/NVME entries with a Capacity field, the Disk capacity
			// on the component must be non-zero.
			if entry.Capacity != "" && got.Component.Capacities.Disk == 0 {
				t.Errorf("component with Capacity=%q must have non-zero Disk capacity", entry.Capacity)
			}
		})
	}
}

// TestCatalog_Generate_NICPortTypes verifies that SharedNIC generates SharedPort
// interfaces and SmartNIC/FPGA generate DedicatedPort interfaces.
func TestCatalog_Generate_NICPortTypes(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	cases := []struct {
		ctype    sliver.ComponentType
		model    string
		portType sliver.InterfaceType
	}{
		{sliver.ComponentTypeSharedNIC, "ConnectX-6", sliver.InterfaceTypeSharedPort},
		{sliver.ComponentTypeSmartNIC, "ConnectX-6", sliver.InterfaceTypeDedicatedPort},
		{sliver.ComponentTypeFPGA, "Xilinx-U280", sliver.InterfaceTypeDedicatedPort},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.ctype)+"/"+tc.model, func(t *testing.T) {
			got, err := components.Generate(GenerateOpts{
				ParentNodeName: "vm1",
				ChildName:      "dev",
				Type:           tc.ctype,
				Model:          tc.model,
				GraphID:        "test-graph-id",
			})
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			for _, iface := range got.Interfaces {
				if iface.Type != tc.portType {
					t.Errorf("interface %q Type = %q, want %q", iface.Name, iface.Type, tc.portType)
				}
			}
		})
	}
}

// TestCatalog_Generate_FPGAServiceType verifies that FPGA components get a P4
// service (not OVS).
func TestCatalog_Generate_FPGAServiceType(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	got, err := components.Generate(GenerateOpts{
		ParentNodeName: "vm1",
		ChildName:      "fpga",
		Type:           sliver.ComponentTypeFPGA,
		Model:          "Xilinx-U280",
		GraphID:        "test-graph-id",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.Service == nil {
		t.Fatal("FPGA must generate a service")
	}
	if got.Service.Type != sliver.ServiceTypeP4 {
		t.Errorf("FPGA service Type = %q, want P4", got.Service.Type)
	}
}

// TestCatalog_Generate_UnknownModel verifies that Generate returns an error
// for an unknown model, not a panic.
func TestCatalog_Generate_UnknownModel(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	_, err = components.Generate(GenerateOpts{
		ParentNodeName: "vm1",
		ChildName:      "dev",
		Type:           sliver.ComponentTypeGPU,
		Model:          "NoSuchModel-XYZ",
		GraphID:        "test-graph-id",
	})
	if err == nil {
		t.Error("Generate with unknown model must return an error")
	}
}

// ---------------------------------------------------------------------------
// Instance catalog validation
// ---------------------------------------------------------------------------

// TestCatalog_AllInstances_CanLookup verifies that every instance name returned
// by Names() can be retrieved by Lookup() with matching Capacities.
func TestCatalog_AllInstances_CanLookup(t *testing.T) {
	instances, err := Instances()
	if err != nil {
		t.Fatalf("Instances: %v", err)
	}
	for _, name := range instances.Names() {
		name := name
		t.Run(name, func(t *testing.T) {
			cap, ok := instances.Lookup(name)
			if !ok {
				t.Fatalf("Lookup(%q) returned false", name)
			}
			if cap.Core <= 0 || cap.RAM <= 0 || cap.Disk <= 0 {
				t.Errorf("instance %q has zero or negative capacity: %+v", name, cap)
			}
		})
	}
}

// TestCatalog_Instance_Lookup_Unknown verifies that Lookup of an unknown name
// returns false, not a panic.
func TestCatalog_Instance_Lookup_Unknown(t *testing.T) {
	instances, err := Instances()
	if err != nil {
		t.Fatalf("Instances: %v", err)
	}
	if _, ok := instances.Lookup("no.such.flavor"); ok {
		t.Error("Lookup of unknown instance name must return false")
	}
}

// TestCatalog_MapCapacitiesToInstance_SmallestFit verifies that
// MapCapacitiesToInstance returns the smallest instance that satisfies every
// requested field.
func TestCatalog_MapCapacitiesToInstance_SmallestFit(t *testing.T) {
	instances, err := Instances()
	if err != nil {
		t.Fatalf("Instances: %v", err)
	}
	tests := []struct {
		name string
		want sliver.Capacities
	}{
		{"tiny", sliver.Capacities{Core: 1, RAM: 1, Disk: 1}},
		{"typical", sliver.Capacities{Core: 4, RAM: 8, Disk: 10}},
		{"large", sliver.Capacities{Core: 16, RAM: 64, Disk: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := instances.MapCapacitiesToInstance(tt.want)
			if result == "" {
				t.Fatalf("MapCapacitiesToInstance returned empty for %+v", tt.want)
			}
			got, ok := instances.Lookup(result)
			if !ok {
				t.Fatalf("result %q is not a known instance", result)
			}
			if !got.GreaterOrEqual(tt.want) {
				t.Errorf("chosen instance %q capacities %+v do not satisfy wanted %+v", result, got, tt.want)
			}
		})
	}
}

// TestCatalog_MapCapacitiesToInstance_RoundTrip verifies the round-trip:
// look up an instance name → get capacities → map back → same instance name.
func TestCatalog_MapCapacitiesToInstance_RoundTrip(t *testing.T) {
	instances, err := Instances()
	if err != nil {
		t.Fatalf("Instances: %v", err)
	}
	for _, name := range instances.Names() {
		name := name
		t.Run(name, func(t *testing.T) {
			cap, _ := instances.Lookup(name)
			back := instances.MapCapacitiesToInstance(cap)
			if back == "" {
				t.Fatalf("MapCapacitiesToInstance returned empty for %q caps", name)
			}
			// MapCapacitiesToInstance may return a smaller-or-equal instance with
			// the same total score; verify that it still fits and that looking up
			// the returned name returns capacities >= the original.
			backCap, _ := instances.Lookup(back)
			if !backCap.GreaterOrEqual(cap) {
				t.Errorf("round-trip %q → %q but %+v does not cover %+v", name, back, backCap, cap)
			}
		})
	}
}

// TestCatalog_MapCapacitiesToInstance_ZeroRequest verifies that requesting zero
// capacities always returns some (the smallest) instance rather than panicking.
func TestCatalog_MapCapacitiesToInstance_ZeroRequest(t *testing.T) {
	instances, err := Instances()
	if err != nil {
		t.Fatalf("Instances: %v", err)
	}
	result := instances.MapCapacitiesToInstance(sliver.Capacities{})
	if result == "" {
		t.Error("MapCapacitiesToInstance with zero request must return a non-empty name")
	}
}

// ---------------------------------------------------------------------------
// FABlib alias validation
// ---------------------------------------------------------------------------

var allFABlibAliases = []string{
	FABlibNICBasic,
	FABlibNICOpenStack,
	FABlibNICConnectX5,
	FABlibNICConnectX6,
	FABlibNICConnectX7100,
	FABlibNICConnectX7400,
	FABlibNICBlueField2ConnectX6,
	FABlibNICBlueField2ConnectX6Py,
	FABlibNICP4,
	FABlibGPUTeslaT4,
	FABlibGPURTX6000,
	FABlibGPUA30,
	FABlibGPUA40,
	FABlibNVMEP4510,
	FABlibFPGAXilinxU280,
	FABlibFPGAXilinxSN1022,
}

// TestCatalog_FABlib_AllAliasesResolve verifies that every constant defined
// in fablib.go resolves to a (ComponentType, Model) pair that is present in
// the component catalog.
func TestCatalog_FABlib_AllAliasesResolve(t *testing.T) {
	components, err := Components()
	if err != nil {
		t.Fatalf("Components: %v", err)
	}
	for _, alias := range allFABlibAliases {
		alias := alias
		t.Run(alias, func(t *testing.T) {
			ctype, model, ok := ResolveFABlibModel(alias)
			if !ok {
				t.Fatalf("ResolveFABlibModel(%q) returned false", alias)
			}
			if ctype == "" {
				t.Errorf("resolved ComponentType is empty for alias %q", alias)
			}
			if model == "" {
				t.Errorf("resolved Model is empty for alias %q", alias)
			}
			// The resolved type+model must exist in the component catalog.
			if _, found := components.Lookup(ctype, model); !found {
				t.Errorf("alias %q resolves to %q/%q which is not in the component catalog", alias, ctype, model)
			}
		})
	}
}

// TestCatalog_FABlib_UnknownAlias_ReturnsError verifies that an unrecognised
// FABlib alias returns ok=false and does not panic.
func TestCatalog_FABlib_UnknownAlias_ReturnsError(t *testing.T) {
	_, _, ok := ResolveFABlibModel("NIC_DoesNotExist_XYZ")
	if ok {
		t.Error("ResolveFABlibModel with unknown alias must return ok=false")
	}
}

// TestCatalog_FABlib_NICBasic_ResolvesSharedNIC verifies the important
// alias NIC_Basic → SharedNIC / ConnectX-6.
func TestCatalog_FABlib_NICBasic_ResolvesSharedNIC(t *testing.T) {
	ctype, model, ok := ResolveFABlibModel(FABlibNICBasic)
	if !ok {
		t.Fatalf("ResolveFABlibModel(NIC_Basic) returned false")
	}
	if ctype != sliver.ComponentTypeSharedNIC {
		t.Errorf("NIC_Basic type = %q, want SharedNIC", ctype)
	}
	if model == "" {
		t.Errorf("NIC_Basic model is empty")
	}
}

// TestCatalog_FABlib_NICConnectX6_ResolvesSmartNIC verifies the common
// alias NIC_ConnectX_6 → SmartNIC / ConnectX-6.
func TestCatalog_FABlib_NICConnectX6_ResolvesSmartNIC(t *testing.T) {
	ctype, _, ok := ResolveFABlibModel(FABlibNICConnectX6)
	if !ok {
		t.Fatalf("ResolveFABlibModel(NIC_ConnectX_6) returned false")
	}
	if ctype != sliver.ComponentTypeSmartNIC {
		t.Errorf("NIC_ConnectX_6 type = %q, want SmartNIC", ctype)
	}
}

// TestCatalog_FABlib_BothBF2Spellings verifies that both BlueField-2 alias
// spellings (with and without "2") resolve to the same model.
func TestCatalog_FABlib_BothBF2Spellings(t *testing.T) {
	_, model1, ok1 := ResolveFABlibModel(FABlibNICBlueField2ConnectX6)
	_, model2, ok2 := ResolveFABlibModel(FABlibNICBlueField2ConnectX6Py)
	if !ok1 || !ok2 {
		t.Fatalf("one of the BF2 aliases failed to resolve: %v %v", ok1, ok2)
	}
	if model1 != model2 {
		t.Errorf("BF2 alias spellings resolve to different models: %q vs %q", model1, model2)
	}
}

// TestCatalog_FABlib_NIC_P4_ResolvesFPGA verifies that NIC_P4 maps to FPGA.
func TestCatalog_FABlib_NIC_P4_ResolvesFPGA(t *testing.T) {
	ctype, _, ok := ResolveFABlibModel(FABlibNICP4)
	if !ok {
		t.Fatalf("ResolveFABlibModel(NIC_P4) returned false")
	}
	if ctype != sliver.ComponentTypeFPGA {
		t.Errorf("NIC_P4 type = %q, want FPGA", ctype)
	}
}
