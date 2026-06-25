package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
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

func TestCodexStackV0DomainWorkSubmitClaimVivoBloqueaReintentoHastaRecovery(t *testing.T) {
	ctx := context.Background()
	for _, status := range []string{
		DomainWorkArtifactSubmissionStatusClaimedV0,
		DomainWorkArtifactSubmissionStatusSubmittingV0,
		DomainWorkArtifactSubmissionStatusSubmittedV0,
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
		if err == nil || err.Error() != "domain_work_submit_recovery_required" || submitted {
			t.Fatalf("status=%s submitted=%v err=%v", status, submitted, err)
		}
	}
}

func TestCodexStackV0DomainWorkSubmitSubmittedConReceiptPromueveSinReenviar(t *testing.T) {
	ctx := context.Background()
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	record := domainWorkSubmissionRecordForRetryTestV0(DomainWorkArtifactSubmissionStatusSubmittedV0)
	record.ReceiptRef = "receipt-ref-submit-effect-001"
	record.EvidenceRefs = []string{"domain-work-submit-receipt-recorded"}
	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, record); err != nil {
		t.Fatalf("record submitted: %v", err)
	}
	stack := StackV0{DomainDelivery: DomainWorkDeliveryBridgeConfigV0{Ledger: ledger}}
	submitted, err := stack.domainWorkSubmissionAlreadyRecordedV0(
		ctx,
		orquestacoreworkflow.OrchestrationRunV0{RunID: record.RunRef},
		orquestacoreworkflow.WorkflowTaskV0{TaskID: record.TaskRef},
		orquestacionnucleoapp.AgentDeliveryObservationV0{DeliveryRef: record.DeliveryRef},
		domainWorkSubmissionForRetryTestV0(),
	)
	if err != nil || !submitted {
		t.Fatalf("submitted=%v err=%v", submitted, err)
	}
	records, err := ledger.ListDomainWorkArtifactSubmissionsV0(
		ctx,
		DomainWorkArtifactSubmissionRecordFilterV0{IdempotencyKey: record.IdempotencyKey},
	)
	if err != nil || len(records) != 1 ||
		records[0].Status != DomainWorkArtifactSubmissionStatusAcceptedV0 ||
		records[0].ReceiptRef != "receipt-ref-submit-effect-001" ||
		!codexStackStringInSetV0(records[0].EvidenceRefs, "domain-work-submit-ledger-recovered") {
		t.Fatalf("records=%+v err=%v", records, err)
	}
}

func TestCodexStackV0DomainWorkSubmitAcceptedYRejectedFuncionalBloqueanReintento(t *testing.T) {
	ctx := context.Background()
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

func TestCodexStackV0DomainWorkSubmitReceiptSinAcceptedNoReenvia(t *testing.T) {
	runtime := newFakeCodexStackRuntimeV0()
	memory := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	ledger := &failAcceptedDomainWorkSubmissionLedgerV0{inner: memory, failAccepted: true}
	domainWork := &claimCheckingDomainWorkExecutorV0{ledger: ledger, t: t}
	stack := mustBuildCodexStackWithDomainWorkForTestV0(t, runtime, domainWork)
	stack.DomainDelivery.Ledger = ledger
	director := postDirectorAPIV0(t, stack)
	postOPESExternalWorkChangeV0(t, stack, opesExternalWorkChangeV0(director.RunRef))

	var recoveryErr error
	for attempt := 1; attempt <= 6; attempt++ {
		_, recoveryErr = stack.DrainRunV0(context.Background(), DrainRunRequestV0{
			RunRef:               director.RunRef,
			CorrelationID:        "corr-domain-work-submit-receipt-ledger-fail",
			MaxBursts:            16,
			MaxStepsPerBurst:     8,
			MaxDispatchesPerWait: 8,
			MaxExternalWaits:     2,
		})
		if recoveryErr != nil {
			break
		}
	}
	if recoveryErr == nil || recoveryErr.Error() != "domain_work_submit_recovery_required" {
		t.Fatalf("recoveryErr=%v", recoveryErr)
	}
	records, err := memory.ListDomainWorkArtifactSubmissionsV0(
		context.Background(),
		DomainWorkArtifactSubmissionRecordFilterV0{Status: DomainWorkArtifactSubmissionStatusSubmittedV0},
	)
	if err != nil || len(records) != 1 || records[0].ReceiptRef == "" {
		t.Fatalf("submitted records=%+v err=%v", records, err)
	}
	if domainWork.submits != 1 {
		t.Fatalf("submits=%d", domainWork.submits)
	}
	_, recoveryErr = stack.DrainRunV0(context.Background(), DrainRunRequestV0{
		RunRef:               director.RunRef,
		CorrelationID:        "corr-domain-work-submit-receipt-replay",
		MaxBursts:            16,
		MaxStepsPerBurst:     8,
		MaxDispatchesPerWait: 8,
		MaxExternalWaits:     2,
	})
	if recoveryErr == nil || recoveryErr.Error() != "domain_work_submit_recovery_required" {
		t.Fatalf("replay recoveryErr=%v", recoveryErr)
	}
	if domainWork.submits != 1 {
		t.Fatalf("replay submit duplicado=%d", domainWork.submits)
	}
}
