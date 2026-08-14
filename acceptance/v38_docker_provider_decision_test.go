package acceptance_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const v38DockerProviderFixturePath = "acceptance/fixtures/v38_docker_provider_decision.json"

func TestV38DockerProviderDecisionMatchesCanonicalRoadmap(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	fixture := agentFirecrackerReadObject(t, filepath.Join(root, v38DockerProviderFixturePath))
	if !v38DockerProviderSemanticsValid(fixture) {
		t.Fatal("la fixture Docker no conserva el corte opt-in no acreditante")
	}

	roadmap := agentFirecrackerReadObject(t, filepath.Join(root, "product/roadmap.json"))
	decisions := agentFirecrackerObjectSlice(t, roadmap["implementation_decisions"])
	if len(decisions) != 2 || decisions[0]["id"] != "agent_microvm_network" || decisions[1]["id"] != "agent_runtime_docker_provider" {
		t.Fatalf("decisiones de implementación inesperadas: %#v", decisions)
	}
	if !reflect.DeepEqual(decisions[1], fixture["decision"]) {
		t.Fatalf("la decisión Docker no coincide con roadmap:\n got: %#v\nwant: %#v", decisions[1], fixture["decision"])
	}

	alignment := fixture["v38_alignment"].(map[string]any)
	capability := agentFirecrackerObjectByID(t, roadmap["capability_entries"], "ORC-28")
	contract := agentFirecrackerObjectByID(t, roadmap["acceptance_contracts"], "AC-V38-AGENT-RUNTIME-ELASTIC")
	if capability["status"] != alignment["capability_status"] || len(capability["evidence_refs"].([]any)) != 0 ||
		contract["status"] != alignment["contract_status"] {
		t.Fatal("Docker anticipó implementación, evidencia o acreditación de V38")
	}
	if _, exists := contract["receipt"]; exists {
		t.Fatal("el contrato V38 planificado anticipa receipt")
	}
	v38 := agentFirecrackerReadObject(t, filepath.Join(root, agentFirecrackerV38FixturePath))
	subgates := v38["subgates"].(map[string]any)
	if !reflect.DeepEqual(alignment["firecracker_required_subgates"], subgates["order"]) ||
		alignment["firecracker_global_accreditation"] != subgates["global_accreditation"] {
		t.Fatal("la decisión Docker relajó los subgates Firecracker A+B+C")
	}
	v38AssertNoPrematureEvidence(t, filepath.Join(root, "product/evidence"))
	v38DockerAssertPublishedModuleWithoutReplace(t, root)
	v38DockerAssertClientAdoptionBlocked(t, root, fixture["client_adoption_gate"].(map[string]any))
	v38DockerAssertNoDirectRuntime(t, root)
}

func TestV38DockerProviderDecisionRejectsSemanticDrift(t *testing.T) {
	root := evidenceRepositoryRoot(t)
	cases := []struct {
		name  string
		path  []string
		value any
	}{
		{"aplicado", []string{"decision", "status"}, "applied"},
		{"fuera_de_puertos", []string{"decision", "adapter_boundary"}, "docker_adapter_private_api"},
		{"fallback", []string{"decision", "selection"}, "automatic_fallback"},
		{"cliente_local", []string{"decision", "client_adoption"}, "local_replace"},
		{"motor_en_orquesta", []string{"decision", "engine_owner"}, "orquesta"},
		{"socket_directo", []string{"decision", "orquesta_docker_socket"}, "allowed"},
		{"db_compartida", []string{"decision", "forbidden_sharing"}, []any{"filesystem", "secrets"}},
		{"fallback_permitido", []string{"decision", "fallback"}, "allowed"},
		{"acredita_v38", []string{"decision", "v38_accreditation"}, "accredit_v38"},
		{"firecracker_incompleto", []string{"v38_alignment", "firecracker_required_subgates"}, []any{"A", "B"}},
		{"evidencia_nueva", []string{"v38_alignment", "new_evidence"}, true},
		{"wiring_anticipado", []string{"implementation_state", "runtime_wiring"}, "present"},
		{"cliente_disponible", []string{"client_adoption_gate", "status"}, "ready"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			fixture := agentFirecrackerReadObject(t, filepath.Join(root, v38DockerProviderFixturePath))
			agentFirecrackerSet(t, fixture, test.path, test.value)
			if v38DockerProviderSemanticsValid(fixture) {
				t.Fatal("la mutación contractual fue aceptada")
			}
		})
	}
	for _, invalid := range []string{
		"require " + v38DockerClientModule + " v1.2.3\nreplace\t" + v38DockerClientModule + " => ../local\n",
		"require " + v38DockerClientModule + " v1.2.3\nreplace (\n  " + v38DockerClientModule + " => ../local\n)\n",
	} {
		if v38DockerPublishedModuleWithoutReplace(invalid) {
			t.Fatal("un replace local con whitespace válido fue aceptado")
		}
	}
}

