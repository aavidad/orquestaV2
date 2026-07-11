package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestagoal "orquesta/modulos/orquesta-goal"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestastatefile "orquesta/modulos/orquesta-state-file"
)

func TestAutoprogrammingBatchE2ETemporalRestartGitGateFinalizerYReplayV0(t *testing.T) {
	t.Run("dos goals cierran, integran y finalizan una vez", func(t *testing.T) {
		fixture := newServerAutoprogrammingBatchE2EV0(t, false)

		handled, complete, _, err := fixture.stack.ReconcileClosedAutoprogrammingBatchRunV0(fixture.ctx, fixture.runs[0])
		if err != nil {
			t.Fatalf("primer cierre batch: %v", err)
		}
		first := fixture.loadBatchV0(t)
		if !handled || complete || first.Status != orquestaautoprogramming.AutoprogrammingBatchStatusGoalsRunningV0 ||
			first.Members[0].FocalStatus != orquestaautoprogramming.AutoprogrammingBatchFocalClosedV0 ||
			first.Members[1].FocalStatus != orquestaautoprogramming.AutoprogrammingBatchFocalRunningV0 {
			t.Fatalf("primer goal cerro o promovio batch: complete=%v batch=%+v", complete, first)
		}
		fixture.assertEffectsV0(t, fixture.baseCommitCount, 0, 0, 0)

		fixture.restartStoreV0(t)
		handled, complete, _, err = fixture.stack.ReconcileClosedAutoprogrammingBatchRunV0(fixture.ctx, fixture.runs[1])
		if err != nil {
			t.Fatalf("segundo cierre batch tras restart: %v", err)
		}
		closed := fixture.loadBatchV0(t)
		if !handled || !complete || closed.Status != orquestaautoprogramming.AutoprogrammingBatchStatusClosedV0 {
			t.Fatalf("batch no cerrado: complete=%v batch=%+v", complete, closed)
		}
		fixture.assertIntegratedChainV0(t, closed)
		fixture.assertEffectsV0(t, fixture.baseCommitCount+2, 1, 1, 1)
		if head := fixture.gitV0(t, fixture.repo, "rev-parse", "HEAD"); head != closed.IntegratedRevision {
			t.Fatalf("HEAD=%s integrated_revision=%s", head, closed.IntegratedRevision)
		}
		if status := fixture.gitV0(t, fixture.repo, "status", "--porcelain", "--untracked-files=all"); status != "" {
			t.Fatalf("checkout canonico sucio tras finalizacion: %q", status)
		}

		version := closed.StoreVersion
		handled, complete, _, err = fixture.stack.ReconcileClosedAutoprogrammingBatchRunV0(fixture.ctx, fixture.runs[1])
		if err != nil || !handled || !complete {
			t.Fatalf("replay cierre batch: handled=%v complete=%v err=%v", handled, complete, err)
		}
		replayed := fixture.loadBatchV0(t)
		if replayed.StoreVersion != version || replayed.Status != orquestaautoprogramming.AutoprogrammingBatchStatusClosedV0 {
			t.Fatalf("replay mutante: before=%d after=%+v", version, replayed)
		}
		fixture.assertEffectsV0(t, fixture.baseCommitCount+2, 1, 1, 1)
	})

	t.Run("cambio post gate bloquea sin commit", func(t *testing.T) {
		fixture := newServerAutoprogrammingBatchE2EV0(t, true)
		if handled, complete, _, err := fixture.stack.ReconcileClosedAutoprogrammingBatchRunV0(fixture.ctx, fixture.runs[0]); err != nil || !handled || complete {
			t.Fatalf("primer cierre negativo: handled=%v complete=%v err=%v", handled, complete, err)
		}
		fixture.restartStoreV0(t)

		handled, complete, _, err := fixture.stack.ReconcileClosedAutoprogrammingBatchRunV0(fixture.ctx, fixture.runs[1])
		if err == nil || err.Error() != "autoprogramming_batch_promotion_canonical_dirty" || !handled || complete {
			t.Fatalf("cambio post-gate no freno finalizador: handled=%v complete=%v err=%v", handled, complete, err)
		}
		pending := fixture.loadBatchV0(t)
		if pending.Status != orquestaautoprogramming.AutoprogrammingBatchStatusPromotionPendingV0 ||
			pending.PromotionReceipt.ReceiptRef != "" {
			t.Fatalf("batch post-gate inesperado: %+v", pending)
		}
		fixture.assertEffectsV0(t, fixture.baseCommitCount+2, 1, 1, 0)
		if status := fixture.gitV0(t, fixture.repo, "status", "--porcelain", "--untracked-files=all"); !strings.Contains(status, "post-gate.txt") {
			t.Fatalf("cambio post-gate no quedo sin commit: %q", status)
		}

		handled, complete, _, err = fixture.stack.ReconcileClosedAutoprogrammingBatchRunV0(fixture.ctx, fixture.runs[1])
		if err != nil || !handled || complete {
			t.Fatalf("replay claim post-gate: handled=%v complete=%v err=%v", handled, complete, err)
		}
		blocked := fixture.loadBatchV0(t)
		if blocked.Status != orquestaautoprogramming.AutoprogrammingBatchStatusBlockedV0 ||
			blocked.BlockRef != "batch-block-ref-orphan-promotion-claim" {
			t.Fatalf("claim post-gate no bloqueado: %+v", blocked)
		}
		fixture.assertEffectsV0(t, fixture.baseCommitCount+2, 1, 1, 0)
	})
}

