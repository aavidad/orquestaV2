package gitlocal

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"orquesta/internal/identity"
	"orquesta/internal/ports"
)

func (adapter *Adapter) Prepare(ctx context.Context, request ports.WorkspacePrepareRequest) (ports.WorkspacePrepared, error) {
	if err := adapter.ensureAvailable(); err != nil {
		return ports.WorkspacePrepared{}, &Error{Code: CodeUnavailable}
	}
	if err := ports.ValidateWorkspacePrepareRequest(request); err != nil {
		return ports.WorkspacePrepared{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.WorkspacePrepared{}, err
	}
	unlock := adapter.lockScopes("effect:"+request.IdempotencyKey, "workspace:"+request.WorkspaceRef.String())
	defer unlock()
	if err := adapter.prepareStateConflict(request); err != nil {
		return ports.WorkspacePrepared{}, err
	}
	repository, binding, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return ports.WorkspacePrepared{}, err
	}
	message := effectMarkerMessage("prepare", prepareRequestDigest(request))
	claimed, err := adapter.effectMarkerMatches(ctx, repository, markerRef(request.IdempotencyKey),
		message, CodeWorkspaceConflict)
	if err != nil {
		return ports.WorkspacePrepared{}, err
	}
	target, base, format, err := adapter.prepareTarget(ctx, repository, firstNonEmpty(request.TargetRef, binding.TargetRef))
	if err != nil {
		return ports.WorkspacePrepared{}, err
	}
	// Pin the base before publishing the effect claim. Therefore a replayable
	// claim can never observe a later target tip after a crash.
	base, err = adapter.ensureWorkspaceBaseRef(ctx, repository, request.WorkspaceRef, base, format)
	if err != nil {
		return ports.WorkspacePrepared{}, err
	}
	if !claimed {
		if err := adapter.ensureEffectClaim(ctx, repository, format, request.IdempotencyKey, message,
			request.PreparedAt, CodeWorkspaceConflict); err != nil {
			return ports.WorkspacePrepared{}, err
		}
	}
	return adapter.prepareWorkspace(ctx, repository, target, base, format, request)
}

func (adapter *Adapter) prepareWorkspace(
	ctx context.Context,
	repository, target, targetBase string,
	format ports.GitObjectFormat,
	request ports.WorkspacePrepareRequest,
) (ports.WorkspacePrepared, error) {
	path := adapter.workspacePath(request.WorkspaceRef)
	if err := ensureSafeNewWorkspacePath(adapter.root, path); err != nil {
		return ports.WorkspacePrepared{}, err
	}
	base, gitDir, err := adapter.ensureWorkspace(ctx, repository, path, targetBase, format, request.WorkspaceRef)
	if err != nil {
		return ports.WorkspacePrepared{}, err
	}
	result := ports.WorkspacePrepared{
		WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef, ExecutionRef: request.ExecutionRef,
		TargetRef: target, BaseOID: base, ObjectFormat: format,
		WriteSetDigest: request.WriteSetDigest, AdapterRef: adapterRef,
		ReceiptRef: digestRef("workspace-receipt:", request.IdempotencyKey), PreparedAt: request.PreparedAt,
	}
	if err := ports.ValidateWorkspacePrepared(request, result); err != nil {
		return ports.WorkspacePrepared{}, err
	}
	if err := adapter.ensureWorkspaceBindingMarker(ctx, repository, request, result); err != nil {
		return ports.WorkspacePrepared{}, err
	}
	adapter.rememberPrepared(preparedRecord{request: request, result: result, path: path, gitFile: gitDir})
	return result, nil
}

