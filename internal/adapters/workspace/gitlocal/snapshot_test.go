package gitlocal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func TestSnapshotStreamRejectsTreeDiffTestAndPolicyDrift(t *testing.T) {
	fixture := newCommittedSnapshot(t, "", snapshotFixtureWrite{"allowed.txt", []byte("v17 snapshot\n")})
	adapter, request, workspace := fixture.adapter, fixture.request, fixture.workspace
	if err := drainSnapshot(adapter, request); err != nil {
		t.Fatalf("verify exact snapshot: %v", err)
	}
	assertWorkspaceMode(t, workspace, 0o700)

	fresh, err := newTestAdapter(Config{Root: adapter.root, GitCommand: testGitExecutable(t), Locator: adapter.loc, Now: adapter.now})
	gitTestNoError(t, err)
	t.Cleanup(func() { _ = fresh.Close() })
	if err := drainSnapshot(fresh, request); err != nil {
		t.Fatalf("restart lost verifiable snapshot: %v", err)
	}
	assertWorkspaceMode(t, workspace, 0o700)

	for name, mutate := range map[string]func(*ports.SnapshotVerificationRequest){
		"tree": func(candidate *ports.SnapshotVerificationRequest) {
			candidate.Subject.TreeOID = strings.Repeat("9", len(candidate.Subject.TreeOID))
			candidate.SubjectDigest = ports.TestSubjectDigest(candidate.Subject)
		},
		"diff": func(candidate *ports.SnapshotVerificationRequest) {
			candidate.Subject.DiffDigest = strings.Repeat("9", 64)
			candidate.SubjectDigest = ports.TestSubjectDigest(candidate.Subject)
		},
		"required_tests": func(candidate *ports.SnapshotVerificationRequest) {
			candidate.Subject.RequiredTestsDigest = strings.Repeat("9", 64)
		},
		"policy": func(candidate *ports.SnapshotVerificationRequest) {
			candidate.Subject.PolicyDigest = strings.Repeat("9", 64)
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := request
			mutate(&candidate)
			if err := drainSnapshot(adapter, candidate); err == nil {
				t.Fatal("drifted snapshot accepted")
			}
		})
	}

	gitTestNoError(t, os.WriteFile(filepath.Join(workspace, "allowed.txt"), []byte("dirty worktree ignored\n"), 0o600))
	if err := drainSnapshot(adapter, request); err != nil {
		t.Fatalf("dirty worktree affected object snapshot: %v", err)
	}
}

func TestSnapshotStreamRehashesSHA256Objects(t *testing.T) {
	fixture := newCommittedSnapshot(t, "sha256", snapshotFixtureWrite{"allowed.txt", []byte("sha256 snapshot\n")})
	if fixture.change.ObjectFormat != ports.GitObjectFormatSHA256 {
		t.Fatalf("object format=%q", fixture.change.ObjectFormat)
	}
	gitTestNoError(t, drainSnapshot(fixture.adapter, fixture.request))
}

func drainSnapshot(adapter *Adapter, request ports.SnapshotVerificationRequest) error {
	stream, err := adapter.OpenSnapshotStream(context.Background(), request)
	if err != nil {
		return err
	}
	_, readErr := io.Copy(io.Discard, stream)
	return errors.Join(readErr, stream.Close())
}

func TestCommitDiffDigestBindsContentNotOnlyPaths(t *testing.T) {
	commit := func(content string) ports.CommitResult {
		return newCommittedSnapshot(t, "", snapshotFixtureWrite{"allowed.txt", []byte(content)}).change
	}
	first, second := commit("first content\n"), commit("second content\n")
	if first.DiffDigest == second.DiffDigest {
		t.Fatal("different bytes at the same path produced the same diff digest")
	}
	legacy := sha256.Sum256([]byte("allowed.txt\x00"))
	if first.DiffDigest == hex.EncodeToString(legacy[:]) || second.DiffDigest == hex.EncodeToString(legacy[:]) {
		t.Fatal("diff digest still represents path names only")
	}
}

