package gitlocal

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type testLocator struct{ binding LocalRepositoryBinding }

func (value testLocator) LocateLocalRepository(context.Context, identity.RepositoryRef) (LocalRepositoryBinding, error) {
	return value.binding, nil
}

func TestWorkspacePrepareIsIdempotentAndUniquePerExecution(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	first, err := adapter.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("prepare replay differs: %#v %#v", first, second)
	}
	retry := request
	retry.AttemptRef, retry.ActionFence = "attempt:retry", 9
	replayed, err := adapter.Prepare(context.Background(), retry)
	if err != nil {
		t.Fatal(err)
	}
	if replayed != first {
		t.Fatalf("prepare retry differs: %#v %#v", replayed, first)
	}
	fresh, err := New(Config{Root: adapter.root, GitCommand: adapter.git, Locator: adapter.loc, Now: adapter.now})
	if err != nil {
		t.Fatal(err)
	}
	changedTarget := retry
	changedTarget.TargetRef = "refs/heads/other"
	if _, err := adapter.Prepare(context.Background(), changedTarget); ErrorCodeOf(err) != CodeWorkspaceConflict {
		t.Fatalf("changed target err=%v", err)
	}
	changedWriteSet := retry
	changedWriteSet.WriteSet = []string{"different.txt"}
	changedWriteSet.WriteSetDigest = ports.WorkspaceWriteSetDigest(changedWriteSet.WriteSet)
	if _, err := adapter.Prepare(context.Background(), changedWriteSet); ErrorCodeOf(err) != CodeWorkspaceConflict {
		t.Fatalf("changed write-set err=%v", err)
	}
	other := request
	other.WorkspaceRef = mustWorkspace(t, "workspace:other")
	other.ExecutionRef = mustExecution(t, "execution:other")
	if _, err := adapter.Prepare(context.Background(), other); ErrorCodeOf(err) != CodeWorkspaceConflict {
		t.Fatalf("same-key changed workspace err=%v", err)
	}
	if _, err := fresh.Prepare(context.Background(), other); ErrorCodeOf(err) != CodeWorkspaceConflict {
		t.Fatalf("restart same-key changed workspace err=%v", err)
	}
	other.IdempotencyKey = "prepare:other"
	other.IntentRef, other.AttemptRef, other.ActionFence = "intent:other", "attempt:other", 2
	if _, err := adapter.Prepare(context.Background(), other); err != nil {
		t.Fatal(err)
	}
	if adapter.workspacePath(request.WorkspaceRef) == adapter.workspacePath(other.WorkspaceRef) {
		t.Fatal("workspace path reused")
	}
}

func TestWorkspacePrepareRequiresExplicitTargetBinding(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	adapter.loc = testLocator{binding: LocalRepositoryBinding{Path: adapter.loc.(testLocator).binding.Path}}
	request.TargetRef = ""
	if _, err := adapter.Prepare(context.Background(), request); ErrorCodeOf(err) != CodeRepositoryInvalid {
		t.Fatalf("implicit target err=%v", err)
	}
	adapter.loc = testLocator{binding: LocalRepositoryBinding{
		Path: adapter.loc.(testLocator).binding.Path, TargetRef: "HEAD",
	}}
	if _, err := adapter.Prepare(context.Background(), request); ErrorCodeOf(err) != CodeRepositoryInvalid {
		t.Fatalf("symbolic target err=%v", err)
	}
}

func TestWorkspacePrepareReconcilesCrashAfterWorktreeAdd(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	repository, binding, err := adapter.repository(context.Background(), request.RepositoryRef)
	if err != nil {
		t.Fatal(err)
	}
	_, base, format, err := adapter.prepareTarget(context.Background(), repository, binding.TargetRef)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.workspacePath(request.WorkspaceRef)
	if _, err := adapter.gitRun(context.Background(), repository, nil, "worktree", "add", "--lock", "-b", workspaceBranch(request.WorkspaceRef), path, base); err != nil {
		t.Fatal(err)
	}
	if err := adapter.hardenCreatedWorkspaceGitMetadata(context.Background(), repository, path); err != nil {
		t.Fatal(err)
	}
	if _, found, err := adapter.gitRefOID(context.Background(), repository, workspaceBaseRef(request.WorkspaceRef)); err != nil || found {
		t.Fatalf("unexpected pre-crash marker: found=%v err=%v", found, err)
	}
	prepared, err := adapter.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.BaseOID != base || prepared.ObjectFormat != format {
		t.Fatalf("reconciled=%#v", prepared)
	}
	if marker, found, err := adapter.gitRefOID(context.Background(), repository, workspaceBaseRef(request.WorkspaceRef)); err != nil || !found || marker != base {
		t.Fatalf("marker=%q found=%v err=%v", marker, found, err)
	}
}

