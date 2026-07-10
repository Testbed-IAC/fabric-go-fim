package client

import "testing"

func TestManagementIPFromSliverPayload(t *testing.T) {
	t.Parallel()
	// Realized FABRIC slivers carry the management IP under "MgmtIp".
	got := managementIPFromSliverPayload(map[string]interface{}{
		"Name":   "worker-0",
		"MgmtIp": "2001:400:a100:3030:f816:3eff:fe42:a83",
		"Type":   "VM",
	})
	if got != "2001:400:a100:3030:f816:3eff:fe42:a83" {
		t.Errorf("MgmtIp = %q", got)
	}
	// Legacy fallbacks still work.
	if got := managementIPFromSliverPayload(map[string]interface{}{"management_ip": "10.0.0.5"}); got != "10.0.0.5" {
		t.Errorf("fallback = %q", got)
	}
	// Absent / empty → "".
	if got := managementIPFromSliverPayload(map[string]interface{}{"MgmtIp": ""}); got != "" {
		t.Errorf("empty MgmtIp = %q, want ''", got)
	}
	if got := managementIPFromSliverPayload(nil); got != "" {
		t.Errorf("nil payload = %q", got)
	}
}

func TestNameFromSliverPayload(t *testing.T) {
	t.Parallel()
	if got := nameFromSliverPayload(map[string]interface{}{"Name": "worker-1"}); got != "worker-1" {
		t.Errorf("Name = %q", got)
	}
	if got := nameFromSliverPayload(map[string]interface{}{"Type": "OVS"}); got != "" {
		t.Errorf("missing Name = %q, want ''", got)
	}
}

func TestImageFromSliverPayload(t *testing.T) {
	t.Parallel()
	// ImageRef is "<image>,<type>" — only the image name is returned.
	if got := imageFromSliverPayload(map[string]interface{}{"ImageRef": "default_ubuntu_22,qcow2"}); got != "default_ubuntu_22" {
		t.Errorf("ImageRef = %q, want default_ubuntu_22", got)
	}
	// No type suffix → returned as-is.
	if got := imageFromSliverPayload(map[string]interface{}{"ImageRef": "default_rocky_9"}); got != "default_rocky_9" {
		t.Errorf("no-suffix = %q", got)
	}
	// Network services / absent / "None" → "".
	for _, p := range []map[string]interface{}{
		{"Name": "cluster-net"},
		{"ImageRef": ""},
		{"ImageRef": "None"},
		nil,
	} {
		if got := imageFromSliverPayload(p); got != "" {
			t.Errorf("payload %v image = %q, want ''", p, got)
		}
	}
}
