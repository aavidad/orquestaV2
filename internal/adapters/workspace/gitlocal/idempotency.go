package gitlocal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
