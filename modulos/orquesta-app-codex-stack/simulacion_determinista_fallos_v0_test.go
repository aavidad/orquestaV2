package orquestaappcodexstack

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	orquestaestadovivo "orquesta/modulos/orquesta-estado-vivo"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruncoordinator "orquesta/modulos/orquesta-run-coordinator"
	orquestaruntimecodexappserver "orquesta/modulos/orquesta-runtime-codex-appserver"
	orquestaserver "orquesta/modulos/orquesta-server"
)

const simulacionDeterministaSeedV0 int64 = 291201

func TestSimulacionDeterministaFallosGoalFirstV0(t *testing.T) {
	started := time.Now()
	points := simulacionInjectionPointsV0(t, simulacionDeterministaSeedV0)
	if len(points) == 0 {
		t.Fatalf("sin puntos de inyeccion")
	}
	modes := []simulacionFaultModeV0{
		simulacionFaultBackendKillV0,
		simulacionFaultDelayMaterializationV0,
		simulacionFaultDuplicateObservationV0,
		simulacionFaultProcessCutV0,
	}

	first := make([]string, 0, len(points)*len(modes))
	for _, point := range points {
		for _, mode := range modes {
			result, err := runSimulacionGoalFirstV0(simulacionDeterministaSeedV0, point, mode)
			if err != nil {
				t.Fatalf("%v", err)
			}
			first = append(first, result.CanonicalStringV0())
		}
	}

	second := make([]string, 0, len(points)*len(modes))
	for _, point := range simulacionInjectionPointsV0(t, simulacionDeterministaSeedV0) {
		for _, mode := range modes {
			result, err := runSimulacionGoalFirstV0(simulacionDeterministaSeedV0, point, mode)
			if err != nil {
				t.Fatalf("segunda pasada: %v", err)
			}
			second = append(second, result.CanonicalStringV0())
		}
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("simulacion no determinista seed=%d\nfirst=%v\nsecond=%v", simulacionDeterministaSeedV0, first, second)
	}
	if elapsed, limit := time.Since(started), simulacionDeterministaFallosElapsedLimitV0(); limit > 0 && elapsed >= limit {
		t.Fatalf("simulacion lenta seed=%d elapsed=%s points=%d", simulacionDeterministaSeedV0, elapsed, len(points))
	}
}

func TestSimulacionDeterministaDetectaTransicionBackendIlegalV0(t *testing.T) {
	point := simulacionInjectionPointV0{
		Ref:  "backend-illegal-apagado-vivo",
		Kind: "backend_transition",
		BackendEdge: simulacionBackendTransitionEdgeV0{
			From: orquestaruntimecodexappserver.BackendApagadoV0,
			Observation: orquestaruntimecodexappserver.ObservacionBackendV0{
				SessionObserved: true,
			},
		},
	}

	_, err := runSimulacionGoalFirstV0(simulacionDeterministaSeedV0, point, simulacionFaultDelayMaterializationV0)
	if err == nil {
		t.Fatalf("se esperaba deteccion de transicion ilegal")
	}
	message := err.Error()
	for _, want := range []string{
		orquestaruntimecodexappserver.BackendTransitionIssueIllegalV0,
		"seed=291201",
		"backend-illegal-apagado-vivo",
		"sequence=",
	} {
		if !strings.Contains(message, want) {
			t.Fatalf("error sin evidencia reproducible %q en %s", want, message)
		}
	}
}

