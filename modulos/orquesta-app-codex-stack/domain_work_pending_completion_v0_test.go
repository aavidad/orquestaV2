package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestadomainworkmemory "orquesta/modulos/orquesta-domain-work-memory"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestaruncontrol "orquesta/modulos/orquesta-run-control"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestCodexStackV0DrainQueueStatusNoParaDomainWorkJobAceptadoSinReceiptV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-domain-work-job-pending-receipt-001"
	creator, jobRef := newDomainWorkAcceptedJobForRunV0(t, runRef)
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	stack := StackV0{
		DomainDelivery: DomainWorkDeliveryBridgeConfigV0{
			Enabled:    true,
			Ledger:     ledger,
			JobRecords: creator,
		},
	}
	result := domainWorkQuiescentEmptyRunResultV0(runRef)

	status, evidenceRefs, err := stack.stackDrainQueueStatusAndEvidenceForCoordinatorV0(ctx, result)
	if err != nil {
		t.Fatalf("stackDrainQueueStatusAndEvidenceForCoordinatorV0: %v", err)
	}
	if status != "" || !codexStackStringInSetV0(evidenceRefs, domainWorkAcceptedJobPendingReceiptEvidenceRefV0) {
		t.Fatalf("status=%q evidence=%v", status, evidenceRefs)
	}

	if err := ledger.RecordDomainWorkArtifactSubmissionV0(ctx, DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-domain-work-job-pending-receipt-accepted-001",
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         runRef,
		JobRef:         jobRef,
		DeliveryRef:    "delivery-ref-domain-work-job-pending-receipt-001",
		DomainRef:      "opes",
		ArtifactRef:    "artifact-ref-domain-work-job-pending-receipt-001",
		ArtifactType:   "local_html_site",
		ReceiptRef:     "receipt-ref-domain-work-job-pending-receipt-001",
	}); err != nil {
		t.Fatalf("RecordDomainWorkArtifactSubmissionV0: %v", err)
	}
	status, evidenceRefs, err = stack.stackDrainQueueStatusAndEvidenceForCoordinatorV0(ctx, result)
	if err != nil {
		t.Fatalf("stackDrainQueueStatusAndEvidenceForCoordinatorV0 accepted: %v", err)
	}
	if status != orquestarunqueue.RunStatusStoppedV0 || len(evidenceRefs) != 0 {
		t.Fatalf("status=%q evidence=%v", status, evidenceRefs)
	}
}

func TestCodexStackV0RecoverNoTerminalizaStopConDomainWorkJobAceptadoSinReceiptV0(t *testing.T) {
	ctx := context.Background()
	runRef := "run-ref-domain-work-stop-job-pending-receipt-001"
	creator, _ := newDomainWorkAcceptedJobForRunV0(t, runRef)
	stack := mustBuildCodexStackWithDomainWorkForTestV0(
		t,
		newFakeCodexStackRuntimeV0(),
		&fakeCodexStackDomainWorkExecutorV0{},
	)
	stack.DomainDelivery.JobRecords = creator
	state, err := stack.Stores.RunControl.StopRunV0(ctx, orquestaruncontrol.StopRunCommandV0{
		RunRef:       runRef,
		RequestedBy:  "orquesta-director",
		Reason:       "reinicio ordenado con job domain work pendiente",
		EvidenceRefs: []string{"evidence-ref-test-domain-work-job-pending-stop"},
	})
	if err != nil {
		t.Fatalf("StopRunV0: %v", err)
	}

	completed, err := stack.completeQueuedRunControlIfStopHasNoPendingAgentsV0(
		ctx,
		globalTickCommandForTestV0(),
		orquestarunqueue.RunSchedulingCandidateV0{
			RunRef:        runRef,
			AppRef:        "project-ref-orquesta-server",
			Status:        orquestarunqueue.RunStatusRunningV0,
			PriorityScore: 80,
		},
		state,
		orquestacoreworkflow.OrchestrationRunV0{
			SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
			RunID:         runRef,
			Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
			CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		},
	)
	if err != nil {
		t.Fatalf("completeQueuedRunControlIfStopHasNoPendingAgentsV0: %v", err)
	}
	if completed {
		t.Fatalf("run-control terminalizado con job DomainWork aceptado sin receipt")
	}
}

func newDomainWorkAcceptedJobForRunV0(
	t *testing.T,
	runRef string,
) (*orquestadomainworkmemory.InMemoryDomainWorkJobCreatorV0, string) {
	t.Helper()
	creator := orquestadomainworkmemory.NewInMemoryDomainWorkJobCreatorV0()
	job, err := creator.CreateDomainWorkJobV0(context.Background(), orquestadomainwork.DomainWorkJobRequestV0{
		SchemaVersion:  orquestadomainwork.DomainWorkJobRequestSchemaV0,
		RequestID:      "request-ref-" + runRef,
		CorrelationID:  "corr-" + runRef,
		IdempotencyKey: "change-ref-" + runRef,
		RequestedBy:    orquestadomainwork.DomainWorkDefaultRequestedByV0,
		DomainRef:      "opes",
		WorkKind:       "generate_html_site",
		Objective:      "generar HTML local aislado",
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{{
			Kind: "run_ref",
			Ref:  runRef,
		}},
	})
	if err != nil {
		t.Fatalf("CreateDomainWorkJobV0: %v", err)
	}
	if job.Status != orquestadomainwork.DomainWorkStatusAcceptedV0 {
		t.Fatalf("job=%+v", job)
	}
	return creator, job.JobRef
}

func domainWorkQuiescentEmptyRunResultV0(
	runRef string,
) orquestacionnucleoapp.ManagedProgressiveLoopResultV0 {
	return orquestacionnucleoapp.ManagedProgressiveLoopResultV0{
		Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
		Final: orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run: orquestacoreworkflow.OrchestrationRunV0{
				SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
				RunID:         runRef,
				Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
				CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
			},
		},
	}
}
