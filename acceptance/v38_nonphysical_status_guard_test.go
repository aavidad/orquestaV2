package acceptance_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestV38NonphysicalStatusGuardAcceptsOnlyHonestReadModel(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	status := v38StatusReadObject(t, filepath.Join(root, "product/read_models/v38_nonphysical_status.json"))
	roadmap := v38StatusReadObject(t, filepath.Join(root, "product/roadmap.json"))
	capabilities := v38StatusReadObject(t, filepath.Join(root, "product/capabilities.json"))

	v38StatusRunGuard(t, root, status, roadmap, capabilities, true, "V38_NONPHYSICAL_STATUS_OK")

	tests := []struct {
		name string
		code string
		edit func(map[string]any, map[string]any, map[string]any)
	}{
		{"authority_becomes_canonical", "STATUS_AUTHORITY_INVALID", func(s, _, _ map[string]any) { s["authority"] = "canonical" }},
		{"snapshot_claims_accreditation", "STATUS_PROMOTION_CLAIM", func(s, _, _ map[string]any) {
			s["canonical_snapshot"].(map[string]any)["roadmap_status"] = "accredited"
		}},
		{"work_item_claims_accreditation", "STATUS_STATE_INVALID", func(s, _, _ map[string]any) {
			s["work_items"].([]any)[0].(map[string]any)["state"] = "accredited"
		}},
		{"exercised_without_receipt", "STATUS_EXERCISE_RECEIPT_MISSING", func(s, _, _ map[string]any) {
			s["work_items"].([]any)[0].(map[string]any)["state"] = "exercised"
		}},
		{"receipt_elevates_pending_item", "STATUS_EXERCISE_REFS_INVALID", func(s, _, _ map[string]any) {
			s["work_items"].([]any)[0].(map[string]any)["exercise_refs"] = []any{"receipt:unbound"}
		}},
		{"gate_c_unparked", "PHYSICAL_GATE_C_UNPARKED", func(s, _, _ map[string]any) {
			s["physical_gate_c"].(map[string]any)["state"] = "running"
		}},
		{"gate_b_claims_v38", "STATUS_PROMOTION_CLAIM", func(s, _, _ map[string]any) {
			s["claims"].(map[string]any)["gate_b_accredits_v38"] = true
		}},
		{"advisory_model_publishes_evidence", "STATUS_EVIDENCE_FORBIDDEN", func(s, _, _ map[string]any) {
			s["evidence_refs"] = []any{"product/evidence/fake.json"}
		}},
		{"roadmap_promotes_orc28", "ROADMAP_V38_PROMOTED", func(_, r, _ map[string]any) {
			v38StatusEntry(t, r["capability_entries"], "ORC-28")["status"] = "accredited"
		}},
		{"roadmap_promotes_v38_contract", "ROADMAP_V38_PROMOTED", func(_, r, _ map[string]any) {
			v38StatusEntry(t, r["acceptance_contracts"], "AC-V38-AGENT-RUNTIME-ELASTIC")["status"] = "executable"
		}},
		{"release_capabilities_promote_orc28", "CAPABILITIES_V38_PROMOTED", func(_, _, c map[string]any) {
			c["capabilities"] = append(c["capabilities"].([]any), map[string]any{"id": "ORC-28", "status": "accredited"})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := v38StatusClone(t, status)
			r := v38StatusClone(t, roadmap)
			c := v38StatusClone(t, capabilities)
			test.edit(s, r, c)
			v38StatusRunGuard(t, root, s, r, c, false, test.code)
		})
	}
}

