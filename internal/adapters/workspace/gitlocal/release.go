package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/ports"
)

func (adapter *Adapter) Release(ctx context.Context, request ports.WorkspaceReleaseRequest) (ports.WorkspaceReleaseReceipt, error) {
	if err := adapter.ensureAvailable(); err != nil {
		return ports.WorkspaceReleaseReceipt{}, &Error{Code: CodeUnavailable}
	}
	result, err := workspaceReleaseReceipt(request)
	if err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	unlock := adapter.lockScopes("effect:"+request.IdempotencyKey, "workspace:"+request.WorkspaceRef.String())
	defer unlock()
	if err := adapter.releaseStateConflict(request); err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	repository, _, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	format, err := adapter.repositoryObjectFormat(ctx, repository)
	if err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	message := effectMarkerMessage("release", releaseRequestDigest(request))
	if err := adapter.ensureEffectClaim(ctx, repository, format, request.IdempotencyKey, message,
		request.RequestedAt, CodeWorkspaceConflict); err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	if err := adapter.releaseWorkspace(ctx, repository, request); err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	adapter.rememberRelease(releaseRecord{request: request, result: result})
	return result, nil
}

func workspaceReleaseReceipt(request ports.WorkspaceReleaseRequest) (ports.WorkspaceReleaseReceipt, error) {
	result := ports.WorkspaceReleaseReceipt{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, Released: true,
		ReceiptRef: digestRef("workspace-release:", request.IdempotencyKey), ReleasedAt: request.RequestedAt,
	}
	if err := ports.ValidateWorkspaceReleaseReceipt(request, result); err != nil {
		return ports.WorkspaceReleaseReceipt{}, err
	}
	return result, nil
}

func (adapter *Adapter) releaseWorkspace(
	ctx context.Context,
	repository string,
	request ports.WorkspaceReleaseRequest,
) error {
	path := adapter.workspacePath(request.WorkspaceRef)
	record, recorded := adapter.preparedRecord(request.WorkspaceRef)
	if recorded && (record.request.RepositoryRef != request.RepositoryRef || record.request.ExecutionRef != request.ExecutionRef) {
		return &Error{Code: CodeWorkspaceNotFound}
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return adapter.verifyReleasedWorkspace(ctx, repository, path, request.WorkspaceRef)
	} else if err != nil {
		return &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	gitDir, err := adapter.verifyWorkspace(ctx, repository, path, "", request.WorkspaceRef)
	if err != nil {
		return err
	}
	if recorded && (record.path != path || record.gitFile != gitDir) {
		return &Error{Code: CodeWorkspaceUnsafe}
	}
	dirty, err := adapter.workspaceDirty(ctx, path)
	if err != nil {
		return err
	}
	if dirty {
		return &Error{Code: CodeWorkspaceDirty}
	}
	unlockErr := adapter.unlockWorktree(ctx, repository, path)
	if _, err := adapter.gitRun(ctx, repository, nil, "worktree", "remove", path); err != nil {
		if unlockErr != nil {
			return unlockErr
		}
		return err
	}
	return nil
}

func (adapter *Adapter) unlockWorktree(ctx context.Context, repository, path string) error {
	_, err := adapter.gitRun(ctx, repository, nil, "worktree", "unlock", path)
	return err
}

func (adapter *Adapter) verifyReleasedWorkspace(
	ctx context.Context,
	repository, path string,
	ref ports.ExecutionWorkspaceRef,
) error {
	if _, found, err := adapter.gitRefOID(ctx, repository, workspaceBaseRef(ref)); err != nil || !found {
		if err != nil {
			return err
		}
		return &Error{Code: CodeWorkspaceNotFound}
	}
	if _, found, err := adapter.gitRefOID(ctx, repository, "refs/heads/"+workspaceBranch(ref)); err != nil || !found {
		if err != nil {
			return err
		}
		return &Error{Code: CodeWorkspaceNotFound}
	}
	registered, err := adapter.worktreeRegistered(ctx, repository, path)
	if err != nil {
		return err
	}
	if registered {
		return &Error{Code: CodeWorkspaceUnsafe}
	}
	return nil
}

func (adapter *Adapter) worktreeRegistered(ctx context.Context, repository, path string) (bool, error) {
	output, err := adapter.gitRun(ctx, repository, nil, "worktree", "list", "--porcelain")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "worktree ") && filepath.Clean(strings.TrimPrefix(line, "worktree ")) == path {
			return true, nil
		}
	}
	return false, nil
}
