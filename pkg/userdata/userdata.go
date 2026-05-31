// Package userdata defines the typed FABlib-compatible user-data envelope that
// FABRIC stores on a node sliver's UserData property.
//
// FABlib persists routes, post-boot tasks, post-update commands, and storage
// flags inside the node's user-data JSON under a "fablib_data" style envelope
// (see fablib/node.py). Modelling that envelope as typed Go lets a writer (the
// Terraform provider) and a reader (drift detection, FABlib) agree on the same
// shape while preserving any keys this package does not understand.
//
// Encode produces deterministic, sorted-key JSON so that generated GraphML is
// stable across runs, and enforces the same 2048-byte cap that sliver
// validation applies to the UserData blob.
package userdata

import (
	"encoding/json"
	"fmt"
	"sort"
)

// MaxBytes is the maximum encoded size of a node user-data envelope. It matches
// the UserData blob cap enforced by pkg/sliver validation; exceeding it would
// make the resulting sliver fail to serialize.
const MaxBytes = 2048

// Post-boot task type values, matching FABlib's add_post_boot_* helpers.
const (
	// TaskExecute runs a command on the node after boot.
	TaskExecute = "execute"
	// TaskUploadFile uploads a single file to the node after boot.
	TaskUploadFile = "upload_file"
	// TaskUploadDirectory uploads a directory to the node after boot.
	TaskUploadDirectory = "upload_directory"
)

// reservedKeys are the envelope keys this package models with typed fields.
// Any other key found during Decode is preserved verbatim in NodeData.Extra so
// round-tripping never destroys data written by FABlib or future tooling.
var reservedKeys = map[string]struct{}{
	"routes":               {},
	"post_boot_tasks":      {},
	"post_update_commands": {},
	"storage":              {},
	"storage_cluster":      {},
}

// Route is a static route declared into a node's user-data. FABlib emits each
// route as {"subnet": ..., "next_hop": ...}.
type Route struct {
	Subnet  string `json:"subnet"`
	NextHop string `json:"next_hop"`
}

// PostBootTask is a single post-boot task. FABlib stores these as heterogeneous
// tuples — ("execute", cmd), ("upload_file", local, remote) — so a task
// serializes as a JSON array whose first element is Type and whose remaining
// elements are Args.
type PostBootTask struct {
	Type string
	Args []string
}

// MarshalJSON renders the task as the FABlib tuple form [Type, Args...].
func (t PostBootTask) MarshalJSON() ([]byte, error) {
	tuple := make([]string, 0, len(t.Args)+1)
	tuple = append(tuple, t.Type)
	tuple = append(tuple, t.Args...)
	return json.Marshal(tuple)
}

// UnmarshalJSON parses the FABlib tuple form [Type, Args...] back into the task.
func (t *PostBootTask) UnmarshalJSON(data []byte) error {
	var tuple []string
	if err := json.Unmarshal(data, &tuple); err != nil {
		return fmt.Errorf("%w: post-boot task must be a JSON array of strings: %v", ErrInvalid, err)
	}
	if len(tuple) == 0 {
		return fmt.Errorf("%w: post-boot task must have at least a type element", ErrInvalid)
	}
	t.Type = tuple[0]
	t.Args = append([]string(nil), tuple[1:]...)
	return nil
}

// NodeData is the typed view of a node's user-data envelope. Fields this
// package understands are modelled explicitly; any other key is held in Extra.
type NodeData struct {
	Routes         []Route
	PostBootTasks  []PostBootTask
	PostUpdate     []string
	Storage        bool
	StorageCluster string
	// Extra holds envelope keys this package does not model, so Decode→Encode
	// preserves data written by FABlib or other tooling.
	Extra map[string]json.RawMessage
}

// IsEmpty reports whether the envelope carries no data at all. Callers use it to
// avoid writing an empty "{}" user-data blob onto a sliver.
func (d NodeData) IsEmpty() bool {
	return len(d.Routes) == 0 &&
		len(d.PostBootTasks) == 0 &&
		len(d.PostUpdate) == 0 &&
		!d.Storage &&
		d.StorageCluster == "" &&
		len(d.Extra) == 0
}