type serverAutoprogrammingBatchE2EV0 struct {
	ctx                 context.Context
	repo                string
	stateRoot           string
	gateCountPath       string
	integrationReceipts string
	gateReceipts        string
	baseCommit          string
	baseCommitCount     int
	batchRef            string
	runs                []orquestacoreworkflow.OrchestrationRunV0
	stack               orquestaappcodexstack.StackV0
	gateProbe           *serverAutoprogrammingBatchGateProbeV0
	finalizerProbe      *serverAutoprogrammingBatchFinalizerProbeV0
}

func newServerAutoprogrammingBatchE2EV0(t *testing.T, dirtyAfterGate bool) *serverAutoprogrammingBatchE2EV0 {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	stateDir := filepath.Join(root, "state")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatalf("go no encontrado: %v", err)
	}
	gateCountPath := filepath.Join(root, "gate-count")
	countingGo := filepath.Join(root, "counting-go")
	countingScript := "#!/bin/sh\nprintf x >> " + serverAutoprogrammingBatchShellQuoteV0(gateCountPath) + "\nexec " + serverAutoprogrammingBatchShellQuoteV0(realGo) + " \"$@\"\n"
	if err := os.WriteFile(countingGo, []byte(countingScript), 0o700); err != nil {
		t.Fatalf("write counting go: %v", err)
	}
	configPath := filepath.Join(repo, serverProjectConfigFileNameV0)
	config := `{
		"schema_version":"orquesta_config.v0",
		"required_test_runner":{"enabled":true,"go_command":` + strconv.Quote(countingGo) + `,"output_dir":` + strconv.Quote(filepath.Join(root, "gate-output")) + `},
		"autoprogramming":{"promotion":{"enabled":true,"archive_dir":` + strconv.Quote(filepath.Join(root, "archive")) + `,"commit_message":"test: integrate batch member"}}
	}`
	files := map[string]string{
		serverProjectConfigFileNameV0: config,
		"go.mod":                      "module example.com/orquesta-batch-e2e\n\ngo 1.22\n",
		"base.go":                     "package batch\n\nfunc Base() string { return \"base\" }\n",
		"batch_test.go":               "package batch\n\nimport \"testing\"\n\nfunc TestIntegratedBatch(t *testing.T) { if Alpha()+Beta() != 3 { t.Fatal(\"batch not integrated\") } }\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	serverAutoprogrammingBatchRunGitV0(t, repo, "init", "-q")
	serverAutoprogrammingBatchRunGitV0(t, repo, "config", "user.name", "Orquesta Batch E2E")
	serverAutoprogrammingBatchRunGitV0(t, repo, "config", "user.email", "batch-e2e@example.invalid")
	serverAutoprogrammingBatchRunGitV0(t, repo, "add", ".")
	serverAutoprogrammingBatchRunGitV0(t, repo, "commit", "-q", "-m", "test: batch base")

	stack, err := buildStackFromProjectConfigV0(orquestaserver.ConfigV0{
		ProjectWorkDir:                    repo,
		IdleSelfImprovementProjectWorkDir: repo,
		StateDir:                          stateDir,
		ProjectConfigFilePath:             configPath,
		RuntimeWorkDir:                    filepath.Join(root, "runtime"),
	}, serverCodexGoalBackendV0{}, serverProjectConfigFileV0{})
	if err != nil {
		t.Fatalf("buildStackFromProjectConfigV0: %v", err)
	}
	store, ok := stack.Stores.AutoprogrammingBatchStore.(*orquestastatefile.StoreV0)
	if !ok {
		t.Fatalf("batch store no es StoreV0 real: %T", stack.Stores.AutoprogrammingBatchStore)
	}
	fixture := &serverAutoprogrammingBatchE2EV0{
		ctx: ctx, repo: repo, stateRoot: store.RootDirV0(), gateCountPath: gateCountPath,
		integrationReceipts: stack.AutoprogrammingPromotion.BatchIntegrationReceiptDir,
		baseCommit:          serverAutoprogrammingBatchGitV0(t, repo, "rev-parse", "HEAD"),
		baseCommitCount:     serverAutoprogrammingBatchGitCountV0(t, repo), stack: stack,
	}

	requestRef := "request-ref-autoprogramming-batch-e2e"
	projectRef := "project-ref-autoprogramming-batch-e2e"
	worktreeRef := "worktree-ref-autoprogramming-batch-e2e"
	workspaces := make([]orquestaruntimeworktree.GoalWorkspaceV0, 2)
	for index := range workspaces {
		goalRef := fmt.Sprintf("goal-ref-autoprogramming-batch-e2e-%d", index+1)
		workspace, issues := fixture.stack.AutoprogrammingPromotion.GoalWorkspaceProvisioner.PrepareGoalWorkspaceV0(ctx, orquestaruntimeworktree.GoalWorkspaceRequestV0{
			RunRef: requestRef, GoalRef: goalRef, ProjectRef: projectRef, WorktreeRef: worktreeRef,
			SourceWorkDir: repo, WorkspaceRoot: fixture.stack.AutoprogrammingPromotion.GoalWorkspaceRoot,
			BaseRevision: fixture.baseCommit,
		})
		if len(issues) > 0 {
			t.Fatalf("PrepareGoalWorkspaceV0 %d: %+v", index, issues)
		}
		workspaces[index] = workspace
	}
	if workspaces[0].ProjectWorkDir == workspaces[1].ProjectWorkDir ||
		!serverAutoprogrammingBatchPhysicalWorktreeV0(t, repo, workspaces[0].ProjectWorkDir) ||
		!serverAutoprogrammingBatchPhysicalWorktreeV0(t, repo, workspaces[1].ProjectWorkDir) {
		t.Fatalf("worktrees no fisicos/disjuntos: %+v", workspaces)
	}
	if err := os.WriteFile(filepath.Join(workspaces[0].ProjectWorkDir, "alpha.go"), []byte("package batch\n\nfunc Alpha() int { return 1 }\n"), 0o600); err != nil {
		t.Fatalf("write alpha: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workspaces[1].ProjectWorkDir, "beta.go"), []byte("package batch\n\nfunc Beta() int { return 2 }\n"), 0o600); err != nil {
		t.Fatalf("write beta: %v", err)
	}

	command := "go test -count=1 ./..."
	commandSHA := sha256.Sum256([]byte(command))
	members := make([]orquestaautoprogramming.AutoprogrammingBatchMemberV0, 2)
	fixture.runs = make([]orquestacoreworkflow.OrchestrationRunV0, 2)
	for index, workspace := range workspaces {
		taskRef := fmt.Sprintf("task-ref-autoprogramming-batch-e2e-%d", index+1)
		goalRef := fmt.Sprintf("goal-ref-autoprogramming-batch-e2e-%d", index+1)
		runRef := fmt.Sprintf("run-ref-autoprogramming-batch-e2e-%d", index+1)
		path := []string{"alpha.go", "beta.go"}[index]
		members[index] = orquestaautoprogramming.AutoprogrammingBatchMemberV0{
			TaskRef: taskRef, GoalRef: goalRef, RunRef: runRef, WorkspaceRef: workspace.WorkspaceID, WriteSet: []string{path},
		}
		fixture.runs[index] = orquestacoreworkflow.OrchestrationRunV0{
			SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0, RunID: runRef,
			ProjectRef: projectRef, AppSpecRef: "app-spec-ref-autoprogramming-batch-e2e",
			Status: orquestacoreworkflow.OrchestrationRunStatusClosedV0, CurrentPhase: orquestacoreworkflow.OrchestrationPhaseCierreV0,
		}
	}
	created := orquestaautoprogramming.NewAutoprogrammingBatchV0(orquestaautoprogramming.AutoprogrammingBatchPlanV0{
		BatchRef: "batch-ref-autoprogramming-e2e", RequestRef: requestRef, ProjectRef: projectRef,
		BaseRevision: fixture.baseCommit, Members: members,
		FrozenTests: []orquestaautoprogramming.AutoprogrammingBatchTestV0{{Command: command, SHA256: hex.EncodeToString(commandSHA[:])}},
	})
	if !created.Accepted {
		t.Fatalf("NewAutoprogrammingBatchV0: %+v", created.Issues)
	}
	fixture.batchRef = created.Batch.BatchRef
	batch, err := store.CompareAndSwapAutoprogrammingBatchV0(ctx, 0, created.Batch)
	if err != nil {
		t.Fatalf("persist batch: %v", err)
	}
	for _, member := range members {
		batch = serverAutoprogrammingBatchTransitionV0(t, ctx, store, batch,
			orquestaautoprogramming.RegisterAutoprogrammingBatchLaunchV0(batch, batch.StoreVersion, "launch-"+member.TaskRef, member.TaskRef))
	}
	for index, member := range members {
		state := orquestagoal.GoalWorkStateV0{
			SchemaVersion: orquestagoal.GoalWorkStateSchemaV0, RunRef: member.RunRef, GoalRef: member.GoalRef,
			ExternalGoalRef: "external-" + member.GoalRef, Status: orquestagoal.GoalStatusCompleteV0,
			Spec: orquestagoal.GoalWorkSpecV0{
				SchemaVersion: orquestagoal.GoalWorkSpecSchemaV0, GoalRef: member.GoalRef, RequestRef: requestRef,
				RunRef: member.RunRef, ProjectRef: projectRef, WorkKind: orquestaautoprogramming.AutoprogrammingGoalWorkKindV0,
				Objective: "Materialize one disjoint batch member.", DirectorKind: orquestagoal.GoalDirectorKindCodexGoalV0,
				ContextRefs: []orquestagoal.GoalContextRefV0{
					{Kind: "autoprogramming_batch", Ref: fixture.batchRef, Required: true},
					{Kind: "autoprogramming_batch_task", Ref: member.TaskRef, Required: true},
					{Kind: "goal_workspace", Ref: workspaces[index].WorkspaceID, Required: true},
					{Kind: "worktree", Ref: worktreeRef, Required: true},
				},
				WriteSet: []orquestagoal.GoalWriteScopeV0{{Path: member.WriteSet[0]}},
			},
			LaunchReceipt: orquestagoal.GoalLaunchReceiptV0{
				SchemaVersion: orquestagoal.GoalWorkLaunchReceiptSchemaV0, Status: orquestagoal.GoalStatusRunningV0,
				GoalRef: member.GoalRef, ExternalGoalRef: "external-" + member.GoalRef,
			},
			LastClosure:  &orquestagoal.GoalClosureValidationV0{Status: orquestagoal.GoalStatusAcceptedV0, Accepted: true},
			EvidenceRefs: []string{"evidence-ref-autoprogramming-batch-e2e-goal-closed"},
		}
		if err := store.SaveGoalWorkStateV0(ctx, state); err != nil {
			t.Fatalf("SaveGoalWorkStateV0 %s: %v", member.RunRef, err)
		}
	}

	realRunner := fixture.stack.AutoprogrammingPromotion.BatchTestRunner
	fixture.gateProbe = &serverAutoprogrammingBatchGateProbeV0{delegate: realRunner, repo: repo, dirtyAfterRun: dirtyAfterGate}
	fixture.stack.AutoprogrammingPromotion.BatchTestRunner = fixture.gateProbe
	fixture.gateReceipts = realRunner.(*serverAutoprogrammingBatchTestRunnerV0).ReceiptDir
	realFinalizer := fixture.stack.AutoprogrammingPromotion.BatchPromotionFinalizer
	fixture.finalizerProbe = &serverAutoprogrammingBatchFinalizerProbeV0{delegate: realFinalizer, reconciler: fixture.stack.AutoprogrammingPromotion.BatchPromotionReconciler}
	fixture.stack.AutoprogrammingPromotion.BatchPromotionFinalizer = fixture.finalizerProbe
	fixture.stack.AutoprogrammingPromotion.BatchPromotionReconciler = fixture.finalizerProbe
	return fixture
}

func (fixture *serverAutoprogrammingBatchE2EV0) restartStoreV0(t *testing.T) {
	t.Helper()
	reopened, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{RootDir: fixture.stateRoot})
	if err != nil {
		t.Fatalf("reopen StoreV0: %v", err)
	}
	fixture.stack.Stores.AutoprogrammingBatchStore = reopened
	fixture.stack.Stores.AppGoalStateStore = reopened
	fixture.stack.Ports.GoalStateStore = reopened
	if _, err := reopened.LoadAutoprogrammingBatchV0(fixture.ctx, fixture.batchRef); err != nil {
		t.Fatalf("batch no sobrevivio restart: %v", err)
	}
	for _, run := range fixture.runs {
		if _, err := reopened.LoadGoalWorkStateV0(fixture.ctx, run.RunID); err != nil {
			t.Fatalf("goal %s no sobrevivio restart: %v", run.RunID, err)
		}
	}
}

