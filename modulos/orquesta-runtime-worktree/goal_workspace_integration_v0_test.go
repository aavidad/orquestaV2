package orquestaruntimeworktree

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitGoalWorkspaceIntegrationConnectorV0IntegratesDisjointWorkspacesV0(t *testing.T) {
	canonical, first, second, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, first, "first.txt", "first\n")
	writeAppVCSFileV0(t, second, "second.txt", "second\n")
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	firstResult, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), goalWorkspaceIntegrationRequestForTestV0("integration-ref-disjoint-001", first, canonical, base, []string{"first.txt"}, receiptDir))
	if len(issues) > 0 || firstResult.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", firstResult, issues)
	}
	secondResult, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), goalWorkspaceIntegrationRequestForTestV0("integration-ref-disjoint-002", second, canonical, base, []string{"second.txt"}, receiptDir))
	if len(issues) > 0 || secondResult.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("second=%+v issues=%+v", secondResult, issues)
	}
	for _, result := range []GoalWorkspaceIntegrationResultV0{firstResult, secondResult} {
		receipt, found, receiptIssues := loadGoalWorkspaceIntegrationReceiptV0(filepath.Join(receiptDir, result.IntegrationRef+".json"))
		if len(receiptIssues) > 0 || !found || receipt.SourceCommit != result.SourceCommit || receipt.IntegratedCommit != result.IntegratedCommit {
			t.Fatalf("receipt=%+v found=%t issues=%+v result=%+v", receipt, found, receiptIssues, result)
		}
	}
	for _, rel := range []string{"first.txt", "second.txt"} {
		if _, err := os.Stat(filepath.Join(canonical, rel)); err != nil {
			t.Fatalf("canonical missing %s: %v", rel, err)
		}
	}
	for _, workspace := range []string{first, second} {
		if _, err := os.Stat(workspace); err != nil {
			t.Fatalf("source workspace removed: %v", err)
		}
	}
	requireGoalWorkspaceIntegrationCleanV0(t, canonical)
	if strings.Contains(strings.Join(firstResult.EvidenceRefs, " "), canonical) || strings.Contains(strings.Join(secondResult.EvidenceRefs, " "), second) {
		t.Fatalf("evidence references leak local paths: first=%+v second=%+v", firstResult.EvidenceRefs, secondResult.EvidenceRefs)
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0ReplaysReceiptV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "replay.txt", "replay\n")
	request := goalWorkspaceIntegrationRequestForTestV0("integration-ref-replay-001", source, canonical, base, []string{"replay.txt"}, receiptDir)
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	first, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 || first.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", first, issues)
	}
	head := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD"))
	writeAppVCSFileV0(t, canonical, "local-only.txt", "dirty\n")
	replay, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 || replay.Status != GoalWorkspaceIntegrationStatusReplayedV0 || replay.IntegratedCommit != first.IntegratedCommit {
		t.Fatalf("replay=%+v issues=%+v", replay, issues)
	}
	if got := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD")); got != head {
		t.Fatalf("replay created a second commit: got=%s want=%s", got, head)
	}
	if status := strings.TrimSpace(runAppVCSGitV0(t, canonical, "status", "--porcelain")); status == "" {
		t.Fatal("replay cleaned unrelated canonical changes")
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0AbortsConflictAndRetainsWorkspaceV0(t *testing.T) {
	canonical, first, second, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, first, "README.md", "first\n")
	writeAppVCSFileV0(t, second, "README.md", "second\n")
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	if result, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), goalWorkspaceIntegrationRequestForTestV0("integration-ref-conflict-001", first, canonical, base, []string{"README.md"}, receiptDir)); len(issues) > 0 || result.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", result, issues)
	}
	result, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), goalWorkspaceIntegrationRequestForTestV0("integration-ref-conflict-002", second, canonical, base, []string{"README.md"}, receiptDir))
	if result.Status != GoalWorkspaceIntegrationStatusBlockedV0 || !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	requireGoalWorkspaceIntegrationCleanV0(t, canonical)
	requireGoalWorkspaceIntegrationCleanV0(t, second)
	if body, err := os.ReadFile(filepath.Join(canonical, "README.md")); err != nil || string(body) != "first\n" {
		t.Fatalf("canonical changed after conflict: body=%q err=%v", body, err)
	}
	if _, err := os.Stat(second); err != nil {
		t.Fatalf("conflicting source workspace removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(receiptDir, "integration-ref-conflict-002.json")); !os.IsNotExist(err) {
		t.Fatalf("conflicted integration wrote receipt: %v", err)
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0BlocksDirtyCanonicalV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "source.txt", "source\n")
	goalWorkspaceRunGitForTestV0(t, source, "add", "source.txt")
	goalWorkspaceRunGitForTestV0(t, source, "commit", "-m", "source change")
	writeAppVCSFileV0(t, canonical, "dirty.txt", "dirty\n")
	result, issues := (GitGoalWorkspaceIntegrationConnectorV0{}).IntegrateGoalWorkspaceV0(
		context.Background(),
		goalWorkspaceIntegrationRequestForTestV0("integration-ref-dirty-001", source, canonical, base, []string{"source.txt"}, receiptDir),
	)
	if result.Status != GoalWorkspaceIntegrationStatusBlockedV0 || !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if status := strings.TrimSpace(runAppVCSGitV0(t, canonical, "status", "--porcelain")); status == "" {
		t.Fatal("dirty canonical was modified or cleaned")
	}
	if _, err := os.Stat(filepath.Join(receiptDir, "integration-ref-dirty-001.json")); !os.IsNotExist(err) {
		t.Fatalf("dirty canonical wrote receipt: %v", err)
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0BlocksReceiptIdentityMismatchV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "identity.txt", "identity\n")
	request := goalWorkspaceIntegrationRequestForTestV0("integration-ref-identity-001", source, canonical, base, []string{"identity.txt"}, receiptDir)
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	if result, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request); len(issues) > 0 || result.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", result, issues)
	}
	request.CommitMessage = "test: changed receipt identity"
	result, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request)
	if result.Status != GoalWorkspaceIntegrationStatusBlockedV0 || !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0BlocksCanonicalOutsideBaseHistoryV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "source.txt", "source\n")
	goalWorkspaceRunGitForTestV0(t, canonical, "checkout", "--orphan", "unrelated")
	goalWorkspaceRunGitForTestV0(t, canonical, "rm", "-rf", ".")
	writeAppVCSFileV0(t, canonical, "unrelated.txt", "unrelated\n")
	goalWorkspaceRunGitForTestV0(t, canonical, "add", "unrelated.txt")
	goalWorkspaceRunGitForTestV0(t, canonical, "commit", "-m", "unrelated root")

	result, issues := (GitGoalWorkspaceIntegrationConnectorV0{}).IntegrateGoalWorkspaceV0(
		context.Background(),
		goalWorkspaceIntegrationRequestForTestV0("integration-ref-unrelated-001", source, canonical, base, []string{"source.txt"}, receiptDir),
	)
	if result.Status != GoalWorkspaceIntegrationStatusBlockedV0 || !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	requireGoalWorkspaceIntegrationCleanV0(t, canonical)
}

