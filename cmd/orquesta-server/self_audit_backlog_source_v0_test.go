package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
	orquestaserver "orquesta/modulos/orquesta-server"
)

func TestSelfAuditBacklogSectionsV0StaticcheckFindingEmiteSeccionEstableV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "modulos/orquesta-server/foo_v0.go:10:2: this value of err is never used (SA4006)\n",
		}
	})
	defer restore()

	first := selfAuditBacklogSectionsV0(context.Background(), t.TempDir())
	second := selfAuditBacklogSectionsV0(context.Background(), t.TempDir())

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("sections first=%+v second=%+v", first, second)
	}
	section := first[0]
	if section.Ref == "" ||
		section.Ref != second[0].Ref ||
		section.SourceKind != "self_audit" ||
		section.Scope[0] != "modulos/orquesta-server/foo_v0.go" ||
		section.Tests[0] != "staticcheck ./..." ||
		len(section.AcceptanceChecks) != 1 ||
		section.AcceptanceChecks[0].CriterionRef == "" ||
		section.AcceptanceChecks[0].CriterionRef != second[0].AcceptanceChecks[0].CriterionRef ||
		section.AcceptanceChecks[0].Description != "staticcheck no reporta el hallazgo SA4006 para modulos/orquesta-server/foo_v0.go" ||
		section.AcceptanceChecks[0].Command != "staticcheck ./..." ||
		!strings.Contains(section.Criteria[0], "staticcheck no reporta el hallazgo SA4006") ||
		!containsStringForTestV0(section.Inputs, "self_audit_tool:staticcheck") {
		t.Fatalf("section=%+v second=%+v", section, second[0])
	}
}

func TestSelfAuditBacklogSectionsV0GovulncheckGlobalEmiteSeccionV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "govulncheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "Vulnerability #1: GO-2026-4341\n  More info: https://pkg.go.dev/vuln/GO-2026-4341\n",
		}
	})
	defer restore()

	sections := selfAuditBacklogSectionsV0(context.Background(), t.TempDir())

	if len(sections) != 1 {
		t.Fatalf("sections=%+v", sections)
	}
	section := sections[0]
	if section.SourceKind != "self_audit" ||
		section.SourcePath != "self_audit://govulncheck" ||
		section.Scope[0] != "go.mod" ||
		section.Tests[0] != "govulncheck ./..." ||
		!strings.Contains(section.Criteria[0], "GO-2026-4341") ||
		!containsStringForTestV0(section.Inputs, "self_audit_tool:govulncheck") {
		t.Fatalf("section=%+v", section)
	}
}