func (fixture *serverAutoprogrammingBatchE2EV0) loadBatchV0(t *testing.T) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	batch, err := fixture.stack.Stores.AutoprogrammingBatchStore.LoadAutoprogrammingBatchV0(fixture.ctx, fixture.batchRef)
	if err != nil {
		t.Fatalf("LoadAutoprogrammingBatchV0: %v", err)
	}
	return batch
}

func (fixture *serverAutoprogrammingBatchE2EV0) assertIntegratedChainV0(t *testing.T, batch orquestaautoprogramming.AutoprogrammingBatchV0) {
	t.Helper()
	if len(batch.Members) != 2 || batch.Members[0].ParentRevision != fixture.baseCommit ||
		batch.Members[1].ParentRevision != batch.Members[0].IntegrationRevision ||
		batch.IntegratedRevision != batch.Members[1].IntegrationRevision {
		t.Fatalf("cadena de integracion invalida: %+v", batch.Members)
	}
	for index, member := range batch.Members {
		parent := fixture.gitV0(t, fixture.repo, "rev-parse", member.IntegrationRevision+"^")
		if parent != member.ParentRevision || member.IntegrationOrder != uint64(index+1) || member.IntegrationReceiptRef == "" {
			t.Fatalf("miembro integrado %d invalido: parent=%s member=%+v", index, parent, member)
		}
	}
}

