package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestWorkspaceEffectsReplayEveryCrashFrontierExactlyOnce(t *testing.T) {
	fixture := v16LoadFixture(t)
	if len(fixture.CrashFrontiers) != 7 {
		t.Fatalf("fixture crash frontiers=%v", fixture.CrashFrontiers)
	}
	for _, frontier := range fixture.CrashFrontiers {
		frontier := frontier
		t.Run(frontier, func(t *testing.T) {
			v16RunCrashFrontier(t, fixture, frontier)
		})
	}
}

func v16RunCrashFrontier(t *testing.T, fixture v16E2EFixture, frontier string) {
	harness := newV16Harness(t, fixture, map[string]v16Write{
		"crash-change": {"src/crash.txt": "survives " + frontier + "\n"},
	})
	goalRef := harness.submit(t, harness.access, "request:v16-crash:"+frontier,
		"crash-change", []string{"src/crash.txt"})
	integrationRequestRef := v16InjectCrashFrontier(t, harness, fixture, goalRef, frontier)
	v16FinishCrashReplay(t, harness, fixture, goalRef, frontier, integrationRequestRef)
}

func v16InjectCrashFrontier(
	t *testing.T,
	harness *v16Harness,
	fixture v16E2EFixture,
	goalRef goal.GoalRef,
	frontier string,
) string {
	integrationRequestRef := "request:v16-crash-final:" + frontier
	switch frontier {
	case "before_adapter_call":
		harness.restart(t)
	case "after_branch_marker_before_worktree":
		v16InjectBranchMarkerCrash(t, harness, fixture, goalRef)
	case "after_worktree_before_state_receipt":
		v16InjectWorktreeCrash(t, harness, fixture, goalRef)
	case "after_commit_object_before_ref", "after_isolated_ref_before_state_receipt":
		v16InjectCommitCrash(t, harness, goalRef, frontier)
	case "after_target_and_marker_before_state_receipt":
		integrationRequestRef = "request:v16-crash-integrate:" + frontier
		v16InjectIntegrationCrash(t, harness, goalRef, integrationRequestRef)
	case "during_release":
		v16InjectReleaseCrash(t, harness, goalRef)
	}
	return integrationRequestRef
}

func v16InjectBranchMarkerCrash(
	t *testing.T,
	harness *v16Harness,
	fixture v16E2EFixture,
	goalRef goal.GoalRef,
) {
	record := harness.get(t, harness.access, goalRef)
	workspace := record.Executions[0].ExecutionWorkspaceRef
	base := v16Git(t, harness.git, harness.seed, "rev-parse", fixture.GitFixture.TargetRef)
	v16Git(t, harness.git, harness.seed, "update-ref", v16WorkspaceBaseRef(workspace), base)
	v16Git(t, harness.git, harness.seed, "update-ref", v16WorkspaceBranchRef(workspace), base)
	harness.restart(t)
}

func v16InjectWorktreeCrash(
	t *testing.T,
	harness *v16Harness,
	fixture v16E2EFixture,
	goalRef goal.GoalRef,
) {
	record := harness.get(t, harness.access, goalRef)
	workspace := record.Executions[0].ExecutionWorkspaceRef
	base := v16Git(t, harness.git, harness.seed, "rev-parse", fixture.GitFixture.TargetRef)
	v16Git(t, harness.git, harness.seed, "update-ref", v16WorkspaceBaseRef(workspace), base)
	v16Git(t, harness.git, harness.seed, "worktree", "add", "--lock", "-b",
		v16WorkspaceBranch(workspace), harness.workspacePath(workspace), base)
	v16HardenInjectedWorktree(t, harness, workspace)
	harness.restart(t)
}

func v16HardenInjectedWorktree(t *testing.T, harness *v16Harness, workspace ports.ExecutionWorkspaceRef) {
	path := harness.workspacePath(workspace)
	gitDir := v16Git(t, harness.git, path, "rev-parse", "--path-format=absolute", "--git-dir")
	for target, mode := range map[string]os.FileMode{
		filepath.Join(path, ".git"): 0o600,
		filepath.Dir(gitDir):        0o700,
		gitDir:                      0o700,
	} {
		if err := os.Chmod(target, mode); err != nil {
			t.Fatal(err)
		}
	}
}

func v16InjectCommitCrash(t *testing.T, harness *v16Harness, goalRef goal.GoalRef, frontier string) {
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent, application.ActionObserveAgent)
	claim := harness.claim(t, application.ActionCommitChange)
	record := harness.get(t, harness.access, goalRef)
	change := v16CreateCommitObject(t, harness, record, claim)
	if frontier == "after_isolated_ref_before_state_receipt" {
		v16Git(t, harness.git, harness.seed, "update-ref",
			v16WorkspaceBranchRef(record.WorkspaceBindings[0].Ref), change, record.WorkspaceBindings[0].BaseOID)
	}
	harness.restartAfterLease(t)
}

