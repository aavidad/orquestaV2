package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
)

func TestCodexStackV0DomainWorkSubmitRegistraClaimAntesDeEfecto(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	domainWork := &claimCheckingDomainWorkExecutorV0{ledger: ledger, t: t}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	stack.DomainDelivery.Ledger = ledger
	director := postDirectorAPIV0(t, stack)
	postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))

	for attempt := 1; attempt <= 6; attempt++ {
		if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-domain-work-claim-drain",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     2,
		}); err != nil {
			t.Fatalf("DrainRunV0 intento %d: %v", attempt, err)
		}
		if domainWork.submits > 0 {
			break
		}
	}
	if domainWork.submits != 1 {
		t.Fatalf("submits=%d", domainWork.submits)
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{Status: DomainWorkArtifactSubmissionStatusAcceptedV0},
	)
	if err != nil || len(records) != 1 || records[0].ReceiptRef == "" {
		t.Fatalf("records=%+v err=%v", records, err)
	}
	if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-domain-work-claim-replay",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     2,
	}); err != nil {
		t.Fatalf("DrainRunV0 replay: %v", err)
	}
	if domainWork.submits != 1 {
		t.Fatalf("replay submit duplicado=%d", domainWork.submits)
	}
}

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
		return orquestamcp.MCPDomainWorkToolResultV0{
			Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
			RequestID:     input.RequestID,
			CorrelationID: input.CorrelationID,
			Action:        input.Action,
		}, nil
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
	return orquestamcp.MCPDomainWorkToolResultV0{
		Estado:        orquestamcp.MCPDomainWorkEstadoOKV0,
		RequestID:     input.RequestID,
		CorrelationID: input.CorrelationID,
		Action:        input.Action,
		Receipt: &orquestadomainwork.DomainWorkArtifactReceiptV0{
			SchemaVersion: orquestadomainwork.DomainWorkArtifactReceiptSchemaV0,
			Status:        orquestadomainwork.DomainWorkStatusAcceptedV0,
			JobRef:        input.ArtifactSubmission.JobRef,
			ArtifactRef:   input.ArtifactSubmission.ArtifactRef,
			ReceiptRef:    "receipt-ref-claim-" + input.ArtifactSubmission.ArtifactRef,
			EvidenceRefs:  []string{"receipt-evidence-ref-claim"},
		},
	}, nil
}