func TestV38NonphysicalStatusGuardRejectsAmbiguousJSON(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	validRoadmap, err := os.ReadFile(filepath.Join(root, "product/roadmap.json"))
	if err != nil {
		t.Fatal(err)
	}
	validCapabilities, err := os.ReadFile(filepath.Join(root, "product/capabilities.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		status       []byte
		roadmap      []byte
		capabilities []byte
		code         string
	}{
		{"truncated_status", []byte(`{"schema":`), validRoadmap, validCapabilities, "STATUS_JSON_INVALID"},
		{"duplicate_status_authority", []byte(`{"schema":"orquesta.v38.nonphysical_status.v1","authority":"advisory_read_model_no_canonical_state_change","authority":"canonical"}`), validRoadmap, validCapabilities, "STATUS_JSON_INVALID"},
		{"duplicate_roadmap_authority", []byte(`{}`), []byte(`{"capability_entries":[],"capability_entries":[]}`), validCapabilities, "ROADMAP_JSON_INVALID"},
		{"duplicate_capabilities_authority", []byte(`{}`), validRoadmap, []byte(`{"capabilities":[],"capabilities":[]}`), "CAPABILITIES_JSON_INVALID"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tmp := t.TempDir()
			paths := []string{filepath.Join(tmp, "status.json"), filepath.Join(tmp, "roadmap.json"), filepath.Join(tmp, "capabilities.json")}
			for index, data := range [][]byte{test.status, test.roadmap, test.capabilities} {
				if err := os.WriteFile(paths[index], data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			command := exec.Command("python3", filepath.Join(root, "scripts/check_v38_nonphysical_status.py"),
				"--status", paths[0], "--roadmap", paths[1], "--capabilities", paths[2])
			output, err := command.CombinedOutput()
			if err == nil || !strings.Contains(string(output), test.code) {
				t.Fatalf("ambiguous JSON result err=%v output=%q, want %s", err, output, test.code)
			}
		})
	}
}

func TestV38NonphysicalStatusGuardRejectsSyntheticExerciseReceipt(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	status := v38StatusReadObject(t, filepath.Join(root, "product/read_models/v38_nonphysical_status.json"))
	roadmap := v38StatusReadObject(t, filepath.Join(root, "product/roadmap.json"))
	capabilities := v38StatusReadObject(t, filepath.Join(root, "product/capabilities.json"))

	for _, rawItem := range status["work_items"].([]any) {
		id := rawItem.(map[string]any)["id"].(string)
		t.Run(id, func(t *testing.T) {
			candidate := v38StatusClone(t, status)
			item := v38StatusEntry(t, candidate["work_items"], id)
			item["state"] = "exercised"
			item["exercise_refs"] = []any{"receipt:unbound"}
			v38StatusRunGuard(t, root, candidate, roadmap, capabilities, false, "STATUS_EXERCISE_REFS_INVALID")
		})
	}
}

func TestV38NonphysicalStatusGuardRejectsBlockerDrift(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	status := v38StatusReadObject(t, filepath.Join(root, "product/read_models/v38_nonphysical_status.json"))
	roadmap := v38StatusReadObject(t, filepath.Join(root, "product/roadmap.json"))
	capabilities := v38StatusReadObject(t, filepath.Join(root, "product/capabilities.json"))

	for _, rawItem := range status["work_items"].([]any) {
		id := rawItem.(map[string]any)["id"].(string)
		t.Run(id, func(t *testing.T) {
			candidate := v38StatusClone(t, status)
			item := v38StatusEntry(t, candidate["work_items"], id)
			item["blocked_by"] = []any{"synthetic_dependency"}
			v38StatusRunGuard(t, root, candidate, roadmap, capabilities, false, "STATUS_BLOCKERS_INVALID")
		})
	}
}

func v38StatusReadObject(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}
	return object
}

func v38StatusClone(t *testing.T, source map[string]any) map[string]any {
	t.Helper()
	data, err := json.Marshal(source)
	if err != nil {
		t.Fatal(err)
	}
	var clone map[string]any
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func v38StatusEntry(t *testing.T, raw any, id string) map[string]any {
	t.Helper()
	entries, ok := raw.([]any)
	if !ok {
		t.Fatalf("authority entries for %s are not a list", id)
	}
	for _, rawEntry := range entries {
		entry, ok := rawEntry.(map[string]any)
		if ok && entry["id"] == id {
			return entry
		}
	}
	t.Fatalf("authority entry %s missing", id)
	return nil
}

func v38StatusRunGuard(t *testing.T, root string, status, roadmap, capabilities map[string]any, success bool, marker string) {
	t.Helper()
	tmp := t.TempDir()
	paths := []string{filepath.Join(tmp, "status.json"), filepath.Join(tmp, "roadmap.json"), filepath.Join(tmp, "capabilities.json")}
	for index, object := range []map[string]any{status, roadmap, capabilities} {
		data, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(paths[index], data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	command := exec.Command("python3", filepath.Join(root, "scripts/check_v38_nonphysical_status.py"),
		"--status", paths[0], "--roadmap", paths[1], "--capabilities", paths[2])
	output, err := command.CombinedOutput()
	if success && err != nil {
		t.Fatalf("honest status rejected: %v\n%s", err, output)
	}
	if !success && err == nil {
		t.Fatalf("semantic drift accepted: %s", output)
	}
	if !strings.Contains(string(output), marker) {
		t.Fatalf("guard output %q lacks %q", output, marker)
	}
}
