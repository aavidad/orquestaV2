package gitlocal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"orquesta/internal/ports"
)

func (adapter *Adapter) PreviewIntegration(ctx context.Context, request ports.IntegrationPreviewRequest) (ports.IntegrationPreview, error) {
	if adapter == nil {
		return ports.IntegrationPreview{}, &Error{Code: CodeUnavailable}
	}
	if err := ports.ValidateIntegrationPreviewRequest(request); err != nil {
		return ports.IntegrationPreview{}, err
	}
	repository, _, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return ports.IntegrationPreview{}, err
	}
	current, err := adapter.gitOID(ctx, repository, request.TargetRef)
	if err != nil {
		return ports.IntegrationPreview{}, err
	}
	if current != request.TargetOID {
		return adapter.previewResult(request, ports.MergeStatusStale, "", []byte("stale target"))
	}
	output, err := adapter.gitRun(ctx, repository, nil, "merge-tree", "--write-tree", request.TargetOID, request.SourceOID)
	if err != nil {
		return adapter.previewResult(request, ports.MergeStatusConflicted, "", []byte("merge conflict"))
	}
	return adapter.previewResult(request, ports.MergeStatusClean, trimOID(output), nil)
}

func (adapter *Adapter) Integrate(ctx context.Context, request ports.IntegrationRequest) (ports.IntegrationResult, error) {
	if adapter == nil {
		return ports.IntegrationResult{}, &Error{Code: CodeUnavailable}
	}
	if err := ports.ValidateIntegrationRequest(request); err != nil {
		return ports.IntegrationResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ports.IntegrationResult{}, err
	}
	unlock := adapter.lockScopes("effect:" + request.IdempotencyKey)
	defer unlock()
	repository, _, err := adapter.repository(ctx, request.RepositoryRef)
	if err != nil {
		return ports.IntegrationResult{}, err
	}
	marker := markerRef(request.IdempotencyKey)
	if result, found, err := adapter.integrationReplay(ctx, repository, marker, request); err != nil || found {
		return result, err
	}
	return adapter.integrateUnseen(ctx, repository, marker, request)
}

func (adapter *Adapter) integrateUnseen(
	ctx context.Context,
	repository, marker string,
	request ports.IntegrationRequest,
) (ports.IntegrationResult, error) {
	current, err := adapter.gitOID(ctx, repository, request.TargetRef)
	if err != nil {
		return ports.IntegrationResult{}, err
	}
	if current != request.ExpectedTargetOID {
		return adapter.integrationResult(request, ports.IntegrationStatusStale, request.ExpectedTargetOID, "", "stale target", "")
	}
	preview, err := adapter.PreviewIntegration(ctx, integrationPreviewRequest(request))
	if err != nil {
		return ports.IntegrationResult{}, err
	}
	if preview.Status != ports.MergeStatusClean {
		return adapter.integrationResult(request, ports.IntegrationStatusConflicted,
			request.ExpectedTargetOID, "", "merge conflict", "")
	}
	commitRaw, err := adapter.gitCommitTree(ctx, repository, preview.CandidateTreeOID,
		request.ExpectedTargetOID, request.SourceOID, request.RequestedAt, integrationCommitMessage(request))
	if err != nil {
		return ports.IntegrationResult{}, err
	}
	after := trimOID(commitRaw)
	transaction := fmt.Sprintf("start\nupdate %s %s %s\ncreate %s %s\nprepare\ncommit\n",
		request.TargetRef, after, request.ExpectedTargetOID, marker, after)
	adapter.observeIntegrationCAS(integrationCASReady, request)
	if _, err := adapter.gitRun(ctx, repository, []byte(transaction), "update-ref", "--stdin"); err != nil {
		return adapter.reconcileIntegrationCAS(ctx, repository, marker, request)
	}
	return adapter.integrationResult(request, ports.IntegrationStatusIntegrated, after, preview.CandidateTreeOID, "", marker)
}

func integrationPreviewRequest(request ports.IntegrationRequest) ports.IntegrationPreviewRequest {
	return ports.IntegrationPreviewRequest{
		ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetOID: request.ExpectedTargetOID,
		ObjectFormat: request.ObjectFormat, IdempotencyKey: request.IdempotencyKey, RequestedAt: request.RequestedAt,
	}
}

