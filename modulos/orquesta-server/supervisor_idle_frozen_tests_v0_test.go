package orquestaserver

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestarunsupervisor "orquesta/modulos/orquesta-run-supervisor"
)

func TestRuntimeV0IdleSelfImprovementFrozenTestsOptInEncadenaDefinidorEImplementadorV0(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	clock := &mutableClockForGoalFirstChainTestV0{now: now}
	projectDir := t.TempDir()
	store := &memoryStateStoreV0{}
	goalStates := newMemoryGoalStateStoreV0()
	base := idleSelfImprovementFrozenTestsRequestForTestV0()
	supervisor := &frozenTestsGoalSupervisorForTestV0{
		projectDir:    projectDir,
		planBatches:   [][]IdleSelfImprovementRequestV0{{base}, {base}},
		launched:      []orquestagoal.GoalWorkSpecV0{},
		started:       make(chan struct{}, 2),
		frozenRelPath: "modulos/orquesta-server/frozen_actor_test.go",
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                          t.TempDir(),
		ProjectWorkDir:                    projectDir,
		IdleSelfImprovementProjectWorkDir: projectDir,
		TickInterval:                      time.Hour,
		IdleSelfImprovementAfter:          time.Minute,
		IdleSelfImprovementMaxRequests:    1,
		IdleSelfImprovementGoalFirst:      true,
		IdleSelfImprovementFrozenTests:    true,
		IdleSelfImprovementProjectRef:     "project-ref-frozen-tests",
		AuditDisabled:                     true,
	}, RuntimeDepsV0{
		Supervisor:     supervisor,
		GoalStateStore: goalStates,
		StateStore:     store,
		Clock:          clock,
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	markNoExecutionSinceForTestV0(runtime, now)

	runtime.runSupervisorTickV0(ctx)
	waitFrozenTestsLaunchForTestV0(t, supervisor)
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	definer := supervisor.launched[0]
	if definer.RequestRef != base.RequestRef+idleSelfImprovementFrozenTestsDefinerSuffixV0 ||
		definer.WorkProfileKind != "required_tests" ||
		len(definer.RequiredTests) != 0 ||
		!orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(definer, orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0) ||
		!containsGoalRuleRefForTestV0(definer.RuleRefs, orquestaautoprogramming.FrozenRequiredTestsRuleRefV0, orquestagoal.GoalRuleEnforcementHardV0) {
		t.Fatalf("definer spec=%+v", definer)
	}

	clock.now = now.Add(2 * time.Minute)
	runtime.runSupervisorTickV0(ctx)
	if supervisor.observeCalls != 1 ||
		store.snapshotV0().IdleSelfImprovementGoalClosure == nil ||
		!store.snapshotV0().IdleSelfImprovementGoalClosure.Accepted {
		t.Fatalf("definer closure observe=%d state=%+v", supervisor.observeCalls, store.snapshotV0())
	}

	clock.now = now.Add(4 * time.Minute)
	runtime.runSupervisorTickV0(ctx)
	waitFrozenTestsLaunchForTestV0(t, supervisor)
	waitRuntimeAsyncWorkForTestV0(t, runtime)
	if len(supervisor.launched) != 2 {
		t.Fatalf("launched=%d specs=%+v", len(supervisor.launched), supervisor.launched)
	}
	implementer := supervisor.launched[1]
	frozen := orquestaautoprogramming.ParseFrozenRequiredTestContextRefsV0(implementer.ContextRefs)
	if implementer.RequestRef != base.RequestRef+idleSelfImprovementFrozenTestsImplementerSuffixV0 ||
		implementer.WorkProfileKind != "implementation" ||
		len(implementer.RequiredTests) != 1 ||
		!orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(implementer, orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0) ||
		len(frozen) != 1 ||
		frozen[0].Path != supervisor.frozenRelPath ||
		!containsGoalRuleRefForTestV0(implementer.RuleRefs, orquestaautoprogramming.FrozenRequiredTestsRuleRefV0, orquestagoal.GoalRuleEnforcementHardV0) {
		t.Fatalf("implementer spec=%+v frozen=%+v", implementer, frozen)
	}
}

func TestRuntimeV0IdleSelfImprovementFrozenTestsSinOptInNoCambiaSpecV0(t *testing.T) {
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                     t.TempDir(),
		IdleSelfImprovementGoalFirst: true,
		AuditDisabled:                true,
	}, RuntimeDepsV0{
		Supervisor:     &frozenTestsGoalSupervisorForTestV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	base := idleSelfImprovementFrozenTestsRequestForTestV0()
	requests := runtime.idleSelfImprovementFrozenTestsRequestsV0([]IdleSelfImprovementRequestV0{base})
	if len(requests) != 1 || requests[0].RequestRef != base.RequestRef {
		t.Fatalf("requests=%+v", requests)
	}
	spec := runtime.idleSelfImprovementGoalWorkSpecV0(base)
	if spec.RequestRef != base.RequestRef ||
		spec.WorkProfileKind != "implementation" ||
		orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(spec, orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0) {
		t.Fatalf("spec=%+v", spec)
	}
}

func TestRuntimeV0IdleSelfImprovementFrozenTestsFallbackBloqueaHashModificadoV0(t *testing.T) {
	projectDir := t.TempDir()
	rel := "modulos/orquesta-server/frozen_actor_test.go"
	writeFrozenTestFileForServerTestV0(t, projectDir, rel, "package orquestaserver\n\nfunc TestFrozenOriginal(t *testing.T) {}\n")
	sum, err := fileSHA256ForServerTestV0(filepath.Join(projectDir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("sha: %v", err)
	}
	frozenRef, ok := orquestaautoprogramming.FrozenRequiredTestContextRefV0(
		orquestaautoprogramming.FrozenRequiredTestV0{Path: rel, SHA256: sum},
	)
	if !ok {
		t.Fatalf("frozen ref invalida")
	}
	runtime, err := NewRuntimeV0(ConfigV0{
		StateDir:                          t.TempDir(),
		ProjectWorkDir:                    projectDir,
		IdleSelfImprovementProjectWorkDir: projectDir,
		AuditDisabled:                     true,
	}, RuntimeDepsV0{
		Supervisor:     &frozenTestsGoalSupervisorForTestV0{},
		GoalStateStore: newMemoryGoalStateStoreV0(),
		StateStore:     &memoryStateStoreV0{},
	})
	if err != nil {
		t.Fatalf("NewRuntimeV0: %v", err)
	}
	spec := runtime.idleSelfImprovementGoalWorkSpecV0(idleSelfImprovementFrozenTestsRequestForTestV0())
	spec.ContextRefs = append(spec.ContextRefs,
		orquestaautoprogramming.FrozenRequiredTestsPhaseContextRefV0(orquestaautoprogramming.FrozenRequiredTestsImplementerPhaseV0),
		frozenRef,
	)
	runtime.tracker.MarkIdleSelfImprovementPreparedV0(IdleSelfImprovementResultV0{
		Accepted:    true,
		RequestRef:  spec.RequestRef,
		RunRef:      spec.RunRef,
		GoalRef:     spec.GoalRef,
		Status:      orquestagoal.GoalStatusAcceptedV0,
		Message:     "goal_first_launched",
		GoalSpec:    spec,
		GoalReceipt: orquestagoal.GoalLaunchReceiptV0{Status: orquestagoal.GoalStatusAcceptedV0, GoalRef: spec.GoalRef},
	}, time.Now().UTC())
	writeFrozenTestFileForServerTestV0(t, projectDir, rel, "package orquestaserver\n\nfunc TestFrozenModified(t *testing.T) {}\n")
	result := runtime.idleSelfImprovementResultWithFrozenTestGuardV0(orquestagoal.GoalWorkResultV0{
		SchemaVersion: orquestagoal.GoalWorkResultSchemaV0,
		Status:        orquestagoal.GoalStatusCompleteV0,
		GoalRef:       spec.GoalRef,
		Summary:       "implementado",
	})
	if result.Status != orquestagoal.GoalStatusBlockedV0 ||
		!goalWorkResultHasIssueCodeV0(result, orquestaautoprogramming.FrozenRequiredTestsModifiedIssueCodeV0) {
		t.Fatalf("result=%+v", result)
	}
}

type frozenTestsGoalSupervisorForTestV0 struct {
	projectDir    string
	planBatches   [][]IdleSelfImprovementRequestV0
	planCalls     int
	launchCalls   int
	observeCalls  int
	launched      []orquestagoal.GoalWorkSpecV0
	started       chan struct{}
	frozenRelPath string
}

func (fake *frozenTestsGoalSupervisorForTestV0) RunGlobalSupervisorV0(
	context.Context,
	orquestarunsupervisor.RunSupervisorCommandV0,
) (orquestarunsupervisor.RunSupervisorResultV0, error) {
	return orquestarunsupervisor.RunSupervisorResultV0{
		StopReason: orquestarunsupervisor.RunSupervisorStopNoExecutionV0,
	}, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) PlanIdleSelfImprovementV0(
	_ context.Context,
	request IdleSelfImprovementPlanRequestV0,
) (IdleSelfImprovementPlanResultV0, error) {
	fake.planCalls++
	index := fake.planCalls - 1
	if index < len(fake.planBatches) {
		return IdleSelfImprovementPlanResultV0{Requests: append([]IdleSelfImprovementRequestV0(nil), fake.planBatches[index]...)}, nil
	}
	return IdleSelfImprovementPlanResultV0{Requests: []IdleSelfImprovementRequestV0{request.BaseRequest}}, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) FilterIdleSelfImprovementRequestsV0(
	_ context.Context,
	request IdleSelfImprovementRequestFilterRequestV0,
) (IdleSelfImprovementRequestFilterResultV0, error) {
	return IdleSelfImprovementRequestFilterResultV0{Requests: append([]IdleSelfImprovementRequestV0(nil), request.Requests...)}, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) IdleSelfImprovementBlockersV0(
	context.Context,
	IdleSelfImprovementBlockerRequestV0,
) (IdleSelfImprovementBlockerResultV0, error) {
	return IdleSelfImprovementBlockerResultV0{}, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) RetryableIdleSelfImprovementRunRefsV0(
	context.Context,
	IdleSelfImprovementRunFreshnessRequestV0,
) (IdleSelfImprovementRunFreshnessResultV0, error) {
	return IdleSelfImprovementRunFreshnessResultV0{}, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) LaunchGoalWorkV0(
	_ context.Context,
	spec orquestagoal.GoalWorkSpecV0,
) (orquestagoal.GoalLaunchReceiptV0, error) {
	fake.launchCalls++
	fake.launched = append(fake.launched, copyGoalWorkSpecForServerStateV0(spec))
	if fake.started != nil {
		fake.started <- struct{}{}
	}
	return orquestagoal.GoalLaunchReceiptV0{
		SchemaVersion:   orquestagoal.GoalWorkLaunchReceiptSchemaV0,
		Status:          orquestagoal.GoalStatusAcceptedV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: "external-" + spec.GoalRef,
		EvidenceRefs:    []string{"evidence-ref-frozen-tests-launch"},
	}, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) ObserveGoalWorkV0(
	_ context.Context,
	request orquestagoal.GoalObservationRequestV0,
) (orquestagoal.GoalWorkResultV0, error) {
	fake.observeCalls++
	spec := fake.specForGoalRefV0(request.GoalRef)
	result := orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         request.GoalRef,
		ExternalGoalRef: request.ExternalGoalRef,
		Summary:         "goal frozen tests completo",
		EvidenceRefs:    []string{"evidence-ref-frozen-tests-result"},
	}
	if orquestaautoprogramming.GoalHasFrozenRequiredTestsPhaseV0(spec, orquestaautoprogramming.FrozenRequiredTestsDefinerPhaseV0) {
		writeFrozenTestFileForServerTestV0(nil, fake.projectDir, fake.frozenRelPath, "package orquestaserver\n\nimport \"testing\"\n\nfunc TestFrozenActorCritical(t *testing.T) {}\n")
		result.ArtifactPaths = []string{fake.frozenRelPath}
		result.MaterializedArtifacts = []orquestagoal.GoalMaterializedArtifactV0{{
			ArtifactRef:  "artifact-ref-frozen-required-test",
			Path:         fake.frozenRelPath,
			ArtifactType: "required_test",
			Status:       orquestagoal.GoalMaterializedArtifactStatusValidV0,
			EvidenceRefs: []string{"evidence-ref-frozen-test-file"},
		}}
		result.Checklist = orquestagoal.GoalWorkChecklistV0{
			ExpectedRefs:  []string{"frozen_required_test"},
			CompletedRefs: []string{"frozen_required_test"},
			EvidenceRefs:  []string{"evidence-ref-frozen-test-file"},
		}
		return result, nil
	}
	for _, test := range spec.RequiredTests {
		result.RequiredTestResults = append(result.RequiredTestResults, orquestagoal.GoalRequiredTestResultV0{
			TestRef:      test.TestRef,
			Status:       "passed",
			EvidenceRefs: []string{"evidence-ref-frozen-tests-required-test-passed"},
		})
	}
	return result, nil
}

func (fake *frozenTestsGoalSupervisorForTestV0) specForGoalRefV0(goalRef string) orquestagoal.GoalWorkSpecV0 {
	for _, spec := range fake.launched {
		if spec.GoalRef == goalRef {
			return spec
		}
	}
	return orquestagoal.GoalWorkSpecV0{}
}

func idleSelfImprovementFrozenTestsRequestForTestV0() IdleSelfImprovementRequestV0 {
	return IdleSelfImprovementRequestV0{
		RequestRef:         "request-ref-frozen-tests-actor-critico",
		ProjectRef:         "project-ref-frozen-tests",
		FailureSummary:     "corregir actor critico con tests congelados",
		SuggestedArea:      "modulos-orquesta-server",
		WriteSet:           []string{"modulos/orquesta-server"},
		RequiredTests:      []string{"go test -count=1 ./modulos/orquesta-server"},
		AcceptanceCriteria: []string{"tests congelados antes del implementador"},
		EvidenceRefs:       []string{"evidence-ref-frozen-tests-input"},
	}
}

func waitFrozenTestsLaunchForTestV0(t *testing.T, supervisor *frozenTestsGoalSupervisorForTestV0) {
	t.Helper()
	select {
	case <-supervisor.started:
	case <-time.After(time.Second):
		t.Fatalf("launch congelado no observado")
	}
}

func writeFrozenTestFileForServerTestV0(t *testing.T, projectDir string, rel string, body string) {
	if t != nil {
		t.Helper()
	}
	path := filepath.Join(projectDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		if t != nil {
			t.Fatalf("mkdir frozen test: %v", err)
		}
		return
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		if t != nil {
			t.Fatalf("write frozen test: %v", err)
		}
		return
	}
}

func fileSHA256ForServerTestV0(path string) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.ToLower(hexSHA256ForServerTestV0(body)), nil
}

func hexSHA256ForServerTestV0(body []byte) string {
	sum := sha256ForServerTestV0(body)
	return sum
}

func sha256ForServerTestV0(body []byte) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h[:])
}
