package orquestaruntimeworktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitGoalWorkspaceIntegrationConnectorV0IntegratesChainedExpectedParentsV0(t *testing.T) {
	canonical, first, second, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, first, "first.txt", "first\n")
	writeAppVCSFileV0(t, second, "second.txt", "second\n")
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	firstResult, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), goalWorkspaceIntegrationRequestForTestV0("integration-ref-disjoint-001", first, canonical, base, []string{"first.txt"}, receiptDir))
	if len(issues) > 0 || firstResult.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", firstResult, issues)
	}
	if firstResult.ExpectedParentRevision != base {
		t.Fatalf("first expected parent=%s want=%s", firstResult.ExpectedParentRevision, base)
	}
	if parent := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", firstResult.IntegratedCommit+"^")); parent != firstResult.ExpectedParentRevision {
		t.Fatalf("first integrated parent=%s want=%s", parent, firstResult.ExpectedParentRevision)
	}
	secondRequest := goalWorkspaceIntegrationRequestForTestV0("integration-ref-disjoint-002", second, canonical, base, []string{"second.txt"}, receiptDir)
	secondRequest.ExpectedParentRevision = firstResult.IntegratedCommit
	secondResult, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), secondRequest)
	if len(issues) > 0 || secondResult.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("second=%+v issues=%+v", secondResult, issues)
	}
	if secondResult.ExpectedParentRevision != firstResult.IntegratedCommit {
		t.Fatalf("second expected parent=%s want=%s", secondResult.ExpectedParentRevision, firstResult.IntegratedCommit)
	}
	if parent := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", secondResult.IntegratedCommit+"^")); parent != secondResult.ExpectedParentRevision {
		t.Fatalf("second integrated parent=%s want=%s", parent, secondResult.ExpectedParentRevision)
	}
	for _, result := range []GoalWorkspaceIntegrationResultV0{firstResult, secondResult} {
		receipt, found, receiptIssues := loadGoalWorkspaceIntegrationReceiptV0(filepath.Join(receiptDir, result.IntegrationRef+".json"))
		if len(receiptIssues) > 0 || !found || receipt.SourceCommit != result.SourceCommit || receipt.IntegratedCommit != result.IntegratedCommit || receipt.ExpectedParentRevision != result.ExpectedParentRevision {
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

func TestGitGoalWorkspaceIntegrationConnectorV0IntegraSinIdentidadGitConfiguradaV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	if err := os.WriteFile(filepath.Join(source, "identity-free.txt"), []byte("goal output\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	goalWorkspaceRunGitForTestV0(t, canonical, "config", "--unset-all", "user.name")
	goalWorkspaceRunGitForTestV0(t, canonical, "config", "--unset-all", "user.email")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	result, issues := (GitGoalWorkspaceIntegrationConnectorV0{}).IntegrateGoalWorkspaceV0(
		context.Background(),
		goalWorkspaceIntegrationRequestForTestV0(
			"integration-ref-without-git-identity-001",
			source,
			canonical,
			base,
			[]string{"identity-free.txt"},
			receiptDir,
		),
	)
	if len(issues) > 0 || result.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	wantIdentity := goalWorkspaceIntegrationGitUserNameV0 + " <" + goalWorkspaceIntegrationGitUserEmailV0 + ">"
	if got := strings.TrimSpace(runAppVCSGitV0(t, canonical, "show", "-s", "--format=%an <%ae>|%cn <%ce>", result.IntegratedCommit)); got != wantIdentity+"|"+wantIdentity {
		t.Fatalf("identidad commit=%q", got)
	}
	if got := strings.TrimSpace(runAppVCSGitV0(t, source, "show", "-s", "--format=%an <%ae>|%cn <%ce>", result.SourceCommit)); got != wantIdentity+"|"+wantIdentity {
		t.Fatalf("identidad source commit=%q", got)
	}
	head := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD"))
	replay, replayIssues := (GitGoalWorkspaceIntegrationConnectorV0{}).IntegrateGoalWorkspaceV0(
		context.Background(),
		goalWorkspaceIntegrationRequestForTestV0(
			"integration-ref-without-git-identity-001", source, canonical, base,
			[]string{"identity-free.txt"}, receiptDir,
		),
	)
	if len(replayIssues) > 0 || replay.Status != GoalWorkspaceIntegrationStatusReplayedV0 ||
		replay.IntegratedCommit != result.IntegratedCommit ||
		strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD")) != head {
		t.Fatalf("replay=%+v issues=%+v", replay, replayIssues)
	}
	requireGoalWorkspaceIntegrationCleanV0(t, canonical)
	requireGoalWorkspaceIntegrationCleanV0(t, source)
	for _, key := range []string{"user.name", "user.email"} {
		command := exec.Command("git", "-C", canonical, "config", "--local", "--get", key)
		if output, err := command.CombinedOutput(); err == nil || strings.TrimSpace(string(output)) != "" {
			t.Fatalf("config %s persistida: output=%q err=%v", key, output, err)
		}
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0ReplaysLegacyRequestFromReceiptV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "replay.txt", "replay\n")
	request := goalWorkspaceIntegrationRequestForTestV0("integration-ref-replay-001", source, canonical, base, []string{"replay.txt"}, receiptDir)
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	first, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), request)
	if len(issues) > 0 || first.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", first, issues)
	}
	writeAppVCSFileV0(t, canonical, "local-only.txt", "later\n")
	goalWorkspaceRunGitForTestV0(t, canonical, "add", "local-only.txt")
	goalWorkspaceRunGitForTestV0(t, canonical, "commit", "-m", "later canonical change")
	head := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD"))
	legacyReplayRequest := request
	legacyReplayRequest.ExpectedParentRevision = ""
	replay, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), legacyReplayRequest)
	if len(issues) > 0 || replay.Status != GoalWorkspaceIntegrationStatusReplayedV0 || replay.IntegratedCommit != first.IntegratedCommit {
		t.Fatalf("replay=%+v issues=%+v", replay, issues)
	}
	if got := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD")); got != head {
		t.Fatalf("replay created a second commit: got=%s want=%s", got, head)
	}
	if parent := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", replay.IntegratedCommit+"^")); parent != replay.ExpectedParentRevision {
		t.Fatalf("receipt replay accepted wrong integrated parent: got=%s want=%s", parent, replay.ExpectedParentRevision)
	}
	if _, err := os.Stat(filepath.Join(canonical, "local-only.txt")); err != nil {
		t.Fatalf("replay lost later canonical commit: %v", err)
	}
	receipt, found, receiptIssues := loadGoalWorkspaceIntegrationReceiptV0(filepath.Join(receiptDir, request.IntegrationRef+".json"))
	if len(receiptIssues) > 0 || !found {
		t.Fatalf("receipt=%+v found=%t issues=%+v", receipt, found, receiptIssues)
	}
	receipt.IntegratedCommit = head
	if receiptIssues := writeGoalWorkspaceIntegrationReceiptV0(filepath.Join(receiptDir, request.IntegrationRef+".json"), receipt); len(receiptIssues) > 0 {
		t.Fatalf("write tampered receipt: %+v", receiptIssues)
	}
	blocked, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), legacyReplayRequest)
	if blocked.Status != GoalWorkspaceIntegrationStatusBlockedV0 || !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("tampered receipt replay=%+v issues=%+v", blocked, issues)
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0BlocksStaleExpectedParentWithoutCherryPickV0(t *testing.T) {
	canonical, source, _, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, source, "source.txt", "source\n")
	writeAppVCSFileV0(t, canonical, "later.txt", "later\n")
	goalWorkspaceRunGitForTestV0(t, canonical, "add", "later.txt")
	goalWorkspaceRunGitForTestV0(t, canonical, "commit", "-m", "advance canonical head")
	head := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD"))

	result, issues := (GitGoalWorkspaceIntegrationConnectorV0{}).IntegrateGoalWorkspaceV0(
		context.Background(),
		goalWorkspaceIntegrationRequestForTestV0("integration-ref-stale-parent-001", source, canonical, base, []string{"source.txt"}, receiptDir),
	)
	if result.Status != GoalWorkspaceIntegrationStatusBlockedV0 || !goalWorkspaceHasIssueForTestV0(issues, WorktreeIssueWorkspaceConflictV0) {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if got := strings.TrimSpace(runAppVCSGitV0(t, canonical, "rev-parse", "HEAD")); got != head {
		t.Fatalf("stale integration changed HEAD: got=%s want=%s", got, head)
	}
	if result.SourceCommit == "" {
		t.Fatalf("expected source commit in blocked result: %+v", result)
	}
	if _, err := (GitGoalWorkspaceIntegrationConnectorV0{}).VCS.gitOutputV0(context.Background(), canonical, "merge-base", "--is-ancestor", result.SourceCommit, "HEAD"); err == nil {
		t.Fatalf("stale source commit was cherry-picked: %s", result.SourceCommit)
	}
	if _, err := os.Stat(filepath.Join(receiptDir, "integration-ref-stale-parent-001.json")); !os.IsNotExist(err) {
		t.Fatalf("stale integration wrote receipt: %v", err)
	}
}