func (fixture *serverAutoprogrammingBatchE2EV0) assertEffectsV0(t *testing.T, commits, gateRuns, finalizations, promotionReceipts int) {
	t.Helper()
	if got := serverAutoprogrammingBatchGitCountV0(t, fixture.repo); got != commits {
		t.Fatalf("commits=%d want=%d", got, commits)
	}
	gateCount, err := os.ReadFile(fixture.gateCountPath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read gate count: %v", err)
	}
	if len(gateCount) != gateRuns || fixture.gateProbe.runs != gateRuns {
		t.Fatalf("gate executions file=%d port=%d want=%d", len(gateCount), fixture.gateProbe.runs, gateRuns)
	}
	if fixture.finalizerProbe.finalizations != finalizations {
		t.Fatalf("finalizations=%d want=%d", fixture.finalizerProbe.finalizations, finalizations)
	}
	if got := serverAutoprogrammingBatchReceiptSchemaCountV0(t, fixture.gateReceipts, serverAutoprogrammingBatchTestReceiptSchemaV0); got != gateRuns {
		t.Fatalf("gate receipts=%d want=%d", got, gateRuns)
	}
	if got := serverAutoprogrammingBatchReceiptSchemaCountV0(t, fixture.integrationReceipts, orquestaruntimeworktree.GoalWorkspaceIntegrationSchemaVersionV0); got != commits-fixture.baseCommitCount {
		t.Fatalf("integration receipts=%d want=%d", got, commits-fixture.baseCommitCount)
	}
	if got := serverAutoprogrammingBatchReceiptSchemaCountV0(t, fixture.integrationReceipts, serverAutoprogrammingBatchPromotionReceiptSchemaV0); got != promotionReceipts {
		t.Fatalf("promotion receipts=%d want=%d", got, promotionReceipts)
	}
}