func v16InjectIntegrationCrash(
	t *testing.T,
	harness *v16Harness,
	goalRef goal.GoalRef,
	integrationRequestRef string,
) {
	harness.process(t, application.ActionPrepareWorkspace, application.ActionLaunchAgent,
		application.ActionObserveAgent, application.ActionCommitChange, application.ActionAttestTest)
	record := harness.get(t, harness.access, goalRef)
	before := record.WorkspaceBindings[0].BaseOID
	if _, err := harness.runtime.Orchestrator().IntegrateChange(context.Background(), harness.access,
		application.IntegrateChangeRequest{
			RequestRef: integrationRequestRef, GoalRef: goalRef,
			ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: before,
		}); err != nil {
		t.Fatal(err)
	}
	claim := harness.claim(t, application.ActionIntegrateChange)
	v16ApplyIntegrationOutsideSQLite(t, harness, record, claim)
	harness.restartAfterLease(t)
}

func v16InjectReleaseCrash(t *testing.T, harness *v16Harness, goalRef goal.GoalRef) {
	harness.driveToIntegrated(t, goalRef, "request:v16-release-integrate")
	record := harness.get(t, harness.access, goalRef)
	workspace := record.WorkspaceBindings[0].Ref
	v16Git(t, harness.git, harness.seed, "worktree", "unlock", harness.workspacePath(workspace))
	harness.restart(t)
	v16ReleaseRecoveredWorkspace(t, harness, record)
}

func v16FinishCrashReplay(
	t *testing.T,
	harness *v16Harness,
	fixture v16E2EFixture,
	goalRef goal.GoalRef,
	frontier, integrationRequestRef string,
) {
	if frontier != "during_release" {
		record := harness.get(t, harness.access, goalRef)
		if len(record.ChangeSets) == 0 || record.Executions[0].State != application.ExecutionAwaitingIntegration {
			harness.driveToAttested(t, record)
			record = harness.get(t, harness.access, goalRef)
		}
		if len(record.IntegrationReceipts) == 0 {
			before := record.WorkspaceBindings[0].BaseOID
			if _, err := harness.runtime.Orchestrator().IntegrateChange(context.Background(), harness.access,
				application.IntegrateChangeRequest{
					RequestRef: integrationRequestRef, GoalRef: goalRef,
					ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: before,
				}); err != nil {
				t.Fatal(err)
			}
			harness.process(t, application.ActionIntegrateChange)
		}
	}
	// The replay is not closed until the newly persisted result itself survives
	// recovery validation. This catches a marker replay with a stale preview.
	harness.restart(t)
	v16AssertCrashClosed(t, harness, fixture, goalRef, frontier)
}

func v16AssertCrashClosed(
	t *testing.T,
	harness *v16Harness,
	fixture v16E2EFixture,
	goalRef goal.GoalRef,
	frontier string,
) {
	closed := harness.get(t, harness.access, goalRef)
	if closed.Goal.State() != goal.GoalStateSucceeded || len(closed.WorkspaceBindings) != 1 ||
		len(closed.ChangeSets) != 1 || len(closed.IntegrationReceipts) != 1 ||
		len(closed.MergeObservations) != 1 || closed.MergeObservations[0].Status != ports.MergeStatusClean ||
		closed.IntegrationReceipts[0].Status != ports.IntegrationStatusIntegrated ||
		len(closed.Artifacts) != 3 || len(closed.Attestations) != 2 ||
		len(closed.EffectReceipts) != 5 || harness.launches.Load() != 1 {
		t.Fatalf("frontier %s duplicated/lost effects: state=%s bindings=%d changes=%d artifacts=%d attestations=%d integrations=%d effect_receipts=%d launches=%d",
			frontier, closed.Goal.State(), len(closed.WorkspaceBindings), len(closed.ChangeSets),
			len(closed.Artifacts), len(closed.Attestations), len(closed.IntegrationReceipts),
			len(closed.EffectReceipts), harness.launches.Load())
	}
	log := v16Git(t, harness.git, harness.seed, "log", "--format=%s", fixture.GitFixture.TargetRef)
	if count := strings.Count(log, "orquesta integration"); count != 1 {
		t.Fatalf("frontier %s integration commits=%d", frontier, count)
	}
	v16AssertFactsPathFree(t, harness, closed)
}
