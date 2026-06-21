package orquestaappcodexstack

import (
	"context"
	"errors"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestamcp "orquesta/modulos/orquesta-mcp"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
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

func TestCodexStackV0DomainWorkSubmitAcceptedYRejectedFuncionalBloqueanReintento(t *testing.T) {
	ctx := context.Background()
	for _, status := range []string{
		DomainWorkArtifactSubmissionStatusClaimedV0,
		DomainWorkArtifactSubmissionStatusSubmittingV0,
	} {
		ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
		record := domainWorkSubmissionRecordForRetryTestV0(status)
		if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
			t.Fatalf("record status=%s: %v", status, err)
		}
		stack := StackV0{DomainDelivery: DomainWorkDeliveryBridgeConfigV0{Ledger: ledger}}

		submitted, err := stack.domainWorkSubmissionAlreadyRecordedV0(
			ctx,
			orquestacoreworkflow.OrchestrationRunV0{RunID: record.RunRef},
			orquestacoreworkflow.WorkflowTaskV0{TaskID: record.TaskRef},
			orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: record.DeliveryRef},
			domainWorkSubmissionForRetryTestV0(),
		)
		if err != nil || submitted {
			t.Fatalf("status=%s submitted=%v err=%v", status, submitted, err)
		}
	}

	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	rejected := domainWorkSubmissionRecordForRetryTestV0(DomainWorkArtifactSubmissionStatusRejectedV0)
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, rejected); err != nil {
		t.Fatalf("record rejected: %v", err)
	}
	stack := StackV0{DomainDelivery: DomainWorkDeliveryBridgeConfigV0{Ledger: ledger}}
	submitted, err := stack.domainWorkSubmissionAlreadyRecordedV0(
		ctx,
		orquestacoreworkflow.OrchestrationRunV0{RunID: rejected.RunRef},
		orquestacoreworkflow.WorkflowTaskV0{TaskID: rejected.TaskRef},
		orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: rejected.DeliveryRef},
		domainWorkSubmissionForRetryTestV0(),
	)
	if err != nil || !submitted {
		t.Fatalf("rejected funcional submitted=%v err=%v", submitted, err)
	}

	ledger = NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	rejectedTransient := domainWorkSubmissionRecordForRetryTestV0(DomainWorkArtifactSubmissionStatusRejectedV0)
	for _, transientIssue := range []string{
		"domain-work-submit-execute-error",
		"domain_work_port_no_disponible",
		"opes_http_timeout",
	} {
		ledger = NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
		rejectedTransient.IssueRefs = []string{"domain-work-submit-artifact-rejected", transientIssue}
		if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, rejectedTransient); err != nil {
			t.Fatalf("record rejected transient %s: %v", transientIssue, err)
		}
		stack = StackV0{DomainDelivery: DomainWorkDeliveryBridgeConfigV0{Ledger: ledger}}
		submitted, err = stack.domainWorkSubmissionAlreadyRecordedV0(
			ctx,
			orquestacoreworkflow.OrchestrationRunV0{RunID: rejectedTransient.RunRef},
			orquestacoreworkflow.WorkflowTaskV0{TaskID: rejectedTransient.TaskRef},
			orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: rejectedTransient.DeliveryRef},
			domainWorkSubmissionForRetryTestV0(),
		)
		if err != nil || submitted {
			t.Fatalf("rejected transitorio %s submitted=%v err=%v", transientIssue, submitted, err)
		}
	}

	ledger = NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	accepted := domainWorkSubmissionRecordForRetryTestV0(DomainWorkArtifactSubmissionStatusAcceptedV0)
	accepted.ReceiptRef = "receipt-ref-retry"
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, accepted); err != nil {
		t.Fatalf("record accepted: %v", err)
	}
	stack = StackV0{DomainDelivery: DomainWorkDeliveryBridgeConfigV0{Ledger: ledger}}
	submitted, err = stack.domainWorkSubmissionAlreadyRecordedV0(
		ctx,
		orquestacoreworkflow.OrchestrationRunV0{RunID: accepted.RunRef},
		orquestacoreworkflow.WorkflowTaskV0{TaskID: accepted.TaskRef},
		orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: accepted.DeliveryRef},
		domainWorkSubmissionForRetryTestV0(),
	)
	if err != nil || !submitted {
		t.Fatalf("accepted submitted=%v err=%v", submitted, err)
	}
}

func TestCodexStackV0DomainWorkSubmitRegistraRejectedSiExecutorFallaYReintenta(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	domainWork := &failOnceDomainWorkExecutorV0{}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	stack.DomainDelivery.Ledger = ledger
	director := postDirectorAPIV0(t, stack)
	postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))

	for attempt := 1; attempt <= 6 && domainWork.submits == 0; attempt++ {
		_, _ = stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-domain-work-submit-fail",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     2,
		})
	}
	if domainWork.submits != 1 {
		t.Fatalf("primer submit no ejecutado: submits=%d", domainWork.submits)
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{Status: DomainWorkArtifactSubmissionStatusRejectedV0},
	)
	if err != nil || len(records) != 1 ||
		!codexStackStringInSetV0(records[0].IssueRefs, "domain-work-submit-execute-error") {
		t.Fatalf("rejected records=%+v err=%v", records, err)
	}

	for attempt := 1; attempt <= 6; attempt++ {
		if _, err := stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-domain-work-submit-retry",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     2,
		}); err != nil {
			t.Fatalf("DrainRunV0 retry %d: %v", attempt, err)
		}
		accepted, err := ledger.ListDomainWorkArtifactSubmissionsV0(
			context.Background(),
			DomainWorkArtifactSubmissionRecordFilterV0{Status: DomainWorkArtifactSubmissionStatusAcceptedV0},
		)
		if err != nil {
			t.Fatalf("ListDomainWorkArtifactSubmissionsV0: %v", err)
		}
		if len(accepted) == 1 {
			return
		}
	}
	t.Fatalf("retry no aceptado: submits=%d", domainWork.submits)
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

type failOnceDomainWorkExecutorV0 struct {
	submits int
}

func (executor *failOnceDomainWorkExecutorV0) Execute(
	_ context.Context,
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
	if executor.submits == 1 {
		return orquestamcp.MCPDomainWorkToolResultV0{}, errors.New("domain_work_http_unavailable")
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
			ReceiptRef:    "receipt-ref-retry-" + input.ArtifactSubmission.ArtifactRef,
			EvidenceRefs:  []string{"receipt-evidence-ref-retry"},
		},
	}, nil
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