func TestGitGoalWorkspaceIntegrationConnectorV0ReconstructsMissingReceiptV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "recovery.txt", "recovery\n")
	request := goalWorkspaceIntegrationRequestForTestV0("integration-ref-recovery-001", source, canonical, base, []string{"recovery.txt"}, receiptDir)
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	first, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 || first.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", first, issues)
	}
	head := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD"))
	if err := os.Remove(filepath.Join(receiptDir, request.IntegrationRef+".json")); err != nil {
		t.Fatalf("remove receipt: %v", err)
	}
	recovered, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 || recovered.Status != GoalWorkspaceIntegrationStatusReplayedV0 || recovered.IntegratedCommit != first.IntegratedCommit {
		t.Fatalf("recovered=%+v issues=%+v", recovered, issues)
	}
	if got := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD")); got != head {
		t.Fatalf("recovery created duplicate commit: got=%s want=%s", got, head)
	}
	receipt, found, receiptIssues := loadGoalWorkspaceIntegrationReceiptV0(filepath.Join(receiptDir, request.IntegrationRef+".json"))
	if len(receiptIssues) > 0 || !found || receipt.IntegratedCommit != first.IntegratedCommit {
		t.Fatalf("receipt=%+v found=%t issues=%+v", receipt, found, receiptIssues)
	}
}

func newGoalWorkspaceIntegrationReposV0(t *testing.T) (string, string, string, string) {
	t.Helper()
	canonical := initAppVCSTestRepoV0(t)
	base := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD"))
	root := t.TempDir()
	first := filepath.Join(root, "goal-first")
	second := filepath.Join(root, "goal-second")
	goalWorkspaceRunGitForTestV0(t, canonical, "worktree", "add", "--detach", first, base)
	goalWorkspaceRunGitForTestV0(t, canonical, "worktree", "add", "--detach", second, base)
	return canonical, first, second, base
}

func goalWorkspaceIntegrationRequestForTestV0(integrationRef string, source string, canonical string, base string, writeSet []string, receiptDir string) GoalWorkspaceIntegrationRequestV0 {
	return GoalWorkspaceIntegrationRequestV0{
		IntegrationRef:     integrationRef,
		SourceWorkspaceDir: source,
		CanonicalWorkDir:   canonical,
		BaseRevision:       base,
		WriteSet:           writeSet,
		CommitMessage:      "test: integrate goal workspace",
		ReceiptDir:         receiptDir,
	}
}

func requireGoalWorkspaceIntegrationCleanV0(t *testing.T, repo string) {
	t.Helper()
	if status := strings.TrimSpace(runAppVCSGitV0(t, repo, "status", "--porcelain", "--untracked-files=all")); status != "" {
		t.Fatalf("repo must be clean: %s", status)
	}
}
