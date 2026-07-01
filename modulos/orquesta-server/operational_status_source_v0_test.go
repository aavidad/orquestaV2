package orquestaserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaobservability "orquesta/modulos/orquesta-observability"
)

func TestResidentOperationalStatusSourceV0DerivaDiagnosticoCompactoRedactado(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{
		Addr:           "127.0.0.1:8787",
		ProjectWorkDir: "/tmp/proyecto-secreto",
		RuntimeWorkDir: "/tmp/runtime-secreto",
	}, now)
	tracker.MarkServingV0("127.0.0.1:8787", now)
	tracker.MarkStartupReadyV0(StartupCheckResultV0{
		Status:       StartupCheckStatusReadyV0,
		Ready:        true,
		Message:      "ready with /tmp/runtime-secreto",
		EvidenceRefs: []string{"evidence-ref-startup-ready"},
	}, now)
	query := residentOperationalStatusQueryV0("corr-ref-resident-operational-status")

	diagnostic, err := (ResidentOperationalStatusSourceV0{
		Tracker: tracker,
		Clock:   fixedServerClockV0{now: now},
	}).QueryOperationalStatusV0(query)

	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic invalid: %v", err)
	}
	body, _ := json.Marshal(diagnostic)
	for _, forbidden := range []string{"proyecto-secreto", "runtime-secreto", "project_work_dir", "runtime_work_dir"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("diagnostic leaks %q: %s", forbidden, string(body))
		}
	}
	if diagnostic.Estado != orquestaobservability.DiagnosticoEstadoOKV0 || len(diagnostic.Warnings) == 0 {
		t.Fatalf("diagnostic=%+v", diagnostic)
	}
}

