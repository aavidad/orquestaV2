package orquestaappcodexstack

import (
	"context"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestaappchangedirectorsource "orquesta/modulos/orquesta-app-change-director-source"
	orquestaappdirectorservice "orquesta/modulos/orquesta-app-director-service"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestOperationalClosureSourceV0CierraOPESDomainWorkConReceiptAceptado(t *testing.T) {
	ctx := context.Background()
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "accepted")
	if err := fixture.Ledger.RecordDomainWorkArtifactSubmissionV0(ctx, DomainWorkArtifactSubmissionRecordV0{
		IdempotencyKey: "idem-opes-closure-accepted",
		Status:         DomainWorkArtifactSubmissionStatusAcceptedV0,
		RunRef:         fixture.Run.RunID,
		TaskRef:        fixture.Task.TaskID,
		DeliveryRef:    fixture.DeliveryRef,
		ReceiptRef:     "receipt-ref-opes-closure-accepted",
		EvidenceRefs:   []string{"opes-artifact-ref-accepted"},
	}); err != nil {
		t.Fatalf("record receipt: %v", err)
	}

	got, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		ctx,
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{
			Run:        fixture.Run,
			OccurredAt: "2026-05-24T12:00:00Z",
		},
	)
	if err != nil || !ok {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0 ok=%v err=%v request=%+v", ok, err, got)
	}
	if got.TaskID != fixture.Task.TaskID ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "receipt-ref-opes-closure-accepted") ||
		!codexStackOperationalClosureContainsV0(got.EvidenceRefs, "evidence-ref-opes-domain-work-accepted") {
		t.Fatalf("cierre OPES sin receipt causal: %+v", got)
	}
}

func TestOperationalClosureSourceV0NoCierraOPESSinReceiptAceptado(t *testing.T) {
	fixture := newOPESDomainWorkClosureFixtureForTestV0(t, "missing-receipt")

	_, ok, err := fixture.Source.BuildOperationalDirectorClosureRequestV0(
		context.Background(),
		orquestaappdirectorservice.AppDirectorOperationalClosureRequestV0{Run: fixture.Run},
	)
	if err != nil {
		t.Fatalf("BuildOperationalDirectorClosureRequestV0: %v", err)
	}
	if ok {
		t.Fatalf("no debe cerrar OPES sin receipt aceptado de DomainWork")
	}
}

type opesDomainWorkClosureFixtureForTestV0 struct {
	Run         orquestacoreworkflow.OrchestrationRunV0
	Task        orquestacoreworkflow.WorkflowTaskV0
	DeliveryRef string
	Ledger      *InMemoryDomainWorkArtifactSubmissionLedgerV0
	Source      codexStackOperationalClosureSourceV0
}

func newOPESDomainWorkClosureFixtureForTestV0(
	t *testing.T,
	suffix string,
) opesDomainWorkClosureFixtureForTestV0 {
	t.Helper()
	runRef := "run-stack-opes-domain-closure-" + suffix
	changeRef := "opes-job-job-ref-closure-" + suffix
	taskRef := orquestaappchangedirectorsource.AppChangeTaskRefV0(changeRef)
	deliveryRef := "delivery-ref-opes-domain-closure-" + suffix
	testRef := "opes-domain-test-draft_content_block-job-ref-closure-" + suffix
	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:   orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:          taskRef,
		RunID:           runRef,
		PhaseID:         orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind: orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:           "Resolver job OPES con cierre causal",
		WriteSet:        []string{"external/opes/draft_content_block/job-ref-closure-" + suffix},
		AcceptanceCriteria: []string{
			"OPES acepta el artefacto y devuelve receipt por API publica",
		},
		RequiredTests: []string{testRef},
		FunctionContractRefs: []orquestacoreworkflow.WorkflowFunctionContractRefV0{{
			FunctionName: "ApplyExternalDomainWorkV0",
		}},
	}
	run := stackOperationalClosureRunForTestV0(runRef, taskRef)
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{"accepted-review-ref-" + deliveryRef}
	reader := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	if err := reader.AppendRunEventsV0(context.Background(), runRef, []orquestacoreworkflow.OrchestrationEventV0{
		stackOperationalClosureDeliveryEventForTestV0(t, runRef, 1, taskRef, deliveryRef),
		stackOperationalClosureReviewRequestedEventForTestV0(t, runRef, 2, deliveryRef),
		stackOperationalClosureReviewResultEventForTestV0(t, runRef, 3, deliveryRef, orquestacoreworkflow.ReviewResultStatusAcceptedV0),
		stackOperationalClosureAcceptedReviewEventForTestV0(t, runRef, 4, deliveryRef),
	}); err != nil {
		t.Fatalf("AppendRunEventsV0: %v", err)
	}
	ledger := NewInMemoryDomainWorkArtifactSubmissionLedgerV0()
	return opesDomainWorkClosureFixtureForTestV0{
		Run:         run,
		Task:        task,
		DeliveryRef: deliveryRef,
		Ledger:      ledger,
		Source: codexStackOperationalClosureSourceV0{
			TaskStore:                 orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			EventReader:               reader,
			RequiredTestEvidenceStore: orquestacionnucleoapp.NewInMemoryRequiredTestEvidenceStoreV0(opesDomainRequiredTestEvidenceForTestV0(runRef, taskRef, deliveryRef, testRef)),
			AppChangeStore:            orquestaappchange.NewInMemoryAppChangeStoreV0(opesDomainWorkAppChangeRecordForTestV0(runRef, changeRef)),
			DomainSubmissionLedger:    ledger,
		},
	}
}

func opesDomainWorkAppChangeRecordForTestV0(
	runRef string,
	changeRef string,
) orquestaappchange.AppChangeRecordV0 {
	return orquestaappchange.AppChangeRecordV0{
		Request: orquestaappchange.PrepareAppChangeRequestV0(orquestaappchange.AppChangeRequestV0{
			RunRef:     runRef,
			AppRef:     "opes",
			ChangeRef:  changeRef,
			UserIntent: "Resolver job OPES por contrato externo.",
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-closure",
				WorkKind:   "draft_content_block",
			},
		}),
	}
}

func opesDomainRequiredTestEvidenceForTestV0(
	runRef string,
	taskRef string,
	deliveryRef string,
	testRef string,
) orquestacionnucleoapp.RequiredTestEvidenceV0 {
	return orquestacionnucleoapp.RequiredTestEvidenceV0{
		SchemaVersion:     orquestacionnucleoapp.RequiredTestEvidenceSchemaVersionV0,
		EvidenceRef:       "test-evidence-ref-" + deliveryRef,
		RunRef:            runRef,
		TaskRef:           taskRef,
		TestCommand:       testRef,
		Status:            orquestacionnucleoapp.RequiredTestEvidenceStatusPassedV0,
		DeliveryRef:       deliveryRef,
		ReviewRequestID:   "review-request-ref-" + deliveryRef,
		ReviewResultRef:   "review-result-ref-" + deliveryRef,
		AcceptedReviewRef: "accepted-review-ref-" + deliveryRef,
		OccurredAt:        "2026-05-24T12:00:00Z",
		EvidenceRefs:      []string{"domain-test-output-ref-" + deliveryRef},
	}
}
