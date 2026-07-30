package acceptance_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const v38AgentRuntimeElasticFixture = "acceptance/fixtures/v38_agent_runtime_elastic_plan.json"

var v38ExactTests = []string{
	"TestV38AgentRuntimeElasticPlanMatchesCanonicalRoadmap",
	"TestV38AgentRuntimeElasticPlanRejectsSemanticDrift",
	"TestV38AgentRuntimeElasticPlanRejectsInvalidJSON",
	"TestV38AgentRuntimeElasticPlanRequiresExactRunPassEvidence",
}

func TestV38AgentRuntimeElasticPlanMatchesCanonicalRoadmap(t *testing.T) {
	fixture := v38Fixture(t)
	if !v38SemanticsValid(fixture) {
		t.Fatal("la fixture V38 no conserva el contrato completo")
	}
	root := evidenceRepositoryRoot(t)
	var roadmap struct {
		CatalogSize       int            `json:"catalog_size"`
		OperatorDecisions map[string]any `json:"operator_decisions"`
		Verticals         []struct {
			ID string `json:"id"`
		} `json:"verticals"`
		AcceptanceContracts []struct {
			ID, Vertical, Status, Command string
			Assertions                    []string `json:"assertions"`
		} `json:"acceptance_contracts"`
		CapabilityEntries []struct {
			ID           string `json:"id"`
			OwnerContext string `json:"owner_context"`
		} `json:"capability_entries"`
	}
	data, err := os.ReadFile(filepath.Join(root, "product/roadmap.json"))
	if err != nil || json.Unmarshal(data, &roadmap) != nil {
		t.Fatalf("roadmap V38 ilegible: %v", err)
	}
	if roadmap.CatalogSize != 257 || len(roadmap.Verticals) != 38 ||
		len(roadmap.AcceptanceContracts) != 38 ||
		roadmap.OperatorDecisions["priority_vertical"] != "agent_runtime_elastic" {
		t.Fatal("V38 alteró catálogo, conteos o prioridad")
	}
	foundContract, owned := false, []string{}
	for _, contract := range roadmap.AcceptanceContracts {
		if contract.ID == fixture.ContractID {
			foundContract = contract.Vertical == "agent_runtime_elastic" && contract.Status == "planned"
			for _, name := range v38ExactTests {
				if !strings.Contains(contract.Command, name) {
					t.Fatalf("el comando planned omite %s", name)
				}
			}
			assertions := strings.Join(contract.Assertions, "\n")
			for _, marker := range []string{
				"capacidad física reservable y la cuota del proveedor como hechos separados",
				"cuenta y la colocación opacas se eligen antes de la reclamación atómica",
				"runtime.codex.max_concurrent_executions es solo un guardarraíl local del conector Codex",
				"todos los hilos turnos y agentes Codex viven dentro de la microVM",
				"puerto StateRepository con una sola fuente transaccional activa",
				"PostgreSQL es el adaptador productivo futuro",
			} {
				if !strings.Contains(assertions, marker) {
					t.Fatalf("el contrato V38 omite la regla planificada %q", marker)
				}
			}
		}
	}
	for _, entry := range roadmap.CapabilityEntries {
		if entry.OwnerContext == "agent_runtime_elastic" {
			owned = append(owned, entry.ID)
		}
	}
	if !foundContract || !reflect.DeepEqual(owned, []string{"ORC-28"}) {
		t.Fatalf("contrato o propiedad V38 inválidos: contrato=%v propiedad=%v", foundContract, owned)
	}
	v38AssertNoPrematureEvidence(t, filepath.Join(root, "product/evidence"))
}

