package acceptance_test

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/adapters/state/sqlite"
	"orquesta/internal/ports"
)

const v38DockerProviderFixturePath = "acceptance/fixtures/v38_docker_provider_decision.json"

var (
	_ func(*microvm.Cliente, context.Context) (microvm.RespuestaCapacidades, error)                                                                            = (*microvm.Cliente).NegociarDocker
	_ func(*microvm.FirmanteConcesiones, microvm.PlanLanzamientoContenedorV1, time.Time, time.Duration) (microvm.SolicitudLanzarORecuperarContenedorV1, error) = (*microvm.FirmanteConcesiones).PrepararContenedor
	_ func(*microvm.Cliente, context.Context, string, json.RawMessage) (microvm.RespuestaContenedorV1, error)                                                  = (*microvm.Cliente).LanzarORecuperarContenedorCodificada
	_ func(*microvm.Cliente, context.Context, string) (microvm.RespuestaOperacionContenedorV1, error)                                                          = (*microvm.Cliente).ConsultarOperacionContenedor
	_ func(*microvm.Cliente, context.Context, string, string, json.RawMessage) (microvm.RespuestaInicioSesionContenedorV1, error)                              = (*microvm.Cliente).IniciarSesionContenedorCodificada
	_ func(*microvm.Cliente, context.Context, string, string, string, json.RawMessage) (microvm.RespuestaEntradaSesionContenedorV1, error)                     = (*microvm.Cliente).EnviarEntradaSesionContenedorCodificada
	_ func(*microvm.Cliente, context.Context, string) (microvm.RespuestaContenedorV1, error)                                                                   = (*microvm.Cliente).ObservarContenedor
	_ func(*microvm.Cliente, context.Context, string, string, microvm.ConsultaEventosSesionContenedorV1) (microvm.RespuestaEventosSesionContenedorV1, error)   = (*microvm.Cliente).LeerEventosSesionContenedor
	_ ports.AgentProviderRequestJournal                                                                                                                        = (*sqlite.Repository)(nil)
)

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
	v38DockerAssertClientAdoptionReady(t, root, fixture["client_adoption_gate"].(map[string]any))
	v38DockerAssertCompositionBlockerReferences(t, root, fixture["composition_gate"].(map[string]any))
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
		{"launcher_ausente", []string{"implementation_state", "neutral_agent_launcher"}, "absent"},
		{"observador_ausente", []string{"implementation_state", "neutral_agent_observer"}, "absent"},
		{"journal_ausente", []string{"implementation_state", "prepared_request_journal"}, "absent"},
		{"cliente_bloqueado", []string{"client_adoption_gate", "status"}, "blocked"},
		{"superficie_observer_ausente", []string{"client_adoption_gate", "required_symbols"}, []any{"NegociarDocker"}},
		{"composicion_anticipada", []string{"composition_gate", "status"}, "satisfied"},
		{"contrato_neutral_ausente", []string{"composition_gate", "neutral_application_contract"}, "absent"},
		{"autoridad_incompleta", []string{"composition_gate", "required_application_authorities"}, []any{"execution_workspace_ref"}},
		{"bloqueo_externo_oculto", []string{"composition_gate", "external_unblock_dependency"}, "none"},
		{"workaround_compartido", []string{"composition_gate", "forbidden_resolutions"}, []any{"shared_database"}},
		{"dependencia_obsoleta", []string{"client_adoption_gate", "next_safe_dependency"}, "v38_docker_opt_in_runtime_composition"},
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
	if !agentFirecrackerHasExactKeys(fixture, "schema_version", "fixture_id", "decision", "v38_alignment", "implementation_state", "composition_gate", "client_adoption_gate") ||
		fixture["schema_version"] != float64(7) || fixture["fixture_id"] != "v38_docker_provider_decision" {
		return false
	}
	decision, decisionOK := fixture["decision"].(map[string]any)
	alignment, alignmentOK := fixture["v38_alignment"].(map[string]any)
	state, stateOK := fixture["implementation_state"].(map[string]any)
	composition, compositionOK := fixture["composition_gate"].(map[string]any)
	gate, gateOK := fixture["client_adoption_gate"].(map[string]any)
	if !decisionOK || !alignmentOK || !stateOK || !compositionOK || !gateOK ||
		!agentFirecrackerHasExactKeys(decision, "id", "status", "capability_refs", "provider", "transport", "adapter_boundary", "selection", "client_adoption", "engine_owner", "orquesta_docker_socket", "forbidden_sharing", "fallback", "v38_accreditation", "test_refs") ||
		!agentFirecrackerHasExactKeys(alignment, "canonical_vertical", "capability_id", "capability_status", "acceptance_contract", "contract_status", "firecracker_required_subgates", "firecracker_global_accreditation", "docker_accreditation", "new_capability", "new_acceptance_contract", "new_receipt", "new_evidence") ||
		!agentFirecrackerHasExactKeys(state, "runtime_wiring", "neutral_agent_launcher", "neutral_agent_observer", "docker_client_usage", "prepared_request_journal", "docker_engine_smoke", "local_replace") ||
		!agentFirecrackerHasExactKeys(composition, "status", "neutral_application_contract", "required_application_authorities", "published_contract_gap", "external_unblock_dependency", "forbidden_resolutions", "test_refs") ||
		!agentFirecrackerHasExactKeys(gate, "status", "pinned_module_version", "module_zip_sum", "module_source", "observed_external_candidate_commit", "required_symbols", "unblock", "forbidden_resolutions", "next_safe_dependency") {
		return false
	}
	return decision["id"] == "agent_runtime_docker_provider" && decision["status"] == "implemented" &&
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
		state["runtime_wiring"] == "absent" && state["neutral_agent_launcher"] == "implemented_reviewed_not_runtime_wired" &&
		state["neutral_agent_observer"] == "implemented_reviewed_not_runtime_wired" && state["docker_client_usage"] == "published_module_used_by_adapter_not_runtime_wired" &&
		state["prepared_request_journal"] == "same_state_repository_used_by_adapter_not_runtime_wired" &&
		state["docker_engine_smoke"] == "not_claimed" && state["local_replace"] == "forbidden" &&
		composition["status"] == "blocked_external_contract_required" && composition["neutral_application_contract"] == "implemented_port_contract_not_runtime_wired_v2" && reflect.DeepEqual(composition["required_application_authorities"], []any{
			"execution_workspace_ref", "artifact_access_ref", "mcp_access_ref", "mailbox_endpoint_ref", "egress_authority",
		}) && composition["published_contract_gap"] == "docker_client_missing_complete_isolated_application_authority_delegation" &&
		composition["external_unblock_dependency"] == "v38_docker_published_isolated_session_delegation" &&
		reflect.DeepEqual(composition["forbidden_resolutions"], []any{"drop_runtime_authority", "shared_database", "shared_filesystem", "shared_secrets", "orquesta_docker_socket", "silent_fallback"}) &&
		reflect.DeepEqual(composition["test_refs"], []any{"internal/ports/microvm_session_broker_test.go", "internal/application/execution_session_test.go", "internal/adapters/agent/agentmicrovm/docker_adapter_test.go"}) &&
		gate["status"] == "satisfied_published_module_ready_for_neutral_adapter" && gate["pinned_module_version"] == "v0.0.0-20260814005716-2768389c82c0" &&
		gate["module_zip_sum"] == "h1:YRl+uPkDB0Glq34Hqq/4D7ZLu+LQa5LgCHl1WwlJars=" &&
		gate["module_source"] == "authorized_local_gomodcache_private_zip" &&
		gate["observed_external_candidate_commit"] == "2768389c82c02e4eeb3599152037fe3a52e2dc76" && reflect.DeepEqual(gate["required_symbols"], []any{
		"NegociarDocker", "PrepararContenedor", "LanzarORecuperarContenedorCodificada", "ConsultarOperacionContenedor",
		"IniciarSesionContenedorCodificada", "EnviarEntradaSesionContenedorCodificada", "ObservarContenedor", "LeerEventosSesionContenedor",
	}) &&
		gate["unblock"] == "satisfied_published_versioned_module_available_in_authorized_local_supply_chain" && reflect.DeepEqual(gate["forbidden_resolutions"], []any{"copy_external_worktree", "local_replace", "go_work", "private_module_remote_download"}) &&
		gate["next_safe_dependency"] == "v38_docker_stop_durable_intent_contract"
}

