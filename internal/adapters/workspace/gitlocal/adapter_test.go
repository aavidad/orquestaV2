package gitlocal

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

type testLocator struct{ binding LocalRepositoryBinding }

type durableBindingResolverFunc func(
	context.Context,
	ports.ExecutionWorkspaceRef,
) (DurableWorkspaceBinding, bool, error)

func (function durableBindingResolverFunc) ResolveDurableWorkspaceBinding(
	ctx context.Context,
	ref ports.ExecutionWorkspaceRef,
) (DurableWorkspaceBinding, bool, error) {
	return function(ctx, ref)
}

func gitTestNoError(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func newTestAdapter(config Config) (*Adapter, error) { return newTestAdapterAfterHash(config, nil) }

func newTestAdapterAfterHash(config Config, afterHash func()) (*Adapter, error) {
	return newAdapter(config, gitPinPolicy{ownerUID: uint32(os.Geteuid()), afterHash: afterHash})
}

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
	fresh, err := newTestAdapter(Config{Root: adapter.root, GitCommand: testGitExecutable(t), Locator: adapter.loc, Now: adapter.now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fresh.Close() })
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

func TestResolveExecutionWorkspaceRecoversDurableBindingAfterRestart(t *testing.T) {
	original, request := testAdapterAndPrepare(t)
	prepared, err := original.Prepare(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	config := Config{
		Root: original.root, GitCommand: testGitExecutable(t), Locator: original.loc, Now: original.now,
	}
	if err := original.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := newTestAdapter(config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reopened.Close() })
	var calls atomic.Int64
	resolver := durableBindingResolverFunc(func(
		_ context.Context,
		ref ports.ExecutionWorkspaceRef,
	) (DurableWorkspaceBinding, bool, error) {
		calls.Add(1)
		if ref != request.WorkspaceRef {
			return DurableWorkspaceBinding{}, false, nil
		}
		return DurableWorkspaceBinding{Request: request, Prepared: prepared}, true, nil
	})
	if err := reopened.BindDurableWorkspaceBindingResolver(resolver); err != nil {
		t.Fatal(err)
	}
	const readers = 8
	var wait sync.WaitGroup
	wait.Add(readers)
	errorsByReader := make([]error, readers)
	paths := make([]string, readers)
	for index := 0; index < readers; index++ {
		go func(index int) {
			defer wait.Done()
			paths[index], errorsByReader[index] = reopened.ResolveExecutionWorkspace(
				context.Background(), request.WorkspaceRef,
			)
		}(index)
	}
	wait.Wait()
	for index := range paths {
		if errorsByReader[index] != nil || paths[index] != reopened.workspacePath(request.WorkspaceRef) {
			t.Fatalf("reader %d path=%q error=%v", index, paths[index], errorsByReader[index])
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("durable resolver calls=%d", calls.Load())
	}
	if _, found := reopened.preparedRecord(request.WorkspaceRef); found {
		t.Fatal("resolution recovery contaminated exact prepare replay cache")
	}
	replayed, err := reopened.Prepare(context.Background(), request)
	if err != nil || replayed != prepared {
		t.Fatalf("Prepare replay after Resolve=%+v error=%v want=%+v", replayed, err, prepared)
	}
}

func TestResolveExecutionWorkspaceRecoveryRejectsUntrustedDurableFacts(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(testing.TB, *Adapter, ports.WorkspacePrepareRequest, ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared)
	}{
		{"request_target", func(_ testing.TB, _ *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			request.TargetRef = "refs/heads/other"
			return request, prepared
		}},
		{"request_idempotency", func(_ testing.TB, _ *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			request.IdempotencyKey = "prepare:foreign"
			prepared.ReceiptRef = digestRef("workspace-receipt:", request.IdempotencyKey)
			return request, prepared
		}},
		{"request_time", func(_ testing.TB, _ *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			request.PreparedAt = request.PreparedAt.Add(time.Second)
			prepared.PreparedAt = request.PreparedAt
			return request, prepared
		}},
		{"adapter", func(_ testing.TB, _ *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			prepared.AdapterRef = "workspace:foreign"
			return request, prepared
		}},
		{"receipt", func(_ testing.TB, _ *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			prepared.ReceiptRef = "workspace-receipt:foreign"
			return request, prepared
		}},
		{"base", func(_ testing.TB, _ *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			prepared.BaseOID = strings.Repeat("f", len(prepared.BaseOID))
			return request, prepared
		}},
		{"prepare_marker", func(t testing.TB, adapter *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			repository, _, err := adapter.repository(context.Background(), request.RepositoryRef)
			gitTestNoError(t, err)
			_, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", "-d", markerRef(request.IdempotencyKey))
			gitTestNoError(t, err)
			return request, prepared
		}},
		{"binding_marker", func(t testing.TB, adapter *Adapter, request ports.WorkspacePrepareRequest, prepared ports.WorkspacePrepared) (ports.WorkspacePrepareRequest, ports.WorkspacePrepared) {
			repository, _, err := adapter.repository(context.Background(), request.RepositoryRef)
			gitTestNoError(t, err)
			digest := workspaceBindingDigest(request, prepared)
			_, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", "-d",
				markerRef("workspace-binding:"+request.WorkspaceRef.String()+":"+digest))
			gitTestNoError(t, err)
			return request, prepared
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			original, request := testAdapterAndPrepare(t)
			prepared, err := original.Prepare(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			request, prepared = test.mutate(t, original, request, prepared)
			reopened, err := newTestAdapter(Config{
				Root: original.root, GitCommand: testGitExecutable(t), Locator: original.loc, Now: original.now,
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = reopened.Close() })
			if err := reopened.BindDurableWorkspaceBindingResolver(durableBindingResolverFunc(func(
				context.Context, ports.ExecutionWorkspaceRef,
			) (DurableWorkspaceBinding, bool, error) {
				return DurableWorkspaceBinding{Request: request, Prepared: prepared}, true, nil
			})); err != nil {
				t.Fatal(err)
			}
			if _, err := reopened.ResolveExecutionWorkspace(
				context.Background(), request.WorkspaceRef,
			); ErrorCodeOf(err) != CodeWorkspaceUnsafe {
				t.Fatalf("untrusted recovery error=%v code=%q", err, ErrorCodeOf(err))
			}
		})
	}
}

func TestResolveExecutionWorkspaceRecoveryClassifiesResolverFailures(t *testing.T) {
	tests := []struct {
		name string
		run  durableBindingResolverFunc
		code ErrorCode
	}{
		{
			name: "not_found",
			run: func(context.Context, ports.ExecutionWorkspaceRef) (DurableWorkspaceBinding, bool, error) {
				return DurableWorkspaceBinding{}, false, nil
			},
			code: CodeWorkspaceNotFound,
		},
		{
			name: "authority_error",
			run: func(context.Context, ports.ExecutionWorkspaceRef) (DurableWorkspaceBinding, bool, error) {
				return DurableWorkspaceBinding{}, false, errors.New("state unavailable")
			},
			code: CodeWorkspaceUnsafe,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter, request := testAdapterAndPrepare(t)
			if err := adapter.BindDurableWorkspaceBindingResolver(test.run); err != nil {
				t.Fatal(err)
			}
			if _, err := adapter.ResolveExecutionWorkspace(context.Background(), request.WorkspaceRef); ErrorCodeOf(err) != test.code {
				t.Fatalf("resolver failure err=%v code=%q want=%q", err, ErrorCodeOf(err), test.code)
			}
		})
	}
}