func TestV38AgentRuntimeElasticPlanRejectsSemanticDrift(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*v38ElasticFixture)
	}{
		{"propiedad_global", func(f *v38ElasticFixture) { f.CapabilityOwnership = append(f.CapabilityOwnership, "OPS-16") }},
		{"capacidad_fisica_derivada_de_cuota", func(f *v38ElasticFixture) { f.Capacity.PhysicalCapacity = "provider_quota" }},
		{"cuota_desconocida_abierta", func(f *v38ElasticFixture) { f.Capacity.UnknownQuota = "unlimited" }},
		{"cuota_agotada_libera_hueco", func(f *v38ElasticFixture) { f.Capacity.ExhaustedQuota = "release_physical_slot" }},
		{"cuota_convertida_en_capacidad", func(f *v38ElasticFixture) { f.Capacity.ProviderQuota = "physical_slot_count" }},
		{"reserva_sin_cerca", func(f *v38ElasticFixture) { f.Capacity.Reservation = "atomic_without_fence" }},
		{"colocacion_despues_de_reclamar", func(f *v38ElasticFixture) { f.Placement.Selection = "selected_after_claim" }},
		{"reclamacion_no_fija_colocacion", func(f *v38ElasticFixture) { f.Placement.Claim = "placement_not_bound" }},
		{"lanzador_reselecciona", func(f *v38ElasticFixture) { f.Placement.Launcher = "may_reselect_account" }},
		{"kvm_en_nucleo", func(f *v38ElasticFixture) { f.Subgates.A.KVM = "required" }},
		{"firecracker_con_fallback", func(f *v38ElasticFixture) { f.Subgates.B.Selection = "automatic_fallback" }},
		{"cierre_sin_a_b", func(f *v38ElasticFixture) { f.Subgates.C.StatusEffect = "accredit_v38" }},
		{"demanda_truncada", func(f *v38ElasticFixture) { f.Demand.LogicalCohorts = []int{1, 16, 70} }},
		{"salta_escalon_10", func(f *v38ElasticFixture) { f.Demand.PhysicalSteps = []int{1, 5, 16, 20} }},
		{"stop_sin_prioridad", func(f *v38ElasticFixture) { f.Scheduling.StopPriority = "fifo" }},
		{"observe_bloqueado", func(f *v38ElasticFixture) { f.Scheduling.ObserveProgress = "optional" }},
		{"admision_global_no_atomica", func(f *v38ElasticFixture) { f.Scheduling.GlobalAdmission = "best_effort" }},
		{"techo_global_codex", func(f *v38ElasticFixture) { f.Scheduling.GlobalCeiling = "codex_connector_limit" }},
		{"limite_codex_gobierna_despacho", func(f *v38ElasticFixture) {
			f.Scheduling.RuntimeCodexMaxConcurrentExecutions = "global_dispatch_budget"
		}},
		{"selector_privado", func(f *v38ElasticFixture) { f.Scheduling.PrivateLaunchOnlySelector = "allowed" }},
		{"trabajadores_ociosos", func(f *v38ElasticFixture) { f.Scheduling.IdleWorkerGoroutines = "allowed" }},
		{"turno_codex_en_anfitrion", func(f *v38ElasticFixture) { f.CodexIsolation.GuestExecution = "host_turn_allowed" }},
		{"app_server_anfitrion_completo", func(f *v38ElasticFixture) { f.CodexIsolation.HostAppServer = "full_runtime" }},
		{"app_server_anfitrion_ejecuta_agente", func(f *v38ElasticFixture) { f.CodexIsolation.HostForbiddenScope = "none" }},
		{"app_server_anfitrion_gobierna_lifecycle", func(f *v38ElasticFixture) { f.CodexIsolation.HostAuthority = "agent_lifecycle" }},
		{"sqlite_sustituye_puerto", func(f *v38ElasticFixture) { f.StateRepository.Port = "SQLite" }},
		{"sqlite_productivo", func(f *v38ElasticFixture) { f.StateRepository.SQLite = "production" }},
		{"postgresql_no_productivo", func(f *v38ElasticFixture) { f.StateRepository.PostgreSQL = "optional_local" }},
		{"dos_fuentes_activas", func(f *v38ElasticFixture) { f.StateRepository.ActiveSource = "sqlite_and_postgresql" }},
		{"escritura_doble", func(f *v38ElasticFixture) { f.StateRepository.DualWrite = "allowed" }},
		{"parada_cohorte", func(f *v38ElasticFixture) { f.Shutdown.Scope = "whole_cohort" }},
		{"borrado", func(f *v38ElasticFixture) { f.Preservation.AutomaticDeletion = true }},
		{"reintento_incierto", func(f *v38ElasticFixture) { f.Recovery.Retry = "unknown_applied_allowed" }},
		{"latencia_omitida", func(f *v38ElasticFixture) { f.MeasuredLatencies = f.MeasuredLatencies[:6] }},
		{"evidencia_anticipada", func(f *v38ElasticFixture) { f.CatalogGuards.EvidenceAbsentWhilePlanned = false }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := v38Fixture(t)
			test.mutate(&got)
			decoded, err := v38DecodeFixture(v38CanonicalBytes(t, got))
			if err != nil {
				t.Fatal(err)
			}
			if v38SemanticsValid(decoded) {
				t.Fatal("la mutación semántica fue aceptada")
			}
		})
	}
	if !v38IsPrematureEvidence("otra_V38.json", nil) ||
		!v38IsPrematureEvidence("otra.json", []byte(`{"vertical":"agent_runtime_elastic"}`)) {
		t.Fatal("el cierre de evidencia V38 no cubre nombres y registros")
	}
}

