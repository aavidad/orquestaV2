package acceptance_test

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const agentFirecrackerSingleVMFixturePath = "acceptance/fixtures/agent_firecracker_single_vm.json"

func TestAgentFirecrackerSingleVMFixtureRatchetsExistingNetworkDecision(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := agentFirecrackerReadObject(t, filepath.Join(root, agentFirecrackerSingleVMFixturePath))
	if err := agentFirecrackerValidateFixture(fixture); err != nil {
		t.Fatal(err)
	}

	roadmap := agentFirecrackerReadObject(t, filepath.Join(root, "product/roadmap.json"))
	decisions := agentFirecrackerObjectSlice(t, roadmap["implementation_decisions"])
	if len(decisions) != 1 {
		t.Fatalf("implementation decisions = %d, want one authority", len(decisions))
	}
	authority := fixture["authority"].(map[string]any)
	if !reflect.DeepEqual(decisions[0], authority["roadmap_decision"]) {
		t.Fatalf("roadmap authority drifted:\n got: %#v\nwant: %#v", decisions[0], authority["roadmap_decision"])
	}
	if roadmap["catalog_size"] != float64(257) ||
		agentFirecrackerHasID(roadmap["verticals"], "V39") ||
		agentFirecrackerHasIDPrefix(roadmap["acceptance_contracts"], "AC-V39") {
		t.Fatal("neutral characterization changed the canonical catalog or created V39")
	}

	for _, relative := range agentFirecrackerStringSlice(t, fixture["decision_files"]) {
		info, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relative)))
		if err != nil || !info.Mode().IsRegular() {
			t.Fatalf("decision file %q is absent or non-regular: %v", relative, err)
		}
	}
	for _, relative := range []string{
		"acceptance/fixtures/v39_agent_firecracker_single_vm.json",
		"acceptance/v39_agent_firecracker_contract_test.go",
		"product/evidence/agent_firecracker_single_vm.json",
		"product/evidence/v39_agent_firecracker_single_vm.json",
	} {
		if _, err := os.Lstat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("forbidden V39/receipt artifact exists at %q: %v", relative, err)
		}
	}
}

func TestAgentFirecrackerSingleVMProfileRejectsSemanticDrift(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	cases := []struct {
		name  string
		path  []string
		value any
	}{
		{"authority_renamed", []string{"authority", "decision_id"}, "another_decision"},
		{"authority_promoted", []string{"authority", "status"}, "applied"},
		{"transport_changed", []string{"connectivity", "transport"}, "tap"},
		{"service_changed", []string{"connectivity", "allowed_services"}, []any{"orquesta_broker", "direct_proxy"}},
		{"guest_ip_enabled", []string{"connectivity", "forbidden_connectivity"}, []any{"tap", "bridge", "nat", "inbound", "east_west", "direct_internet"}},
		{"credential_reusable", []string{"credential_store_proof", "use"}, "reusable"},
		{"credential_replay", []string{"credential_store_proof", "replay"}, "allowed"},
		{"attestor_network", []string{"test_attestor_separation", "network"}, "present"},
		{"attestor_vsock", []string{"test_attestor_separation", "vsock"}, "present"},
		{"microvm_limit_expanded", []string{"bounded_example", "maximum_microvms"}, float64(16)},
		{"runtime_wiring_claimed", []string{"bounded_example", "runtime_wiring"}, "present"},
		{"physical_execution_claimed", []string{"bounded_example", "physical_execution"}, "passed"},
		{"vertical_created", []string{"catalog_absences", "new_vertical"}, true},
		{"receipt_created", []string{"catalog_absences", "new_receipt"}, true},
		{"guest_claimed", []string{"catalog_absences", "guest_agent_claim"}, true},
		{"required_test_weakened", []string{"required_test", "command"}, "go test ./acceptance"},
		{"empty_decision_files", []string{"decision_files"}, []any{}},
		{"unknown_top_level_claim", []string{"physical_receipt"}, "receipt:invented"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := agentFirecrackerReadObject(t, filepath.Join(root, agentFirecrackerSingleVMFixturePath))
			agentFirecrackerSet(t, fixture, test.path, test.value)
			if err := agentFirecrackerValidateFixture(fixture); err == nil {
				t.Fatal("expanded, promoted or weakened characterization was accepted")
			}
		})
	}
}

