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
	fixture := newServiceDomainWorkClosureFixtureForTestV0(t, "positive")

	loop, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(fixture.RunRef),
		StartAppDirectorPortsV0{
			RunStore:                 orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run),
			EventSink:                orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			EventReader:              serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
			DirectorTaskStore:        orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(fixture.Task),
			OperationalClosureSource: fixture.Source,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    fixture.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if !fixture.Source.Called || len(fixture.Source.LastRequest.RequiredTestEvidenceRefs) != 0 {
		t.Fatalf("source no valido cierre domain_work sin required tests: called=%v request=%+v", fixture.Source.Called, fixture.Source.LastRequest)
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(loop.Run.ClosedTasks, fixture.TaskRef) ||
		!serviceStringInSetV0(loop.Run.Validations, fixture.Source.ValidationRef) ||
		!serviceStringInSetV0(loop.Run.Closures, fixture.Source.ClosureRef) {
		t.Fatalf("run domain_work no cerrado: %+v", loop.Run)
	}
}

func TestMaybeCloseOperationalDirectorV0NoCierraDomainWorkConArtefactoNoCausal(t *testing.T) {
	fixture := newServiceDomainWorkClosureFixtureForTestV0(t, "no-causal")
	fixture.Artifact.ExternalRefs = []orquestadomainwork.DomainWorkExternalRefV0{
		{Kind: "run_ref", Ref: fixture.RunRef},
		{Kind: "task_ref", Ref: fixture.TaskRef},
		{Kind: "delivery_ref", Ref: "delivery-ref-domain-work-fake-app-otro"},
		{Kind: "accepted_review_ref", Ref: fixture.AcceptedReviewRef},
	}
	fixture.Source.Artifact = fixture.Artifact
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	_, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		serviceContinueClosureRequestForTestV0(fixture.RunRef),
		StartAppDirectorPortsV0{
			RunStore:                 runStore,
			EventSink:                eventSink,
			EventReader:              serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
			DirectorTaskStore:        orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(fixture.Task),
			OperationalClosureSource: fixture.Source,
		},
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    fixture.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err == nil || len(issues) != 0 || !fixture.Source.Called {
		t.Fatalf("domain_work no causal debe fallar antes de cerrar: err=%v issues=%+v called=%v", err, issues, fixture.Source.Called)
	}
	run, loadErr := runStore.LoadRunV0(context.Background(), fixture.RunRef)
	if loadErr != nil {
		t.Fatalf("LoadRunV0: %v", loadErr)
	}
	if run.Status == orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(run.ClosedTasks) != 0 ||
		len(run.Validations) != 0 ||
		len(run.Closures) != 0 ||
		serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventTaskClosedV0) != 0 ||
		serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0) != 0 ||
		serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0) != 0 {
		t.Fatalf("domain_work no causal cerro o emitio eventos de cierre: run=%+v events=%+v", run, eventSink.EventsV0())
	}
}

func TestMaybeCloseOperationalDirectorV0DomainWorkReplayNoDuplicaCierre(t *testing.T) {
	fixture := newServiceDomainWorkClosureFixtureForTestV0(t, "replay")
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	ports := StartAppDirectorPortsV0{
		RunStore:                 runStore,
		EventSink:                eventSink,
		EventReader:              serviceOperationalClosureEventReaderForTestV0{Events: fixture.Events},
		DirectorTaskStore:        orquestacionnucleoapp.NewInMemoryWorkflowTaskStoreV0(fixture.Task),
		OperationalClosureSource: fixture.Source,
	}
	request := serviceContinueClosureRequestForTestV0(fixture.RunRef)
	first, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    fixture.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 first err=%v issues=%+v", err, issues)
	}
	request.OccurredAt = "2026-05-22T22:15:01Z"
	replay, issues, err := maybeCloseOperationalDirectorV0(
		context.Background(),
		request,
		ports,
		orquestacionnucleoapp.ProgressiveLoopResultV0{
			Status: orquestacionnucleoapp.ProgressiveLoopStatusQuiescentV0,
			Run:    first.Run,
		},
		orquestacionnucleoapp.ProgressiveLoopRequestV0{},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 replay err=%v issues=%+v", err, issues)
	}
	if replay.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		len(replay.Run.ClosedTasks) != 1 ||
		len(replay.Run.Validations) != 1 ||
		len(replay.Run.Closures) != 1 ||
		serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventTaskClosedV0) != 1 ||
		serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventFinalValidationRegisteredV0) != 1 ||
		serviceCountEventsByTypeV0(eventSink.EventsV0(), orquestacoreworkflow.OrchestrationEventRunClosedV0) != 1 {
		t.Fatalf("domain_work replay no idempotente: run=%+v events=%+v", replay.Run, eventSink.EventsV0())
	}
}