func TestSimulacionCriteriosAutomejoraIdleYProyeccionPublicaV0(t *testing.T) {
	defaultConfig := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{})
	if defaultConfig.IdleSelfImprovementAfter != 60*time.Second {
		t.Fatalf("idle default=%s want=60s", defaultConfig.IdleSelfImprovementAfter)
	}
	disabled := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		IdleSelfImprovementIdleDisabled: true,
		IdleSelfImprovementAfter:        60 * time.Second,
	})
	if disabled.IdleSelfImprovementAfter != 0 {
		t.Fatalf("idle disabled after=%s want=0", disabled.IdleSelfImprovementAfter)
	}
	capacity := orquestaserver.NormalizeConfigV0(orquestaserver.ConfigV0{
		IdleSelfImprovementMaxRequests: 4,
		IdleSelfImprovementTargetQueue: 2,
	})
	if capacity.IdleSelfImprovementTargetQueue != 4 {
		t.Fatalf("target_queue=%d want max_requests=4", capacity.IdleSelfImprovementTargetQueue)
	}

	public := []orquestaserver.ServerPublicStatusV0{
		orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			LastSupervisorStopPublic:   orquestaserver.SupervisorPublicStopWaitingOutboxV0,
			LastSupervisorStopCategory: orquestaserver.SupervisorPublicCategoryWaitOutboxV0,
		}),
		orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			LastSupervisorStopPublic:   orquestaserver.SupervisorPublicStopWaitingExternalV0,
			LastSupervisorStopCategory: orquestaserver.SupervisorPublicCategoryWaitExternalV0,
		}),
		orquestaserver.NewServerPublicStatusV0(orquestaserver.StateV0{
			LastSupervisorStopPublic:   orquestaserver.SupervisorPublicStopRunningLiveV0,
			LastSupervisorStopCategory: orquestaserver.SupervisorPublicCategoryExternalProcessV0,
		}),
	}
	got := []string{
		public[0].LastSupervisorStopCategory,
		public[1].LastSupervisorStopCategory,
		public[2].LastSupervisorStopCategory,
	}
	want := []string{
		orquestaserver.SupervisorPublicCategoryWaitOutboxV0,
		orquestaserver.SupervisorPublicCategoryWaitExternalV0,
		orquestaserver.SupervisorPublicCategoryExternalProcessV0,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("proyeccion publica stop categories=%v want=%v", got, want)
	}

	reconciled := orquestaruncoordinator.ReconcileExternalWorkPublicStatusV0(
		orquestaruncoordinator.ExternalWorkReconciliationInputV0{
			RunRef:             "run-ref-simulacion-public-status",
			PendingOutboxCount: 1,
		},
	)
	if reconciled.PublicStatus != orquestaruncoordinator.ExternalWorkPublicStatusPendingV0 ||
		reconciled.Action != orquestaruncoordinator.ExternalWorkReconcileActionObserveV0 {
		t.Fatalf("outbox pendiente no queda running observable: %+v", reconciled)
	}
}

type simulacionFaultModeV0 string

const (
	simulacionFaultBackendKillV0          simulacionFaultModeV0 = "backend_kill"
	simulacionFaultDelayMaterializationV0 simulacionFaultModeV0 = "delay_materialization"
	simulacionFaultDuplicateObservationV0 simulacionFaultModeV0 = "duplicate_observation"
	simulacionFaultProcessCutV0           simulacionFaultModeV0 = "process_cut"
)

type simulacionInjectionPointV0 struct {
	Ref         string
	Kind        string
	BackendEdge simulacionBackendTransitionEdgeV0
}

type simulacionBackendTransitionEdgeV0 struct {
	From        orquestaruntimecodexappserver.EstadoBackendAppServerV0
	To          orquestaruntimecodexappserver.EstadoBackendAppServerV0
	Observation orquestaruntimecodexappserver.ObservacionBackendV0
	Issues      []string
}

type simulacionRunResultV0 struct {
	PointRef    string
	Fault       simulacionFaultModeV0
	FinalStatus string
	Phase       orquestaestadovivo.FaseCicloVidaV0
	Conflict    bool
	Steps       []string
}

func (result simulacionRunResultV0) CanonicalStringV0() string {
	return strings.Join([]string{
		result.PointRef,
		string(result.Fault),
		result.FinalStatus,
		string(result.Phase),
		fmt.Sprintf("conflict=%t", result.Conflict),
		strings.Join(result.Steps, ">"),
	}, "|")
}

