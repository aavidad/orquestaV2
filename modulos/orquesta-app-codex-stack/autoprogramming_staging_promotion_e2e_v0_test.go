package orquestaappcodexstack

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
	orquestaruntimeworktree "orquesta/modulos/orquesta-runtime-worktree"
)

func TestCodexStackAutoprogrammingPromotionV0E2ERepoTemporalReplayV0(t *testing.T) {
	ctx := context.Background()
	repo := initAutoprogrammingPromotionE2ERepoV0(t)
	baseCommits := gitCommitCountForPromotionE2EV0(t, repo)
	if err := os.WriteFile(filepath.Join(repo, "feature.md"), []byte("v2\n"), 0o600); err != nil {
		t.Fatalf("write feature: %v", err)
	}

	stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
	stack = withAutoprogrammingPromotionStoresForTestV0(stack)
	port := &gitBackedAutoprogrammingPromotionPortForTestV0{
		projectWorkDir: repo,
		archiveDir:     filepath.Join(t.TempDir(), "archive"),
	}
	stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{Enabled: true, Port: port}
	seedAutoprogrammingPromotionRunV0(t, ctx, stack, "run-autoprogramming-promotion-e2e-001", []string{"feature.md"})

	run := mustLoadCodexStackRunForTestV0(t, stack, "run-autoprogramming-promotion-e2e-001")
	status, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, promotionE2ELoopResultV0(run))
	if err != nil {
		t.Fatalf("queue status: %v", err)
	}
	if status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue_status=%q", status)
	}
	if port.promotions != 1 || port.archives != 1 {
		t.Fatalf("port=%+v", port)
	}
	if got := gitCommitCountForPromotionE2EV0(t, repo); got != baseCommits+1 {
		t.Fatalf("commits=%d want %d", got, baseCommits+1)
	}
	if paths := gitLastCommitPathsForPromotionE2EV0(t, repo); strings.Join(paths, ",") != "feature.md" {
		t.Fatalf("commit paths=%v", paths)
	}
	if port.lastPromotion.WorktreeRef != "worktree-ref-autoprogramming-promotion-001" ||
		port.lastPromotion.BranchRef != "branch-ref-autoprogramming-promotion-001" ||
		strings.Join(port.lastPromotionResult.ChangedPaths, ",") != "feature.md" {
		t.Fatalf("promotion refs/result=%+v %+v", port.lastPromotion, port.lastPromotionResult)
	}

	status, err = stack.stackDrainQueueStatusForCoordinatorV0(ctx, promotionE2ELoopResultV0(run))
	if err != nil {
		t.Fatalf("queue status replay: %v", err)
	}
	if status != orquestarunqueue.RunStatusClosedV0 {
		t.Fatalf("queue_status replay=%q", status)
	}
	if got := gitCommitCountForPromotionE2EV0(t, repo); got != baseCommits+1 {
		t.Fatalf("commits replay=%d want %d", got, baseCommits+1)
	}
	if files := archiveManifestCountForPromotionE2EV0(t, port.archiveDir); files != 1 {
		t.Fatalf("archive files=%d", files)
	}
}

func TestCodexStackAutoprogrammingPromotionV0NoCierraColaConEfectoIncompletoV0(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name          string
		promoteStatus string
		archiveStatus string
	}{
		{name: "promotion_blocked", promoteStatus: orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0},
		{name: "pending_push", promoteStatus: orquestaautoprogramming.AutoprogrammingStagingEffectPendingPushV0},
		{name: "archive_blocked", promoteStatus: orquestaautoprogramming.AutoprogrammingStagingEffectPromotedV0, archiveStatus: orquestaautoprogramming.AutoprogrammingStagingEffectBlockedV0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stack := mustBuildCodexStackForTestV0(t, newFakeCodexStackRuntimeV0())
			stack = withAutoprogrammingPromotionStoresForTestV0(stack)
			stack.AutoprogrammingPromotion = AutoprogrammingPromotionConfigV0{
				Enabled: true,
				Port: &statusAutoprogrammingPromotionPortForTestV0{
					promoteStatus: tc.promoteStatus,
					archiveStatus: tc.archiveStatus,
				},
			}
			runRef := "run-autoprogramming-promotion-incomplete-" + tc.name
			seedAutoprogrammingPromotionRunV0(t, ctx, stack, runRef, []string{"feature.md"})

			run := mustLoadCodexStackRunForTestV0(t, stack, runRef)
			status, err := stack.stackDrainQueueStatusForCoordinatorV0(ctx, promotionE2ELoopResultV0(run))
			if err != nil {
				t.Fatalf("queue status: %v", err)
			}
			if status != "" {
				t.Fatalf("queue_status=%q want pending retry", status)
			}
		})
	}
}

func promotionE2ELoopResultV0(run orquestacoreworkflow.OrchestrationRunV0) orquestacionnucleoapp.ManagedProgressiveLoopResultV0 {
	return orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{Run: run},
	}
}

type gitBackedAutoprogrammingPromotionPortForTestV0 struct {
	projectWorkDir      string
	archiveDir          string
	connector           orquestaruntimeworktree.GitStagingPromotionConnectorV0
	promotions          int
	archives            int
	lastPromotion       orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0
	lastPromotionResult orquestaruntimeworktree.StagingPromotionResultV0
}