func TestHandlerV0OperationalStatusQueryV0(t *testing.T) {
	now := time.Date(2026, 5, 25, 10, 0, 0, 0, time.UTC)
	tracker := NewStatusTrackerV0(ConfigV0{Addr: "127.0.0.1:8787"}, now)
	tracker.MarkServingV0("127.0.0.1:8787", now)
	handler := NewHandlerV0(HandlerConfigV0{
		Tracker: tracker,
		OperationalStatusSource: ResidentOperationalStatusSourceV0{
			Tracker: tracker,
			Clock:   fixedServerClockV0{now: now},
		},
	})
	payload, _ := json.Marshal(residentOperationalStatusQueryV0("corr-ref-handler-operational-status"))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v0/operational-status/query", bytes.NewReader(payload))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "diagnostico_compacto.v0") ||
		!strings.Contains(rec.Body.String(), "runtime_not_available") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestResidentOperationalProgressV0NormalizaTotalSiCompletedSuperaTotalV0(t *testing.T) {
	progress := residentOperationalProgressV0(StateV0{
		Status:               "running",
		SupervisorTicks:      1,
		SupervisorExecutions: 3,
	})

	if progress.Completed != 3 || progress.Total != 3 {
		t.Fatalf("progress=%+v", progress)
	}
	if progress.Percent == nil || *progress.Percent != 100 {
		t.Fatalf("percent=%v progress=%+v", progress.Percent, progress)
	}
}

func TestResidentOperationalStatusV0NoBloqueaPorContadoresHistoricosRecuperadosV0(t *testing.T) {
	state := StateV0{
		Status:                     "running",
		LastSupervisorStatus:       "ok",
		LastSupervisorAt:           "2026-06-25T10:00:00Z",
		SupervisorErrorTicks:       7,
		ResidentDirectorStatus:     "ok",
		ResidentDirectorErrorTicks: 3,
		ResidentDirectorLastTickAt: "2026-06-25T10:00:00Z",
		ResidentDirectorLastError:  "",
		ExternalBridgeStatus:       "disabled",
	}

	if got := residentOperationalEstadoV0(state); got != orquestaobservability.DiagnosticoEstadoOKV0 {
		t.Fatalf("estado=%s", got)
	}
	if got := supervisorEstadoV0(state); got != orquestaobservability.DiagnosticoEstadoOKV0 {
		t.Fatalf("supervisor estado=%s", got)
	}
	if blockers := residentOperationalBlockersV0(state); blockersContainRefForTestV0(blockers, "blocker-ref-server-operational-status") ||
		blockersContainRefForTestV0(blockers, "blocker-ref-server-resident-director") {
		t.Fatalf("blockers historicos no recuperados: %+v", blockers)
	}
}

func TestResidentOperationalStatusV0BloqueaSiResidenteDisabledConWaitingOutboxV0(t *testing.T) {
	state := StateV0{
		Status:               "running",
		LastSupervisorStatus: SupervisorPublicStatusWaitingOutboxV0,
		EffectiveConfig: ServerEffectiveConfigV0{
			Settings: []ServerConfigSettingV0{{
				Key:   "ORQUESTA_SERVER_RESIDENT_DIRECTOR_ENABLED",
				Value: "false",
			}},
		},
	}

	if got := residentOperationalEstadoV0(state); got != orquestaobservability.DiagnosticoEstadoDegradedV0 {
		t.Fatalf("estado=%s", got)
	}
	if blockers := residentOperationalBlockersV0(state); !blockersContainRefForTestV0(
		blockers,
		"blocker-ref-server-resident-director-disabled-waiting-outbox",
	) {
		t.Fatalf("missing resident disabled blocker: %+v", blockers)
	}
}

func TestResidentOperationalStatusV0BloqueaPorErrorVigenteAunqueContadoresSeanCeroV0(t *testing.T) {
	state := StateV0{
		Status:               "running",
		LastSupervisorStatus: "error",
		LastSupervisorError:  "supervisor_error",
	}

	if got := residentOperationalEstadoV0(state); got != orquestaobservability.DiagnosticoEstadoDegradedV0 {
		t.Fatalf("estado=%s", got)
	}
	if got := supervisorEstadoV0(state); got != orquestaobservability.DiagnosticoEstadoDegradedV0 {
		t.Fatalf("supervisor estado=%s", got)
	}
	if blockers := residentOperationalBlockersV0(state); !blockersContainRefForTestV0(blockers, "blocker-ref-server-operational-status") {
		t.Fatalf("missing supervisor blocker: %+v", blockers)
	}
}

func TestResidentOperationalStatusSourceV0ExponeGoalFirstCompactoV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 0, 0, 0, time.UTC)
	state := StateV0{
		SchemaVersion:            StateSchemaVersionV0,
		Status:                   "running",
		LastHeartbeatAt:          now.Add(-3 * time.Second).Format(time.RFC3339),
		IdleSelfImprovementCheck: now.Format(time.RFC3339),
		SupervisorTicks:          4,
		SupervisorExecutions:     2,
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "idle_self_improvement",
			ReasonCode:    "goal_closure_accepted",
			Status:        "accepted",
			GoalRefs: []string{
				"goal-ref-operational-status-001",
				"external-goal-ref-operational-status-001",
			},
			RequestRefs:  []string{"request-ref-operational-status-goal-001"},
			EvidenceRefs: []string{"evidence-ref-operational-status-goal-message"},
		},
		IdleSelfImprovementGoalSpec: &orquestagoal.GoalWorkSpecV0{
			SchemaVersion:   orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:         "goal-ref-operational-status-001",
			RequestRef:      "request-ref-operational-status-goal-001",
			RunRef:          "run-ref-operational-status-goal-001",
			ProjectRef:      "project-ref-operational-status-goal-001",
			DomainRef:       "domain-ref-operational-status-goal-001",
			WorkKind:        "autoprogramming",
			WorkProfileKind: "self_improvement",
			Objective:       "NO_DEBE_FILTRARSE objetivo con ruta /tmp/secreta",
			DirectorKind:    orquestagoal.GoalDirectorKindCodexGoalV0,
			ContextRefs: []orquestagoal.GoalContextRefV0{{
				Ref:     "context-ref-operational-status-goal-001",
				Purpose: "fixture",
			}},
			RuleRefs: []orquestagoal.GoalRuleRefV0{{
				Ref:         "rule-ref-operational-status-goal-001",
				Enforcement: orquestagoal.GoalRuleEnforcementAdvisoryV0,
			}},
			SkillRefs:     []string{"skill-ref-operational-status-goal-001"},
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server", Purpose: "fixture"}},
			RequiredTests: []orquestagoal.GoalRequiredTestV0{{TestRef: "test-ref-operational-status-goal-001"}},
			ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
				ArtifactRef:  "artifact-ref-operational-status-goal-001",
				ArtifactType: "patch",
				Required:     true,
			}},
			EvidenceRefs: []string{"evidence-ref-operational-status-goal-spec"},
			ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
				RequireRequiredTests: true,
				RequireArtifacts:     true,
				RequiredEvidenceRefs: []string{"evidence-ref-operational-status-goal-required"},
			},
		},
		IdleSelfImprovementGoalReceipt: &orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-operational-status-001",
			ExternalGoalRef: "external-goal-ref-operational-status-001",
			EvidenceRefs:    []string{"evidence-ref-operational-status-goal-receipt"},
		},
		IdleSelfImprovementGoalResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
			Status:          orquestagoal.GoalStatusCompleteV0,
			GoalRef:         "goal-ref-operational-status-001",
			ExternalGoalRef: "external-goal-ref-operational-status-001",
			Summary:         "NO_DEBE_FILTRARSE resumen completo",
			ArtifactRefs:    []string{"artifact-ref-operational-status-goal-001"},
			RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
				TestRef:      "test-ref-operational-status-goal-001",
				Status:       "passed",
				EvidenceRefs: []string{"evidence-ref-operational-status-goal-test"},
			}},
			DomainReceiptRefs: []string{"domain-receipt-ref-operational-status-goal-001"},
			EvidenceRefs:      []string{"evidence-ref-operational-status-goal-result"},
		},
		IdleSelfImprovementGoalClosure: &orquestagoal.GoalClosureValidationV0{
			Status:       orquestagoal.GoalStatusAcceptedV0,
			Accepted:     true,
			EvidenceRefs: []string{"evidence-ref-operational-status-goal-required"},
		},
	}
	query := residentOperationalStatusQueryV0("corr-ref-operational-status-goal-001")

	diagnostic := buildResidentOperationalStatusDiagnosticV0(query, state, now)
	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic invalid: %v", err)
	}
	if len(diagnostic.Contadores) > 24 {
		t.Fatalf("too many counters: %d %+v", len(diagnostic.Contadores), diagnostic.Contadores)
	}
	assertOperationalCounterForTestV0(t, diagnostic, "external_bridge_ticks", 0)
	assertOperationalCounterForTestV0(t, diagnostic, "external_bridge_errors", 0)
	assertOperationalCounterForTestV0(t, diagnostic, "state_persist_failures", 0)
	assertOperationalCounterForTestV0(t, diagnostic, "audit_failures", 0)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_spec_present", 1)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_receipt_present", 1)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_result_present", 1)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_closure_present", 1)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_closure_accepted", 1)
	if !diagnosticoContainsHealthForTestV0(diagnostic, "server.health.idle_self_improvement_goal") {
		t.Fatalf("missing goal health in %+v", diagnostic.Salud)
	}
	if !diagnosticoContainsActivityForTestV0(diagnostic, "activity-ref-server-idle-goal") {
		t.Fatalf("missing goal activity in %+v", diagnostic.ActividadReciente)
	}
	for _, ref := range []string{
		"goal-ref-operational-status-001",
		"external-goal-ref-operational-status-001",
		"request-ref-operational-status-goal-001",
		"run-ref-operational-status-goal-001",
		"project-ref-operational-status-goal-001",
	} {
		if !diagnosticoContainsReferenceForTestV0(diagnostic, ref) {
			t.Fatalf("missing reference %q in %+v", ref, diagnostic.Referencias)
		}
	}
	body, _ := json.Marshal(diagnostic)
	for _, forbidden := range []string{
		"NO_DEBE_FILTRARSE",
		"/tmp/secreta",
		"objective",
		"write_set",
		"required_tests",
	} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("diagnostic leaks %q: %s", forbidden, string(body))
		}
	}
}