type simulacionTraceV0 struct {
	Seed  int64
	Point simulacionInjectionPointV0
	Fault simulacionFaultModeV0
	Steps []string
}

func (trace *simulacionTraceV0) AddV0(step string) {
	trace.Steps = append(trace.Steps, strings.TrimSpace(step))
}

func (trace simulacionTraceV0) ErrorV0(format string, args ...interface{}) error {
	return fmt.Errorf(format+"; seed=%d point=%s fault=%s sequence=%s",
		append(args, trace.Seed, trace.Point.Ref, trace.Fault, strings.Join(trace.Steps, ">"))...,
	)
}

func runSimulacionGoalFirstV0(
	seed int64,
	point simulacionInjectionPointV0,
	fault simulacionFaultModeV0,
) (simulacionRunResultV0, error) {
	trace := &simulacionTraceV0{Seed: seed, Point: point, Fault: fault}
	ctx := context.Background()
	store := newGoalFirstQueueStateStoreForTestV0()
	backend := &simulacionBackendGoalFirstV0{
		goalRef:         fmt.Sprintf("goal-ref-simulacion-%d", seed),
		externalGoalRef: fmt.Sprintf("external-goal-ref-simulacion-%d", seed),
	}
	ports := orquestagoal.GoalWorkLifecyclePortsV0{
		Launcher:         backend,
		Observer:         backend,
		ClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
		StateStore:       store,
	}
	spec := simulacionGoalWorkSpecV0(seed)
	trace.AddV0("start_goal")
	started, err := orquestagoal.StartGoalWorkV0(ctx, orquestagoal.GoalWorkStartRequestV0{
		RunRef:       spec.RunRef,
		Spec:         spec,
		EvidenceRefs: []string{"evidence-ref-simulacion-start"},
	}, ports)
	if err != nil {
		return simulacionRunResultV0{}, trace.ErrorV0("StartGoalWorkV0: %v", err)
	}
	if started.State.Status != orquestagoal.GoalStatusRunningV0 {
		return simulacionRunResultV0{}, trace.ErrorV0("estado inicial=%s want running", started.State.Status)
	}

	if point.Kind == "backend_transition" {
		if err := simulacionApplyBackendTransitionV0(point.BackendEdge, trace); err != nil {
			return simulacionRunResultV0{}, err
		}
	}
	if fault == simulacionFaultBackendKillV0 {
		backend.killed = true
		trace.AddV0("fault_backend_kill")
	}

	trace.AddV0("observe_initial")
	observed, err := orquestagoal.ObserveGoalWorkV0(ctx, orquestagoal.GoalWorkObserveRequestV0{RunRef: spec.RunRef}, ports)
	if err != nil {
		return simulacionRunResultV0{}, trace.ErrorV0("ObserveGoalWorkV0 inicial: %v", err)
	}
	if observed.State.Status == orquestagoal.GoalStatusRunningV0 && fault != simulacionFaultBackendKillV0 {
		trace.AddV0("materialize_result")
		if fault != simulacionFaultDelayMaterializationV0 {
			backend.materialized = true
		}
	}

	for tick := 0; tick < 6; tick++ {
		if fault == simulacionFaultDelayMaterializationV0 && tick == 1 {
			trace.AddV0("materialize_result_delayed")
			backend.materialized = true
		}
		trace.AddV0(fmt.Sprintf("observe_tick_%d", tick))
		observed, err = orquestagoal.ObserveGoalWorkV0(ctx, orquestagoal.GoalWorkObserveRequestV0{RunRef: spec.RunRef}, ports)
		if err != nil {
			return simulacionRunResultV0{}, trace.ErrorV0("ObserveGoalWorkV0 tick=%d: %v", tick, err)
		}
		if orquestagoal.GoalWorkResultTerminalV0(observed.State.Status) {
			break
		}
	}
	if fault == simulacionFaultDuplicateObservationV0 {
		trace.AddV0("duplicate_terminal_observation")
		observed, err = orquestagoal.ObserveGoalWorkV0(ctx, orquestagoal.GoalWorkObserveRequestV0{RunRef: spec.RunRef}, ports)
		if err != nil {
			return simulacionRunResultV0{}, trace.ErrorV0("ObserveGoalWorkV0 duplicate: %v", err)
		}
	}
	state, err := store.LoadGoalWorkStateV0(ctx, spec.RunRef)
	if err != nil {
		return simulacionRunResultV0{}, trace.ErrorV0("LoadGoalWorkStateV0: %v", err)
	}
	trace.AddV0("reconcile_projection")
	projection := orquestaestadovivo.ConstruirProyeccionCicloVidaV0(
		simulacionEvidenciasEstadoV0(state, fault),
		time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
		time.Hour,
	)
	if len(projection.Nodos) != 1 {
		return simulacionRunResultV0{}, trace.ErrorV0("nodos=%d want=1", len(projection.Nodos))
	}
	node := projection.Nodos[0]
	if state.Status == orquestagoal.GoalStatusRunningV0 {
		return simulacionRunResultV0{}, trace.ErrorV0("goal running eterno")
	}
	if terminal := state.LastResult != nil && orquestagoal.GoalWorkResultTerminalV0(state.LastResult.Status); terminal {
		switch node.Fase {
		case orquestaestadovivo.FaseTerminalAceptadoV0,
			orquestaestadovivo.FaseTerminalReworkV0,
			orquestaestadovivo.FaseConflictoV0:
		default:
			return simulacionRunResultV0{}, trace.ErrorV0("terminal desaparece fase=%s status=%s", node.Fase, state.LastResult.Status)
		}
	}
	if fault == simulacionFaultProcessCutV0 {
		if node.Fase != orquestaestadovivo.FaseConflictoV0 ||
			len(node.Conflictos) != 1 ||
			node.Conflictos[0].Codigo != orquestaestadovivo.CodigoConflictoProcesoVivoTrasTerminalV0 {
			return simulacionRunResultV0{}, trace.ErrorV0("conflicto no visible: %+v", node)
		}
	}
	return simulacionRunResultV0{
		PointRef:    point.Ref,
		Fault:       fault,
		FinalStatus: state.Status,
		Phase:       node.Fase,
		Conflict:    len(node.Conflictos) > 0,
		Steps:       append([]string(nil), trace.Steps...),
	}, nil
}