// Encode serializes the envelope to deterministic, sorted-key JSON. It returns
// ErrTooLarge if the result exceeds MaxBytes. Managed fields take precedence
// over any same-named key in Extra.
func (d NodeData) Encode() ([]byte, error) {
	out := make(map[string]json.RawMessage, len(d.Extra)+5)
	for key, value := range d.Extra {
		if _, reserved := reservedKeys[key]; reserved {
			continue
		}
		out[key] = value
	}
	if len(d.Routes) > 0 {
		raw, err := json.Marshal(d.Routes)
		if err != nil {
			return nil, fmt.Errorf("%w: encoding routes: %v", ErrInvalid, err)
		}
		out["routes"] = raw
	}
	if len(d.PostBootTasks) > 0 {
		raw, err := json.Marshal(d.PostBootTasks)
		if err != nil {
			return nil, fmt.Errorf("%w: encoding post-boot tasks: %v", ErrInvalid, err)
		}
		out["post_boot_tasks"] = raw
	}
	if len(d.PostUpdate) > 0 {
		raw, err := json.Marshal(d.PostUpdate)
		if err != nil {
			return nil, fmt.Errorf("%w: encoding post-update commands: %v", ErrInvalid, err)
		}
		out["post_update_commands"] = raw
	}
	if d.Storage {
		out["storage"] = json.RawMessage("true")
	}
	if d.StorageCluster != "" {
		raw, err := json.Marshal(d.StorageCluster)
		if err != nil {
			return nil, fmt.Errorf("%w: encoding storage cluster: %v", ErrInvalid, err)
		}
		out["storage_cluster"] = raw
	}
	encoded, err := marshalSorted(out)
	if err != nil {
		return nil, err
	}
	if len(encoded) > MaxBytes {
		return nil, fmt.Errorf("%w: encoded user-data is %d bytes, exceeds the %d-byte limit", ErrTooLarge, len(encoded), MaxBytes)
	}
	return encoded, nil
}

// Decode parses an envelope, routing reserved keys to typed fields and every
// other key to Extra. A zero-length input decodes to an empty NodeData.
func Decode(data []byte) (NodeData, error) {
	var d NodeData
	if len(data) == 0 {
		return d, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return d, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	for key, value := range raw {
		var err error
		switch key {
		case "routes":
			err = json.Unmarshal(value, &d.Routes)
		case "post_boot_tasks":
			err = json.Unmarshal(value, &d.PostBootTasks)
		case "post_update_commands":
			err = json.Unmarshal(value, &d.PostUpdate)
		case "storage":
			err = json.Unmarshal(value, &d.Storage)
		case "storage_cluster":
			err = json.Unmarshal(value, &d.StorageCluster)
		default:
			if d.Extra == nil {
				d.Extra = map[string]json.RawMessage{}
			}
			d.Extra[key] = value
		}
		if err != nil {
			return NodeData{}, fmt.Errorf("%w: decoding %q: %v", ErrInvalid, key, err)
		}
	}
	return d, nil
}

// marshalSorted marshals a string-keyed map with keys in sorted order. The
// standard library already sorts map keys, but doing it explicitly documents
// the determinism guarantee that GraphML stability depends on.
func marshalSorted(m map[string]json.RawMessage) ([]byte, error) {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	buf := make([]byte, 0, 64)
	buf = append(buf, '{')
	for i, key := range keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		encodedKey, err := json.Marshal(key)
		if err != nil {
			return nil, fmt.Errorf("%w: encoding key %q: %v", ErrInvalid, key, err)
		}
		buf = append(buf, encodedKey...)
		buf = append(buf, ':')
		buf = append(buf, m[key]...)
	}
	buf = append(buf, '}')
	return buf, nil
}
