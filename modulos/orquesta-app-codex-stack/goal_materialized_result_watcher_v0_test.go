package orquestaappcodexstack

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntimecodexgoal "orquesta/modulos/orquesta-runtime-codex-goal"
)

func TestGoalMaterializedResultWatcherV0DespiertaReconciliacionSinWatchdogV0(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	projectRoot := t.TempDir()
	stack.Ports.GoalObserver = goalFirstReconciledObserverV0{
		Inner:      observer,
		StateStore: stack.Stores.AppGoalStateStore,
		MaterializedResultSource: stackGoalMaterializedRefsSourceV0{
			Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectRoot}},
		},
	}
	wakeupCh := make(chan GoalMaterializedResultWakeupV0, 1)
	observed := make(chan error, 1)
	watcher := NewGoalMaterializedResultWatcherV0(GoalMaterializedResultWatcherConfigV0{
		ProjectWorkDir: projectRoot,
		StateStore:     stack.Stores.AppGoalStateStore,
		Interval:       20 * time.Millisecond,
		Wakeup: func(_ context.Context, wakeup GoalMaterializedResultWakeupV0) bool {
			wakeupCh <- wakeup
			return true
		},
	})
	go watcher.RunBackgroundV0(ctx)
	watcher.NotifyActiveGoalsChangedV0()
	waitGoalMaterializedResultWatcherStatsForTestV0(t, watcher, func(stats GoalMaterializedResultWatcherStatsV0) bool {
		return stats.ActiveRefreshes > 0
	})
	go func() {
		select {
		case <-ctx.Done():
			observed <- ctx.Err()
		case <-wakeupCh:
			result, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{
				List: orquestagoal.GoalWorkStateListRequestV0{MaxItems: 3},
			})
			if err != nil {
				observed <- err
				return
			}
			if len(result.Observations) != 1 ||
				!result.Observations[0].Terminal ||
				!result.Observations[0].Accepted {
				observed <- errGoalMaterializedWatcherUnexpectedResultForTestV0(result.Observations)
				return
			}
			observed <- nil
		}
	}()
	state, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	materializedDir := filepath.Dir(goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(projectRoot, state))
	if err := os.MkdirAll(materializedDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	materializedResult := orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		ArtifactPaths:   goalFirstQueueTechnicalArtifactPathsV0(),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-materialized-result-watcher-required-test",
		),
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
	}
	raw, err := json.Marshal(materializedResult)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(projectRoot, state), raw, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	select {
	case err := <-observed:
		if err != nil {
			t.Fatalf("observed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("result terminal no disparo reconciliacion antes de 2s: stats=%+v", watcher.StatsV0())
	}
	if len(observer.requests) != 0 {
		t.Fatalf("observer backend llamado pese a result materializado: %+v", observer.requests)
	}
	all := listGoalFirstQueueCandidatesForTestV0(t, stack)
	if len(all) != 1 ||
		all[0].RunRef != runRef ||
		all[0].Status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue no sincronizada por wakeup materializado: %+v", all)
	}
	if stats := watcher.StatsV0(); stats.Wakeups != 1 || stats.ResultChecks == 0 {
		t.Fatalf("watcher stats=%+v", stats)
	}
}