func simulacionGoalWorkSpecV0(seed int64) orquestagoal.GoalWorkSpecV0 {
	return orquestagoal.GoalWorkSpecV0{
		GoalRef:         fmt.Sprintf("goal-ref-simulacion-%d", seed),
		RequestRef:      fmt.Sprintf("request-ref-simulacion-%d", seed),
		RunRef:          fmt.Sprintf("run-ref-simulacion-%d", seed),
		ProjectRef:      "project-ref-orquesta-server",
		WorkKind:        "idle_self_improvement",
		WorkProfileKind: "implementation",
		Objective:       "Simulacion determinista goal-first con inyeccion de fallos.",
		DirectorKind:    orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet: []orquestagoal.GoalWriteScopeV0{
			{Path: "modulos/orquesta-app-codex-stack", Purpose: "harness"},
			{Path: "modulos/orquesta-estado-vivo", Purpose: "projection"},
		},
		RequiredTests: []orquestagoal.GoalRequiredTestV0{{
			TestRef: "test-ref-simulacion-goal-first",
			Command: "go test -count=1 ./modulos/orquesta-app-codex-stack -run TestSimulacion",
		}},
		ArtifactContracts: []orquestagoal.GoalArtifactContractV0{{
			ArtifactRef:  "artifact-ref-simulacion-determinista",
			ArtifactType: "simulation_result",
			Required:     true,
		}},
		EvidenceRefs: []string{"evidence-ref-simulacion-spec"},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireRequiredTests:         true,
			RequireArtifacts:             true,
			RequireArtifactPaths:         true,
			RequireMaterializedArtifacts: true,
			RequiredEvidenceRefs:         []string{"evidence-ref-simulacion-result"},
		},
	}
}

