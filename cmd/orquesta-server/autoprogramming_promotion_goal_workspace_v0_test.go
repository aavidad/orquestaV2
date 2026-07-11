package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestServerAutoprogrammingPromotionPortV0IntegratesGoalWorkspaceAndReplaysV0(t *testing.T) {
	canonical := newCodexGoalWorkspaceAdapterGitRepoV0(t)
	workspaceRoot := codexGoalWorkspaceRootForSourceV0(canonical)
	t.Cleanup(func() { _ = os.RemoveAll(workspaceRoot) })
	provisioner := orquestaruntimeworktree.GitGoalWorkspaceProvisionerV0{}
	workspaceRequest := orquestaruntimeworktree.GoalWorkspaceRequestV0{
		RunRef: "request-ref-server-integration-001", GoalRef: "goal-ref-server-integration-001",
		ProjectRef: "project-ref-server-integration-001", WorktreeRef: "worktree-ref-server-integration-001",
		SourceWorkDir: canonical, WorkspaceRoot: workspaceRoot,
	}
	workspace, issues := provisioner.PrepareGoalWorkspaceV0(context.Background(), workspaceRequest)
	if len(issues) > 0 {
		t.Fatalf("prepare workspace: %+v", issues)
	}
	if err := os.WriteFile(filepath.Join(workspace.ProjectWorkDir, "integrated.txt"), []byte("integrated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	port := serverAutoprogrammingPromotionPortV0{
		ProjectWorkDir: canonical, CommitMessage: "test: integrate server goal",
		GoalWorkspaceRoot: workspaceRoot, IntegrationReceiptDir: filepath.Join(workspaceRoot, "integration-receipts"),
		WorkspaceProvisioner: provisioner, IntegrationConnector: orquestaruntimeworktree.GitGoalWorkspaceIntegrationConnectorV0{},
	}
	command := orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0{
		PromotionRef: "promotion-ref-server-integration-001", RequestRef: workspaceRequest.RunRef,
		RunRef: "run-ref-server-integration-001", GoalRef: workspaceRequest.GoalRef,
		ProjectRef: workspaceRequest.ProjectRef, WorktreeRef: workspaceRequest.WorktreeRef,
		BranchRef: "branch-ref-server-integration-001", WriteSet: []string{"integrated.txt"},
	}
	first, err := port.PromoteAutoprogrammingStagingV0(context.Background(), command)
	if err != nil || first.Status != orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0 ||
		first.IntegrationStatus != orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0 ||
		strings.TrimSpace(first.IntegrationReceiptRef) == "" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if body, err := os.ReadFile(filepath.Join(canonical, "integrated.txt")); err != nil || string(body) != "integrated\n" {
		t.Fatalf("canonical body=%q err=%v", body, err)
	}
	head := codexGoalWorkspaceAdapterGitOutputV0(t, canonical, "rev-parse", "HEAD")
	replayed, err := port.PromoteAutoprogrammingStagingV0(context.Background(), command)
	if err != nil || replayed.IntegrationStatus != orquestaautoprogramming.AutoprogrammingStagingIntegrationStatusIntegratedV0 {
		t.Fatalf("replayed=%+v err=%v", replayed, err)
	}
	if got := codexGoalWorkspaceAdapterGitOutputV0(t, canonical, "rev-parse", "HEAD"); got != head {
		t.Fatalf("replay duplicated commit: got=%s want=%s", got, head)
	}
}

func codexGoalWorkspaceAdapterGitOutputV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	command := append([]string{"-C", repo}, args...)
	out, err := exec.Command("git", command...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
