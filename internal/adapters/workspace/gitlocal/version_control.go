package gitlocal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"orquesta/internal/ports"
)

func (adapter *Adapter) Commit(ctx context.Context, request ports.CommitRequest) (ports.CommitResult, error) {
	if err := adapter.ensureAvailable(); err != nil {
		return ports.CommitResult{}, &Error{Code: CodeUnavailable}
	}
	if err := ports.ValidateCommitRequest(request); err != nil {
		return ports.CommitResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.CommitResult{}, err
	}
	unlock := adapter.lockScopes("effect:"+request.IdempotencyKey, "workspace:"+request.WorkspaceRef.String(),
		"change:"+request.ChangeSetRef.String())
	defer unlock()
	if err := adapter.commitStateConflict(request); err != nil {
		return ports.CommitResult{}, err
	}
	repository, _, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return ports.CommitResult{}, err
	}
	message := effectMarkerMessage("commit", commitRequestDigest(request))
	if err := adapter.ensureEffectClaim(ctx, repository, request.ObjectFormat, request.IdempotencyKey, message,
		request.CommittedAt, CodeChangeConflict); err != nil {
		return ports.CommitResult{}, err
	}
	record, recorded := adapter.preparedRecord(request.WorkspaceRef)
	if err := validatePreparedCommit(request, record, recorded); err != nil {
		return ports.CommitResult{}, err
	}
	return adapter.commitWorkspace(ctx, repository, request, record, recorded)
}

func validatePreparedCommit(request ports.CommitRequest, record preparedRecord, recorded bool) error {
	if !recorded {
		return nil
	}
	if record.request.RepositoryRef != request.RepositoryRef || record.request.ExecutionRef != request.ExecutionRef ||
		record.result.BaseOID != request.BaseOID || record.request.WriteSetDigest != request.WriteSetDigest {
		return &Error{Code: CodeWorkspaceConflict}
	}
	return nil
}

func (adapter *Adapter) commitWorkspace(
	ctx context.Context,
	repository string,
	request ports.CommitRequest,
	record preparedRecord,
	recorded bool,
) (ports.CommitResult, error) {
	if base, err := adapter.workspaceBaseOID(ctx, repository, request.WorkspaceRef); err != nil || base != request.BaseOID {
		return ports.CommitResult{}, &Error{Code: CodeBaseStale, Cause: err}
	}
	path := adapter.workspacePath(request.WorkspaceRef)
	gitDir, err := adapter.verifyWorkspace(ctx, repository, path, "", request.WorkspaceRef)
	if err != nil {
		return ports.CommitResult{}, err
	}
	if recorded && (record.path != path || gitDir != record.gitFile) {
		return ports.CommitResult{}, &Error{Code: CodeWorkspaceUnsafe, Cause: errUnsafeWorkspace}
	}
	head, err := adapter.gitOID(ctx, path, "HEAD")
	if err != nil {
		return ports.CommitResult{}, err
	}
	if head != request.BaseOID {
		return adapter.replayCommit(ctx, request, path, head)
	}
	return adapter.createWorkspaceCommit(ctx, request, path)
}