type simulacionBackendGoalFirstV0 struct {
	goalRef         string
	externalGoalRef string
	materialized    bool
	killed          bool
	terminalCount   int
}

func (backend *simulacionBackendGoalFirstV0) LaunchGoalWorkV0(
	context.Context,
	orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	return orquestagoal.GoalLaunchReceiptV0{
		Status:          orquestagoal.GoalStatusRunningV0,
		GoalRef:         backend.goalRef,
		ExternalGoalRef: backend.externalGoalRef,
		EvidenceRefs:    []string{"evidence-ref-simulacion-launch"},
	}, nil
}

func (backend *simulacionBackendGoalFirstV0) ObserveGoalWorkV0(
	context.Context,
	orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	if backend.killed {
		backend.terminalCount++
		return orquestagoal.GoalWorkResultV0{
			Status:          orquestagoal.GoalStatusBlockedV0,
			GoalRef:         backend.goalRef,
			ExternalGoalRef: backend.externalGoalRef,
			Summary:         "backend fake cortado de forma determinista",
			EvidenceRefs:    []string{"evidence-ref-simulacion-backend-killed"},
			Issues:          []orquestagoal.GoalWorkIssueV0{{Code: "backend_killed", Field: "fake_backend"}},
		}, nil
	}
	if !backend.materialized {
		return orquestagoal.GoalWorkResultV0{
			Status:          orquestagoal.GoalStatusRunningV0,
			GoalRef:         backend.goalRef,
			ExternalGoalRef: backend.externalGoalRef,
			Summary:         "result pendiente de materializacion",
			EvidenceRefs:    []string{"evidence-ref-simulacion-running"},
		}, nil
	}
	backend.terminalCount++
	return orquestagoal.GoalWorkResultV0{
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         backend.goalRef,
		ExternalGoalRef: backend.externalGoalRef,
		Summary:         "result materializado y validado en simulacion",
		ArtifactRefs:    []string{"artifact-ref-simulacion-determinista"},
		ArtifactPaths:   []string{"modulos/orquesta-app-codex-stack/docs/simulacion_goal_first_harness_seed_291201.json"},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-simulacion-determinista",
			Path:         "modulos/orquesta-app-codex-stack/docs/simulacion_goal_first_harness_seed_291201.json",
			ArtifactType: "simulation_result",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-simulacion-result"},
		}},
		RequiredTestResults: []orquestagoal.GoalRequiredTestResultV0{{
			TestRef:      "test-ref-simulacion-goal-first",
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-simulacion-test"},
		}},
		EvidenceRefs: []string{"evidence-ref-simulacion-result"},
	}, nil
}

func simulacionEvidenciasEstadoV0(
	state orquestagoal.GoalWorkStateV0,
	fault simulacionFaultModeV0,
) []orquestaestadovivo.EvidenciaEstadoV0 {
	evidencias := []orquestaestadovivo.EvidenciaEstadoV0{{
		RunRef:       state.RunRef,
		GoalRef:      state.GoalRef,
		Fuente:       "run_marker",
		Estado:       state.Status,
		ObservadoEn:  "2026-07-03T11:59:00Z",
		EvidenceRefs: state.EvidenceRefs,
	}}
	if state.LastResult != nil && orquestagoal.GoalWorkResultTerminalV0(state.LastResult.Status) {
		evidencias = append(evidencias, orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:       state.RunRef,
			GoalRef:      state.GoalRef,
			Fuente:       "receipt",
			Estado:       state.LastResult.Status,
			Terminal:     true,
			Aceptado:     state.LastClosure != nil && state.LastClosure.Accepted,
			ObservadoEn:  "2026-07-03T11:59:30Z",
			EvidenceRefs: append(append([]string(nil), state.LastResult.EvidenceRefs...), "evidence-ref-simulacion-terminal"),
		})
	}
	if state.Status == orquestagoal.GoalStatusRunningV0 || fault == simulacionFaultProcessCutV0 {
		evidencias = append(evidencias, orquestaestadovivo.EvidenciaEstadoV0{
			RunRef:                      state.RunRef,
			GoalRef:                     state.GoalRef,
			Fuente:                      "process_snapshot",
			Scope:                       orquestaestadovivo.ScopeGoalExecutionV0,
			RuntimeIdentityRef:          "runtime-ref-simulacion-process-live",
			RuntimeObservationAttempted: true,
			RuntimeObservado:            true,
			ProcesoVivo:                 true,
			ObservadoEn:                 "2026-07-03T11:59:45Z",
			EvidenceRefs:                []string{"evidence-ref-simulacion-process-live"},
		})
	}
	return evidencias
}