func v38DockerProviderSemanticsValid(fixture map[string]any) bool {
	if !agentFirecrackerHasExactKeys(fixture, "schema_version", "fixture_id", "decision", "v38_alignment", "implementation_state", "client_adoption_gate") ||
		fixture["schema_version"] != float64(2) || fixture["fixture_id"] != "v38_docker_provider_decision" {
		return false
	}
	decision, decisionOK := fixture["decision"].(map[string]any)
	alignment, alignmentOK := fixture["v38_alignment"].(map[string]any)
	state, stateOK := fixture["implementation_state"].(map[string]any)
	gate, gateOK := fixture["client_adoption_gate"].(map[string]any)
	if !decisionOK || !alignmentOK || !stateOK || !gateOK ||
		!agentFirecrackerHasExactKeys(decision, "id", "status", "capability_refs", "provider", "transport", "adapter_boundary", "selection", "client_adoption", "engine_owner", "orquesta_docker_socket", "forbidden_sharing", "fallback", "v38_accreditation", "test_refs") ||
		!agentFirecrackerHasExactKeys(alignment, "canonical_vertical", "capability_id", "capability_status", "acceptance_contract", "contract_status", "firecracker_required_subgates", "firecracker_global_accreditation", "docker_accreditation", "new_capability", "new_acceptance_contract", "new_receipt", "new_evidence") ||
		!agentFirecrackerHasExactKeys(state, "runtime_wiring", "docker_client_usage", "docker_engine_smoke", "local_replace") ||
		!agentFirecrackerHasExactKeys(gate, "status", "pinned_module_version", "observed_external_candidate_commit", "required_symbols", "unblock", "forbidden_resolutions", "next_safe_dependency") {
		return false
	}
	return decision["id"] == "agent_runtime_docker_provider" && decision["status"] == "planned_not_applied" &&
		reflect.DeepEqual(decision["capability_refs"], []any{"ORC-28"}) && decision["provider"] == "docker" &&
		decision["transport"] == "agentmicrovm.local.v1_http1_over_unix_socket" &&
		decision["adapter_boundary"] == "agent_launcher_and_existing_neutral_agent_ports_only" &&
		decision["selection"] == "opt_in_explicit_no_fallback" &&
		decision["client_adoption"] == "published_versioned_agente_microvm_go_module_only_no_local_replace" &&
		decision["engine_owner"] == "agente_microvm_sibling_process_only" && decision["orquesta_docker_socket"] == "forbidden" &&
		reflect.DeepEqual(decision["forbidden_sharing"], []any{"database", "filesystem", "secrets"}) && decision["fallback"] == "forbidden" &&
		decision["v38_accreditation"] == "none_firecracker_a_b_c_remain_mandatory" &&
		reflect.DeepEqual(decision["test_refs"], []any{"acceptance/v38_docker_provider_decision_test.go"}) &&
		alignment["canonical_vertical"] == "agent_runtime_elastic" && alignment["capability_id"] == "ORC-28" && alignment["capability_status"] == "declared" &&
		alignment["acceptance_contract"] == "AC-V38-AGENT-RUNTIME-ELASTIC" && alignment["contract_status"] == "planned" &&
		reflect.DeepEqual(alignment["firecracker_required_subgates"], []any{"A", "B", "C"}) && alignment["firecracker_global_accreditation"] == "only_after_a_b_c_pass_on_same_candidate" &&
		alignment["docker_accreditation"] == "none" && alignment["new_capability"] == false && alignment["new_acceptance_contract"] == false && alignment["new_receipt"] == false && alignment["new_evidence"] == false &&
		state["runtime_wiring"] == "absent" && state["docker_client_usage"] == "absent_in_this_cut" && state["docker_engine_smoke"] == "not_claimed" && state["local_replace"] == "forbidden" &&
		gate["status"] == "blocked_current_pinned_module_lacks_docker_contract" && gate["pinned_module_version"] == "v0.0.0-20260805221511-928ef309a173" &&
		gate["observed_external_candidate_commit"] == "2768389c82c02e4eeb3599152037fe3a52e2dc76" && reflect.DeepEqual(gate["required_symbols"], []any{"NegociarDocker", "PrepararContenedor", "LanzarORecuperarContenedor"}) &&
		gate["unblock"] == "published_versioned_module_available_in_authorized_local_supply_chain" && reflect.DeepEqual(gate["forbidden_resolutions"], []any{"copy_external_worktree", "local_replace", "go_work", "remote_download"}) &&
		gate["next_safe_dependency"] == "v38_firecracker_b_local_contract_gates"
}