func TestResidentOperationalStatusSourceV0MarcaGoalFirstBloqueadoV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 5, 0, 0, time.UTC)
	state := StateV0{
		SchemaVersion:            StateSchemaVersionV0,
		Status:                   "running",
		LastHeartbeatAt:          now.Format(time.RFC3339),
		IdleSelfImprovementCheck: now.Format(time.RFC3339),
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "idle_self_improvement",
			ReasonCode:    idleSelfImprovementGoalInvalidReasonV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRefs:      []string{"goal-ref-operational-status-blocked-001"},
		},
		IdleSelfImprovementGoalSpec: &orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-operational-status-blocked-001",
			Objective:     "NO_DEBE_FILTRARSE objetivo bloqueado",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}},
		},
		IdleSelfImprovementGoalResult: &orquestagoal.GoalWorkResultV0{
			SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
			Status:        orquestagoal.GoalStatusInvalidV0,
			GoalRef:       "goal-ref-operational-status-blocked-001",
			Summary:       "NO_DEBE_FILTRARSE resumen bloqueado",
		},
		IdleSelfImprovementGoalClosure: &orquestagoal.GoalClosureValidationV0{
			Status:      orquestagoal.GoalStatusBlockedV0,
			NeedsRework: true,
			Issues:      []orquestagoal.GoalWorkIssueV0{{Code: orquestagoal.ErrGoalClosureInvalidV0, Field: "status"}},
		},
	}

	diagnostic := buildResidentOperationalStatusDiagnosticV0(
		residentOperationalStatusQueryV0("corr-ref-operational-status-goal-blocked-001"),
		state,
		now,
	)

	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic invalid: %v", err)
	}
	if !diagnosticoContainsBlockerForTestV0(diagnostic, "blocker-ref-server-idle-goal") {
		t.Fatalf("missing goal blocker in %+v", diagnostic.Bloqueos)
	}
	if !diagnosticoContainsHealthEstadoForTestV0(
		diagnostic,
		"server.health.idle_self_improvement_goal",
		orquestaobservability.DiagnosticoEstadoBlockedV0,
	) {
		t.Fatalf("missing blocked goal health in %+v", diagnostic.Salud)
	}
	body, _ := json.Marshal(diagnostic)
	if strings.Contains(string(body), "NO_DEBE_FILTRARSE") {
		t.Fatalf("diagnostic leaks blocked goal payload: %s", string(body))
	}
}