func simulacionInjectionPointsV0(t *testing.T, seed int64) []simulacionInjectionPointV0 {
	t.Helper()
	edges := simulacionEnumerarTransicionesBackendV0(t)
	points := make([]simulacionInjectionPointV0, 0, len(edges))
	for _, edge := range edges {
		if simulacionStringInSetV0(edge.Issues, orquestaruntimecodexappserver.BackendTransitionIssueIllegalV0) {
			continue
		}
		points = append(points, simulacionInjectionPointV0{
			Ref:         simulacionBackendEdgeRefV0(edge),
			Kind:        "backend_transition",
			BackendEdge: edge,
		})
	}
	sort.Slice(points, func(i, j int) bool { return points[i].Ref < points[j].Ref })
	rng := rand.New(rand.NewSource(seed))
	rng.Shuffle(len(points), func(i, j int) {
		points[i], points[j] = points[j], points[i]
	})
	return points
}

func simulacionEnumerarTransicionesBackendV0(t *testing.T) []simulacionBackendTransitionEdgeV0 {
	t.Helper()
	states, actions := simulacionConstantesBackendRealesV0(t)
	observations := simulacionObservacionesBackendV0(actions)
	seen := map[string]simulacionBackendTransitionEdgeV0{}
	for _, state := range states {
		for _, obs := range observations {
			to, issues := orquestaruntimecodexappserver.TransicionBackendV0(state, obs)
			edge := simulacionBackendTransitionEdgeV0{
				From:        state,
				To:          to,
				Observation: obs,
				Issues:      append([]string(nil), issues...),
			}
			seen[simulacionBackendEdgeKeyV0(edge)] = edge
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]simulacionBackendTransitionEdgeV0, 0, len(keys))
	for _, key := range keys {
		out = append(out, seen[key])
	}
	return out
}

func simulacionConstantesBackendRealesV0(t *testing.T) ([]orquestaruntimecodexappserver.EstadoBackendAppServerV0, []string) {
	t.Helper()
	file, err := parser.ParseFile(
		token.NewFileSet(),
		"../orquesta-runtime-codex-appserver/estado_backend_v0.go",
		nil,
		0,
	)
	if err != nil {
		t.Fatalf("parse estado_backend_v0.go: %v", err)
	}
	stateSet := map[string]bool{}
	actionSet := map[string]bool{"": true}
	ast.Inspect(file, func(node ast.Node) bool {
		decl, ok := node.(*ast.GenDecl)
		if !ok || decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range valueSpec.Names {
				value := simulacionStringConstValueV0(valueSpec, i)
				if value == "" {
					continue
				}
				if simulacionASTExprNameV0(valueSpec.Type) == "EstadoBackendAppServerV0" {
					stateSet[value] = true
				}
				if strings.HasPrefix(name.Name, "AccionBackend") {
					actionSet[value] = true
				}
			}
		}
		return true
	})
	states := make([]orquestaruntimecodexappserver.EstadoBackendAppServerV0, 0, len(stateSet))
	for state := range stateSet {
		states = append(states, orquestaruntimecodexappserver.EstadoBackendAppServerV0(state))
	}
	actions := make([]string, 0, len(actionSet))
	for action := range actionSet {
		actions = append(actions, action)
	}
	sort.Slice(states, func(i, j int) bool { return states[i] < states[j] })
	sort.Strings(actions)
	if len(states) == 0 || len(actions) <= 1 {
		t.Fatalf("constantes backend insuficientes states=%v actions=%v", states, actions)
	}
	return states, actions
}