func (port *gitBackedAutoprogrammingPromotionPortForTestV0) PromoteAutoprogrammingStagingV0(
	ctx context.Context,
	command orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	port.promotions++
	port.lastPromotion = command
	result, issues := port.connector.PromoteStagingWorktreeV0(ctx, orquestaruntimeworktree.StagingPromotionRequestV0{
		PromotionRef:   command.PromotionRef,
		RunRef:         command.RunRef,
		ProjectRef:     command.ProjectRef,
		RepoRef:        "repo-ref-autoprogramming-promotion-e2e",
		AppRef:         "app-ref-autoprogramming-promotion-e2e",
		WorktreeRef:    command.WorktreeRef,
		BranchRef:      command.BranchRef,
		ProjectWorkDir: port.projectWorkDir,
		CommitMessage:  "test: promote autoprogramming staging",
		WriteSet:       command.WriteSet,
		EvidenceRefs:   command.EvidenceRefs,
	})
	port.lastPromotionResult = result
	return autoprogrammingPromotionEffectForTestV0(result, issues), nil
}

func (port *gitBackedAutoprogrammingPromotionPortForTestV0) ArchiveAutoprogrammingStagingV0(
	ctx context.Context,
	command orquestaautoprogramming.AutoprogrammingStagingCleanupCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	port.archives++
	result, issues := port.connector.ArchiveStagingWorktreeV0(ctx, orquestaruntimeworktree.StagingPromotionRequestV0{
		ArchiveRef:   command.ArchiveRef,
		PromotionRef: command.PromotionRef,
		RunRef:       command.RunRef,
		ProjectRef:   command.ProjectRef,
		RepoRef:      "repo-ref-autoprogramming-promotion-e2e",
		AppRef:       "app-ref-autoprogramming-promotion-e2e",
		WorktreeRef:  command.WorktreeRef,
		BranchRef:    command.BranchRef,
		ArchiveDir:   port.archiveDir,
		WriteSet:     command.WriteSet,
		EvidenceRefs: command.EvidenceRefs,
	})
	return autoprogrammingPromotionEffectForTestV0(result, issues), nil
}

func autoprogrammingPromotionEffectForTestV0(
	result orquestaruntimeworktree.StagingPromotionResultV0,
	issues []orquestaruntimeworktree.WorktreeIssueV0,
) orquestaautoprogramming.AutoprogrammingStagingEffectResultV0 {
	out := orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{
		SchemaVersion:  orquestaautoprogramming.AutoprogrammingStagingPromotionSchemaVersionV0,
		Status:         result.Status,
		PromotionRef:   result.PromotionRef,
		ArchiveRef:     result.ArchiveRef,
		RunRef:         result.RunRef,
		ProjectRef:     result.ProjectRef,
		WorktreeRef:    result.WorktreeRef,
		BranchRef:      result.BranchRef,
		ChangedPaths:   result.ChangedPaths,
		CommitRef:      result.CommitRef,
		CommitShortRef: result.CommitShortRef,
		Retryable:      result.Retryable,
		EvidenceRefs:   result.EvidenceRefs,
	}
	for _, issue := range issues {
		out.Issues = append(out.Issues, orquestaautoprogramming.AutoprogrammingRequestIssueV0{Code: string(issue.Code), Field: issue.Field})
	}
	return out
}

type statusAutoprogrammingPromotionPortForTestV0 struct {
	promoteStatus string
	archiveStatus string
}

func (port *statusAutoprogrammingPromotionPortForTestV0) PromoteAutoprogrammingStagingV0(
	context.Context,
	orquestaautoprogramming.AutoprogrammingStagingPromotionCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{Status: port.promoteStatus, EvidenceRefs: []string{"evidence-ref-promotion-status"}}, nil
}

func (port *statusAutoprogrammingPromotionPortForTestV0) ArchiveAutoprogrammingStagingV0(
	context.Context,
	orquestaautoprogramming.AutoprogrammingStagingCleanupCommandV0,
) (orquestaautoprogramming.AutoprogrammingStagingEffectResultV0, error) {
	return orquestaautoprogramming.AutoprogrammingStagingEffectResultV0{Status: port.archiveStatus, EvidenceRefs: []string{"evidence-ref-archive-status"}}, nil
}

func initAutoprogrammingPromotionE2ERepoV0(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGitForPromotionE2EV0(t, repo, "init")
	runGitForPromotionE2EV0(t, repo, "config", "user.email", "orquesta@example.invalid")
	runGitForPromotionE2EV0(t, repo, "config", "user.name", "Orquesta Test")
	if err := os.WriteFile(filepath.Join(repo, "feature.md"), []byte("v1\n"), 0o600); err != nil {
		t.Fatalf("write base: %v", err)
	}
	runGitForPromotionE2EV0(t, repo, "add", "feature.md")
	runGitForPromotionE2EV0(t, repo, "commit", "-m", "test: base")
	return repo
}

func runGitForPromotionE2EV0(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=Never")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func gitCommitCountForPromotionE2EV0(t *testing.T, repo string) int {
	t.Helper()
	out := runGitForPromotionE2EV0(t, repo, "rev-list", "--count", "HEAD")
	if out != "1" && out != "2" {
		t.Fatalf("unexpected commit count %q", out)
	}
	if out == "2" {
		return 2
	}
	return 1
}

func gitLastCommitPathsForPromotionE2EV0(t *testing.T, repo string) []string {
	t.Helper()
	out := runGitForPromotionE2EV0(t, repo, "show", "--pretty=", "--name-only", "HEAD")
	return compactStringsV0(strings.Split(out, "\n"))
}

func archiveManifestCountForPromotionE2EV0(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read archive dir: %v", err)
	}
	return len(entries)
}