func TestResidentOperationalStatusSourceV0ExponeGoalBackendActivoConColaVaciaV0(t *testing.T) {
	now := time.Date(2026, 7, 1, 10, 10, 0, 0, time.UTC)
	state := StateV0{
		SchemaVersion:              StateSchemaVersionV0,
		Status:                     "running",
		LastHeartbeatAt:            now.Format(time.RFC3339),
		LastSupervisorStatus:       SupervisorPublicStatusQueueIdleGoalBackendActiveV0,
		LastSupervisorStopPublic:   SupervisorPublicStopQueueIdleButGoalBackendActiveV0,
		LastSupervisorStopCategory: SupervisorPublicCategoryGoalBackendV0,
		LastSupervisorQueueSize:    0,
		GoalObserverStatus:         "ok",
		GoalObserverLastSuccessAt:  now.Add(-time.Second).Format(time.RFC3339),
		GoalObserverOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "goal_observer",
			ReasonCode:    "ok",
			Status:        "ok",
			RunRefs: []string{
				"run-ref-bug-069-active-a",
				"run-ref-bug-069-active-b",
			},
			GoalRefs: []string{
				"goal-ref-bug-069-active-a",
				"external-goal-ref-bug-069-active-a",
				"goal-ref-bug-069-active-b",
			},
			Counters: map[string]int{
				"observed": 2,
				"terminal": 0,
				"issues":   0,
			},
		},
	}

	diagnostic := buildResidentOperationalStatusDiagnosticV0(
		residentOperationalStatusQueryV0("corr-ref-bug-069-goal-backend-active"),
		state,
		now,
	)

	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic invalid: %v", err)
	}
	assertOperationalCounterForTestV0(t, diagnostic, "queue_size", 0)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_backend_active", 2)
	assertOperationalCounterForTestV0(t, diagnostic, "goal_backend_observed", 2)
	if !diagnosticoContainsReferenceForTestV0(diagnostic, "run-ref-bug-069-active-a") ||
		!diagnosticoContainsReferenceForTestV0(diagnostic, "goal-ref-bug-069-active-a") {
		t.Fatalf("missing goal backend refs in %+v", diagnostic.Referencias)
	}
}