func TestSnapshotRequiresExactPersistedWorkspaceBindingWithoutSubjectMutation(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	repository := adapter.loc.(testLocator).binding.Path
	marker := markerRef("workspace-binding:" + request.Subject.WorkspaceRef.String() + ":" + request.Subject.WorkspaceBindingDigest)
	if _, err := adapter.gitRun(context.Background(), repository, nil, "update-ref", "-d", marker); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.OpenSnapshotStream(context.Background(), request); ErrorCodeOf(err) != CodeSnapshotInvalid {
		t.Fatalf("missing exact binding marker err=%v", err)
	}
}

func TestSnapshotRejectsDifferentWorkspaceBindingWithRecomputedSubjectDigest(t *testing.T) {
	adapter, request, _ := committedSnapshot(t)
	other := testPrepareRequestForOtherWorkspace(t, request.Subject)
	prepared, err := adapter.Prepare(context.Background(), other)
	gitTestNoError(t, err)
	repository := adapter.loc.(testLocator).binding.Path
	if _, err := adapter.gitRun(context.Background(), repository, nil, "update-ref",
		"refs/heads/"+workspaceBranch(prepared.WorkspaceRef), request.Subject.HeadOID, prepared.BaseOID); err != nil {
		t.Fatal(err)
	}

	request.Subject.WorkspaceRef = prepared.WorkspaceRef
	request.Subject.ExecutionRef = other.ExecutionRef
	request.Subject.WorkspaceBindingDigest = workspaceBindingDigest(other, prepared)
	request.SubjectDigest = ports.TestSubjectDigest(request.Subject)
	if err := drainSnapshot(adapter, request); ErrorCodeOf(err) != CodeSnapshotInvalid {
		t.Fatalf("rehashed alternate workspace binding err=%v", err)
	}
}

func TestSnapshotChangeMarkerTamperFailsClosed(t *testing.T) {
	for _, mutation := range []string{"delete", "move", "replace", "tag", "object_replace"} {
		t.Run(mutation, func(t *testing.T) {
			adapter, request, _ := committedSnapshot(t)
			repository := adapter.loc.(testLocator).binding.Path
			marker := snapshotChangeMarkerRef(request.Subject.ChangeSetRef)
			markerOID := trimOID(gitTestOutput(t, repository, "rev-parse", "--verify", marker))
			var err error
			switch mutation {
			case "delete":
				_, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", "-d", marker, markerOID)
			case "move":
				if _, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", marker+"-moved", markerOID); err == nil {
					_, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", "-d", marker, markerOID)
				}
			case "replace":
				_, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", marker,
					request.Subject.HeadOID, markerOID)
			case "tag":
				tag := []byte("object " + markerOID + "\ntype commit\ntag snapshot-marker\n" +
					"tagger Orquesta <orquesta@local> 1700000000 +0000\n\nmarker tag\n")
				var tagOID []byte
				tagOID, err = adapter.gitRun(context.Background(), repository, tag, "mktag")
				if err == nil {
					_, err = adapter.gitRun(context.Background(), repository, nil, "update-ref", marker,
						trimOID(tagOID), markerOID)
				}
			case "object_replace":
				object := gitObjectContent(t, repository, "commit", markerOID)
				object[0] ^= 1
				objectPath := filepath.Join(repository, ".git", "objects", markerOID[:2], markerOID[2:])
				if err = os.Chmod(objectPath, 0o600); err == nil {
					err = writeLooseGitObject(objectPath, "commit", object)
				}
			}
			gitTestNoError(t, err)
			if err := drainSnapshot(adapter, request); ErrorCodeOf(err) != CodeSnapshotInvalid {
				t.Fatalf("mutation=%s err=%v", mutation, err)
			}
		})
	}
}