func TestGoalMaterializedResultWatcherV0DetectaResultadoTerminalSobrescritoV0(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stack, observer, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	runRef := started.Run.RunID
	spec := launcher.specs[0]
	projectRoot := t.TempDir()
	stack.Ports.GoalObserver = goalFirstReconciledObserverV0{
		Inner:      observer,
		StateStore: stack.Stores.AppGoalStateStore,
		MaterializedResultSource: stackGoalMaterializedRefsSourceV0{
			Config: ConfigV0{Codex: CodexRuntimeConfigV0{ProjectWorkDir: projectRoot}},
		},
	}
	state, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, runRef)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	materializedDir := filepath.Dir(goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(projectRoot, state))
	if err := os.MkdirAll(materializedDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	resultPath := goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(projectRoot, state)
	if err := os.WriteFile(resultPath, []byte(`{"schema_version":"orquesta_goal_result.v0","status":"pending"}`), 0o600); err != nil {
		t.Fatalf("WriteFile pending result: %v", err)
	}

	wakeupCh := make(chan GoalMaterializedResultWakeupV0, 1)
	observed := make(chan error, 1)
	watcher := NewGoalMaterializedResultWatcherV0(GoalMaterializedResultWatcherConfigV0{
		ProjectWorkDir: projectRoot,
		StateStore:     stack.Stores.AppGoalStateStore,
		Interval:       20 * time.Millisecond,
		Wakeup: func(_ context.Context, wakeup GoalMaterializedResultWakeupV0) bool {
			wakeupCh <- wakeup
			return true
		},
	})
	go watcher.RunBackgroundV0(ctx)
	watcher.NotifyActiveGoalsChangedV0()
	waitGoalMaterializedResultWatcherStatsForTestV0(t, watcher, func(stats GoalMaterializedResultWatcherStatsV0) bool {
		return stats.ActiveRefreshes > 0 && stats.ResultChecks > 0
	})
	go func() {
		select {
		case <-ctx.Done():
			observed <- ctx.Err()
		case <-wakeupCh:
			result, err := stack.ObserveActiveGoalWorksV0(ctx, orquestagoal.GoalWorkObserveActiveRequestV0{
				List: orquestagoal.GoalWorkStateListRequestV0{MaxItems: 3},
			})
			if err != nil {
				observed <- err
				return
			}
			if len(result.Observations) != 1 ||
				!result.Observations[0].Terminal ||
				!result.Observations[0].Accepted {
				observed <- errGoalMaterializedWatcherUnexpectedResultForTestV0(result.Observations)
				return
			}
			observed <- nil
		}
	}()
	raw, err := json.Marshal(orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		ArtifactPaths:   goalFirstQueueTechnicalArtifactPathsV0(),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-materialized-result-watcher-overwrite-required-test",
		),
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(resultPath, raw, 0o600); err != nil {
		t.Fatalf("WriteFile terminal result: %v", err)
	}

	select {
	case err := <-observed:
		if err != nil {
			t.Fatalf("observed: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("result terminal sobrescrito no disparo reconciliacion antes de 2s: stats=%+v", watcher.StatsV0())
	}
	if len(observer.requests) != 0 {
		t.Fatalf("observer backend llamado pese a result materializado sobrescrito: %+v", observer.requests)
	}
}

func TestGoalMaterializedResultWatcherV0ResuelveRootPorGoalV0(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stack, _, launcher, started := startGoalFirstQueueSyncStackForTestV0(t)
	state, err := stack.Ports.GoalStateStore.LoadGoalWorkStateV0(ctx, started.Run.RunID)
	if err != nil {
		t.Fatalf("LoadGoalWorkStateV0: %v", err)
	}
	canonicalRoot := t.TempDir()
	workspaceRoot := t.TempDir()
	materializedDir := filepath.Dir(goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(workspaceRoot, state))
	if err := os.MkdirAll(materializedDir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	spec := launcher.specs[0]
	raw, err := json.Marshal(orquestagoal.GoalWorkResultV0{
		SchemaVersion:   orquestagoal.GoalWorkResultSchemaV0,
		Status:          orquestagoal.GoalStatusCompleteV0,
		GoalRef:         spec.GoalRef,
		ExternalGoalRef: started.ExternalGoalRef,
		ArtifactRefs:    goalFirstQueueRequiredArtifactRefsV0(spec),
		ArtifactPaths:   goalFirstQueueTechnicalArtifactPathsV0(),
		RequiredTestResults: goalFirstQueueRequiredTestResultsV0(
			spec,
			"evidence-ref-goal-materialized-result-workspace-required-test",
		),
		EvidenceRefs: spec.ClosurePolicy.RequiredEvidenceRefs,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if err := os.WriteFile(goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(workspaceRoot, state), raw, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	wakeupCh := make(chan GoalMaterializedResultWakeupV0, 1)
	watcher := NewGoalMaterializedResultWatcherV0(GoalMaterializedResultWatcherConfigV0{
		ProjectWorkDir:      canonicalRoot,
		ProjectRootResolver: fixedGoalMaterializedResultRootResolverForTestV0{root: workspaceRoot},
		StateStore:          stack.Stores.AppGoalStateStore,
		Interval:            20 * time.Millisecond,
		Wakeup: func(_ context.Context, wakeup GoalMaterializedResultWakeupV0) bool {
			wakeupCh <- wakeup
			return true
		},
	})
	go watcher.RunBackgroundV0(ctx)
	watcher.NotifyActiveGoalsChangedV0()
	select {
	case wakeup := <-wakeupCh:
		if wakeup.RunRef != state.RunRef || wakeup.GoalRef != state.GoalRef {
			t.Fatalf("wakeup no causal: %+v", wakeup)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("resultado en workspace no detectado: stats=%+v", watcher.StatsV0())
	}
}

type fixedGoalMaterializedResultRootResolverForTestV0 struct {
	root string
	err  error
}

func (resolver fixedGoalMaterializedResultRootResolverForTestV0) ResolveGoalMaterializedResultProjectRootV0(
	context.Context,
	orquestagoal.GoalWorkStateV0,
) (string, error) {
	return resolver.root, resolver.err
}

func goalMaterializedResultWatcherCanonicalReceiptPathForTestV0(
	projectRoot string,
	state orquestagoal.GoalWorkStateV0,
) string {
	goalRef := goalMaterializedStateGoalRefV0(state)
	return filepath.Join(
		projectRoot,
		filepath.FromSlash(orquestaruntimecodexgoal.CodexGoalRuntimeReceiptRelativeDirV0(goalRef)),
		orquestaruntimecodexgoal.CodexGoalResultFileNameForGoalRefV0(goalRef),
	)
}

func TestGoalMaterializedResultWatcherV0SinGoalsActivosNoPollV0(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher := NewGoalMaterializedResultWatcherV0(GoalMaterializedResultWatcherConfigV0{
		ProjectWorkDir: t.TempDir(),
		StateStore:     newGoalFirstQueueStateStoreForTestV0(),
		Interval:       10 * time.Millisecond,
		Wakeup: func(context.Context, GoalMaterializedResultWakeupV0) bool {
			t.Fatalf("sin goals activos no debe despertar")
			return false
		},
	})
	go watcher.RunBackgroundV0(ctx)
	waitGoalMaterializedResultWatcherStatsForTestV0(t, watcher, func(stats GoalMaterializedResultWatcherStatsV0) bool {
		return stats.NoActiveSleeps > 0
	})
	statsBefore := watcher.StatsV0()
	time.Sleep(50 * time.Millisecond)
	statsAfter := watcher.StatsV0()
	if statsBefore.DirectoryPolls != 0 ||
		statsAfter.DirectoryPolls != 0 ||
		statsAfter.ResultChecks != 0 ||
		statsAfter.Wakeups != 0 ||
		statsAfter.ActiveRefreshes != statsBefore.ActiveRefreshes {
		t.Fatalf("watcher hizo trabajo observable sin goals activos: before=%+v after=%+v", statsBefore, statsAfter)
	}
}

func waitGoalMaterializedResultWatcherStatsForTestV0(
	t *testing.T,
	watcher *GoalMaterializedResultWatcherV0,
	accept func(GoalMaterializedResultWatcherStatsV0) bool,
) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if accept(watcher.StatsV0()) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("watcher stats no alcanzaron condicion: %+v", watcher.StatsV0())
}

type errGoalMaterializedWatcherUnexpectedResultForTestV0 []orquestagoal.GoalWorkObserveResultV0

func (err errGoalMaterializedWatcherUnexpectedResultForTestV0) Error() string {
	return "goal materialized watcher unexpected result"
}