func TestGitGoalWorkspaceIntegrationConnectorV0DefaultsExpectedParentToLockedCanonicalHeadV0(t *testing.T) {
	canonical, first, second, base := newGoalWorkspaceIntegrationReposV0(t)
	receiptDir := filepath.Join(t.TempDir(), "receipts")
	writeAppVCSFileV0(t, first, "first.txt", "first\n")
	writeAppVCSFileV0(t, second, "second.txt", "second\n")
	connector := GitGoalWorkspaceIntegrationConnectorV0{}
	firstResult, issues := connector.IntegrateGoalWorkspaceV0(
		context.Background(),
		goalWorkspaceIntegrationRequestForTestV0("integration-ref-legacy-first-001", first, canonical, base, []string{"first.txt"}, receiptDir),
	)
	if len(issues) > 0 || firstResult.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("first=%+v issues=%+v", firstResult, issues)
	}
	secondRequest := goalWorkspaceIntegrationRequestForTestV0("integration-ref-legacy-second-001", second, canonical, base, []string{"second.txt"}, receiptDir)
	secondRequest.ExpectedParentRevision = ""
	secondResult, issues := connector.IntegrateGoalWorkspaceV0(context.Background(), secondRequest)
	if len(issues) > 0 || secondResult.Status != GoalWorkspaceIntegrationStatusIntegratedV0 {
		t.Fatalf("second=%+v issues=%+v", secondResult, issues)
	}
	if secondResult.ExpectedParentRevision != firstResult.IntegratedCommit {
		t.Fatalf("legacy expected parent=%s want=%s", secondResult.ExpectedParentRevision, firstResult.IntegratedCommit)
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

func TestGitGoalWorkspaceIntegrationConnectorV0RechazaWriteSetControlAntesDeEfectosV0(t *testing.T) {
	root := t.TempDir()
	receiptDir := filepath.Join(root, "receipts")
	result, issues := (GitGoalWorkspaceIntegrationConnectorV0{}).IntegrateGoalWorkspaceV0(
		context.Background(),
		GoalWorkspaceIntegrationRequestV0{
			IntegrationRef:     "integration-ref-control-write-set",
			SourceWorkspaceDir: filepath.Join(root, "source"),
			CanonicalWorkDir:   filepath.Join(root, "canonical"),
			BaseRevision:       "base-revision-control-write-set",
			WriteSet:           []string{".git"},
			CommitMessage:      "test: rejected control write set",
			ReceiptDir:         receiptDir,
		},
	)
	if result.Status != GoalWorkspaceIntegrationStatusBlockedV0 || len(issues) != 1 || issues[0].Code != WorktreeIssueControlPathV0 {
		t.Fatalf("result=%+v issues=%+v", result, issues)
	}
	if _, err := os.Stat(receiptDir); !os.IsNotExist(err) {
		t.Fatalf("integration creó receipts: %v", err)
	}
}

func goalWorkspaceIntegrationRequestForTestV0(integrationRef string, source string, canonical string, base string, writeSet []string, receiptDir string) GoalWorkspaceIntegrationRequestV0 {
	return GoalWorkspaceIntegrationRequestV0{
		IntegrationRef:         integrationRef,
		SourceWorkspaceDir:     source,
		CanonicalWorkDir:       canonical,
		BaseRevision:           base,
		ExpectedParentRevision: base,
		WriteSet:               writeSet,
		CommitMessage:          "test: integrate goal workspace",
		ReceiptDir:             receiptDir,
	}
}

func requireGoalWorkspaceIntegrationCleanV0(t *testing.T, repo string) {
	t.Helper()
	if status := strings.TrimSpace(runAppVCSGitV0(t, repo, "status", "--porcelain", "--untracked-files=all")); status != "" {
		t.Fatalf("repo must be clean: %s", status)
	}
}