func (fixture *serverAutoprogrammingBatchE2EV0) gitV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	return serverAutoprogrammingBatchGitV0(t, repo, args...)
}

type serverAutoprogrammingBatchGateProbeV0 struct {
	delegate      orquestaappcodexstack.AutoprogrammingBatchTestRunnerPortV0
	repo          string
	dirtyAfterRun bool
	runs          int
	reconciles    int
}

func (probe *serverAutoprogrammingBatchGateProbeV0) RunAutoprogrammingBatchTestV0(ctx context.Context, request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0) (orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0, error) {
	probe.runs++
	result, err := probe.delegate.RunAutoprogrammingBatchTestV0(ctx, request)
	if err == nil && probe.dirtyAfterRun {
		err = os.WriteFile(filepath.Join(probe.repo, "post-gate.txt"), []byte("must remain uncommitted\n"), 0o600)
	}
	return result, err
}

func (probe *serverAutoprogrammingBatchGateProbeV0) ReconcileAutoprogrammingBatchTestClaimV0(ctx context.Context, request orquestaappcodexstack.AutoprogrammingBatchTestRunRequestV0) (orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0, bool, error) {
	probe.reconciles++
	reconciler, ok := probe.delegate.(orquestaappcodexstack.AutoprogrammingBatchTestClaimReconcilerPortV0)
	if !ok {
		return orquestaappcodexstack.AutoprogrammingBatchTestRunResultV0{}, false, fmt.Errorf("gate reconciler unavailable")
	}
	return reconciler.ReconcileAutoprogrammingBatchTestClaimV0(ctx, request)
}