func TestV38AgentRuntimeElasticPlanRejectsInvalidJSON(t *testing.T) {
	base := v38ReadFixtureBytes(t, filepath.Join(evidenceRepositoryRoot(t), v38AgentRuntimeElasticFixture))
	invalid := [][]byte{
		bytes.Replace(base, []byte("{\n"), []byte("{\n  \"desconocida\": true,\n"), 1),
		bytes.Replace(base, []byte(`"schema_version": 2,`), []byte(`"schema_version": 2, "schema_version": 2,`), 1),
		append(append([]byte{}, base...), []byte("{}")...),
		bytes.Replace(base, []byte(`"schema_version": 2`), []byte(`"schema_version":2`), 1),
		bytes.Replace(base, []byte(`"logical_cohorts": [`), []byte(`"desconocida": true, "logical_cohorts": [`), 1),
	}
	for _, data := range invalid {
		if _, err := v38DecodeFixture(data); err == nil {
			t.Fatal("JSON V38 inválido aceptado")
		}
	}
}

func TestV38AgentRuntimeElasticPlanRequiresExactRunPassEvidence(t *testing.T) {
	var exact strings.Builder
	for _, name := range v38ExactTests {
		exact.WriteString(`{"Action":"run","Test":"` + name + "\"}\n")
		exact.WriteString(`{"Action":"pass","Test":"` + name + "\"}\n")
	}
	packagePass := `{"Action":"pass","Package":"orquesta/acceptance"}`
	missing := strings.Replace(exact.String(), `"Action":"pass"`, `"Action":"output"`, 1)
	if !v38HasExactRunPass(exact.String(), v38ExactTests) ||
		v38HasExactRunPass(packagePass, v38ExactTests) ||
		v38HasExactRunPass(missing, v38ExactTests) {
		t.Fatal("el ratchet run/pass acepta un paquete verde sin cada prueba exacta")
	}
}

func v38Fixture(t *testing.T) v38ElasticFixture {
	t.Helper()
	data := v38ReadFixtureBytes(t, filepath.Join(evidenceRepositoryRoot(t), v38AgentRuntimeElasticFixture))
	fixture, err := v38DecodeFixture(data)
	if err != nil {
		t.Fatal(err)
	}
	return fixture
}

