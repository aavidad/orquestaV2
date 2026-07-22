package gitlocal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"io"
	"strconv"
	"strings"
	"time"

	"orquesta/internal/ports"
)

// AttemptRef and ActionFence are retry-envelope metadata, not stable semantic
// payload. A legitimate retry may advance both while retaining one idempotency
// key; the digests below bind only the semantic request that key protects.
func prepareRequestDigest(request ports.WorkspacePrepareRequest) string {
	fields := []string{
		"orquesta.gitlocal.prepare.v1", request.WorkspaceRef.String(), request.PrincipalRef.String(),
		request.ActorRef.String(), request.ProjectRef.String(), request.RepositoryRef.String(), request.GoalRef.String(),
		request.WorkItemRef.String(), request.ExecutionRef.String(), strconv.FormatUint(request.ExecutionAttempt, 10),
		strconv.FormatUint(uint64(request.PlanGeneration), 10), strconv.FormatUint(uint64(request.AppSpecGeneration), 10),
		request.AppSpecHash, strconv.Itoa(len(request.WriteSet)),
	}
	fields = append(fields, request.WriteSet...)
	fields = append(fields, request.WriteSetDigest, request.TargetRef, request.IntentRef,
		request.IdempotencyKey, request.PreparedAt.UTC().Format(time.RFC3339Nano))
	return digestFields(fields)
}

func commitRequestDigest(request ports.CommitRequest) string {
	fields := []string{
		"orquesta.gitlocal.commit.v1", request.ChangeSetRef.String(), request.WorkspaceRef.String(),
		request.PrincipalRef.String(), request.ActorRef.String(), request.ProjectRef.String(), request.RepositoryRef.String(),
		request.GoalRef.String(), request.WorkItemRef.String(), request.ExecutionRef.String(),
		strconv.FormatUint(request.ExecutionAttempt, 10), strconv.FormatUint(uint64(request.PlanGeneration), 10),
		strconv.FormatUint(uint64(request.AppSpecGeneration), 10), request.AppSpecHash, request.BaseOID,
		string(request.ObjectFormat), strconv.Itoa(len(request.WriteSet)),
	}
	fields = append(fields, request.WriteSet...)
	fields = append(fields, request.WriteSetDigest, request.ParentChangeRef.String(), request.IntentRef,
		request.IdempotencyKey, request.CommittedAt.UTC().Format(time.RFC3339Nano))
	return digestFields(fields)
}

func releaseRequestDigest(request ports.WorkspaceReleaseRequest) string {
	return digestFields([]string{
		"orquesta.gitlocal.release.v1", request.WorkspaceRef.String(), request.RepositoryRef.String(),
		request.ExecutionRef.String(), request.IdempotencyKey, request.RequestedAt.UTC().Format(time.RFC3339Nano),
	})
}

