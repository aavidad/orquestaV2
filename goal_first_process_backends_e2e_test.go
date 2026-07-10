package orquesta_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestafactory "orquesta/modulos/orquesta-factory"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruntime "orquesta/modulos/orquesta-runtime"
	orquestaruntimeclaude "orquesta/modulos/orquesta-runtime-claude"
	orquestaruntimegemini "orquesta/modulos/orquesta-runtime-gemini"
)

func TestGoalFirstProcessBackendsE2EV0ReworkThenClose(t *testing.T) {
	for _, backend := range []struct {
		name       string
		resultPath string
		newBackend func(root, commandPath string) goalFirstProcessBackendV0
	}{
		{
			name:       "claude",
			resultPath: filepath.Join("docs", orquestaruntimeclaude.ClaudeGoalResultFileNameV0),
			newBackend: func(root, commandPath string) goalFirstProcessBackendV0 {
				projectDir := filepath.Join(root, "project")
				runtimeDir := filepath.Join(root, "runtime")
				backend := &orquestaruntimeclaude.ClaudeGoalProcessBackendV0{
					Control: orquestaruntimeclaude.ClaudeGoalBackendV0{ProjectWorkDir: projectDir, RuntimeWorkDir: runtimeDir},
					Profile: orquestaruntimeclaude.ClaudeConnectorProfileV0{
						SchemaVersion:  orquestaruntimeclaude.ClaudeConnectorProfileSchemaVersionV0,
						OptIn:          true,
						CommandPath:    commandPath,
						ProjectWorkDir: projectDir,
						RuntimeWorkDir: runtimeDir,
						Model:          "fake",
						Effort:         "medium",
						PermissionMode: "bypassPermissions",
						OutputFormat:   "text",
					},
					ProcessRuntime: orquestaruntime.NewProcessRuntimeConnectorV0(),
				}
				return goalFirstProcessBackendV0{
					launcher: backend,
					observer: backend,
					stop: func(ctx context.Context, goalRef string) {
						_, _ = backend.StopClaudeGoalV0(ctx, orquestaruntimeclaude.ClaudeGoalStopRequestV0{GoalRef: goalRef, Action: "stop", Reason: "test_cleanup", Forced: true})
					},
				}
			},
		},
		{
			name:       "gemini",
			resultPath: orquestaruntimegemini.GeminiGoalResultFileNameV0,
			newBackend: func(root, commandPath string) goalFirstProcessBackendV0 {
				projectDir := filepath.Join(root, "project")
				runtimeDir := filepath.Join(root, "runtime")
				backend := &orquestaruntimegemini.GeminiGoalProcessBackendV0{
					Control: orquestaruntimegemini.GeminiGoalBackendV0{ProjectWorkDir: projectDir, RuntimeWorkDir: runtimeDir},
					Profile: orquestaruntimegemini.GeminiConnectorProfileV0{
						SchemaVersion:  orquestaruntimegemini.GeminiConnectorProfileSchemaVersionV0,
						OptIn:          true,
						CommandPath:    commandPath,
						ProjectWorkDir: projectDir,
						RuntimeWorkDir: runtimeDir,
						ApprovalMode:   "auto_edit",
						OutputFormat:   "text",
					},
					ProcessRuntime: orquestaruntime.NewProcessRuntimeConnectorV0(),
				}
				return goalFirstProcessBackendV0{
					launcher: backend,
					observer: backend,
					stop: func(ctx context.Context, goalRef string) {
						_, _ = backend.StopGeminiGoalV0(ctx, orquestaruntimegemini.GeminiGoalStopRequestV0{GoalRef: goalRef, Action: "stop", Reason: "test_cleanup", Forced: true})
					},
				}
			},
		},
	} {
		t.Run(backend.name, func(t *testing.T) {
			request := goalFirstProcessRequestV0(backend.name)
			preview, err := orquestaappdirectorservice.BuildStartAppDirectorGoalWorkPreviewV0(context.Background(), request)
			if err != nil {
				t.Fatalf("BuildStartAppDirectorGoalWorkPreviewV0: %v", err)
			}
			if preview.Status != orquestaappdirectorservice.StartAppDirectorStatusPreviewReadyV0 {
				t.Fatalf("preview=%+v", preview)
			}

			first := goalFirstProcessResultV0(preview.GoalSpec, false, backend.resultPath)
			reworkSpec := preview.GoalSpec
			reworkSpec.GoalRef += "-rework-1"
			second := goalFirstProcessResultV0(reworkSpec, true, backend.resultPath)
			root := t.TempDir()
			commandPath := goalFirstProcessFakeCommandV0(t, root, first, second)
			processBackend := backend.newBackend(root, commandPath)
			t.Cleanup(func() {
				processBackend.stop(context.Background(), preview.GoalSpec.GoalRef)
				processBackend.stop(context.Background(), reworkSpec.GoalRef)
			})

			runs := orquestacionnucleoapp.NewInMemoryRunStoreV0()
			events := orquestacionnucleoapp.NewInMemoryEventSinkV0()
			states := newGoalFirstProcessStateStoreV0()
			ports := orquestaappdirectorservice.StartAppDirectorPortsV0{
				RunStore:             runs,
				EventSink:            events,
				GoalLauncher:         processBackend.launcher,
				GoalReworkLauncher:   processBackend.launcher,
				GoalObserver:         processBackend.observer,
				GoalClosureValidator: orquestagoal.DefaultGoalWorkClosureValidatorV0{},
				GoalStateStore:       states,
			}
			started, err := orquestaappdirectorservice.StartAppDirectorV0(context.Background(), request, ports)
			if err != nil {
				t.Fatalf("StartAppDirectorV0: %v", err)
			}
			if started.DirectorExecutionMode != orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0 || started.GoalRef != preview.GoalSpec.GoalRef {
				t.Fatalf("started=%+v preview=%+v", started, preview)
			}

			firstObservation := goalFirstProcessWaitObservationV0(t, request.RunRef, ports, func(result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0) bool {
				return result.GoalRef == reworkSpec.GoalRef
			})
			if firstObservation.Run.Status != orquestacoreworkflow.OrchestrationRunStatusActiveV0 || firstObservation.Closure.Accepted || firstObservation.GoalResult.Status != orquestagoal.GoalStatusRunningV0 {
				t.Fatalf("first observation=%+v", firstObservation)
			}
			state, err := states.LoadGoalWorkStateV0(context.Background(), request.RunRef)
			if err != nil {
				t.Fatalf("LoadGoalWorkStateV0 after rework: %v", err)
			}
			if state.GoalRef != reworkSpec.GoalRef || !strings.Contains(strings.Join(state.Spec.AcceptanceCriteria, "\n"), "required_tests") {
				t.Fatalf("rework state=%+v", state)
			}

			closed := goalFirstProcessWaitObservationV0(t, request.RunRef, ports, func(result orquestaappdirectorservice.ObserveAppDirectorGoalResultV0) bool {
				return result.Closure.Accepted
			})
			if closed.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 || closed.GoalResult.GoalRef != reworkSpec.GoalRef || !closed.Closure.Accepted || len(closed.GoalResult.RequiredTestResults) != len(preview.GoalSpec.RequiredTests) {
				t.Fatalf("closed=%+v", closed)
			}
			if len(closed.Closure.EvidenceRefs) == 0 || len(closed.GoalResult.ArtifactRefs) != len(preview.GoalSpec.ArtifactContracts) {
				t.Fatalf("closure/result evidence missing: closure=%+v result=%+v", closed.Closure, closed.GoalResult)
			}
		})
	}
}