func (adapter *Adapter) createWorkspaceCommit(
	ctx context.Context,
	request ports.CommitRequest,
	path string,
) (ports.CommitResult, error) {
	paths, err := adapter.changedPaths(ctx, path, request.BaseOID)
	if err != nil {
		return ports.CommitResult{}, err
	}
	if len(paths) == 0 {
		return ports.CommitResult{}, &Error{Code: CodeNoChanges}
	}
	if !pathsWithin(paths, request.WriteSet) {
		return ports.CommitResult{}, &Error{Code: CodeWriteSetViolation}
	}
	pathspec := make([]byte, 0)
	for _, path := range paths {
		pathspec = append(pathspec, []byte(path)...)
		pathspec = append(pathspec, 0)
	}
	if _, err := adapter.gitRun(ctx, path, pathspec, "add", "-A", "--pathspec-from-file=-", "--pathspec-file-nul"); err != nil {
		return ports.CommitResult{}, err
	}
	treeRaw, err := adapter.gitRun(ctx, path, nil, "write-tree")
	if err != nil {
		return ports.CommitResult{}, err
	}
	tree := trimOID(treeRaw)
	commitRaw, err := adapter.gitCommitTree(ctx, path, tree, request.BaseOID, "", request.CommittedAt, commitMessage(request))
	if err != nil {
		return ports.CommitResult{}, err
	}
	head := trimOID(commitRaw)
	branch := "refs/heads/" + workspaceBranch(request.WorkspaceRef)
	if _, err := adapter.gitRun(ctx, path, nil, "update-ref", branch, head, request.BaseOID); err != nil {
		actual, lookupErr := adapter.gitOID(ctx, path, "HEAD")
		if lookupErr == nil && actual != request.BaseOID {
			return adapter.replayCommit(ctx, request, path, actual)
		}
		return ports.CommitResult{}, &Error{Code: CodeBaseStale, Cause: err}
	}
	return adapter.rememberCommitResult(ctx, request, path, head, tree, paths)
}

func commitMessage(request ports.CommitRequest) []byte {
	return []byte("orquesta changeset " + request.ChangeSetRef.String() +
		"\n\nrequest-digest " + commitRequestDigest(request) + "\n")
}

// crashFrontierCommitMessage is accepted only after the full-payload effect
// claim exists and the complete deterministic commit OID is recomputed.
func crashFrontierCommitMessage(request ports.CommitRequest) []byte {
	return []byte("orquesta changeset " + request.ChangeSetRef.String() + "\n")
}

func (adapter *Adapter) rememberCommitResult(
	ctx context.Context,
	request ports.CommitRequest,
	workspace string,
	head, tree string,
	paths []string,
) (ports.CommitResult, error) {
	diffDigest, err := adapter.committedDiffDigest(ctx, workspace, request.BaseOID, head)
	if err != nil {
		return ports.CommitResult{}, err
	}
	result := ports.CommitResult{ChangeSetRef: request.ChangeSetRef, WorkspaceRef: request.WorkspaceRef,
		RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef, BaseOID: request.BaseOID,
		ParentOID: request.BaseOID, HeadOID: head, TreeOID: tree, ObjectFormat: request.ObjectFormat,
		DiffDigest: diffDigest, ChangedPaths: paths, WriteSetDigest: request.WriteSetDigest,
		ParentChangeRef: request.ParentChangeRef, AdapterRef: adapterRef,
		ReceiptRef: digestRef("commit-receipt:", request.IdempotencyKey), CommittedAt: request.CommittedAt}
	if err := ports.ValidateCommitResult(request, result); err != nil {
		return ports.CommitResult{}, err
	}
	if err := adapter.ensureSnapshotChangeMarker(ctx, workspace, request, result); err != nil {
		return ports.CommitResult{}, err
	}
	adapter.rememberCommit(commitRecord{request: request, result: result})
	return result, nil
}

func (adapter *Adapter) replayCommit(ctx context.Context, request ports.CommitRequest, path, head string) (ports.CommitResult, error) {
	parents, err := adapter.gitRun(ctx, path, nil, "show", "-s", "--format=%P", head)
	if err != nil || trimOID(parents) != request.BaseOID {
		return ports.CommitResult{}, &Error{Code: CodeChangeConflict, Cause: err}
	}
	message, err := adapter.gitRun(ctx, path, nil, "show", "-s", "--format=%B", head)
	expectedMessage, messageOK := deterministicReplayMessage(message, request)
	if err != nil || !messageOK {
		return ports.CommitResult{}, &Error{Code: CodeChangeConflict, Cause: err}
	}
	treeRaw, err := adapter.gitRun(ctx, path, nil, "show", "-s", "--format=%T", head)
	if err != nil {
		return ports.CommitResult{}, err
	}
	tree := trimOID(treeRaw)
	expectedRaw, err := adapter.gitCommitTree(ctx, path, tree, request.BaseOID, "", request.CommittedAt, expectedMessage)
	if err != nil || trimOID(expectedRaw) != head {
		return ports.CommitResult{}, &Error{Code: CodeChangeConflict, Cause: err}
	}
	paths, err := adapter.committedPaths(ctx, path, head)
	if err != nil || len(paths) == 0 || !pathsWithin(paths, request.WriteSet) {
		return ports.CommitResult{}, &Error{Code: CodeChangeConflict, Cause: err}
	}
	return adapter.rememberCommitResult(ctx, request, path, head, tree, paths)
}