func TestWorkspacePrepareReconcilesCrashAfterBaseMarkerBeforeWorktree(t *testing.T) {
	for _, leaveBranch := range []bool{false, true} {
		t.Run(map[bool]string{false: "marker_only", true: "marker_and_orphan_branch"}[leaveBranch], func(t *testing.T) {
			adapter, request := testAdapterAndPrepare(t)
			repository, binding, err := adapter.repository(context.Background(), request.RepositoryRef)
			if err != nil {
				t.Fatal(err)
			}
			_, base, format, err := adapter.prepareTarget(context.Background(), repository, binding.TargetRef)
			if err != nil {
				t.Fatal(err)
			}
			if err := adapter.createWorkspaceBaseRef(context.Background(), repository, request.WorkspaceRef, base, format); err != nil {
				t.Fatal(err)
			}
			if leaveBranch {
				if _, err := adapter.gitRun(context.Background(), repository, nil, "branch", workspaceBranch(request.WorkspaceRef), base); err != nil {
					t.Fatal(err)
				}
			}
			prepared, err := adapter.Prepare(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.BaseOID != base {
				t.Fatalf("prepared base=%s want=%s", prepared.BaseOID, base)
			}
			if _, err := os.Stat(adapter.workspacePath(request.WorkspaceRef)); err != nil {
				t.Fatalf("reconciled worktree: %v", err)
			}
		})
	}
}

func TestResolveExecutionWorkspaceReturnsOnlyBoundSafePath(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	if _, err := adapter.Prepare(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	path, err := adapter.ResolveExecutionWorkspace(context.Background(), request.WorkspaceRef)
	if err != nil {
		t.Fatal(err)
	}
	if path != adapter.workspacePath(request.WorkspaceRef) {
		t.Fatalf("resolved wrong workspace: %q", path)
	}
	if _, err := adapter.ResolveExecutionWorkspace(context.Background(), mustWorkspace(t, "workspace:missing")); ErrorCodeOf(err) != CodeWorkspaceNotFound {
		t.Fatalf("missing err=%v", err)
	}
}

func TestOutOfWriteSetChangeLeavesGitUnmodified(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.workspacePath(prepare.WorkspaceRef)
	if err := os.WriteFile(filepath.Join(path, "outside.txt"), []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := testCommit(t, prepare, prepared)
	if _, err := adapter.Commit(context.Background(), request); ErrorCodeOf(err) != CodeWriteSetViolation {
		t.Fatalf("err=%v", err)
	}
	branch, err := adapter.gitOID(context.Background(), path, "HEAD")
	if err != nil || branch != prepared.BaseOID {
		t.Fatalf("git changed after rejected commit: %s %v", branch, err)
	}
}

func TestCommitBindsBaseTreeDiffWriteSetAndExecution(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	path := adapter.workspacePath(prepare.WorkspaceRef)
	if err := os.WriteFile(filepath.Join(path, "allowed.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := adapter.Commit(context.Background(), testCommit(t, prepare, prepared))
	if err != nil {
		t.Fatal(err)
	}
	if result.BaseOID != prepared.BaseOID || result.TreeOID == "" || result.HeadOID == "" || len(result.ChangedPaths) != 1 || result.ChangedPaths[0] != "allowed.txt" {
		t.Fatalf("bad commit result: %#v", result)
	}
}

func TestWorkspaceCommitReplaySurvivesAdapterRestart(t *testing.T) {
	adapter, prepare := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), prepare)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(adapter.workspacePath(prepare.WorkspaceRef), "allowed.txt"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := testCommit(t, prepare, prepared)
	first, err := adapter.Commit(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	retry := request
	retry.AttemptRef, retry.ActionFence = "attempt:commit-retry", 99
	retried, err := adapter.Commit(context.Background(), retry)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(retried, first) {
		t.Fatalf("commit retry differs: %#v %#v", retried, first)
	}
	changed := retry
	changed.WriteSet = []string{"different.txt"}
	changed.WriteSetDigest = ports.WorkspaceWriteSetDigest(changed.WriteSet)
	if _, err := adapter.Commit(context.Background(), changed); ErrorCodeOf(err) != CodeChangeConflict {
		t.Fatalf("changed commit err=%v", err)
	}
	fresh, err := New(Config{Root: adapter.root, GitCommand: adapter.git, Locator: adapter.loc, Now: adapter.now})
	if err != nil {
		t.Fatal(err)
	}
	prepareRetry := prepare
	prepareRetry.AttemptRef, prepareRetry.ActionFence = "attempt:prepare-after-restart", 88
	preparedAgain, err := fresh.Prepare(context.Background(), prepareRetry)
	if err != nil {
		t.Fatal(err)
	}
	if preparedAgain != prepared {
		t.Fatalf("prepared replay differs: %#v %#v", preparedAgain, prepared)
	}
	second, err := fresh.Commit(context.Background(), retry)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(second, first) {
		t.Fatalf("commit replay differs: %#v %#v", second, first)
	}
	otherChange := retry
	otherChange.ChangeSetRef, _ = ports.NewChangeSetRef("change:other-same-key")
	if _, err := fresh.Commit(context.Background(), otherChange); ErrorCodeOf(err) != CodeChangeConflict {
		t.Fatalf("restart same-key changed change-set err=%v", err)
	}
}

func testAdapterAndPrepare(t *testing.T) (*Adapter, ports.WorkspacePrepareRequest) {
	t.Helper()
	repository := t.TempDir()
	gitTest(t, repository, "init", "-b", "main")
	gitTest(t, repository, "config", "user.name", "Test")
	gitTest(t, repository, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(repository, "allowed.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitTest(t, repository, "add", "allowed.txt")
	gitTest(t, repository, "commit", "-m", "base")
	if err := os.Chmod(filepath.Join(repository, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(repository, 0o700); err != nil {
		t.Fatal(err)
	}
	adapterRoot := filepath.Join(t.TempDir(), "private")
	adapter, err := New(Config{Root: adapterRoot, GitCommand: "/usr/bin/git", Locator: testLocator{binding: LocalRepositoryBinding{Path: repository, TargetRef: "refs/heads/main"}}, Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }})
	if err != nil {
		t.Fatalf("root=%s err=%v", adapterRoot, err)
	}
	actor, _ := goal.NewActorRef("actor:test")
	project, _ := goal.NewProjectRef("project:test")
	principal, _ := identity.NewPrincipalRef("principal:test")
	repositoryRef, _ := identity.NewRepositoryRef("repository:test")
	goalRef, _ := goal.NewGoalRef("goal:test")
	item, _ := goal.NewWorkItemRef("work:test")
	execution := mustExecution(t, "execution:test")
	workspace := mustWorkspace(t, "workspace:test")
	request := ports.WorkspacePrepareRequest{WorkspaceRef: workspace, PrincipalRef: principal, ActorRef: actor, ProjectRef: project, RepositoryRef: repositoryRef, GoalRef: goalRef, WorkItemRef: item, ExecutionRef: execution, ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1, AppSpecHash: strings.Repeat("a", 64), WriteSet: []string{"allowed.txt"}, WriteSetDigest: ports.WorkspaceWriteSetDigest([]string{"allowed.txt"}), IntentRef: "intent:test", AttemptRef: "attempt:test", ActionFence: 1, IdempotencyKey: "prepare:test", PreparedAt: time.Unix(1_700_000_000, 0).UTC()}
	return adapter, request
}

func testCommit(t *testing.T, prepare ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) ports.CommitRequest {
	t.Helper()
	change, _ := ports.NewChangeSetRef("change:test")
	return ports.CommitRequest{ChangeSetRef: change, WorkspaceRef: prepare.WorkspaceRef, PrincipalRef: prepare.PrincipalRef, ActorRef: prepare.ActorRef, ProjectRef: prepare.ProjectRef, RepositoryRef: prepare.RepositoryRef, GoalRef: prepare.GoalRef, WorkItemRef: prepare.WorkItemRef, ExecutionRef: prepare.ExecutionRef, ExecutionAttempt: 1, PlanGeneration: 1, AppSpecGeneration: 1, AppSpecHash: prepare.AppSpecHash, BaseOID: prepared.BaseOID, ObjectFormat: prepared.ObjectFormat, WriteSet: prepare.WriteSet, WriteSetDigest: prepare.WriteSetDigest, IntentRef: "intent:commit", AttemptRef: "attempt:commit", ActionFence: 2, IdempotencyKey: "commit:test", CommittedAt: time.Unix(1_700_000_010, 0).UTC()}
}
func mustWorkspace(t *testing.T, value string) ports.ExecutionWorkspaceRef {
	t.Helper()
	ref, err := ports.NewExecutionWorkspaceRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
func mustExecution(t *testing.T, value string) goal.ExecutionRef {
	t.Helper()
	ref, err := goal.NewExecutionRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
func gitTest(t *testing.T, directory string, args ...string) {
	t.Helper()
	command := exec.Command("/usr/bin/git", append([]string{"-C", directory}, args...)...)
	command.Env = []string{
		"PATH=/usr/bin:/bin", "HOME=" + directory,
		"GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0",
	}
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