type serviceDomainWorkClosureFixtureForTestV0 struct {
	RunRef            string
	TaskRef           string
	DeliveryRef       string
	ReviewRequestRef  string
	ReviewResultRef   string
	AcceptedReviewRef string
	ArtifactRef       string
	JobRef            string
	Run               orquestacoreworkflow.OrchestrationRunV0
	Task              orquestacoreworkflow.WorkflowTaskV0
	Artifact          orquestadomainwork.DomainWorkArtifactSubmissionV0
	Source            *serviceDomainWorkClosureSourceForTestV0
	Events            []orquestacoreworkflow.OrchestrationEventV0
}

func newServiceDomainWorkClosureFixtureForTestV0(t *testing.T, suffix string) serviceDomainWorkClosureFixtureForTestV0 {
	t.Helper()
	fixture := serviceDomainWorkClosureFixtureForTestV0{
		RunRef:            "run-service-operational-closure-domain-work-" + suffix,
		TaskRef:           "task-ref-domain-work-fake-app-" + suffix,
		DeliveryRef:       "delivery-ref-domain-work-fake-app-" + suffix,
		ReviewRequestRef:  "review-request-ref-domain-work-fake-app-" + suffix,
		ReviewResultRef:   "review-result-ref-domain-work-fake-app-" + suffix,
		AcceptedReviewRef: "accepted-review-ref-domain-work-fake-app-" + suffix,
		ArtifactRef:       "artifact-ref-domain-work-fake-app-" + suffix,
		JobRef:            "job-ref-domain-work-fake-app-" + suffix,
	}
	fixture.Run = serviceContinueClosureRunForTestV0(fixture.RunRef, orquestacoreworkflow.OrchestrationPhaseRevisionV0)
	fixture.Run.Tasks = []string{fixture.TaskRef}
	fixture.Run.Deliveries = []string{fixture.DeliveryRef}
	fixture.Run.AcceptedReviews = []string{fixture.AcceptedReviewRef}
	fixture.Task = orquestacoreworkflow.WorkflowTaskV0{
		SchemaVersion:      orquestacoreworkflow.WorkflowTaskSchemaVersionV0,
		TaskID:             fixture.TaskRef,
		RunID:              fixture.RunRef,
		PhaseID:            orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		WorkProfileKind:    orquestacoreworkflow.WorkProfileDomainWorkV0,
		Title:              "Resolver trabajo de dominio en app externa fake",
		WriteSet:           []string{"external-app-fake"},
		AcceptanceCriteria: []string{"artefacto domain_work aceptado por refs opacas"},
		ContextRefs:        []string{fixture.JobRef},
	}
	fixture.Artifact = orquestadomainwork.NormalizeDomainWorkArtifactSubmissionV0(orquestadomainwork.DomainWorkArtifactSubmissionV0{
		SchemaVersion:  orquestadomainwork.DomainWorkArtifactSubmissionSchemaV0,
		CorrelationID:  "corr-" + fixture.RunRef,
		IdempotencyKey: "idem-" + fixture.ArtifactRef,
		RequestedBy:    "fake-external-app",
		DomainRef:      "fake-external-app",
		JobRef:         fixture.JobRef,
		ArtifactRef:    fixture.ArtifactRef,
		ArtifactType:   "domain_result",
		Summary:        "Artefacto fake offline para cierre hexagonal.",
		ExternalRefs: []orquestadomainwork.DomainWorkExternalRefV0{
			{Kind: "run_ref", Ref: fixture.RunRef},
			{Kind: "task_ref", Ref: fixture.TaskRef},
			{Kind: "delivery_ref", Ref: fixture.DeliveryRef},
			{Kind: "accepted_review_ref", Ref: fixture.AcceptedReviewRef},
		},
		EvidenceRefs: []string{fixture.DeliveryRef, fixture.AcceptedReviewRef},
		CompleteJob:  true,
	})
	fixture.Source = &serviceDomainWorkClosureSourceForTestV0{
		Task:              fixture.Task,
		Artifact:          fixture.Artifact,
		DeliveryRef:       fixture.DeliveryRef,
		AcceptedReviewRef: fixture.AcceptedReviewRef,
		ValidationRef:     "validation-ref-domain-work-fake-app-" + suffix,
		ClosureRef:        "closure-ref-domain-work-fake-app-" + suffix,
	}
	fixture.Events = serviceOperationalClosureEventsForTasksForTestV0(t, fixture.RunRef, serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         fixture.TaskRef,
		DeliveryRef:     fixture.DeliveryRef,
		ReviewRequestID: fixture.ReviewRequestRef,
		ReviewResultRef: fixture.ReviewResultRef,
		AcceptedRef:     fixture.AcceptedReviewRef,
		TestEvidenceRef: fixture.ArtifactRef,
	})
	return fixture
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
	CallCount         int
}

func (source *serviceDomainWorkClosureSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
	_ context.Context,
	request AppDirectorOperationalClosureRequestV0,
) (orquestacionnucleoapp.OperationalDirectorClosureRequestV0, bool, error) {
	source.Called = true
	source.CallCount++
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