func digestFields(fields []string) string {
	digest := sha256.New()
	var size [8]byte
	for _, field := range fields {
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		_, _ = digest.Write(size[:])
		_, _ = digest.Write([]byte(field))
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func effectMarkerMessage(kind, digest string) []byte {
	return []byte("orquesta effect " + kind + "\n\nrequest-digest " + digest + "\n")
}

func workspaceBindingDigest(request ports.WorkspacePrepareRequest, result ports.WorkspacePrepared) string {
	fields := []string{
		result.WorkspaceRef.String(), request.PrincipalRef.String(), request.ActorRef.String(), request.ProjectRef.String(),
		result.RepositoryRef.String(), request.GoalRef.String(), request.WorkItemRef.String(), request.ExecutionRef.String(),
		strconv.FormatUint(request.ExecutionAttempt, 10), strconv.FormatUint(uint64(request.PlanGeneration), 10),
		strconv.FormatUint(uint64(request.AppSpecGeneration), 10), request.AppSpecHash, result.WriteSetDigest,
		result.TargetRef, result.BaseOID, string(result.ObjectFormat), result.AdapterRef, request.IntentRef,
		request.AttemptRef, strconv.FormatUint(request.ActionFence, 10), "effect-receipt:" + request.IntentRef,
		result.PreparedAt.UTC().Format(time.RFC3339Nano),
	}
	digest := sha256.New()
	_, _ = digest.Write([]byte("orquesta.workspace_binding.v1\x00"))
	for _, value := range append(fields, request.WriteSet...) {
		_, _ = digest.Write([]byte(value))
		_, _ = digest.Write([]byte{0})
	}
	return hex.EncodeToString(digest.Sum(nil))
}

func (adapter *Adapter) ensureWorkspaceBindingMarker(
	ctx context.Context,
	repository string,
	request ports.WorkspacePrepareRequest,
	result ports.WorkspacePrepared,
) error {
	digest := workspaceBindingDigest(request, result)
	key := "workspace-binding:" + result.WorkspaceRef.String() + ":" + digest
	return adapter.ensureEffectClaim(ctx, repository, result.ObjectFormat,
		key, workspaceBindingMarkerMessage(result.WorkspaceRef, digest), result.PreparedAt, CodeWorkspaceConflict)
}

func (adapter *Adapter) verifyWorkspaceBindingMarker(
	ctx context.Context,
	repository string,
	subject ports.TestSubject,
) error {
	key := "workspace-binding:" + subject.WorkspaceRef.String() + ":" + subject.WorkspaceBindingDigest
	found, err := adapter.effectMarkerMatches(ctx, repository,
		markerRef(key), workspaceBindingMarkerMessage(subject.WorkspaceRef, subject.WorkspaceBindingDigest), CodeSnapshotInvalid)
	if err != nil {
		return err
	}
	if !found {
		return &Error{Code: CodeSnapshotInvalid}
	}
	return nil
}

func workspaceBindingMarkerMessage(ref ports.ExecutionWorkspaceRef, digest string) []byte {
	return effectMarkerMessage("workspace-binding", digestFields([]string{ref.String(), digest}))
}

func snapshotChangeMarkerRef(ref ports.ChangeSetRef) string {
	return markerRef("snapshot-change:" + ref.String())
}

func snapshotChangeCommitMarkerMessage(request ports.CommitRequest, result ports.CommitResult) []byte {
	return snapshotChangeMarkerMessage([]string{
		request.ChangeSetRef.String(), request.WorkspaceRef.String(), request.RepositoryRef.String(),
		request.GoalRef.String(), request.WorkItemRef.String(), request.ExecutionRef.String(),
		strconv.FormatUint(request.ExecutionAttempt, 10), strconv.FormatUint(uint64(request.PlanGeneration), 10),
		strconv.FormatUint(uint64(request.AppSpecGeneration), 10), request.AppSpecHash,
		result.BaseOID, result.ParentOID, result.HeadOID, result.TreeOID, string(result.ObjectFormat),
		result.DiffDigest, result.WriteSetDigest,
	})
}

func snapshotChangeSubjectMarkerMessage(subject ports.TestSubject) []byte {
	return snapshotChangeMarkerMessage([]string{
		subject.ChangeSetRef.String(), subject.WorkspaceRef.String(), subject.RepositoryRef.String(),
		subject.GoalRef.String(), subject.WorkItemRef.String(), subject.ExecutionRef.String(),
		strconv.FormatUint(subject.ExecutionAttempt, 10), strconv.FormatUint(uint64(subject.PlanGeneration), 10),
		strconv.FormatUint(uint64(subject.AppSpecGeneration), 10), subject.AppSpecHash,
		subject.BaseOID, subject.ParentOID, subject.HeadOID, subject.TreeOID, string(subject.ObjectFormat),
		subject.DiffDigest, subject.WriteSetDigest,
	})
}

func snapshotChangeMarkerMessage(fields []string) []byte {
	values := append([]string{"orquesta.snapshot_change_binding.v1"}, fields...)
	return effectMarkerMessage("snapshot-change", digestFields(values))
}

func (adapter *Adapter) ensureSnapshotChangeMarker(
	ctx context.Context,
	workspace string,
	request ports.CommitRequest,
	result ports.CommitResult,
) error {
	expected := snapshotChangeCommitMarkerMessage(request, result)
	return adapter.ensureEffectClaim(ctx, workspace, request.ObjectFormat,
		"snapshot-change:"+request.ChangeSetRef.String(), expected, request.CommittedAt, CodeChangeConflict)
}

func verifySnapshotChangeMarker(
	ctx context.Context,
	repository *snapshotRepository,
	format ports.GitObjectFormat,
	marker string,
	expected []byte,
	conflict ErrorCode,
) error {
	raw, err := repository.run(ctx, maxSnapshotStreamFieldBytes, "rev-parse", "--verify", marker)
	oid := trimOID(raw)
	if err != nil || ports.ValidateGitOID(oid, format) != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &Error{Code: conflict}
	}
	session, err := openGitObjectSession(ctx, repository, format)
	if err != nil {
		return &Error{Code: conflict}
	}
	var object []byte
	readErr := session.read(oid, "commit", maxSnapshotStreamFieldBytes, func(_ int64, content io.Reader) error {
		object, err = io.ReadAll(content)
		return err
	})
	closeErr := session.Close()
	if readErr != nil || closeErr != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return &Error{Code: conflict}
	}
	_, message, found := bytes.Cut(object, []byte("\n\n"))
	if !found || !bytes.Equal(message, expected) {
		return &Error{Code: conflict}
	}
	return nil
}