func TestServerReadinessV0ExponeGoalFirstSinCambiarReadyV0(t *testing.T) {
	state := StateV0{
		Status:       "running",
		StartupReady: true,
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "idle_self_improvement",
			ReasonCode:    idleSelfImprovementGoalRunningReasonV0,
			Status:        orquestagoal.GoalStatusRunningV0,
			GoalRefs:      []string{"goal-ref-readiness-001", "external-goal-ref-readiness-001"},
		},
		IdleSelfImprovementGoalSpec: &orquestagoal.GoalWorkSpecV0{
			SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0,
			GoalRef:       "goal-ref-readiness-001",
			Objective:     "NO_DEBE_FILTRARSE readiness",
			DirectorKind:  orquestagoal.GoalDirectorKindCodexGoalV0,
			WriteSet:      []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server/private_file.go"}},
		},
		IdleSelfImprovementGoalReceipt: &orquestagoal.GoalLaunchReceiptV0{
			SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         "goal-ref-readiness-001",
			ExternalGoalRef: "external-goal-ref-readiness-001",
		},
	}

	readiness := NewServerReadinessV0(state)

	if !readiness.Ready ||
		!readiness.IdleSelfImprovementGoalActive ||
		readiness.IdleSelfImprovementGoalRef != "goal-ref-readiness-001" ||
		readiness.IdleSelfImprovementGoalStatus != orquestagoal.GoalStatusRunningV0 ||
		readiness.IdleSelfImprovementGoalReasonCode != idleSelfImprovementGoalRunningReasonV0 {
		t.Fatalf("readiness=%+v", readiness)
	}
	body, _ := json.Marshal(readiness)
	for _, forbidden := range []string{"NO_DEBE_FILTRARSE", "private_file.go"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("readiness leaks %q: %s", forbidden, string(body))
		}
	}
}

func TestResidentOperationalStatusSourceV0MarcaCapacidadGoalFirstNoDisponibleSinGoalActivoV0(t *testing.T) {
	now := time.Date(2026, 6, 25, 10, 10, 0, 0, time.UTC)
	state := StateV0{
		SchemaVersion:             StateSchemaVersionV0,
		Status:                    "running",
		LastHeartbeatAt:           now.Format(time.RFC3339),
		IdleSelfImprovementCheck:  now.Format(time.RFC3339),
		IdleSelfImprovementReason: idleSelfImprovementGoalLauncherUnavailableReasonV0,
		IdleSelfImprovementOperationalMessage: &ServerOperationalMessageV0{
			SchemaVersion: serverOperationalMessageSchemaV0,
			Scope:         "idle_self_improvement",
			ReasonCode:    idleSelfImprovementGoalLauncherUnavailableReasonV0,
			Status:        "checked",
		},
	}

	diagnostic := buildResidentOperationalStatusDiagnosticV0(
		residentOperationalStatusQueryV0("corr-ref-operational-status-goal-capability-001"),
		state,
		now,
	)

	if err := orquestaobservability.ValidateDiagnosticoCompactoV0(diagnostic); err != nil {
		t.Fatalf("diagnostic invalid: %v", err)
	}
	if diagnostic.Estado != orquestaobservability.DiagnosticoEstadoDegradedV0 {
		t.Fatalf("estado=%s diagnostic=%+v", diagnostic.Estado, diagnostic)
	}
	if diagnosticoContainsHealthForTestV0(diagnostic, "server.health.idle_self_improvement_goal") {
		t.Fatalf("capacidad sin goal no debe publicar salud de goal activo: %+v", diagnostic.Salud)
	}
	if !diagnosticoContainsHealthEstadoForTestV0(
		diagnostic,
		"server.health.idle_self_improvement_goal_first",
		orquestaobservability.DiagnosticoEstadoDegradedV0,
	) {
		t.Fatalf("missing goal-first capability health in %+v", diagnostic.Salud)
	}
	if !diagnosticoContainsBlockerForTestV0(diagnostic, "blocker-ref-server-idle-goal-first-capability") {
		t.Fatalf("missing goal-first capability blocker in %+v", diagnostic.Bloqueos)
	}
	if len(residentOperationalGoalReferencesV0(state)) != 0 {
		t.Fatalf("capacidad sin goal no debe publicar refs de goal")
	}
}