type goalFirstProcessBackendV0 struct {
	launcher orquestagoal.GoalWorkLauncherPortV0
	observer orquestagoal.GoalWorkObservationPortV0
	stop     func(context.Context, string)
}

type goalFirstProcessStateStoreV0 struct {
	mu     sync.Mutex
	states map[string]orquestagoal.GoalWorkStateV0
}

func newGoalFirstProcessStateStoreV0() *goalFirstProcessStateStoreV0 {
	return &goalFirstProcessStateStoreV0{states: map[string]orquestagoal.GoalWorkStateV0{}}
}

func (store *goalFirstProcessStateStoreV0) SaveGoalWorkStateV0(_ context.Context, state orquestagoal.GoalWorkStateV0) error {
	state, err := orquestagoal.NewGoalWorkStateV0(state)
	if err != nil {
		return err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.states[state.RunRef] = state
	return nil
}

func (store *goalFirstProcessStateStoreV0) LoadGoalWorkStateV0(_ context.Context, runRef string) (orquestagoal.GoalWorkStateV0, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	state, ok := store.states[runRef]
	if !ok {
		return orquestagoal.GoalWorkStateV0{}, fmt.Errorf("goal state not found: %s", runRef)
	}
	return state, nil
}

func goalFirstProcessRequestV0(backend string) orquestaappdirectorservice.StartAppDirectorRequestV0 {
	observability := true
	return orquestaappdirectorservice.StartAppDirectorRequestV0{
		RunRef:                "run-goal-first-process-" + backend,
		ProjectRef:            "project-goal-first-process-" + backend,
		OccurredAt:            "2026-07-11T09:00:00Z",
		CorrelationID:         "corr-goal-first-process-" + backend,
		RequestedBy:           "goal-first-process-e2e",
		DirectorExecutionMode: orquestaappdirectorservice.AppDirectorExecutionModeGoalFirstV0,
		AppSpecRequest: orquestafactory.AppSpecRequestV0{
			SchemaVersion: orquestafactory.AppSpecRequestSchemaV0,
			RequestID:     "request-goal-first-process-" + backend,
			Source:        "orquesta-web",
			Locale:        "es-ES",
			Nombre:        "Agenda " + backend,
			Objetivo:      "Gestionar citas mediante una API local.",
			TipoApp:       "mixed",
			Calidad:       orquestafactory.CalidadRequestV0{Pruebas: "media", Observabilidad: &observability},
		},
	}
}

func goalFirstProcessResultV0(spec orquestagoal.GoalWorkSpecV0, passed bool, resultPathSuffix string) orquestagoal.GoalWorkResultV0 {
	artifactRefs := make([]string, 0, len(spec.ArtifactContracts))
	for _, artifact := range spec.ArtifactContracts {
		if artifact.Required {
			artifactRefs = append(artifactRefs, artifact.ArtifactRef)
		}
	}
	tests := make([]orquestagoal.GoalRequiredTestResultV0, 0, len(spec.RequiredTests))
	for _, required := range spec.RequiredTests {
		status := "failed"
		if passed {
			status = "passed"
		}
		tests = append(tests, orquestagoal.GoalRequiredTestResultV0{
			TestRef: required.TestRef, Status: status, EvidenceRefs: []string{"evidence-ref-fake-process-test-" + status},
		})
	}
	resultPath := filepath.ToSlash(filepath.Join(spec.WriteSet[0].Path, resultPathSuffix))
	return orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       spec.GoalRef,
		Summary:       "offline fake process delivery",
		ArtifactRefs:  artifactRefs,
		ArtifactPaths: []string{resultPath},
		MaterializedArtifacts: []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-fake-process-result-" + spec.GoalRef,
			Path:         resultPath,
			ArtifactType: "goal_result",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-fake-process-result"},
		}},
		RequiredTestResults: tests,
		EvidenceRefs:        append(append([]string{}, spec.ClosurePolicy.RequiredEvidenceRefs...), "evidence-ref-fake-process-result"),
	}
}