func agentFirecrackerValidateFixture(fixture map[string]any) error {
	if !agentFirecrackerHasExactKeys(fixture,
		"schema_version", "fixture_id", "contract_kind", "authority", "bounded_example",
		"connectivity", "credential_store_proof", "test_attestor_separation",
		"catalog_absences", "required_test", "decision_files",
	) {
		return fmt.Errorf("fixture fields drifted")
	}
	authority, ok := fixture["authority"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(authority,
		"decision_id", "status", "capability_refs", "roadmap_decision",
	) {
		return fmt.Errorf("authority is absent")
	}
	wantCapabilityRefs := []any{"AGT-01", "AGT-03", "EVD-13", "ORC-15"}
	if fixture["schema_version"] != float64(1) ||
		fixture["fixture_id"] != "agent_firecracker_single_vm" ||
		fixture["contract_kind"] != "neutral_characterization_of_existing_implementation_decision" ||
		authority["decision_id"] != "agent_microvm_network" ||
		authority["status"] != "planned_not_applied" ||
		!reflect.DeepEqual(authority["capability_refs"], wantCapabilityRefs) {
		return fmt.Errorf("authority semantics drifted")
	}

	bounded, ok := fixture["bounded_example"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(bounded,
		"maximum_microvms", "selection_scope", "runtime_wiring", "physical_execution",
	) ||
		bounded["maximum_microvms"] != float64(1) ||
		bounded["selection_scope"] != "one_execution_attempt" ||
		bounded["runtime_wiring"] != "absent" ||
		bounded["physical_execution"] != "not_claimed" {
		return fmt.Errorf("bounded example drifted")
	}

	connectivity, ok := fixture["connectivity"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(connectivity,
		"transport", "allowed_services", "forbidden_connectivity",
	) ||
		connectivity["transport"] != "vsock_only" ||
		!reflect.DeepEqual(connectivity["allowed_services"], []any{"orquesta_broker", "controlled_egress_proxy"}) ||
		!reflect.DeepEqual(connectivity["forbidden_connectivity"],
			[]any{"guest_ip_network", "tap", "bridge", "nat", "inbound", "east_west", "direct_internet"}) {
		return fmt.Errorf("connectivity semantics drifted")
	}

	credential, ok := fixture["credential_store_proof"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(credential,
		"store", "use", "replay", "test_ref", "test_names",
	) ||
		credential["store"] != "CredentialStore" || credential["use"] != "single_use" ||
		credential["replay"] != "denied" ||
		credential["test_ref"] != "internal/adapters/agent/firecracker/networkauth/verifier_test.go" ||
		!reflect.DeepEqual(credential["test_names"], []any{
			"TestVerifierAuthorizesOnceAndRejectsIdenticalReplay",
			"TestVerifierConcurrentReplayHasSingleWinner",
		}) {
		return fmt.Errorf("CredentialStore single-use proof drifted")
	}
	attestor, ok := fixture["test_attestor_separation"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(attestor,
		"scope", "network", "vsock", "test_ref", "test_name",
	) ||
		attestor["scope"] != "unchanged_no_network_no_vsock" ||
		attestor["network"] != "absent" || attestor["vsock"] != "absent" ||
		attestor["test_ref"] != "internal/e2e/firecrackerattestor/supervisor_test.go" ||
		attestor["test_name"] != "TestValidatePhaseRejectsEveryRequiredPhysicalInvariant" {
		return fmt.Errorf("TestAttestor separation drifted")
	}
	absences, ok := fixture["catalog_absences"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(absences,
		"new_vertical", "new_capability", "new_acceptance_contract", "new_receipt", "guest_agent_claim",
	) {
		return fmt.Errorf("catalog absences missing")
	}
	for _, key := range []string{
		"new_vertical", "new_capability", "new_acceptance_contract", "new_receipt", "guest_agent_claim",
	} {
		if absences[key] != false {
			return fmt.Errorf("forbidden catalog/evidence claim %q", key)
		}
	}

	requiredTest, ok := fixture["required_test"].(map[string]any)
	const requiredTestCommand = "go test -mod=vendor -count=1 ./acceptance -run '^TestAgentFirecrackerSingleVM' && go test -mod=vendor -count=1 ./internal/adapters/agent/firecracker/networkauth -run '^(TestVerifierAuthorizesOnceAndRejectsIdenticalReplay|TestVerifierConcurrentReplayHasSingleWinner)$' && go test -mod=vendor -count=1 ./internal/e2e/firecrackerattestor -run '^TestValidatePhaseRejectsEveryRequiredPhysicalInvariant$'"
	if !ok || !agentFirecrackerHasExactKeys(requiredTest, "command", "reject_no_tests_to_run") ||
		requiredTest["command"] != requiredTestCommand ||
		requiredTest["reject_no_tests_to_run"] != true {
		return fmt.Errorf("required test drifted")
	}
	if !reflect.DeepEqual(agentFirecrackerStringSliceValue(fixture["decision_files"]), []string{
		"product/roadmap.json",
		"product_roadmap_test.go",
		"acceptance/fixtures/agent_firecracker_single_vm.json",
		"acceptance/agent_firecracker_single_vm_contract_test.go",
		"docs/corte_agente_firecracker_uno_2026-07-29.md",
		"docs/arquitectura_comunicacion_agentes_firecracker_2026-07-25.md",
		"docs/decision_atestacion_bubblewrap_microvm_2026-07-25.md",
		"docs/reconstruccion/corte_alcance_v23_firecracker_diferido_2026-07-26.md",
		"docs/reconstruccion/HANDOFF_PARADA_ORQUESTAV2_2026-07-26.md",
		"docs/inventario_bugs_orquesta_2026-06-30.md",
	}) {
		return fmt.Errorf("decision files drifted")
	}
	return nil
}

func agentFirecrackerHasExactKeys(object map[string]any, keys ...string) bool {
	if len(object) != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, exists := object[key]; !exists {
			return false
		}
	}
	return true
}

func agentFirecrackerStringSliceValue(value any) []string {
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	result := make([]string, len(raw))
	for index, item := range raw {
		text, stringValue := item.(string)
		if !stringValue {
			return nil
		}
		result[index] = text
	}
	return result
}

func agentFirecrackerReadObject(t *testing.T, path string) map[string]any {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		t.Fatalf("decode trailing %s: %v", path, err)
	}
	return value
}

func agentFirecrackerObjectSlice(t *testing.T, value any) []map[string]any {
	t.Helper()
	raw, ok := value.([]any)
	if !ok {
		t.Fatalf("value is not an array: %#v", value)
	}
	result := make([]map[string]any, len(raw))
	for index, item := range raw {
		entry, object := item.(map[string]any)
		if !object {
			t.Fatalf("array item %d is not an object: %#v", index, item)
		}
		result[index] = entry
	}
	return result
}

func agentFirecrackerHasID(value any, id string) bool {
	for _, entry := range value.([]any) {
		if strings.EqualFold(entry.(map[string]any)["id"].(string), id) {
			return true
		}
	}
	return false
}

func agentFirecrackerHasIDPrefix(value any, prefix string) bool {
	for _, entry := range value.([]any) {
		if strings.HasPrefix(entry.(map[string]any)["id"].(string), prefix) {
			return true
		}
	}
	return false
}

func agentFirecrackerStringSlice(t *testing.T, value any) []string {
	t.Helper()
	raw, ok := value.([]any)
	if !ok {
		t.Fatalf("value is not an array: %#v", value)
	}
	result := make([]string, len(raw))
	for index, item := range raw {
		text, stringValue := item.(string)
		if !stringValue {
			t.Fatalf("array item %d is not a string: %#v", index, item)
		}
		result[index] = text
	}
	return result
}

func agentFirecrackerSet(t *testing.T, root map[string]any, path []string, value any) {
	t.Helper()
	current := root
	for _, key := range path[:len(path)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("path %v does not address an object", path)
		}
		current = next
	}
	current[path[len(path)-1]] = value
}