func (adapter *Adapter) ensureEffectClaim(
	ctx context.Context,
	repository string,
	format ports.GitObjectFormat,
	key string,
	message []byte,
	at time.Time,
	conflict ErrorCode,
) error {
	if found, err := adapter.effectMarkerMatches(ctx, repository, markerRef(key), message, conflict); err != nil || found {
		return err
	}
	treeRaw, err := adapter.gitRun(ctx, repository, nil, "mktree")
	if err != nil {
		return err
	}
	commitRaw, err := adapter.gitCommitTree(ctx, repository, trimOID(treeRaw), "", "", at, message)
	if err != nil {
		return err
	}
	zeroes := strings.Repeat("0", map[ports.GitObjectFormat]int{
		ports.GitObjectFormatSHA1: 40, ports.GitObjectFormatSHA256: 64,
	}[format])
	_, updateErr := adapter.gitRun(ctx, repository, nil, "update-ref", markerRef(key), trimOID(commitRaw), zeroes)
	if updateErr == nil {
		return nil
	}
	if found, err := adapter.effectMarkerMatches(ctx, repository, markerRef(key), message, conflict); err != nil || found {
		return err
	}
	return updateErr
}

func (adapter *Adapter) effectMarkerMatches(
	ctx context.Context,
	repository, marker string,
	expected []byte,
	conflict ErrorCode,
) (bool, error) {
	oid, found, err := adapter.gitRefOID(ctx, repository, marker)
	if err != nil || !found {
		return found, err
	}
	message, err := adapter.gitRun(ctx, repository, nil, "show", "-s", "--format=%B", oid)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(bytes.TrimSpace(message), bytes.TrimSpace(expected)) {
		return false, &Error{Code: conflict}
	}
	return true, nil
}

func (adapter *Adapter) prepareStateConflict(request ports.WorkspacePrepareRequest) error {
	adapter.stateMu.RLock()
	defer adapter.stateMu.RUnlock()
	if record, found := adapter.prepareKeys[request.IdempotencyKey]; found && !equalPrepare(record.request, request) {
		return &Error{Code: CodeWorkspaceConflict}
	}
	if record, found := adapter.prepared[request.WorkspaceRef]; found && !equalPrepare(record.request, request) {
		return &Error{Code: CodeWorkspaceConflict}
	}
	return nil
}

func (adapter *Adapter) rememberPrepared(record preparedRecord) {
	record.request.WriteSet = append([]string(nil), record.request.WriteSet...)
	adapter.stateMu.Lock()
	adapter.prepared[record.request.WorkspaceRef] = record
	adapter.prepareKeys[record.request.IdempotencyKey] = record
	adapter.stateMu.Unlock()
}

func (adapter *Adapter) preparedRecord(ref ports.ExecutionWorkspaceRef) (preparedRecord, bool) {
	adapter.stateMu.RLock()
	defer adapter.stateMu.RUnlock()
	record, found := adapter.prepared[ref]
	return record, found
}

func (adapter *Adapter) commitStateConflict(request ports.CommitRequest) error {
	adapter.stateMu.RLock()
	defer adapter.stateMu.RUnlock()
	if record, found := adapter.commitKeys[request.IdempotencyKey]; found && !equalCommit(record.request, request) {
		return &Error{Code: CodeChangeConflict}
	}
	if record, found := adapter.commits[request.ChangeSetRef]; found && !equalCommit(record.request, request) {
		return &Error{Code: CodeChangeConflict}
	}
	return nil
}

func (adapter *Adapter) rememberCommit(record commitRecord) {
	record.request.WriteSet = append([]string(nil), record.request.WriteSet...)
	record.result.ChangedPaths = append([]string(nil), record.result.ChangedPaths...)
	adapter.stateMu.Lock()
	adapter.commits[record.request.ChangeSetRef] = record
	adapter.commitKeys[record.request.IdempotencyKey] = record
	adapter.stateMu.Unlock()
}

func (adapter *Adapter) releaseStateConflict(request ports.WorkspaceReleaseRequest) error {
	adapter.stateMu.RLock()
	defer adapter.stateMu.RUnlock()
	if record, found := adapter.releases[request.IdempotencyKey]; found &&
		releaseRequestDigest(record.request) != releaseRequestDigest(request) {
		return &Error{Code: CodeWorkspaceConflict}
	}
	return nil
}

func (adapter *Adapter) rememberRelease(record releaseRecord) {
	adapter.stateMu.Lock()
	adapter.releases[record.request.IdempotencyKey] = record
	adapter.stateMu.Unlock()
}