func (adapter *Adapter) reconcileIntegrationCAS(
	ctx context.Context,
	repository, marker string,
	request ports.IntegrationRequest,
) (ports.IntegrationResult, error) {
	adapter.observeIntegrationCAS(integrationCASReconcile, request)
	if replay, found, err := adapter.integrationReplay(ctx, repository, marker, request); err != nil || found {
		return replay, err
	}
	return adapter.integrationResult(request, ports.IntegrationStatusStale,
		request.ExpectedTargetOID, "", "target changed", "")
}

func (adapter *Adapter) observeIntegrationCAS(stage integrationCASStage, request ports.IntegrationRequest) {
	if adapter.integrationCASObserver != nil {
		adapter.integrationCASObserver(stage, request)
	}
}

func (adapter *Adapter) integrationReplay(
	ctx context.Context,
	repository, marker string,
	request ports.IntegrationRequest,
) (ports.IntegrationResult, bool, error) {
	previous, found, err := adapter.gitRefOID(ctx, repository, marker)
	if err != nil || !found {
		return ports.IntegrationResult{}, found, err
	}
	message, err := adapter.gitRun(ctx, repository, nil, "show", "-s", "--format=%B", previous)
	if err != nil {
		return ports.IntegrationResult{}, false, err
	}
	if !bytes.Equal(bytes.TrimSpace(message), bytes.TrimSpace(integrationCommitMessage(request))) {
		return ports.IntegrationResult{}, false, &Error{Code: CodeChangeConflict}
	}
	tree, err := adapter.gitRun(ctx, repository, nil, "show", "-s", "--format=%T", previous)
	if err != nil {
		return ports.IntegrationResult{}, false, err
	}
	result, err := adapter.integrationResult(
		request, ports.IntegrationStatusIntegrated, previous, trimOID(tree), "", marker,
	)
	return result, true, err
}

func integrationCommitMessage(request ports.IntegrationRequest) []byte {
	digest := sha256.New()
	for _, field := range []string{
		"orquesta.gitlocal.integration.v1", request.ChangeSetRef.String(), request.RepositoryRef.String(),
		request.PrincipalRef.String(), request.ProjectRef.String(), request.SourceOID, request.TargetRef,
		request.ExpectedTargetOID, string(request.ObjectFormat), request.IntentRef, request.IdempotencyKey,
		request.RequestedAt.UTC().Format(time.RFC3339Nano),
	} {
		_, _ = digest.Write([]byte(field))
		_, _ = digest.Write([]byte{0})
	}
	return []byte("orquesta integration " + request.ChangeSetRef.String() +
		"\n\nrequest-digest " + hex.EncodeToString(digest.Sum(nil)) + "\n")
}

func (adapter *Adapter) previewResult(request ports.IntegrationPreviewRequest, status ports.MergeStatus, tree string, conflict []byte) (ports.IntegrationPreview, error) {
	conflictDigest := ""
	if status != ports.MergeStatusClean {
		digest := sha256.Sum256(conflict)
		conflictDigest = hex.EncodeToString(digest[:])
	}
	result := ports.IntegrationPreview{ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetOID: request.TargetOID,
		ObjectFormat: request.ObjectFormat, Status: status, CandidateTreeOID: tree, ConflictDigest: conflictDigest,
		AdapterRef: adapterRef, ObservedAt: adapter.now()}
	if err := ports.ValidateIntegrationPreview(request, result); err != nil {
		return ports.IntegrationPreview{}, err
	}
	return result, nil
}

func (adapter *Adapter) integrationResult(request ports.IntegrationRequest, status ports.IntegrationStatus, after, tree, conflict, marker string) (ports.IntegrationResult, error) {
	conflictDigest := ""
	if status != ports.IntegrationStatusIntegrated {
		digest := sha256.Sum256([]byte(conflict))
		conflictDigest = hex.EncodeToString(digest[:])
	}
	result := ports.IntegrationResult{ChangeSetRef: request.ChangeSetRef, RepositoryRef: request.RepositoryRef,
		SourceOID: request.SourceOID, TargetRef: request.TargetRef, TargetBeforeOID: request.ExpectedTargetOID,
		TargetAfterOID: after, TreeOID: tree, ObjectFormat: request.ObjectFormat, Status: status,
		MarkerRef: marker, ConflictDigest: conflictDigest, AdapterRef: adapterRef,
		ReceiptRef: digestRef("integration-receipt:", request.IdempotencyKey), RecordedAt: request.RequestedAt}
	if err := ports.ValidateIntegrationResult(request, result); err != nil {
		return ports.IntegrationResult{}, err
	}
	return result, nil
}