func TestSelfAuditBacklogSectionsV0RaceAbsPathEmiteSeccionConRutaRelativaV0(t *testing.T) {
	projectDir := t.TempDir()
	racePath := filepath.Join(projectDir, "modulos", "orquesta-server", "race_v0.go")
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "go-test-race" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: strings.Join([]string{
				"WARNING: DATA RACE",
				"Read at 0x00 by goroutine 8:",
				"  orquesta/modulos/orquesta-server.(*runtimeV0).tick()",
				"      " + racePath + ":42 +0x123",
			}, "\n"),
		}
	})
	defer restore()

	sections := selfAuditBacklogSectionsV0(context.Background(), projectDir)

	if len(sections) != 1 {
		t.Fatalf("sections=%+v", sections)
	}
	section := sections[0]
	if section.Scope[0] != "modulos/orquesta-server/race_v0.go" ||
		section.Tests[0] != "go test -race ./..." ||
		!strings.Contains(section.Objective, "data-race") ||
		!strings.Contains(section.Objective, "modulos/orquesta-server/race_v0.go") ||
		!containsStringForTestV0(section.Inputs, "self_audit_tool:go-test-race") {
		t.Fatalf("section=%+v", section)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0IncluyeSelfAuditSoloConFlagV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "cmd/orquesta-server/self_audit_fixture_v0.go:12:3: should replace loop with copy (S1011)\n",
		}
	})
	defer restore()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte("# Backlog\n\nSin secciones Txx.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	disabled, err := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir: projectDir,
	}).loadBacklogSectionsV0()
	if err != nil {
		t.Fatalf("loadBacklogSectionsV0 disabled: %v", err)
	}
	if len(disabled) != 0 {
		t.Fatalf("self-audit no debe cargarse sin flag: %+v", disabled)
	}

	result, err := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir:          projectDir,
		SelfAuditBacklogEnabled: true,
	}).PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 1,
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef:    "request-ref-base",
			CorrelationID: "corr-request-ref-base",
			ProjectRef:    "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if len(result.Requests) != 1 {
		t.Fatalf("requests=%+v", result.Requests)
	}
	request := result.Requests[0]
	if request.FailureKind != "backlog_autoprogramming" ||
		!strings.HasPrefix(request.SuggestedArea, "self-audit-") ||
		!containsStringForTestV0(request.WriteSet, "cmd/orquesta-server/self_audit_fixture_v0.go") ||
		!containsStringForTestV0(request.RequiredTests, "staticcheck ./...") ||
		len(request.AcceptanceChecks) != 1 ||
		!strings.HasPrefix(request.AcceptanceChecks[0].CriterionRef, "criterion-ref-self-audit-") ||
		request.AcceptanceChecks[0].Description != "staticcheck no reporta el hallazgo S1011 para cmd/orquesta-server/self_audit_fixture_v0.go" ||
		request.AcceptanceChecks[0].Command != "staticcheck ./..." ||
		!containsStringForTestV0(request.ContextRefs, "backlog_doc:self_audit://staticcheck") {
		t.Fatalf("request=%+v", request)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0SelfAuditRespetaContextCanceladoV0(t *testing.T) {
	runnerCalls := 0
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, _ selfAuditCommandV0) selfAuditCommandResultV0 {
		runnerCalls++
		return selfAuditCommandResultV0{
			Output: "cmd/orquesta-server/self_audit_fixture_v0.go:12:3: should replace loop with copy (S1011)\n",
		}
	})
	defer restore()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte("# Backlog\n\nSin secciones Txx.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir:          projectDir,
		SelfAuditBacklogEnabled: true,
		Context:                 ctx,
	}).PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 1,
		BaseRequest: orquestaserver.IdleSelfImprovementRequestV0{
			RequestRef: "request-ref-base",
			ProjectRef: "project-ref-orquesta",
		},
	})
	if err != nil {
		t.Fatalf("PlanV0: %v", err)
	}
	if runnerCalls != 0 {
		t.Fatalf("self-audit no debe lanzar comandos con contexto cancelado: calls=%d", runnerCalls)
	}
}

func TestIdleSelfImprovementBacklogPlannerV0DeduplicaSelfAuditConKnownRefV0(t *testing.T) {
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "cmd/orquesta-server/self_audit_fixture_v0.go:12:3: should replace loop with copy (S1011)\n",
		}
	})
	defer restore()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte("# Backlog\n\nSin secciones Txx.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}
	planner := idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir:          projectDir,
		SelfAuditBacklogEnabled: true,
	}
	base := orquestaserver.IdleSelfImprovementRequestV0{
		RequestRef: "request-ref-base",
		ProjectRef: "project-ref-orquesta",
	}
	first, err := planner.PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests: 1,
		BaseRequest: base,
	})
	if err != nil || len(first.Requests) != 1 {
		t.Fatalf("first result=%+v err=%v", first, err)
	}
	second, err := planner.PlanV0(orquestaserver.IdleSelfImprovementPlanRequestV0{
		MaxRequests:      1,
		BaseRequest:      base,
		KnownRequestRefs: []string{first.Requests[0].RequestRef},
		KnownRunRefs:     []string{first.Requests[0].RequestRef + "-retry-001"},
	})
	if err != nil {
		t.Fatalf("second PlanV0: %v", err)
	}
	if len(second.Requests) != 0 || second.Message != "backlog_tareas_ya_visibles_en_cola" {
		t.Fatalf("second=%+v", second)
	}
}

