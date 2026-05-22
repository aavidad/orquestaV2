package orquestaappdirectorservice

import (
	"context"
	"fmt"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMaybeCloseOperationalDirectorV0CierraDomainWorkConAppExternaFake(t *testing.T) {
	runRef := "run-service-operational-closure-domain-work-001"
	taskRef := "task-ref-domain-work-fake-app-001"
	deliveryRef := "delivery-ref-domain-work-fake-app-001"
	reviewRequestRef := "review-request-ref-domain-work-fake-app-001"
	reviewResultRef := "review-result-ref-domain-work-fake-app-001"
	acceptedReviewRef := "accepted-review-ref-domain-work-fake-app-001"
	artifactRef := "artifact-ref-domain-work-fake-app-001"
	jobRef := "job-ref-domain-work-fake-app-001"

	run := serviceContinueClosureRunForTestV0(runRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	run.Tasks = []string{taskRef}
	run.Deliveries = []string{deliveryRef}
	run.AcceptedReviews = []string{acceptedReviewRef}

	task := orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             taskRef,
		RunID:              runRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:              "Resolver trabajo de dominio en app externa fake",
		WriteSet:           []string{"external-app-fake"},
		AcceptanceCriteria: []string{"artefacto domain_work aceptado por refs opacas"},
		ContextRefs:        []string{jobRef},
	}
	artifact := orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		SchemaVersion:  orquestadomainwork.DomainWorkArtifactSubmissionSchemaV0,
		CorrelationID:  "corr-" + runRef,
		IdempotencyKey: "idem-" + artifactRef,
		RequestedBy:    "fake-external-app",
		DomainRef:      "fake-external-app",
		JobRef:         jobRef,
		ArtifactRef:    artifactRef,
		ArtifactType:   "domain_result",
		Summary:        "Artefacto fake offline para cierre hexagonal.",
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: runRef},
			{Kind: "task_ref", Ref: taskRef},
			{Kind: "delivery_ref", Ref: deliveryRef},
			{Kind: "accepted_review_ref", Ref: acceptedReviewRef},
		},
		EvidenceRefs: []string{deliveryRef, acceptedReviewRef},
		CompleteJob:  true,
	})
	source := &serviceDomainWorkClosureSourceForTestV0{
		Task:              task,
		Artifact:          artifact,
		DeliveryRef:       deliveryRef,
		AcceptedReviewRef: acceptedReviewRef,
		ValidationRef:     "validation-ref-domain-work-fake-app-001",
		ClosureRef:        "closure-ref-domain-work-fake-app-001",
	}

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(runRef),
		StartAppDirectorPortsV0{
			RunStore:  orquestacionnucleoapp.NewInMemoryRunStoreV0(run),
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader: serviceOperationalClosureEventReaderForTestV0{Events: serviceOperationalClosureEventsForTasksForTestV0(t, runRef, serviceOperationalClosureTaskRefsForTestV0{
				TaskRef:         taskRef,
				DeliveryRef:     deliveryRef,
				ReviewRequestID: reviewRequestRef,
				ReviewResultRef: reviewResultRef,
				AcceptedRef:     acceptedReviewRef,
				TestEvidenceRef: artifactRef,
			})},
			DirectorTaskStore:        orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(task),
			OperationalClosureSource: source,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if !source.Called || len(source.LastRequest.RequiredTestEvidenceRefs) != 0 {
		t.Fatalf("source no valido cierre domain_work sin required tests: called=%v request=%+v", source.Called, source.LastRequest)
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(loop.Run.ClosedTasks, taskRef) ||
		!serviceStringInSetV0(loop.Run.Validations, source.ValidationRef) ||
		!serviceStringInSetV0(loop.Run.Closures, source.ClosureRef) {
		t.Fatalf("run domain_work no cerrado: %+v", loop.Run)
	}
}

type serviceDomainWorkClosureSourceForTestV0 struct {
	Task              orquestacoreworkflow.WorkflowTaskV0
	Artifact          orquestadomainwork.DomainWorkArtifactSubmissionV0
	DeliveryRef       string
	AcceptedReviewRef string
	ValidationRef     string
	ClosureRef        string
	LastRequest       AppDirectorOperationalClosureRequestV0
	Called            bool
}

func (source *serviceDomainWorkClosureSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.LastRequest = request
	if err := source.validateV0(request); err != nil {
		return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{}, false, err
	}
	return orquestacionnucleoapp.OperationalDirectorClosureRequestV0{
		TaskID:            source.Task.TaskID,
		DeliveryRef:       source.DeliveryRef,
		AcceptedReviewRef: source.AcceptedReviewRef,
		ValidationRef:     source.ValidationRef,
		ClosureRef:        source.ClosureRef,
		Summary:           "Cierre offline de trabajo domain_work validado por app externa fake.",
		EvidenceRefs:      []string{source.Artifact.JobRef, source.Artifact.ArtifactRef},
	}, true, nil
}

func (source *serviceDomainWorkClosureSourceForTestV0) validateV0(
	request AppDirectorOperationalClosureRequestV0,
) error {
	if source.Task.WorkProfileKind != orquestacoreworkflow.WorkProfileDomainWorkV0 {
		return fmt.Errorf("task no es domain_work: %s", source.Task.WorkProfileKind)
	}
	if len(source.Task.RequiredTests) != 0 || len(request.RequiredTestEvidenceRefs) != 0 {
		return fmt.Errorf("domain_work fake no debe requerir required tests")
	}
	if issues := orquestadomainwork.ValidateDomainWorkArtifactSubmissionV0(source.Artifact); len(issues) > 0 {
		return fmt.Errorf("artefacto domain_work invalido: %+v", issues)
	}
	for _, check := range []struct {
		values []string
		want   string
		field  string
	}{
		{values: request.Run.Tasks, want: source.Task.TaskID, field: "run.tasks"},
		{values: request.Run.Deliveries, want: source.DeliveryRef, field: "run.deliveries"},
		{values: request.Run.AcceptedReviews, want: source.AcceptedReviewRef, field: "run.accepted_reviews"},
		{values: source.Task.ContextRefs, want: source.Artifact.JobRef, field: "task.context_refs"},
	} {
		if !serviceStringInSetV0(check.values, check.want) {
			return fmt.Errorf("%s no contiene ref opaca %q", check.field, check.want)
		}
	}
	for _, ref := range []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: request.Run.RunID},
		{Kind: "task_ref", Ref: source.Task.TaskID},
		{Kind: "delivery_ref", Ref: source.DeliveryRef},
		{Kind: "accepted_review_ref", Ref: source.AcceptedReviewRef},
	} {
		if !serviceDomainWorkExternalRefInSetForTestV0(source.Artifact.ExternalRefs, ref) {
			return fmt.Errorf("artefacto sin external ref opaca %+v", ref)
		}
	}
	return nil
}

func serviceDomainWorkExternalRefInSetForTestV0(
	values []orquestadomainwork.DomainWorkExternalRefV0,
	want orquestadomainwork.DomainWorkExternalRefV0,
) bool {
	for _, value := range values {
		if value.Kind == want.Kind && value.Ref == want.Ref {
			return true
		}
	}
	return false
}
