package orquestaappdirectorservice

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
	"testing"

	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
)

func TestMaybeCloseOperationalDirectorV0CierraComposicionExternaNeutralPorRefsOpacas(t *testing.T) {
	fixture := newServiceExternalFakeClosureFixtureForTestV0(t, "neutral")
	runStore := orquestacionnucleoapp.NewInMemoryRunStoreV0(fixture.Run)
	eventSink := orquestacionnucleoapp.NewInMemoryEventSinkV0()

	loop, issues, err := maybeCloseOperationalDirectorV0(
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
		orquestacionnucleoapp.ProgressiveLoopRequestV0{
			WaitAgentRefs:    []string{fixture.AgentRef},
			WaitScopeApplied: true,
		},
	)
	if err != nil || len(issues) != 0 {
		t.Fatalf("maybeCloseOperationalDirectorV0 err=%v issues=%+v", err, issues)
	}
	if !fixture.Source.Called {
		t.Fatalf("source externo no invocado")
	}
	if !fixture.Source.LastRequest.WaitScopeApplied ||
		len(fixture.Source.LastRequest.RequiredTestEvidenceRefs) != 0 {
		t.Fatalf("request al source no conserva frontera neutral: %+v", fixture.Source.LastRequest)
	}
	if loop.Run.Status != orquestacoreworkflow.OrchestrationRunStatusClosedV0 ||
		!serviceStringInSetV0(loop.Run.ClosedTasks, fixture.TaskRef) ||
		!serviceStringInSetV0(loop.Run.Validations, fixture.ValidationRef) ||
		!serviceStringInSetV0(loop.Run.Closures, fixture.ClosureRef) {
		t.Fatalf("run no cerrado por refs opacas: %+v", loop.Run)
	}
}

func TestAppDirectorServiceOperationalClosureExternalFakeV0NoImportaProductAdapters(t *testing.T) {
	for _, file := range productionGoFilesForServiceTestV0(t) {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, src, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatalf("import path %s: %v", spec.Path.Value, err)
			}
			if serviceExternalFakeProductAdapterImportForTestV0(path) {
				t.Fatalf("%s importa adaptador de producto %q", file, path)
			}
		}
	}
}

type serviceExternalFakeClosureFixtureForTestV0 struct {
	RunRef            string
	TaskRef           string
	AgentRef          string
	DeliveryRef       string
	ReviewRequestRef  string
	ReviewResultRef   string
	AcceptedReviewRef string
	ValidationRef     string
	ClosureRef        string
	EntityRef         string
	ArtifactRef       string
	Run               orquestacoreworkflow.OrchestrationRunV0
	Task              orquestacoreworkflow.WorkflowTaskV0
	Events            []orquestacoreworkflow.OrchestrationEventV0
	Source            *serviceExternalFakeClosureSourceForTestV0
}

func newServiceExternalFakeClosureFixtureForTestV0(
	t *testing.T,
	suffix string,
) serviceExternalFakeClosureFixtureForTestV0 {
	t.Helper()
	fixture := serviceExternalFakeClosureFixtureForTestV0{
		RunRef:            "run-service-operational-closure-external-fake-" + suffix,
		TaskRef:           "task-ref-external-fake-" + suffix,
		AgentRef:          "agent-ref-external-fake-" + suffix,
		DeliveryRef:       "delivery-ref-external-fake-" + suffix,
		ReviewRequestRef:  "review-request-ref-external-fake-" + suffix,
		ReviewResultRef:   "review-result-ref-external-fake-" + suffix,
		AcceptedReviewRef: "accepted-review-ref-external-fake-" + suffix,
		ValidationRef:     "validation-ref-external-fake-" + suffix,
		ClosureRef:        "closure-ref-external-fake-" + suffix,
		EntityRef:         "external-entity-ref-" + suffix,
		ArtifactRef:       "external-artifact-ref-" + suffix,
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
		WorkProfileKind:    orquestacoreworkflow.WorkProfileDocumentationV0,
		Title:              "Cerrar trabajo externo fake",
		WriteSet:           []string{"external-composition-fake"},
		AcceptanceCriteria: []string{"artefacto externo aceptado por refs opacas"},
		ContextRefs:        []string{fixture.EntityRef},
	}
	fixture.Events = serviceOperationalClosureEventsForTasksForTestV0(t, fixture.RunRef, serviceOperationalClosureTaskRefsForTestV0{
		TaskRef:         fixture.TaskRef,
		DeliveryRef:     fixture.DeliveryRef,
		ReviewRequestID: fixture.ReviewRequestRef,
		ReviewResultRef: fixture.ReviewResultRef,
		AcceptedRef:     fixture.AcceptedReviewRef,
	})
	fixture.Source = &serviceExternalFakeClosureSourceForTestV0{
		Task:              fixture.Task,
		EntityRef:         fixture.EntityRef,
		ArtifactRef:       fixture.ArtifactRef,
		DeliveryRef:       fixture.DeliveryRef,
		AcceptedReviewRef: fixture.AcceptedReviewRef,
		ValidationRef:     fixture.ValidationRef,
		ClosureRef:        fixture.ClosureRef,
	}
	return fixture
}

type serviceExternalFakeClosureSourceForTestV0 struct {
	Task              orquestacoreworkflow.WorkflowTaskV0
	EntityRef         string
	ArtifactRef       string
	DeliveryRef       string
	AcceptedReviewRef string
	ValidationRef     string
	ClosureRef        string
	LastRequest       AppDirectorOperationalClosureRequestV0
	Called            bool
}

func (source *serviceExternalFakeClosureSourceForTestV0) BuildOperationalDirectorClosureRequestV0(
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
		Summary:           "Cierre offline fake externo validado por refs opacas.",
		EvidenceRefs:      []string{source.EntityRef, source.ArtifactRef},
	}, true, nil
}

func (source *serviceExternalFakeClosureSourceForTestV0) validateV0(
	request AppDirectorOperationalClosureRequestV0,
) error {
	if len(source.Task.RequiredTests) != 0 || len(request.RequiredTestEvidenceRefs) != 0 {
		return fmt.Errorf("composicion externa fake no debe requerir tests de programacion")
	}
	for _, check := range []struct {
		values []string
		want   string
		field  string
	}{
		{values: request.Run.Tasks, want: source.Task.TaskID, field: "run.tasks"},
		{values: request.Run.Deliveries, want: source.DeliveryRef, field: "run.deliveries"},
		{values: request.Run.AcceptedReviews, want: source.AcceptedReviewRef, field: "run.accepted_reviews"},
		{values: source.Task.ContextRefs, want: source.EntityRef, field: "task.context_refs"},
	} {
		if !serviceStringInSetV0(check.values, check.want) {
			return fmt.Errorf("%s no contiene ref opaca %q", check.field, check.want)
		}
	}
	return nil
}

func serviceExternalFakeProductAdapterImportForTestV0(path string) bool {
	for _, fragment := range []string{
		"orquesta/modulos/orquesta-runtime-codex",
		"orquesta/modulos/orquesta-app-codex-stack",
		"orquesta/modulos/orquesta-opes-",
	} {
		if strings.HasPrefix(path, fragment) {
			return true
		}
	}
	return false
}