func v38SemanticsValid(f v38ElasticFixture) bool {
	identity := f.SchemaVersion == 2 && f.FixtureID == "v38_agent_runtime_elastic_plan" && f.ContractID == "AC-V38-AGENT-RUNTIME-ELASTIC" &&
		f.Status == "planned" && f.Priority == "operator_priority_immediate" && reflect.DeepEqual(f.CapabilityOwnership, []string{"ORC-28"})
	nonOwned := reflect.DeepEqual(f.NonOwnedRequirements.Capabilities, []string{"ORC-15", "OPS-16", "OPS-17"}) &&
		strings.Join([]string{f.NonOwnedRequirements.MessagesContinuity, f.NonOwnedRequirements.ExactAgentStop, f.NonOwnedRequirements.AgentEnvironmentPreservation}, "|") ==
			"required_behavior_without_accrediting_orc_15|required_behavior_without_accrediting_ops_16|required_behavior_without_accrediting_ops_17"
	gates := reflect.DeepEqual(f.Subgates.Order, []string{"A", "B", "C"}) && f.Subgates.GlobalAccreditation == "only_after_a_b_c_pass_on_same_candidate" &&
		strings.Join([]string{f.Subgates.A.Scope, f.Subgates.A.KVM, f.Subgates.A.Firecracker, f.Subgates.A.StatusEffect, f.Subgates.B.Scope, f.Subgates.B.Selection, f.Subgates.B.Isolation, f.Subgates.B.StatusEffect, f.Subgates.C.Scope, f.Subgates.C.Subject, f.Subgates.C.StatusEffect}, "|") ==
			"neutral_elastic_core|not_required|not_required|cannot_accredit_v38|firecracker_agent_adapter|opt_in_explicit_no_fallback|one_microvm_per_agent|cannot_accredit_v38|real_physical_wave|same_candidate_as_a_and_b|accredit_v38_only_after_a_b_c"
	capacity := strings.Join([]string{
		f.Capacity.Observation, f.Capacity.PhysicalCapacity, f.Capacity.ProviderQuota,
		f.Capacity.KnownQuota, f.Capacity.UnknownQuota, f.Capacity.ExhaustedQuota,
		f.Capacity.Reservation, f.Capacity.Consumption, f.Capacity.Release,
		f.Capacity.Exhaustion, f.Capacity.Restart,
	}, "|") ==
		"source_timestamp_expiry_and_quality_required|absolute_reservable_resource_fact|separate_admission_fact_never_physical_slot_count|reported_limit_enforced_without_deriving_physical_slots|fail_closed_and_never_converted_into_available_physical_slot|fail_closed_and_never_converted_into_available_physical_slot|atomic_with_lease_and_fence|idempotent_after_accepted_launch_receipt|idempotent_only_after_terminal_or_definitely_not_applied|same_execution_waits_without_consuming_attempt|reconstructs_observations_reservations_leases_and_fences"
	placement := strings.Join([]string{f.Placement.Selection, f.Placement.Claim, f.Placement.Launcher, f.Placement.Reselection}, "|") ==
		"opaque_account_and_placement_selected_before_claim|atomically_binds_account_placement_capacity_quota_lease_and_fence|must_obey_claimed_account_and_placement|forbidden_after_claim"
	demand := reflect.DeepEqual(f.Demand.LogicalCohorts, []int{1, 16, 70, 500}) && reflect.DeepEqual(f.Demand.PhysicalSteps, []int{1, 5, 10, 16, 20}) &&
		f.Demand.CompleteReadySet && f.Demand.HiddenGlobalCeiling == "forbidden" && f.Demand.PartialCapacity == "same_execution_waits_without_consuming_attempt" && f.Demand.LargePhysicalClaim == "only_with_explicit_measured_resources"
	scheduling := strings.Join([]string{
		f.Scheduling.Dispatcher, f.Scheduling.ClaimSelection, f.Scheduling.GlobalAdmission,
		f.Scheduling.GlobalCeiling, f.Scheduling.RuntimeCodexMaxConcurrentExecutions,
		f.Scheduling.StopPriority, f.Scheduling.ObserveProgress, f.Scheduling.ConcurrentActionKind,
		f.Scheduling.NonLaunchActions, f.Scheduling.PrivateLaunchOnlySelector, f.Scheduling.IdleWorkerGoroutines,
	}, "|") ==
		"single_global_application_dispatcher|global_any_or_without_launch_never_private_launch_only|atomic_and_provider_neutral|provider_neutral_never_derived_from_connector_limits|local_connector_guardrail_only_never_global_budget_or_dispatch|before_new_launches|required_when_launch_capacity_is_saturated|launch_agent_only|serialized_by_global_dispatcher|forbidden|forbidden"
	codexIsolation := strings.Join([]string{
		f.CodexIsolation.GuestExecution, f.CodexIsolation.HostAppServer,
		f.CodexIsolation.HostForbiddenScope, f.CodexIsolation.HostAuthority,
	}, "|") ==
		"all_codex_threads_turns_and_agents_inside_microvm|minimal_persistent_identity_and_quota_only|threads_turns_agents_and_execution|no_execution_or_lifecycle_authority"
	stateRepository := strings.Join([]string{
		f.StateRepository.Port, f.StateRepository.ActiveSource, f.StateRepository.SQLite,
		f.StateRepository.PostgreSQL, f.StateRepository.DualWrite,
	}, "|") ==
		"StateRepository|exactly_one_transactional_source_selected_by_composition|local_development_and_tests_only|future_production_adapter|forbidden"
	shutdown := strings.Join([]string{f.Shutdown.Scope, f.Shutdown.Cooperative, f.Shutdown.Forced, f.Shutdown.HungAgent}, "|") ==
		"one_exact_agent_without_stopping_the_cohort|required_first|allowed_after_bounded_cooperative_failure|isolated_and_stopped_individually" &&
		f.Preservation.SealBeforeUnmount && f.Preservation.InventoryBeforeUnmount && !f.Preservation.AutomaticDeletion &&
		f.Preservation.TerminalEnvironmentState == "preserved_pending_review" && f.Preservation.Removal == "separate_authorized_decision_after_review"
	recovery := strings.Join([]string{f.Recovery.BeforeAttempt, f.Recovery.AttemptBeforeEffect, f.Recovery.EffectWithoutReceipt, f.Recovery.UnknownApplied, f.Recovery.ReceiptBeforeObserve, f.Recovery.StopReceiptLost, f.Recovery.Retry, f.Recovery.DuplicateLaunch, f.Recovery.StaleFenceWrite}, "|") ==
		"no_external_effect|same_execution_reclaimed_with_higher_fence|quarantine_and_reconcile_by_external_identity|quarantine_and_never_blind_retry|resume_observation_without_relaunch|reconcile_exact_stop_without_broad_kill|only_definitely_not_applied|forbidden|rejected"
	latencies := reflect.DeepEqual(f.MeasuredLatencies, []string{
		"ready_to_capacity_request", "capacity_provision", "agent_launch", "agent_available",
		"cooperative_stop", "forced_stop", "seal_and_inventory",
	})
	guards := reflect.DeepEqual(f.RequiredTests.Names, v38ExactTests) && f.RequiredTests.EventEvidence == "exact_run_and_pass_required_for_each_test" &&
		f.RequiredTests.RejectNoTestsToRun && f.CatalogGuards == (v38CatalogGuards{257, 38, 38, true, true, true})
	dependencies := reflect.DeepEqual(f.AccreditedPrerequisites, []string{"AGT-01", "AGT-03", "GOV-21", "ORC-10", "EVD-13"}) &&
		reflect.DeepEqual(f.CausalDependencies, []string{"config", "credentials", "recovery_backup", "controls", "budgets_effects", "workspace_git", "test_attestor", "codex_e2e"})
	return identity && nonOwned && gates && capacity && placement && demand && scheduling && codexIsolation &&
		stateRepository && shutdown && recovery && latencies && guards && dependencies
}

func v38HasExactRunPass(output string, names []string) bool {
	for _, name := range names {
		if !strings.Contains(output, `"Action":"run","Test":"`+name+`"`) ||
			!strings.Contains(output, `"Action":"pass","Test":"`+name+`"`) {
			return false
		}
	}
	return true
}