func (adapter *Adapter) ensureWorkspace(
	ctx context.Context,
	repository, path, targetBase string,
	format ports.GitObjectFormat,
	ref ports.ExecutionWorkspaceRef,
) (string, string, error) {
	if _, err := os.Lstat(path); err == nil {
		gitDir, verifyErr := adapter.verifyWorkspace(ctx, repository, path, "", ref)
		if verifyErr != nil {
			return "", "", verifyErr
		}
		base, reconcileErr := adapter.reconcileWorkspaceBase(ctx, repository, path, ref, format)
		if reconcileErr != nil {
			return "", "", reconcileErr
		}
		return base, gitDir, nil
	} else if !os.IsNotExist(err) {
		return "", "", &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	base, err := adapter.ensureWorkspaceBaseRef(ctx, repository, ref, targetBase, format)
	if err != nil {
		return "", "", err
	}
	branchRef := "refs/heads/" + workspaceBranch(ref)
	branchOID, branchFound, err := adapter.gitRefOID(ctx, repository, branchRef)
	if err != nil {
		return "", "", err
	}
	arguments := []string{"worktree", "add", "--lock"}
	if branchFound {
		if branchOID != base {
			return "", "", &Error{Code: CodeWorkspaceConflict}
		}
		arguments = append(arguments, path, workspaceBranch(ref))
	} else {
		arguments = append(arguments, "-b", workspaceBranch(ref), path, base)
	}
	if _, err := adapter.gitRun(ctx, repository, nil, arguments...); err != nil {
		return "", "", err
	}
	if err := hardenOwnedDirectory(path); err != nil {
		return "", "", err
	}
	if err := adapter.hardenCreatedWorkspaceGitMetadata(ctx, repository, path); err != nil {
		return "", "", err
	}
	gitDir, err := adapter.verifyWorkspace(ctx, repository, path, base, ref)
	return base, gitDir, err
}

func (adapter *Adapter) ensureWorkspaceBaseRef(
	ctx context.Context,
	repository string,
	ref ports.ExecutionWorkspaceRef,
	targetBase string,
	format ports.GitObjectFormat,
) (string, error) {
	if existing, found, err := adapter.gitRefOID(ctx, repository, workspaceBaseRef(ref)); err != nil {
		return "", err
	} else if found {
		return existing, nil
	}
	if err := adapter.createWorkspaceBaseRef(ctx, repository, ref, targetBase, format); err != nil {
		return "", err
	}
	return targetBase, nil
}

func (adapter *Adapter) workspaceBaseOID(ctx context.Context, repository string, ref ports.ExecutionWorkspaceRef) (string, error) {
	base, err := adapter.gitOID(ctx, repository, workspaceBaseRef(ref))
	if err != nil {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	return base, nil
}

func (adapter *Adapter) reconcileWorkspaceBase(ctx context.Context, repository, path string, ref ports.ExecutionWorkspaceRef, format ports.GitObjectFormat) (string, error) {
	if base, found, err := adapter.gitRefOID(ctx, repository, workspaceBaseRef(ref)); err != nil {
		return "", err
	} else if found {
		return base, nil
	}
	// A clean exact branch/worktree is the durable evidence left by
	// `worktree add` when the process crashed before creating the base marker.
	// It is adopted in place; no ref is reset and no worktree is deleted.
	dirty, err := adapter.workspaceDirty(ctx, path)
	if err != nil {
		return "", err
	}
	if dirty {
		return "", &Error{Code: CodeWorkspaceUnsafe}
	}
	base, err := adapter.gitOID(ctx, path, "HEAD")
	if err != nil {
		return "", err
	}
	if err := adapter.createWorkspaceBaseRef(ctx, repository, ref, base, format); err != nil {
		return "", err
	}
	return base, nil
}

func (adapter *Adapter) createWorkspaceBaseRef(ctx context.Context, repository string, ref ports.ExecutionWorkspaceRef, base string, format ports.GitObjectFormat) error {
	zeroes := 40
	if format == ports.GitObjectFormatSHA256 {
		zeroes = 64
	}
	if _, err := adapter.gitRun(ctx, repository, nil, "update-ref", workspaceBaseRef(ref), base, strings.Repeat("0", zeroes)); err == nil {
		return nil
	}
	existing, found, lookupErr := adapter.gitRefOID(ctx, repository, workspaceBaseRef(ref))
	if lookupErr != nil {
		return lookupErr
	}
	if !found || existing != base {
		return &Error{Code: CodeWorkspaceConflict}
	}
	return nil
}

func (adapter *Adapter) Inspect(ctx context.Context, request ports.WorkspaceInspectRequest) (ports.WorkspaceInspection, error) {
	if err := adapter.ensureAvailable(); err != nil {
		return ports.WorkspaceInspection{}, &Error{Code: CodeUnavailable}
	}
	record, ok := adapter.preparedRecord(request.WorkspaceRef)
	if !ok || record.request.RepositoryRef != request.RepositoryRef || record.request.ExecutionRef != request.ExecutionRef {
		return ports.WorkspaceInspection{}, &Error{Code: CodeWorkspaceNotFound}
	}
	repository, _, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return ports.WorkspaceInspection{}, err
	}
	gitDir, err := adapter.verifyWorkspace(ctx, repository, record.path, "", request.WorkspaceRef)
	if err != nil || gitDir != record.gitFile {
		return ports.WorkspaceInspection{}, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	head, err := adapter.gitOID(ctx, record.path, "HEAD")
	if err != nil {
		return ports.WorkspaceInspection{}, err
	}
	tree, err := adapter.gitRun(ctx, record.path, nil, "show", "-s", "--format=%T", "HEAD")
	if err != nil {
		return ports.WorkspaceInspection{}, err
	}
	dirty, err := adapter.workspaceDirty(ctx, record.path)
	if err != nil {
		return ports.WorkspaceInspection{}, err
	}
	result := ports.WorkspaceInspection{WorkspaceRef: request.WorkspaceRef, RepositoryRef: request.RepositoryRef,
		ExecutionRef: request.ExecutionRef, BaseOID: record.result.BaseOID, HeadOID: head, TreeOID: trimOID(tree),
		WriteSetDigest: record.request.WriteSetDigest, Dirty: dirty, AdapterRef: adapterRef, InspectedAt: adapter.now()}
	if err := ports.ValidateWorkspaceInspection(request, result); err != nil {
		return ports.WorkspaceInspection{}, err
	}
	return result, nil
}

// ResolveExecutionWorkspace is a composition-only escape hatch for a local
// process adapter (such as Codex). The physical path never crosses the
// application ports, state snapshot, receipt or public API.
func (adapter *Adapter) ResolveExecutionWorkspace(ctx context.Context, ref ports.ExecutionWorkspaceRef) (string, error) {
	if err := adapter.ensureAvailable(); err != nil {
		return "", &Error{Code: CodeUnavailable}
	}
	if ref.String() == "" {
		return "", &Error{Code: CodeWorkspaceNotFound}
	}
	record, prepared := adapter.preparedRecord(ref)
	var path, expectedGitFile string
	var repositoryRef identity.RepositoryRef
	if prepared {
		path, expectedGitFile, repositoryRef = record.path, record.gitFile, record.request.RepositoryRef
	} else if resolved, found := adapter.resolvedWorkspace(ref); found {
		path, expectedGitFile, repositoryRef = resolved.path, resolved.gitFile, resolved.repositoryRef
	} else {
		resolved, err := adapter.recoverWorkspaceResolution(ctx, ref)
		if err != nil {
			return "", err
		}
		path, expectedGitFile, repositoryRef = resolved.path, resolved.gitFile, resolved.repositoryRef
	}
	if _, err := os.Lstat(path); os.IsNotExist(err) {
		return "", &Error{Code: CodeWorkspaceNotFound}
	}
	repository, _, err := adapter.repository(ctx, repositoryRef)
	if err != nil {
		return "", err
	}
	gitDir, err := adapter.verifyWorkspace(ctx, repository, path, "", ref)
	if err != nil || gitDir != expectedGitFile {
		return "", &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	return path, nil
}

func (adapter *Adapter) recoverWorkspaceResolution(
	ctx context.Context,
	ref ports.ExecutionWorkspaceRef,
) (resolvedWorkspaceRecord, error) {
	unlock := adapter.lockScopes("workspace:" + ref.String())
	defer unlock()
	if record, found := adapter.preparedRecord(ref); found {
		return resolvedWorkspaceRecord{
			repositoryRef: record.request.RepositoryRef, path: record.path, gitFile: record.gitFile,
		}, nil
	}
	if record, found := adapter.resolvedWorkspace(ref); found {
		return record, nil
	}
	adapter.stateMu.RLock()
	resolver := adapter.bindingResolver
	adapter.stateMu.RUnlock()
	if resolver == nil {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceNotFound}
	}
	recovered, found, err := resolver.ResolveDurableWorkspaceBinding(ctx, ref)
	if err != nil {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	if !found {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceNotFound}
	}
	request, prepared := recovered.Request, recovered.Prepared
	if request.WorkspaceRef != ref || prepared.WorkspaceRef != ref ||
		prepared.AdapterRef != adapterRef ||
		prepared.ReceiptRef != digestRef("workspace-receipt:", request.IdempotencyKey) ||
		ports.ValidateWorkspacePrepareRequest(request) != nil ||
		ports.ValidateWorkspacePrepared(request, prepared) != nil {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceUnsafe}
	}
	repository, _, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return resolvedWorkspaceRecord{}, err
	}
	path := adapter.workspacePath(ref)
	gitDir, err := adapter.verifyWorkspace(ctx, repository, path, "", ref)
	if err != nil {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	base, err := adapter.workspaceBaseOID(ctx, repository, ref)
	if err != nil || base != prepared.BaseOID {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	prepareMarkerFound, err := adapter.effectMarkerMatches(
		ctx, repository, markerRef(request.IdempotencyKey),
		effectMarkerMessage("prepare", prepareRequestDigest(request)), CodeWorkspaceUnsafe,
	)
	if err != nil || !prepareMarkerFound {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	digest := workspaceBindingDigest(request, prepared)
	key := "workspace-binding:" + ref.String() + ":" + digest
	markerFound, err := adapter.effectMarkerMatches(
		ctx, repository, markerRef(key), workspaceBindingMarkerMessage(ref, digest), CodeWorkspaceUnsafe,
	)
	if err != nil || !markerFound {
		return resolvedWorkspaceRecord{}, &Error{Code: CodeWorkspaceUnsafe, Cause: err}
	}
	record := resolvedWorkspaceRecord{repositoryRef: request.RepositoryRef, path: path, gitFile: gitDir}
	adapter.rememberResolvedWorkspace(ref, record)
	return record, nil
}

func (adapter *Adapter) repository(ctx context.Context, ref identity.RepositoryRef) (string, LocalRepositoryBinding, error) {
	if ref.String() == "" {
		return "", LocalRepositoryBinding{}, &Error{Code: CodeRepositoryInvalid}
	}
	binding, err := adapter.loc.LocateLocalRepository(ctx, ref)
	path := binding.Path
	if err != nil || path == "" || !filepath.IsAbs(path) {
		return "", LocalRepositoryBinding{}, &Error{Code: CodeRepositoryInvalid, Cause: err}
	}
	path = filepath.Clean(path)
	if pathsOverlap(adapter.root, path) {
		return "", LocalRepositoryBinding{}, &Error{Code: CodeRepositoryInvalid, Cause: errUnsafeWorkspace}
	}
	if err := checkRepositoryAncestry(path); err != nil {
		return "", LocalRepositoryBinding{}, &Error{Code: CodeRepositoryInvalid, Cause: err}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", LocalRepositoryBinding{}, &Error{Code: CodeRepositoryInvalid, Cause: err}
	}
	root, err := adapter.gitRun(ctx, path, nil, "rev-parse", "--show-toplevel")
	if err != nil || filepath.Clean(trimOID(root)) != path {
		return "", LocalRepositoryBinding{}, &Error{Code: CodeRepositoryInvalid, Cause: err}
	}
	if err := adapter.validateGitMetadataRoot(ctx, path); err != nil {
		return "", LocalRepositoryBinding{}, err
	}
	if err := adapter.validateRepositoryControls(ctx, path); err != nil {
		return "", LocalRepositoryBinding{}, err
	}
	return path, binding, nil
}

func (adapter *Adapter) validateRepositoryControls(ctx context.Context, repository string) error {
	output, err := adapter.gitRun(ctx, repository, nil, "config", "--local", "--no-includes", "--null", "--list")
	if err != nil {
		return err
	}
	for _, record := range strings.Split(string(output), "\x00") {
		key := record
		if index := strings.IndexAny(key, "\n="); index >= 0 {
			key = key[:index]
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "core.hookspath" || key == "core.fsmonitor" || key == "core.sshcommand" ||
			key == "core.editor" || key == "core.attributesfile" || key == "diff.external" ||
			key == "core.alternaterefscommand" || key == "extensions.worktreeconfig" ||
			key == "interactive.difffilter" || key == "sequence.editor" || key == "gpg.program" ||
			key == "commit.gpgsign" || key == "tag.gpgsign" || key == "user.signingkey" ||
			key == "credential.helper" || strings.HasPrefix(key, "filter.") || strings.HasPrefix(key, "include.") ||
			strings.HasPrefix(key, "includeif.") ||
			strings.HasPrefix(key, "pager.") || (strings.HasPrefix(key, "diff.") &&
			(strings.HasSuffix(key, ".command") || strings.HasSuffix(key, ".textconv"))) ||
			(strings.HasPrefix(key, "merge.") && strings.HasSuffix(key, ".driver")) ||
			(strings.HasPrefix(key, "gpg.") && strings.HasSuffix(key, ".program")) {
			return &Error{Code: CodeWorkspaceUnsafe}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (adapter *Adapter) prepareTarget(ctx context.Context, repository, target string) (string, string, ports.GitObjectFormat, error) {
	if !strings.HasPrefix(target, "refs/heads/") {
		return "", "", "", &Error{Code: CodeRepositoryInvalid}
	}
	if _, err := adapter.gitRun(ctx, repository, nil, "check-ref-format", target); err != nil {
		return "", "", "", &Error{Code: CodeRepositoryInvalid, Cause: err}
	}
	base, err := adapter.gitOID(ctx, repository, target)
	if err != nil {
		return "", "", "", err
	}
	format, err := adapter.repositoryObjectFormat(ctx, repository)
	return target, base, format, err
}

func (adapter *Adapter) repositoryObjectFormat(ctx context.Context, repository string) (ports.GitObjectFormat, error) {
	formatRaw, err := adapter.gitRun(ctx, repository, nil, "rev-parse", "--show-object-format")
	if err != nil {
		return "", err
	}
	format := ports.GitObjectFormat(trimOID(formatRaw))
	if format != ports.GitObjectFormatSHA1 && format != ports.GitObjectFormatSHA256 {
		return "", &Error{Code: CodeRepositoryInvalid}
	}
	return format, nil
}

func ensureSafeNewWorkspacePath(root, path string) error {
	if filepath.Dir(path) != filepath.Join(root, "workspaces") || !strings.HasPrefix(path, filepath.Join(root, "workspaces")+string(filepath.Separator)) {
		return &Error{Code: CodeWorkspaceUnsafe}
	}
	if err := checkPrivateAncestry(root); err != nil {
		return err
	}
	return checkPrivateDirectory(filepath.Join(root, "workspaces"))
}

func (adapter *Adapter) workspaceDirty(ctx context.Context, path string) (bool, error) {
	// Machine format is mandatory; its result is intentionally not parsed for
	// commits (diff/ls-files below carry the authoritative path inventory).
	status, err := adapter.gitRun(ctx, path, nil, "status", "--porcelain=v2", "-z")
	if err != nil {
		return false, err
	}
	return len(status) != 0, nil
}