func v38DockerAssertCompositionBlockerReferences(t *testing.T, root string, composition map[string]any) {
	t.Helper()
	for _, ref := range composition["test_refs"].([]any) {
		path, ok := ref.(string)
		if !ok {
			t.Fatalf("referencia de gate Docker inválida: %#v", ref)
		}
		if info, err := os.Stat(filepath.Join(root, path)); err != nil || info.IsDir() {
			t.Fatalf("referencia de gate Docker ausente %q: %v", path, err)
		}
	}
}

func v38DockerAssertClientAdoptionReady(t *testing.T, root string, gate map[string]any) {
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
	version, versionOK := gate["pinned_module_version"].(string)
	zipSum, zipSumOK := gate["module_zip_sum"].(string)
	if !versionOK || !zipSumOK {
		t.Fatal("el gate de adopción debe fijar versión y hash del módulo")
	}
	exact := v38DockerClientModule + " " + version
	goSum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(goSum), exact+" "+zipSum) != 1 {
		t.Fatalf("go.sum no acredita exactamente el zip publicado %q", exact)
	}
	modules, err := os.ReadFile(filepath.Join(root, "vendor", "modules.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(modules), "# "+exact+"\n") != 1 {
		t.Fatalf("vendor/modules.txt no fija exactamente %q", exact)
	}
	vendorInfo, err := os.Stat(filepath.Join(root, "vendor", filepath.FromSlash(v38DockerClientModule)))
	if err != nil || !vendorInfo.IsDir() || !pinned {
		t.Fatalf("el módulo publicado no coincide con go.mod/vendor: pinned=%v err=%v", pinned, err)
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