func TestRuntimeV0SelfAuditBacklogGoalFirstLanzaSpecOperacionalV0(t *testing.T) {
	requireLocalTCPForTestV0(t)
	restore := replaceSelfAuditRunnerForTestV0(func(_ context.Context, _ string, command selfAuditCommandV0) selfAuditCommandResultV0 {
		if command.ToolRef != "staticcheck" {
			return selfAuditCommandResultV0{}
		}
		return selfAuditCommandResultV0{
			Output: "cmd/orquesta-server/self_audit_fixture_v0.go:12:3: should replace loop with copy (S1011)\n",
		}
	})
	defer restore()

	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "docs"), 0o700); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(projectDir, idleSelfImprovementBacklogDocRelV0),
		[]byte("# Backlog\n\nSin secciones Txx.\n"),
		0o600,
	); err != nil {
		t.Fatalf("write backlog: %v", err)
	}

	supervisor := &selfAuditGoalFirstSupervisorForTestV0{
		projectDir: projectDir,
		started:    make(chan struct{}, 1),
	}
	goalStates := newSelfAuditGoalStateStoreForTestV0()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runtime, err := orquestaserver.NewRuntimeV0(orquestaserver.ConfigV0{
		Addr:                           "127.0.0.1:0",
		StateDir:                       t.TempDir(),
		TickInterval:                   5 * time.Millisecond,
		IdleSelfImprovementAfter:       time.Millisecond,
		IdleSelfImprovementGoalFirst:   true,
		IdleSelfImprovementProjectRef:  "project-ref-self-audit-operational",
		IdleSelfImprovementMaxRequests: 1,
		IdleSelfImprovementTargetQueue: 1,
		AuditDisabled:                  true,
	}, orquestaserver.RuntimeDepsV0{
		Supervisor:                 supervisor,
		GoalStateStore:             goalStates,
		GoalRequiredTestSpecBinder: serverGoalRequiredTestSpecBinderForTestV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	runDone := make(chan error, 1)
	go func() { runDone <- runtime.RunV0(ctx) }()

	select {
	case <-supervisor.started:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatalf("self-audit goal-first no lanzo spec")
	}
	spec := supervisor.lastSpecV0()
	if spec.WorkKind != "idle_self_improvement" ||
		spec.DirectorKind != orquestagoal.GoalDirectorKindCodexGoalV0 ||
		spec.ProjectRef != "project-ref-self-audit-operational" ||
		!strings.Contains(spec.Objective, "S1011") ||
		!containsGoalWritePathSelfAuditTestV0(spec.WriteSet, "cmd/orquesta-server/self_audit_fixture_v0.go") ||
		!containsGoalRequiredTestSelfAuditTestV0(spec.RequiredTests, "staticcheck ./...") ||
		!containsGoalContextRefSelfAuditTestV0(spec.ContextRefs, "backlog_doc:self_audit://staticcheck") ||
		!containsStringPrefixSelfAuditTestV0(spec.EvidenceRefs, "evidence-ref-autoprogramming-backlog-section-self-audit-") ||
		!spec.ClosurePolicy.RequireRequiredTests ||
		!spec.ReworkPolicy.PreferNewGoal {
		cancel()
		t.Fatalf("spec self-audit incompleto: %+v", spec)
	}
	state := waitSelfAuditGoalStateForTestV0(t, goalStates, spec.RunRef)
	if state.Status != orquestagoal.GoalStatusRunningV0 ||
		state.Spec.GoalRef != spec.GoalRef ||
		state.ExternalGoalRef != "external-"+spec.GoalRef {
		cancel()
		t.Fatalf("goal state=%+v spec=%+v", state, spec)
	}
	cancel()
	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("RunV0: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("runtime no paro tras cancel")
	}
	if supervisor.planCallsV0() == 0 || supervisor.launchCallsV0() != 1 {
		t.Fatalf("plan_calls=%d launch_calls=%d", supervisor.planCallsV0(), supervisor.launchCallsV0())
	}
}

func replaceSelfAuditRunnerForTestV0(
	runner func(context.Context, string, selfAuditCommandV0) selfAuditCommandResultV0,
) func() {
	previous := runSelfAuditCommandV0
	runSelfAuditCommandV0 = runner
	return func() { runSelfAuditCommandV0 = previous }
}

type selfAuditGoalFirstSupervisorForTestV0 struct {
	mu          sync.Mutex
	projectDir  string
	planCalls   int
	launchCalls int
	lastSpec    orquestagoal.GoalWorkSpecV0
	started     chan struct{}
}

func (supervisor *selfAuditGoalFirstSupervisorForTestV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (supervisor *selfAuditGoalFirstSupervisorForTestV0) PlanIdleSelfImprovementV0(
	ctx context.Context,
	request orquestaserver.IdleSelfImprovementPlanRequestV0,
) (orquestaserver.IdleSelfImprovementPlanResultV0, error) {
	supervisor.mu.Lock()
	supervisor.planCalls++
	supervisor.mu.Unlock()
	return (idleSelfImprovementBacklogPlannerV0{
		ProjectWorkDir:          supervisor.projectDir,
		SelfAuditBacklogEnabled: true,
	}).PlanV0(request)
}

func (supervisor *selfAuditGoalFirstSupervisorForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	normalized := orquestagoal.NormalizeGoalWorkSpecV0(spec)
	supervisor.mu.Lock()
	supervisor.launchCalls++
	supervisor.lastSpec = normalized
	supervisor.mu.Unlock()
	if supervisor.started != nil {
		select {
		case supervisor.started <- struct{}{}:
		default:
		}
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		GoalRef:         normalized.GoalRef,
		ExternalGoalRef: "external-" + normalized.GoalRef,
		EvidenceRefs:    []string{"evidence-ref-self-audit-goal-first-operational-test"},
	}, nil
}

func (supervisor *selfAuditGoalFirstSupervisorForTestV0) lastSpecV0() orquestagoal.GoalWorkSpecV0 {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return supervisor.lastSpec
}

func (supervisor *selfAuditGoalFirstSupervisorForTestV0) planCallsV0() int {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return supervisor.planCalls
}

func (supervisor *selfAuditGoalFirstSupervisorForTestV0) launchCallsV0() int {
	supervisor.mu.Lock()
	defer supervisor.mu.Unlock()
	return supervisor.launchCalls
}

type selfAuditGoalStateStoreForTestV0 struct {
	mu     sync.Mutex
	states map[string]orquestagoal.GoalWorkStateV0
}

func newSelfAuditGoalStateStoreForTestV0() *selfAuditGoalStateStoreForTestV0 {
	return &selfAuditGoalStateStoreForTestV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
}

func (store *selfAuditGoalStateStoreForTestV0) SaveGoalWorkStateV0(
	_ context.Context,
	state orquestagoal.GoalWorkStateV0,
) error {
	normalized, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.states[normalized.RunRef] = normalized
	return nil
}

func (store *selfAuditGoalStateStoreForTestV0) LoadGoalWorkStateV0(
	_ context.Context,
	runRef string,
) (orquestagoal.GoalWorkStateV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state, ok := store.states[strings.TrimSpace(runRef)]
	if !ok {
		return orquestagoal.GoalWorkStateV0{}, errors.New("goal_state_not_found")
	}
	return state, nil
}

func waitSelfAuditGoalStateForTestV0(
	t *testing.T,
	store *selfAuditGoalStateStoreForTestV0,
	runRef string,
) orquestagoal.GoalWorkStateV0 {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		state, err := store.LoadGoalWorkStateV0(context.Background(), runRef)
		if err == nil {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goal state no persistido para %s", runRef)
	return orquestagoal.GoalWorkStateV0{}
}

func containsGoalWritePathSelfAuditTestV0(values []orquestagoal.GoalWriteScopeV0, path string) bool {
	for _, value := range values {
		if value.Path == path {
			return true
		}
	}
	return false
}

func containsGoalRequiredTestSelfAuditTestV0(values []orquestagoal.GoalRequiredTestV0, command string) bool {
	for _, value := range values {
		if value.Command == command {
			return true
		}
	}
	return false
}

func containsGoalContextRefSelfAuditTestV0(values []orquestagoal.GoalContextRefV0, ref string) bool {
	for _, value := range values {
		if value.Ref == ref {
			return true
		}
	}
	return false
}

func containsStringPrefixSelfAuditTestV0(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
