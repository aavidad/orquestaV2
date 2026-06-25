package orquestaappcodexstack

import (
	"context"
	"errors"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

type claimCheckingDomainWorkExecutorV0 struct {
	ledger  DomainWorkArtifactSubmissionRecordReaderPortV0
	t       *testing.T
	submits int
}

func (executor *claimCheckingDomainWorkExecutorV0) Execute(
	ctx context.Context,
	input orquestamcp.MCPDomainWorkToolInputV0,
) (orquestamcp.MCPDomainWorkToolResultV0, error) {
	if input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 {
		return domainWorkToolOKForClaimTestV0(input), nil
	}
	executor.submits++
	records, err := executor.ledger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{
			IdempotencyKey: input.ArtifactSubmission.IdempotencyKey,
			Status:         DomainWorkArtifactSubmissionStatusSubmittingV0,
		},
	)
	if err != nil || len(records) != 1 {
		executor.t.Fatalf("claim/submitting no registrado: records=%+v err=%v", records, err)
	}
	return domainWorkAcceptedToolResultForClaimTestV0(input, "receipt-ref-claim-", "receipt-evidence-ref-claim"), nil
}

type failOnceDomainWorkExecutorV0 struct {
	submits int
}

func (executor *failOnceDomainWorkExecutorV0) Execute(
	_ context.Context,
	input orquestamcp.MCPDomainWorkToolInputV0,
) (orquestamcp.MCPDomainWorkToolResultV0, error) {
	if input.Action != orquestamcp.MCPDomainWorkActionSubmitArtifactV0 {
		return domainWorkToolOKForClaimTestV0(input), nil
	}
	executor.submits++
	if executor.submits == 1 {
		return orquestamcp.MCPDomainWorkToolResultV0{}, errors.New("domain_work_http_unavailable")
	}
	return domainWorkAcceptedToolResultForClaimTestV0(input, "receipt-ref-retry-", "receipt-evidence-ref-retry"), nil
}

type failAcceptedDomainWorkSubmissionLedgerV0 struct {
	inner        *InMemoryDomainWorkArtifactSubmissionLedgerV0
	failAccepted bool
}

func (ledger *failAcceptedDomainWorkSubmissionLedgerV0) HasDomainWorkArtifactSubmissionV0(
	ctx context.Context,
	idempotencyKey string,
) (bool, error) {
	return ledger.inner.HasDomainWorkArtifactSubmissionV0(ctx, idempotencyKey)
}

func (ledger *failAcceptedDomainWorkSubmissionLedgerV0) RecordDomainWorkArtifactSubmissionV0(
	ctx context.Context,
	record DomainWorkArtifactSubmissionRecordV0,
) error {
	if ledger.failAccepted && record.Status == DomainWorkArtifactSubmissionStatusAcceptedV0 {
		return errors.New("accepted-ledger-write-failed")
	}
	return ledger.inner.RecordDomainWorkArtifactSubmissionV0(ctx, record)
}

func (ledger *failAcceptedDomainWorkSubmissionLedgerV0) ListDomainWorkArtifactSubmissionsV0(
	ctx context.Context,
	filter DomainWorkArtifactSubmissionRecordFilterV0,
) ([]DomainWorkArtifactSubmissionRecordV0, error) {
	return ledger.inner.ListDomainWorkArtifactSubmissionsV0(ctx, filter)
}

func domainWorkToolOKForClaimTestV0(
	input orquestamcp.MCPDomainWorkToolInputV0,
) orquestamcp.MCPDomainWorkToolResultV0 {
	return orquestamcp.MCPDomainWorkToolResultV0{
		Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        input.Action,
	}
}

func domainWorkAcceptedToolResultForClaimTestV0(
	input orquestamcp.MCPDomainWorkToolInputV0,
	receiptPrefix string,
	evidenceRef string,
) orquestamcp.MCPDomainWorkToolResultV0 {
	result := domainWorkToolOKForClaimTestV0(input)
	result.Receipt = &orquestadomainwork.DomainWorkArtifactReceiptV0{
		SchemaVersion: orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
		Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
		JobRef:        input.ArtifactSubmission.JobRef,
		ArtifactRef:   input.ArtifactSubmission.ArtifactRef,
		ReceiptRef:    receiptPrefix + input.ArtifactSubmission.ArtifactRef,
		EvidenceRefs:  []string{evidenceRef},
	}
	return result
}

func domainWorkSubmissionForRetryTestV0() orquestadomainwork.DomainWorkArtifactSubmissionV0 {
	return orquestadomainwork.DomainWorkArtifactSubmissionV0{
		IdempotencyKey: "idem-domain-work-retry",
		DomainRef:      "opes",
		JobRef:         "job-ref-retry",
		ArtifactRef:    "artifact-ref-retry",
		ArtifactType:   "visual_asset",
	}
}

func domainWorkSubmissionRecordForRetryTestV0(status string) DomainWorkArtifactSubmissionRecordV0 {
	return DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-domain-work-retry",
		Status:         status,
		RunRef:         "run-ref-retry",
		TaskRef:        "task-ref-retry",
		DeliveryRef:    "delivery-ref-retry",
		DomainRef:      "opes",
		JobRef:         "job-ref-retry",
		ArtifactRef:    "artifact-ref-retry",
		ArtifactType:   "visual_asset",
		IssueRefs:      []string{"domain-work-submit-artifact-rejected"},
	}
}