func goalFirstProcessFakeCommandV0(t *testing.T, root string, first, second orquestagoal.GoalWorkResultV0) string {
	t.Helper()
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatalf("marshal first result: %v", err)
	}
	secondJSON, err := json.Marshal(second)
	if err != nil {
		t.Fatalf("marshal second result: %v", err)
	}
	resultPath := filepath.ToSlash(filepath.Join(first.ArtifactPaths[0]))
	script := "#!/bin/sh\nset -eu\nprompt=$(cat)\nmkdir -p \"$(dirname '" + resultPath + "')\"\ncase \"$prompt\" in\n  *'" + second.GoalRef + "'*) payload='" + strings.ReplaceAll(string(secondJSON), "'", "'\\\"'\\\"'") + "' ;;\n  *'" + first.GoalRef + "'*) payload='" + strings.ReplaceAll(string(firstJSON), "'", "'\\\"'\\\"'") + "' ;;\n  *) exit 7 ;;\nesac\nprintf '%s\\n' \"$payload\" > '" + resultPath + "'\n"
	path := filepath.Join(root, "fake-goal-process")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake process: %v", err)
	}
	return path
}

func goalFirstProcessWaitObservationV0(
	t *testing.T,
	runRef string,
	ports orquestaappdirectorservice.StartAppDirectorPortsV0,
	ready func(orquestaappdirectorservice.ObserveAppDirectorGoalResultV0) bool,
) orquestaappdirectorservice.ObserveAppDirectorGoalResultV0 {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	var last orquestaappdirectorservice.ObserveAppDirectorGoalResultV0
	for time.Now().Before(deadline) {
		result, err := orquestaappdirectorservice.ObserveAppDirectorGoalV0(context.Background(), orquestaappdirectorservice.ObserveAppDirectorGoalRequestV0{RunRef: runRef}, ports)
		if err != nil {
			t.Fatalf("ObserveAppDirectorGoalV0: %v", err)
		}
		if ready(result) {
			return result
		}
		last = result
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout waiting goal-first process observation: last=%+v", last)
	return orquestaappdirectorservice.ObserveAppDirectorGoalResultV0{}
}