func deterministicReplayMessage(actual []byte, request ports.CommitRequest) ([]byte, bool) {
	for _, candidate := range [][]byte{commitMessage(request), crashFrontierCommitMessage(request)} {
		if bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(candidate)) {
			return candidate, true
		}
	}
	return nil, false
}

func (adapter *Adapter) gitCommitTree(ctx context.Context, repository, tree, parent, secondParent string, at time.Time, message []byte) ([]byte, error) {
	stamp := at.UTC().Format("2006-01-02T15:04:05Z")
	args := []string{"commit-tree", tree}
	if parent != "" {
		args = append(args, "-p", parent)
	}
	if secondParent != "" {
		args = append(args, "-p", secondParent)
	}
	// commit-tree reads these values only for this exact child process. They are
	// deterministic facts from the persisted intent, never ambient identity.
	command, cleanup, err := adapter.pinnedGitCommand(ctx, nil, append([]string{
		"-c", "core.hooksPath=/dev/null", "-c", "commit.gpgSign=false", "-C", repository,
	}, args...)...)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	command.Env = append(gitEnvironment(adapter.root),
		"GIT_AUTHOR_NAME=Orquesta", "GIT_AUTHOR_EMAIL=orquesta@local", "GIT_AUTHOR_DATE="+stamp,
		"GIT_COMMITTER_NAME=Orquesta", "GIT_COMMITTER_EMAIL=orquesta@local", "GIT_COMMITTER_DATE="+stamp)
	return adapter.runGitCommand(ctx, command, message)
}

func (adapter *Adapter) committedPaths(ctx context.Context, workspace, head string) ([]string, error) {
	output, err := adapter.gitRun(ctx, workspace, nil,
		"diff-tree", "--no-ext-diff", "--no-textconv", "--no-commit-id", "--name-only", "-z", "-r", "--no-renames", head)
	if err != nil {
		return nil, err
	}
	paths := splitNUL(output)
	sort.Strings(paths)
	return paths, nil
}

func (adapter *Adapter) changedPaths(ctx context.Context, workspace, base string) ([]string, error) {
	tracked, err := adapter.gitRun(ctx, workspace, nil, "diff", "--no-ext-diff", "--no-textconv", "--name-only", "-z", "--no-renames", base)
	if err != nil {
		return nil, err
	}
	untracked, err := adapter.gitRun(ctx, workspace, nil, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, value := range append(splitNUL(tracked), splitNUL(untracked)...) {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func splitNUL(value []byte) []string {
	return strings.FieldsFunc(string(value), func(r rune) bool { return r == 0 })
}
func pathsWithin(paths, scopes []string) bool {
	for _, path := range paths {
		path = filepath.ToSlash(path)
		if path == ".git" || strings.HasPrefix(path, ".git/") {
			return false
		}
		ok := false
		for _, scope := range scopes {
			if path == scope || strings.HasPrefix(path, scope+"/") {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func (adapter *Adapter) committedDiffDigest(ctx context.Context, repository, parent, head string) (string, error) {
	raw, err := adapter.gitRun(ctx, repository, nil, "diff-tree", "--no-ext-diff", "--no-textconv",
		"--raw", "-z", "-r", "--no-abbrev", "--no-renames", "--no-commit-id", parent, head)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}