func TestDurableWorkspaceResolverBindingIsBootOnly(t *testing.T) {
	resolver := durableBindingResolverFunc(func(
		context.Context, ports.ExecutionWorkspaceRef,
	) (DurableWorkspaceBinding, bool, error) {
		return DurableWorkspaceBinding{}, false, errors.New("unused")
	})
	t.Run("nil_and_double", func(t *testing.T) {
		adapter, _ := testAdapterAndPrepare(t)
		if err := adapter.BindDurableWorkspaceBindingResolver(nil); ErrorCodeOf(err) != CodeConfigInvalid {
			t.Fatalf("nil bind=%v", err)
		}
		if err := adapter.BindDurableWorkspaceBindingResolver(resolver); err != nil {
			t.Fatal(err)
		}
		if err := adapter.BindDurableWorkspaceBindingResolver(resolver); ErrorCodeOf(err) != CodeConfigInvalid {
			t.Fatalf("double bind=%v", err)
		}
	})
	t.Run("after_prepare", func(t *testing.T) {
		adapter, request := testAdapterAndPrepare(t)
		if _, err := adapter.Prepare(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		if err := adapter.BindDurableWorkspaceBindingResolver(resolver); ErrorCodeOf(err) != CodeConfigInvalid {
			t.Fatalf("bind after prepare=%v", err)
		}
	})
	t.Run("after_close", func(t *testing.T) {
		adapter, _ := testAdapterAndPrepare(t)
		if err := adapter.Close(); err != nil {
			t.Fatal(err)
		}
		if err := adapter.BindDurableWorkspaceBindingResolver(resolver); ErrorCodeOf(err) != CodeConfigInvalid {
			t.Fatalf("bind after close=%v", err)
		}
	})
	t.Run("concurrent_close", func(t *testing.T) {
		for index := 0; index < 16; index++ {
			adapter, _ := testAdapterAndPrepare(t)
			start := make(chan struct{})
			var bindErr, closeErr error
			var wait sync.WaitGroup
			wait.Add(2)
			go func() {
				defer wait.Done()
				<-start
				bindErr = adapter.BindDurableWorkspaceBindingResolver(resolver)
			}()
			go func() {
				defer wait.Done()
				<-start
				closeErr = adapter.Close()
			}()
			close(start)
			wait.Wait()
			if closeErr != nil {
				t.Fatalf("close %d: %v", index, closeErr)
			}
			if bindErr != nil && ErrorCodeOf(bindErr) != CodeConfigInvalid {
				t.Fatalf("bind %d: %v", index, bindErr)
			}
		}
	})
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
	fresh, err := newTestAdapter(Config{Root: adapter.root, GitCommand: testGitExecutable(t), Locator: adapter.loc, Now: adapter.now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fresh.Close() })
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
	return testAdapterAndPrepareFormat(t, "")
}

func testAdapterAndPrepareFormat(t *testing.T, format string) (*Adapter, ports.WorkspacePrepareRequest) {
	t.Helper()
	repository := t.TempDir()
	initArgs := []string{"init", "-b", "main"}
	if format != "" {
		initArgs = append(initArgs, "--object-format="+format)
	}
	gitTest(t, repository, initArgs...)
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
	adapter, err := newTestAdapter(Config{Root: adapterRoot, GitCommand: testGitExecutable(t), Locator: testLocator{binding: LocalRepositoryBinding{Path: repository, TargetRef: "refs/heads/main"}}, Now: func() time.Time { return time.Unix(1_700_000_000, 0).UTC() }})
	if err != nil {
		t.Fatalf("root=%s err=%v", adapterRoot, err)
	}
	t.Cleanup(func() { _ = adapter.Close() })
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

func testGitExecutable(t *testing.T) string {
	t.Helper()
	source, err := os.Open("/usr/bin/git")
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	path := filepath.Join(t.TempDir(), "git")
	target, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o500)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(target, source); err != nil {
		_ = target.Close()
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o500); err != nil {
		t.Fatal(err)
	}
	return path
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