func v38DockerAssertClientAdoptionBlocked(t *testing.T, root string, gate map[string]any) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	pinned := false
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		pinned = pinned || len(fields) >= 2 && fields[0] == v38DockerClientModule && fields[1] == gate["pinned_module_version"]
	}
	entries, err := os.ReadDir(filepath.Join(root, "vendor", filepath.FromSlash(v38DockerClientModule)))
	if err != nil || !pinned {
		t.Fatalf("el módulo bloqueado no coincide con go.mod/vendor: pinned=%v err=%v", pinned, err)
	}
	var source strings.Builder
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
			content, readErr := os.ReadFile(filepath.Join(root, "vendor", filepath.FromSlash(v38DockerClientModule), entry.Name()))
			if readErr != nil {
				t.Fatal(readErr)
			}
			source.Write(content)
		}
	}
	for _, symbol := range agentFirecrackerStringSlice(t, gate["required_symbols"]) {
		if strings.Contains(source.String(), symbol) {
			t.Fatalf("el módulo fijado ya contiene %s pero el bloqueo sigue abierto", symbol)
		}
	}
}

func v38DockerAssertPublishedModuleWithoutReplace(t *testing.T, root string) {
	t.Helper()
	for _, relative := range []string{"go.work", "go.work.sum"} {
		if _, err := os.Lstat(filepath.Join(root, relative)); !os.IsNotExist(err) {
			t.Fatalf("%s puede reemplazar el cliente publicado fuera de go.mod: %v", relative, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !v38DockerPublishedModuleWithoutReplace(string(data)) {
		t.Fatal("el contrato agente_microvm debe ser publicado, versionado y sin replace local")
	}
}

const v38DockerClientModule = "github.com/aavidad/agente_microvm/conectores/orquesta"

func v38DockerPublishedModuleWithoutReplace(data string) bool {
	versioned, replaceBlock := false, false
	for _, line := range strings.Split(data, "\n") {
		fields := strings.Fields(strings.SplitN(line, "//", 2)[0])
		if len(fields) == 0 {
			continue
		}
		if replaceBlock {
			if fields[0] == ")" {
				replaceBlock = false
			} else if strings.Trim(fields[0], "\"") == v38DockerClientModule {
				return false
			}
			continue
		}
		if fields[0] == "replace" {
			if len(fields) == 2 && fields[1] == "(" {
				replaceBlock = true
			} else if len(fields) > 1 && strings.Trim(fields[1], "\"") == v38DockerClientModule {
				return false
			}
			continue
		}
		versioned = versioned || len(fields) >= 2 && strings.Trim(fields[0], "\"") == v38DockerClientModule && strings.HasPrefix(fields[1], "v")
	}
	return versioned
}

func v38DockerAssertNoDirectRuntime(t *testing.T, root string) {
	t.Helper()
	for _, relative := range []string{"internal", "cmd/orquesta"} {
		err := filepath.WalkDir(filepath.Join(root, relative), func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return err
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, forbidden := range []string{"docker.sock", "DOCKER_HOST", "github.com/docker/docker"} {
				if strings.Contains(string(data), forbidden) {
					t.Errorf("acceso Docker directo %q en %s", forbidden, path)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
