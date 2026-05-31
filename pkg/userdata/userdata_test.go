package userdata

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestEncodeDeterministicOrdering(t *testing.T) {
	t.Parallel()
	data := NodeData{
		Routes:         []Route{{Subnet: "10.0.0.0/24", NextHop: "10.0.1.1"}},
		PostBootTasks:  []PostBootTask{{Type: TaskExecute, Args: []string{"echo hi"}}, {Type: TaskUploadFile, Args: []string{"/local", "/remote"}}},
		PostUpdate:     []string{"apt update"},
		Storage:        true,
		StorageCluster: "europe",
		Extra:          map[string]json.RawMessage{"instantiated": json.RawMessage(`"False"`)},
	}
	const want = `{"instantiated":"False","post_boot_tasks":[["execute","echo hi"],["upload_file","/local","/remote"]],"post_update_commands":["apt update"],"routes":[{"subnet":"10.0.0.0/24","next_hop":"10.0.1.1"}],"storage":true,"storage_cluster":"europe"}`
	got, err := data.Encode()
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if string(got) != want {
		t.Fatalf("Encode mismatch:\n got: %s\nwant: %s", got, want)
	}
	// Re-encoding the same data must be byte-identical (stable GraphML).
	again, err := data.Encode()
	if err != nil {
		t.Fatalf("second Encode returned error: %v", err)
	}
	if string(again) != string(got) {
		t.Fatalf("Encode is not deterministic:\n first: %s\nsecond: %s", got, again)
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	t.Parallel()
	data := NodeData{
		Routes:        []Route{{Subnet: "192.168.1.0/24", NextHop: "192.168.1.1"}},
		PostBootTasks: []PostBootTask{{Type: TaskUploadDirectory, Args: []string{"/src", "/dst"}}, {Type: TaskExecute, Args: []string{"systemctl restart nginx"}}},
		PostUpdate:    []string{"dnf upgrade -y", "reboot"},
		Storage:       true,
	}
	encoded, err := data.Encode()
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	reEncoded, err := decoded.Encode()
	if err != nil {
		t.Fatalf("re-Encode returned error: %v", err)
	}
	if string(encoded) != string(reEncoded) {
		t.Fatalf("round-trip mismatch:\n first: %s\nsecond: %s", encoded, reEncoded)
	}
	if len(decoded.Routes) != 1 || decoded.Routes[0].Subnet != "192.168.1.0/24" || decoded.Routes[0].NextHop != "192.168.1.1" {
		t.Fatalf("routes not preserved: %+v", decoded.Routes)
	}
	if len(decoded.PostBootTasks) != 2 || decoded.PostBootTasks[0].Type != TaskUploadDirectory || decoded.PostBootTasks[1].Args[0] != "systemctl restart nginx" {
		t.Fatalf("post-boot tasks not preserved: %+v", decoded.PostBootTasks)
	}
	if !decoded.Storage {
		t.Fatalf("storage flag not preserved")
	}
}

// TestDecodePreservesUnknownKeys proves Decode→Encode never destroys keys this
// package does not model — the data FABlib writes (instantiated,
// run_update_commands, post_boot_commands) must survive a provider round-trip.
func TestDecodePreservesUnknownKeys(t *testing.T) {
	t.Parallel()
	const fablibWritten = `{"instantiated":"True","post_boot_commands":["a","b"],"routes":[{"subnet":"10.0.0.0/8","next_hop":"10.0.0.1"}],"run_update_commands":"False"}`
	decoded, err := Decode([]byte(fablibWritten))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if _, ok := decoded.Extra["post_boot_commands"]; !ok {
		t.Fatalf("expected post_boot_commands preserved in Extra, got %+v", decoded.Extra)
	}
	if _, ok := decoded.Extra["instantiated"]; !ok {
		t.Fatalf("expected instantiated preserved in Extra, got %+v", decoded.Extra)
	}
	// Now mutate a typed field and re-encode; unknown keys must remain.
	decoded.PostUpdate = []string{"new command"}
	reEncoded, err := decoded.Encode()
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	roundTripped, err := Decode(reEncoded)
	if err != nil {
		t.Fatalf("re-Decode returned error: %v", err)
	}
	if string(roundTripped.Extra["instantiated"]) != `"True"` {
		t.Fatalf("instantiated not preserved across round-trip: %s", roundTripped.Extra["instantiated"])
	}
	if len(roundTripped.Extra["post_boot_commands"]) == 0 {
		t.Fatalf("post_boot_commands lost across round-trip")
	}
	if len(roundTripped.PostUpdate) != 1 || roundTripped.PostUpdate[0] != "new command" {
		t.Fatalf("post_update_commands not updated: %+v", roundTripped.PostUpdate)
	}
}

// TestEncodeManagedFieldOverridesExtra guards that if Extra carries a reserved
// key (e.g. round-tripped from a decode that also set the typed field), the
// typed field wins and the key is not emitted twice.
func TestEncodeManagedFieldOverridesExtra(t *testing.T) {
	t.Parallel()
	data := NodeData{
		PostUpdate: []string{"typed"},
		Extra:      map[string]json.RawMessage{"post_update_commands": json.RawMessage(`["stale"]`)},
	}
	encoded, err := data.Encode()
	if err != nil {
		t.Fatalf("Encode returned error: %v", err)
	}
	if strings.Count(string(encoded), "post_update_commands") != 1 {
		t.Fatalf("reserved key emitted more than once: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"typed"`) || strings.Contains(string(encoded), `"stale"`) {
		t.Fatalf("typed field did not override Extra: %s", encoded)
	}
}

func TestEncodeSizeCap(t *testing.T) {
	t.Parallel()
	big := strings.Repeat("x", MaxBytes)
	data := NodeData{PostUpdate: []string{big}}
	_, err := data.Encode()
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

func TestDecodeRejectsInvalidJSON(t *testing.T) {
	t.Parallel()
	if _, err := Decode([]byte("{not json")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for malformed JSON, got %v", err)
	}
}

func TestPostBootTaskTupleForm(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(PostBootTask{Type: TaskExecute, Args: []string{"uname -a"}})
	if err != nil {
		t.Fatalf("marshal returned error: %v", err)
	}
	if string(raw) != `["execute","uname -a"]` {
		t.Fatalf("tuple form mismatch: %s", raw)
	}
	var task PostBootTask
	if err := json.Unmarshal([]byte(`["upload_file","/a","/b"]`), &task); err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}
	if task.Type != TaskUploadFile || len(task.Args) != 2 || task.Args[1] != "/b" {
		t.Fatalf("unmarshalled task mismatch: %+v", task)
	}
	if err := json.Unmarshal([]byte(`[]`), &task); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid for empty tuple, got %v", err)
	}
}

func TestIsEmpty(t *testing.T) {
	t.Parallel()
	if !(NodeData{}).IsEmpty() {
		t.Fatalf("zero NodeData should be empty")
	}
	if (NodeData{Storage: true}).IsEmpty() {
		t.Fatalf("NodeData with storage should not be empty")
	}
	if (NodeData{Routes: []Route{{Subnet: "10.0.0.0/24", NextHop: "10.0.0.1"}}}).IsEmpty() {
		t.Fatalf("NodeData with routes should not be empty")
	}
}