type serverAutoprogrammingBatchFinalizerProbeV0 struct {
	delegate      orquestaappcodexstack.AutoprogrammingBatchPromotionFinalizerPortV0
	reconciler    orquestaappcodexstack.AutoprogrammingBatchPromotionClaimReconcilerPortV0
	finalizations int
	reconciles    int
}

func (probe *serverAutoprogrammingBatchFinalizerProbeV0) FinalizeAutoprogrammingBatchPromotionV0(ctx context.Context, request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0) (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, error) {
	probe.finalizations++
	return probe.delegate.FinalizeAutoprogrammingBatchPromotionV0(ctx, request)
}

func (probe *serverAutoprogrammingBatchFinalizerProbeV0) ReconcileAutoprogrammingBatchPromotionV0(ctx context.Context, request orquestaappcodexstack.AutoprogrammingBatchPromotionRequestV0) (orquestaappcodexstack.AutoprogrammingBatchPromotionResultV0, bool, error) {
	probe.reconciles++
	return probe.reconciler.ReconcileAutoprogrammingBatchPromotionV0(ctx, request)
}

func serverAutoprogrammingBatchTransitionV0(t *testing.T, ctx context.Context, store orquestaautoprogramming.AutoprogrammingBatchStorePortV0, current orquestaautoprogramming.AutoprogrammingBatchV0, transition orquestaautoprogramming.AutoprogrammingBatchTransitionResultV0) orquestaautoprogramming.AutoprogrammingBatchV0 {
	t.Helper()
	if !transition.Accepted || transition.Replay {
		t.Fatalf("batch setup transition rejected/replayed: %+v", transition)
	}
	saved, err := store.CompareAndSwapAutoprogrammingBatchV0(ctx, current.StoreVersion, transition.Batch)
	if err != nil {
		t.Fatalf("batch setup CAS: %v", err)
	}
	return saved
}

func serverAutoprogrammingBatchRunGitV0(t *testing.T, repo string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
}

func serverAutoprogrammingBatchGitV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repo}, args...)...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output))
}

func serverAutoprogrammingBatchGitCountV0(t *testing.T, repo string) int {
	t.Helper()
	value := serverAutoprogrammingBatchGitV0(t, repo, "rev-list", "--count", "HEAD")
	count, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("commit count %q: %v", value, err)
	}
	return count
}

func serverAutoprogrammingBatchPhysicalWorktreeV0(t *testing.T, repo, want string) bool {
	t.Helper()
	output := serverAutoprogrammingBatchGitV0(t, repo, "worktree", "list", "--porcelain")
	for _, line := range strings.Split(output, "\n") {
		if strings.TrimSpace(strings.TrimPrefix(line, "worktree ")) == want && strings.HasPrefix(line, "worktree ") {
			return true
		}
	}
	return false
}

func serverAutoprogrammingBatchReceiptSchemaCountV0(t *testing.T, dir, schema string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("glob receipts: %v", err)
	}
	count := 0
	for _, path := range matches {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read receipt %s: %v", filepath.Base(path), err)
		}
		var document struct {
			SchemaVersion string `json:"schema_version"`
		}
		if err := json.Unmarshal(raw, &document); err != nil {
			t.Fatalf("decode receipt %s: %v", filepath.Base(path), err)
		}
		if document.SchemaVersion == schema {
			count++
		}
	}
	return count
}

func serverAutoprogrammingBatchShellQuoteV0(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