func TestConcurrentCommitReplayRestoresSnapshotChangeMarkerAcrossRestart(t *testing.T) {
	fixture := newCommittedSnapshot(t, "", snapshotFixtureWrite{"allowed.txt", []byte("snapshot marker replay\n")})
	adapter, prepare, prepared, change := fixture.adapter, fixture.prepare, fixture.prepared, fixture.change
	commitRequest := testCommit(t, prepare, prepared)
	repository := adapter.loc.(testLocator).binding.Path
	marker := snapshotChangeMarkerRef(change.ChangeSetRef)
	markerOID := trimOID(gitTestOutput(t, repository, "rev-parse", "--verify", marker))
	if _, err := adapter.gitRun(context.Background(), repository, nil, "update-ref", "-d", marker, markerOID); err != nil {
		t.Fatal(err)
	}
	config := Config{Root: adapter.root, GitCommand: testGitExecutable(t), Locator: adapter.loc, Now: adapter.now}
	restarted := make([]*Adapter, 2)
	var err error
	for index := range restarted {
		restarted[index], err = newTestAdapter(config)
		gitTestNoError(t, err)
		t.Cleanup(func() { _ = restarted[index].Close() })
	}
	type outcome struct {
		result ports.CommitResult
		err    error
	}
	start, outcomes := make(chan struct{}), make(chan outcome, len(restarted))
	for _, current := range restarted {
		go func() {
			<-start
			result, replayErr := current.Commit(context.Background(), commitRequest)
			outcomes <- outcome{result, replayErr}
		}()
	}
	close(start)
	for range restarted {
		got := <-outcomes
		if got.err != nil || got.result.HeadOID != change.HeadOID || got.result.DiffDigest != change.DiffDigest {
			t.Fatalf("concurrent replay result=%+v err=%v", got.result, got.err)
		}
	}
	if err := drainSnapshot(restarted[0], gitSnapshotRequest(prepare, prepared, change)); err != nil {
		t.Fatalf("restored marker snapshot: %v", err)
	}
}

func testPrepareRequestForOtherWorkspace(t *testing.T, subject ports.TestSubject) ports.WorkspacePrepareRequest {
	t.Helper()
	workspace := mustWorkspace(t, "workspace:other-snapshot")
	execution := mustExecution(t, "execution:other-snapshot")
	principal, err := identity.NewPrincipalRef("principal:test")
	gitTestNoError(t, err)
	actor, err := goal.NewActorRef("actor:test")
	gitTestNoError(t, err)
	project, err := goal.NewProjectRef("project:test")
	gitTestNoError(t, err)
	return ports.WorkspacePrepareRequest{
		WorkspaceRef: workspace, PrincipalRef: principal, ActorRef: actor, ProjectRef: project,
		RepositoryRef: subject.RepositoryRef, GoalRef: subject.GoalRef, WorkItemRef: subject.WorkItemRef,
		ExecutionRef: execution, ExecutionAttempt: subject.ExecutionAttempt, PlanGeneration: subject.PlanGeneration,
		AppSpecGeneration: subject.AppSpecGeneration, AppSpecHash: subject.AppSpecHash,
		WriteSet: []string{"allowed.txt"}, WriteSetDigest: subject.WriteSetDigest,
		IntentRef: "intent:other-snapshot", AttemptRef: "attempt:other-snapshot", ActionFence: 3,
		IdempotencyKey: "prepare:other-snapshot", PreparedAt: time.Unix(1_700_000_020, 0).UTC(),
	}
}

func TestWorkspaceBindingMarkerMatchesCanonicalApplicationFact(t *testing.T) {
	adapter, request := testAdapterAndPrepare(t)
	prepared, err := adapter.Prepare(context.Background(), request)
	gitTestNoError(t, err)
	binding := application.WorkspaceBinding{
		Ref: prepared.WorkspaceRef, PrincipalRef: request.PrincipalRef, ActorRef: request.ActorRef,
		ProjectRef: request.ProjectRef, RepositoryRef: prepared.RepositoryRef, GoalRef: request.GoalRef,
		WorkItemRef: request.WorkItemRef, ExecutionRef: request.ExecutionRef, ExecutionAttempt: request.ExecutionAttempt,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration, SpecHash: request.AppSpecHash,
		WriteSet: request.WriteSet, WriteSetDigest: prepared.WriteSetDigest, TargetRef: prepared.TargetRef,
		BaseOID: prepared.BaseOID, ObjectFormat: prepared.ObjectFormat, AdapterRef: prepared.AdapterRef,
		EffectIntentRef: request.IntentRef, EffectAttemptRef: request.AttemptRef, EffectFence: request.ActionFence,
		ReceiptRef: "effect-receipt:" + request.IntentRef, PreparedAt: prepared.PreparedAt,
	}
	if got, want := workspaceBindingDigest(request, prepared), binding.Digest(); got != want {
		t.Fatalf("adapter binding digest=%s application=%s", got, want)
	}
}