func simulacionASTExprNameV0(expr ast.Expr) string {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name
}

func simulacionStringConstValueV0(spec *ast.ValueSpec, index int) string {
	if index >= len(spec.Values) {
		return ""
	}
	literal, ok := spec.Values[index].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return ""
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func simulacionObservacionesBackendV0(actions []string) []orquestaruntimecodexappserver.ObservacionBackendV0 {
	issues := []string{"", "codex_app_server_websocket_handshake_failed"}
	var out []orquestaruntimecodexappserver.ObservacionBackendV0
	for _, action := range actions {
		for _, issue := range issues {
			for mask := 0; mask < 1<<8; mask++ {
				out = append(out, orquestaruntimecodexappserver.ObservacionBackendV0{
					ActionRequested:          action,
					SessionObserved:          mask&(1<<0) != 0,
					OwnerMarkerObserved:      mask&(1<<1) != 0,
					OwnerMarkerValid:         mask&(1<<2) != 0,
					SocketObserved:           mask&(1<<3) != 0,
					PreflightOK:              mask&(1<<4) != 0,
					PanePIDLive:              mask&(1<<5) != 0,
					ProcessBySocketObserved:  mask&(1<<6) != 0,
					ProcessByRuntimeObserved: mask&(1<<7) != 0,
					IssueCode:                issue,
				})
			}
		}
	}
	return out
}

func simulacionApplyBackendTransitionV0(
	edge simulacionBackendTransitionEdgeV0,
	trace *simulacionTraceV0,
) error {
	to, issues := orquestaruntimecodexappserver.TransicionBackendV0(edge.From, edge.Observation)
	trace.AddV0("backend:" + simulacionBackendEdgeRefV0(simulacionBackendTransitionEdgeV0{
		From:        edge.From,
		To:          to,
		Observation: edge.Observation,
		Issues:      issues,
	}))
	if simulacionStringInSetV0(issues, orquestaruntimecodexappserver.BackendTransitionIssueIllegalV0) {
		return trace.ErrorV0("transicion backend ilegal issues=%v", issues)
	}
	if strings.TrimSpace(string(edge.To)) != "" && to != edge.To {
		return trace.ErrorV0("transicion backend no determinista got=%s want=%s", to, edge.To)
	}
	return nil
}

func simulacionBackendEdgeRefV0(edge simulacionBackendTransitionEdgeV0) string {
	return strings.ReplaceAll(simulacionBackendEdgeKeyV0(edge), "|", "_")
}

func simulacionBackendEdgeKeyV0(edge simulacionBackendTransitionEdgeV0) string {
	issues := append([]string(nil), edge.Issues...)
	sort.Strings(issues)
	return strings.Join([]string{
		string(edge.From),
		string(edge.To),
		edge.Observation.ActionRequested,
		simulacionBoolTokenV0(edge.Observation.SessionObserved),
		simulacionBoolTokenV0(edge.Observation.OwnerMarkerObserved),
		simulacionBoolTokenV0(edge.Observation.OwnerMarkerValid),
		simulacionBoolTokenV0(edge.Observation.SocketObserved),
		simulacionBoolTokenV0(edge.Observation.PreflightOK),
		simulacionBoolTokenV0(edge.Observation.PanePIDLive),
		simulacionBoolTokenV0(edge.Observation.ProcessBySocketObserved),
		simulacionBoolTokenV0(edge.Observation.ProcessByRuntimeObserved),
		edge.Observation.IssueCode,
		strings.Join(issues, ","),
	}, "|")
}

func simulacionBoolTokenV0(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func simulacionStringInSetV0(values []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
