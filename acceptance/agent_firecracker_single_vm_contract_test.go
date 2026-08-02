package acceptance_test

import (
	"bytes"
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
const agentFirecrackerV38FixturePath = "acceptance/fixtures/v38_agent_runtime_elastic_plan.json"

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
		!agentFirecrackerHasID(roadmap["verticals"], "agent_runtime_elastic") ||
		agentFirecrackerHasID(roadmap["verticals"], "V39") ||
		agentFirecrackerHasIDPrefix(roadmap["acceptance_contracts"], "AC-V39") {
		t.Fatal("la caracterización neutral no está subordinada a V38 o creó V39")
	}
	v38 := agentFirecrackerReadObject(t, filepath.Join(root, agentFirecrackerV38FixturePath))
	alignment := fixture["v38_alignment"].(map[string]any)
	if alignment["contract_status"] != v38["status"] ||
		!reflect.DeepEqual(alignment["subgates"], v38["subgates"]) {
		t.Fatal("la caracterización neutral no coincide con los subgates canónicos V38")
	}
	vertical := agentFirecrackerObjectByID(t, roadmap["verticals"], "agent_runtime_elastic")
	if alignment["v23_dependency"] != "independent" ||
		!reflect.DeepEqual(vertical["depends_on"], v38["causal_dependencies"]) {
		t.Fatal("V38 dejó de ser independiente de V23 o derivaron sus dependencias")
	}
	contract := agentFirecrackerObjectByID(t, roadmap["acceptance_contracts"], "AC-V38-AGENT-RUNTIME-ELASTIC")
	if contract["status"] != "planned" {
		t.Fatalf("el contrato V38 dejó de estar planificado: %#v", contract)
	}
	if _, hasReceipt := contract["receipt"]; hasReceipt {
		t.Fatalf("el contrato V38 planificado anticipa receipt: %#v", contract)
	}
	v38AssertNoPrematureEvidence(t, filepath.Join(root, "product/evidence"))

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
		"product/evidence/v38_agent_runtime_elastic.json",
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
		{"grant_reusable", []string{"signed_grant_proof", "use"}, "reusable"},
		{"grant_replay", []string{"signed_grant_proof", "replay"}, "allowed"},
		{"attestor_network", []string{"test_attestor_separation", "network"}, "present"},
		{"attestor_vsock", []string{"test_attestor_separation", "vsock"}, "present"},
		{"example_agents_expanded", []string{"bounded_example", "agent_count"}, float64(2)},
		{"example_microvms_expanded", []string{"bounded_example", "microvm_count"}, float64(2)},
		{"shared_microvm_claimed", []string{"bounded_example", "isolation"}, "shared_microvm"},
		{"runtime_wiring_claimed", []string{"bounded_example", "runtime_wiring"}, "present"},
		{"physical_execution_claimed", []string{"bounded_example", "physical_execution"}, "passed"},
		{"vertical_changed", []string{"v38_alignment", "canonical_vertical"}, "V39"},
		{"contract_changed", []string{"v38_alignment", "acceptance_contract"}, "AC-V39"},
		{"contract_promoted", []string{"v38_alignment", "contract_status"}, "executable"},
		{"v23_made_dependency", []string{"v38_alignment", "v23_dependency"}, "required"},
		{"subgate_a_requires_kvm", []string{"v38_alignment", "subgates", "a", "kvm"}, "required"},
		{"subgate_b_fallback", []string{"v38_alignment", "subgates", "b", "selection"}, "automatic_fallback"},
		{"subgate_b_shared_vm", []string{"v38_alignment", "subgates", "b", "isolation"}, "shared_microvm"},
		{"subgate_c_other_candidate", []string{"v38_alignment", "subgates", "c", "subject"}, "different_candidate"},
		{"accreditation_without_all_subgates", []string{"v38_alignment", "subgates", "global_accreditation"}, "after_c"},
		{"receipt_created", []string{"v38_alignment", "new_receipt"}, true},
		{"evidence_created", []string{"v38_alignment", "new_evidence"}, true},
		{"v38_accredited", []string{"v38_alignment", "v38_accreditation_claim"}, true},
		{"guest_claimed", []string{"v38_alignment", "guest_agent_claim"}, true},
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

func TestAgentFirecrackerSingleVMRejectsInvalidJSON(t *testing.T) {
	base := agentFirecrackerReadBytes(t, filepath.Join(evidenceRepositoryRoot(t), agentFirecrackerSingleVMFixturePath))
	invalid := [][]byte{
		bytes.Replace(base, []byte("{\n"), []byte("{\n  \"unknown\": true,\n"), 1),
		bytes.Replace(base, []byte(`"schema_version": 1,`), []byte(`"schema_version": 1, "schema_version": 1,`), 1),
		append(append([]byte{}, base...), []byte("{}")...),
		bytes.Replace(base, []byte(`"schema_version": 1`), []byte(`"schema_version":1`), 1),
		bytes.Replace(base, []byte(`"agent_count": 1,`), []byte(`"unknown": true, "agent_count": 1,`), 1),
	}
	for _, data := range invalid {
		fixture, err := agentFirecrackerDecodeObject(data)
		if err == nil && agentFirecrackerValidateFixture(fixture) == nil {
			t.Fatal("JSON inválido o no canónico aceptado")
		}
	}
}

func agentFirecrackerValidateFixture(fixture map[string]any) error {
	if !agentFirecrackerHasExactKeys(fixture,
		"schema_version", "fixture_id", "contract_kind", "authority", "bounded_example",
		"connectivity", "signed_grant_proof", "test_attestor_separation",
		"v38_alignment", "required_test", "decision_files",
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
		"agent_count", "microvm_count", "isolation", "selection_scope", "runtime_wiring", "physical_execution",
	) ||
		bounded["agent_count"] != float64(1) ||
		bounded["microvm_count"] != float64(1) ||
		bounded["isolation"] != "one_microvm_per_agent" ||
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

	grant, ok := fixture["signed_grant_proof"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(grant,
		"authority", "use", "replay", "portable_fixture", "connector_test", "test_names",
	) ||
		grant["authority"] != "agentmicrovm_ed25519_public_trust" || grant["use"] != "single_use" ||
		grant["replay"] != "denied_durable_sqlite" ||
		grant["portable_fixture"] != "contratos/fixtures/lanzamiento_firmado_v1.json" ||
		grant["connector_test"] != "conectores/orquesta/concesion_test.go" ||
		!reflect.DeepEqual(grant["test_names"], []any{
			"TestContratoFirmadoCoincideConElVectorRust",
			"TestFirmanteVinculaAlcanceSinExportarReferenciasInternas",
		}) {
		return fmt.Errorf("signed grant proof drifted")
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
	alignment, ok := fixture["v38_alignment"].(map[string]any)
	if !ok || !agentFirecrackerHasExactKeys(alignment,
		"canonical_vertical", "acceptance_contract", "contract_status", "v23_dependency",
		"subgates", "new_capability", "new_receipt", "new_evidence",
		"v38_accreditation_claim", "guest_agent_claim",
	) {
		return fmt.Errorf("V38 alignment missing")
	}
	if alignment["canonical_vertical"] != "agent_runtime_elastic" ||
		alignment["acceptance_contract"] != "AC-V38-AGENT-RUNTIME-ELASTIC" ||
		alignment["contract_status"] != "planned" ||
		alignment["v23_dependency"] != "independent" ||
		!reflect.DeepEqual(alignment["subgates"], agentFirecrackerV38Subgates()) {
		return fmt.Errorf("canonical V38 alignment drifted")
	}
	for _, key := range []string{
		"new_capability", "new_receipt", "new_evidence",
		"v38_accreditation_claim", "guest_agent_claim",
	} {
		if alignment[key] != false {
			return fmt.Errorf("forbidden catalog/evidence claim %q", key)
		}
	}

	requiredTest, ok := fixture["required_test"].(map[string]any)
	const requiredTestCommand = "go test -mod=vendor -count=1 ./acceptance -run '^TestAgentFirecrackerSingleVM' && go test -mod=vendor -count=1 ./internal/ports -run '^TestAgentMicroVMNetworkAuthorityLivesOnlyInSignedSiblingContract$' && go test -mod=vendor -count=1 ./internal/e2e/firecrackerattestor -run '^TestValidatePhaseRejectsEveryRequiredPhysicalInvariant$'"
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
		"acceptance/fixtures/v38_agent_runtime_elastic_plan.json",
		"acceptance/v38_agent_runtime_elastic_plan_test.go",
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

func agentFirecrackerV38Subgates() map[string]any {
	return map[string]any{
		"order":                []any{"A", "B", "C"},
		"global_accreditation": "only_after_a_b_c_pass_on_same_candidate",
		"a": map[string]any{
			"scope":         "neutral_elastic_core",
			"kvm":           "not_required",
			"firecracker":   "not_required",
			"status_effect": "cannot_accredit_v38",
		},
		"b": map[string]any{
			"scope":         "firecracker_agent_adapter",
			"selection":     "opt_in_explicit_no_fallback",
			"isolation":     "one_microvm_per_agent",
			"status_effect": "cannot_accredit_v38",
		},
		"c": map[string]any{
			"scope":         "real_physical_wave",
			"subject":       "same_candidate_as_a_and_b",
			"status_effect": "accredit_v38_only_after_a_b_c",
		},
	}
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
	data := agentFirecrackerReadBytes(t, path)
	value, err := agentFirecrackerDecodeObject(data)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func agentFirecrackerReadBytes(t *testing.T, path string) []byte {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("fichero JSON no regular %q: %v", path, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func agentFirecrackerDecodeObject(data []byte) (map[string]any, error) {
	if err := v38RejectDuplicateJSON(data); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var value map[string]any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("datos posteriores al JSON: %v", err)
	}
	var canonical bytes.Buffer
	if err := json.Indent(&canonical, bytes.TrimSpace(data), "", "  "); err != nil {
		return nil, err
	}
	if !bytes.Equal(data, append(canonical.Bytes(), '\n')) {
		return nil, fmt.Errorf("JSON no canónico")
	}
	return value, nil
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

func agentFirecrackerObjectByID(t *testing.T, value any, id string) map[string]any {
	t.Helper()
	for _, entry := range agentFirecrackerObjectSlice(t, value) {
		if entry["id"] == id {
			return entry
		}
	}
	t.Fatalf("no existe el objeto %q", id)
	return nil
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