func TestSnapshotTreeModePolicyRejectsGitlinkAndSpecialTypes(t *testing.T) {
	if _, err := snapshotEntryMode("160000"); ErrorCodeOf(err) != CodeSnapshotGitlink {
		t.Fatalf("gitlink code=%q", ErrorCodeOf(err))
	}
	for _, mode := range []string{"100600", "140000", "170000"} {
		if _, err := snapshotEntryMode(mode); ErrorCodeOf(err) != CodeSnapshotInvalid {
			t.Fatalf("special mode=%s code=%q", mode, ErrorCodeOf(err))
		}
	}
	for _, mode := range []string{"100644", "100755", "120000"} {
		if tree, err := snapshotEntryMode(mode); err != nil || tree {
			t.Fatalf("regular mode=%s tree=%v err=%v", mode, tree, err)
		}
	}
}

func TestSnapshotTreeRejectsUnsafePathsDuplicatesAndCollisions(t *testing.T) {
	for _, value := range []string{"", "/absolute", "../escape", "a/../b", ".git/config", "a/.GIT/config", "a\x00b"} {
		if validSnapshotPath(value) {
			t.Fatalf("unsafe path accepted: %q", value)
		}
	}
	if validSymlinkTarget("link", []byte{0xff}) {
		t.Fatal("invalid UTF-8 symlink target accepted")
	}
}

func gitSnapshotRequest(
	prepare ports.WorkspacePrepareRequest,
	prepared ports.WorkspacePrepared,
	change ports.CommitResult,
) ports.SnapshotVerificationRequest {
	requiredTestRef, _ := goal.NewRequiredTestRef("required-test:git-snapshot")
	toolRef, _ := goal.NewToolRef("tool:go")
	requiredTest, _ := goal.NewRequiredTestSpec(goal.RequiredTestSpecInput{
		Ref: requiredTestRef, ToolRef: toolRef, Arguments: []string{"test", "./..."}, WorkingDirectory: ".",
	})
	digest := func(value string) string {
		sum := sha256.Sum256([]byte(value))
		return hex.EncodeToString(sum[:])
	}
	subject := ports.TestSubject{
		GoalRef: prepare.GoalRef, WorkItemRef: prepare.WorkItemRef, ExecutionRef: prepare.ExecutionRef,
		ExecutionAttempt: prepare.ExecutionAttempt, PlanGeneration: prepare.PlanGeneration,
		WorkItemGeneration: 1,
		AppSpecGeneration:  prepare.AppSpecGeneration, AppSpecHash: prepare.AppSpecHash,
		WorkspaceRef: prepare.WorkspaceRef, WorkspaceBindingDigest: workspaceBindingDigest(prepare, prepared),
		ChangeSetRef: change.ChangeSetRef, ChangeSetDigest: digest("change"), RepositoryRef: prepare.RepositoryRef,
		ObjectFormat: change.ObjectFormat, BaseOID: change.BaseOID, ParentOID: change.ParentOID,
		HeadOID: change.HeadOID, TreeOID: change.TreeOID, DiffDigest: change.DiffDigest,
		WriteSetDigest: change.WriteSetDigest, RequiredTestsDigest: goal.RequiredTestsDigest([]goal.RequiredTestSpec{requiredTest}),
		PolicyRef: "policy:bubblewrap:v1", PolicyDigest: digest("policy"),
	}
	return ports.SnapshotVerificationRequest{
		Subject: subject, SubjectDigest: ports.TestSubjectDigest(subject),
		ChangedPaths: append([]string(nil), change.ChangedPaths...), WriteSet: append([]string(nil), prepare.WriteSet...),
	}
}