func residentOperationalStatusQueryV0(correlationID string) orquestaobservability.OperationalStatusQueryV0 {
	return orquestaobservability.OperationalStatusQueryV0{
		SchemaVersion: orquestaobservability.OperationalStatusQuerySchemaVersionV0,
		RequestID:     "request-ref-resident-operational-status",
		CorrelationID: correlationID,
		Consumer: orquestaobservability.OperationalStatusConsumerV0{
			Module:  "orquesta-web",
			Channel: orquestaobservability.OperationalStatusConsumerWebChannelV0,
		},
		Locale:          "es",
		Scope:           orquestaobservability.OperationalStatusScopeSistemaV0,
		IncludeSections: []string{"estado", "progreso", "salud", "bloqueos", "actividad_reciente"},
		Limit:           10,
	}
}

type fixedServerClockV0 struct {
	now time.Time
}

func (clock fixedServerClockV0) Now() time.Time {
	return clock.now
}

func assertOperationalCounterForTestV0(
	t *testing.T,
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	key string,
	want float64,
) {
	t.Helper()
	got, ok := diagnostic.Contadores[key]
	if !ok {
		t.Fatalf("missing counter %q in %+v", key, diagnostic.Contadores)
	}
	if got != want {
		t.Fatalf("counter %q=%v want %v", key, got, want)
	}
}

func diagnosticoContainsReferenceForTestV0(
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	targetRef string,
) bool {
	for _, ref := range diagnostic.Referencias {
		if ref.TargetRef == targetRef {
			return true
		}
	}
	return false
}

func diagnosticoContainsHealthForTestV0(
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	i18nKey string,
) bool {
	for _, item := range diagnostic.Salud {
		if item.I18nKey == i18nKey {
			return true
		}
	}
	return false
}

func diagnosticoContainsHealthEstadoForTestV0(
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	i18nKey string,
	estado string,
) bool {
	for _, item := range diagnostic.Salud {
		if item.I18nKey == i18nKey && item.Estado == estado {
			return true
		}
	}
	return false
}

func diagnosticoContainsActivityForTestV0(
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	activityRef string,
) bool {
	for _, item := range diagnostic.ActividadReciente {
		if item.ActivityRef == activityRef {
			return true
		}
	}
	return false
}

func diagnosticoContainsBlockerForTestV0(
	diagnostic orquestaobservability.DiagnosticoCompactoV0,
	blockerRef string,
) bool {
	for _, item := range diagnostic.Bloqueos {
		if item.BlockerRef == blockerRef {
			return true
		}
	}
	return false
}

func blockersContainRefForTestV0(
	blockers []orquestaobservability.DiagnosticoBloqueoV0,
	blockerRef string,
) bool {
	for _, blocker := range blockers {
		if blocker.BlockerRef == blockerRef {
			return true
		}
	}
	return false
}
