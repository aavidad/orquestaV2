package orquestaexternalworkrun

import (
	"context"
	"errors"
	"reflect"
	"testing"

	orquestaappchange "orquesta/modulos/orquesta-app-change"
	orquestacoreworkflow "orquesta/modulos/orquesta-core-workflow"
	orquestadomainwork "orquesta/modulos/orquesta-domain-work"
	orquestacionnucleoapp "orquesta/modulos/orquesta-orchestration-core"
	orquestarunqueue "orquesta/modulos/orquesta-run-queue"
)

func TestStartExternalWorkRunV0CreaRunOperativoYEncola(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	sink := orquestacionnucleoapp.NewInMemoryEventSinkV0()
	queue := &fakeExternalWorkRunQueueV0{}
	changeStore := orquestaappchange.NewInMemoryAppChangeStoreV0()
	result, err := StartExternalWorkRunV0(
		context.Background(),
		validExternalWorkRunRequestForTestV0(),
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: sink,
			RunQueue:  queue,
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            changeStore,
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusAcceptedV0 ||
		result.RunRef != "run-external-work-opes-job-ref-001-change-ref-001" ||
		result.DirectorQuestionRef != "question-ref-app-change-change-ref-001" {
		t.Fatalf("result=%+v", result)
	}
	run, err := store.LoadRunV0(context.Background(), result.RunRef)
	if err != nil {
		t.Fatalf("LoadRunV0: %v", err)
	}
	if string(run.CurrentPhase) != "programacion" {
		t.Fatalf("current_phase=%s", run.CurrentPhase)
	}
	if len(queue.commands) != 1 ||
		queue.commands[0].RunRef != result.RunRef ||
		queue.commands[0].AppRef != "opes" {
		t.Fatalf("queue=%+v", queue.commands)
	}
	records, err := changeStore.ListAppChangeRecordsV0(
		context.Background(),
		orquestaappchange.AppChangeRecordFilterV0{RunRef: result.RunRef},
	)
	if err != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", err)
	}
	if len(records) != 1 || records[0].Request.ExternalWork == nil {
		t.Fatalf("records=%+v", records)
	}
}

func TestStartExternalWorkRunV0NormalizaQueueDefaultAColaCanonica(t *testing.T) {
	queue := &fakeExternalWorkRunQueueV0{}
	request := validExternalWorkRunRequestForTestV0()
	request.QueueRef = " default "

	result, err := StartExternalWorkRunV0(
		context.Background(),
		request,
		StartExternalWorkRunPortsV0{
			RunStore:  orquestacionnucleoapp.NewInMemoryRunStoreV0(),
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  queue,
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            orquestaappchange.NewInMemoryAppChangeStoreV0(),
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{
			QueueRef:   "global",
			OccurredAt: "2026-05-10T10:00:00Z",
		},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusAcceptedV0 ||
		len(queue.commands) != 1 ||
		queue.commands[0].QueueRef != "global" {
		t.Fatalf("result=%+v queue=%+v", result, queue.commands)
	}
}

func TestStartExternalWorkRunV0NoCreaRunSiFaltaExternalWork(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	queue := &fakeExternalWorkRunQueueV0{}
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork = nil

	result, err := StartExternalWorkRunV0(
		context.Background(),
		request,
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  queue,
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            orquestaappchange.NewInMemoryAppChangeStoreV0(),
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusInvalidV0 ||
		!reflect.DeepEqual(issueCodesV0(result.Issues), []string{ErrExternalWorkRunExternalWorkRequiredV0}) {
		t.Fatalf("result=%+v", result)
	}
	if len(queue.commands) != 0 {
		t.Fatalf("queue no debe mutar: %+v", queue.commands)
	}
	if _, err := store.LoadRunV0(context.Background(), result.RunRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe existir, err=%v", err)
	}
}

func TestStartExternalWorkRunV0BloqueaMissingContextAntesDeCrearRun(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	queue := &fakeExternalWorkRunQueueV0{}
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork.InputFields = append(
		request.AppChangeRequest.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "missing_context",
			Values: []string{"topic_outline"},
		},
	)

	result, err := StartExternalWorkRunV0(
		context.Background(),
		request,
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  queue,
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            orquestaappchange.NewInMemoryAppChangeStoreV0(),
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusInvalidV0 ||
		!reflect.DeepEqual(issueCodesV0(result.Issues), []string{ErrExternalWorkRunMissingContextV0}) {
		t.Fatalf("result=%+v", result)
	}
	if len(queue.commands) != 0 {
		t.Fatalf("queue no debe mutar: %+v", queue.commands)
	}
	if _, err := store.LoadRunV0(context.Background(), result.RunRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe existir, err=%v", err)
	}
}

func TestStartExternalWorkRunV0BloqueaRequiredInputFieldsAusentes(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	queue := &fakeExternalWorkRunQueueV0{}
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork.InputFields = append(
		request.AppChangeRequest.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_input_fields",
			Values: []string{"topic_id", "source_refs"},
		},
	)

	result, err := StartExternalWorkRunV0(
		context.Background(),
		request,
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  queue,
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            orquestaappchange.NewInMemoryAppChangeStoreV0(),
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusInvalidV0 ||
		!reflect.DeepEqual(issueCodesV0(result.Issues), []string{ErrExternalWorkRunRequiredInputMissingV0}) {
		t.Fatalf("result=%+v", result)
	}
	if len(queue.commands) != 0 {
		t.Fatalf("queue no debe mutar: %+v", queue.commands)
	}
	if _, err := store.LoadRunV0(context.Background(), result.RunRef); !orquestacionnucleoapp.IsRunNotFoundErrorV0(err) {
		t.Fatalf("run no debe existir, err=%v", err)
	}
}

func TestStartExternalWorkRunV0AceptaRequiredInputFieldsPresentes(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	queue := &fakeExternalWorkRunQueueV0{}
	request := validExternalWorkRunRequestForTestV0()
	request.AppChangeRequest.ExternalWork.InputFields = append(
		request.AppChangeRequest.ExternalWork.InputFields,
		orquestadomainwork.DomainWorkFieldV0{
			Name:   "required_input_fields",
			Values: []string{"topic_id"},
		},
	)

	result, err := StartExternalWorkRunV0(
		context.Background(),
		request,
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  queue,
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            orquestaappchange.NewInMemoryAppChangeStoreV0(),
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusAcceptedV0 || len(queue.commands) != 1 {
		t.Fatalf("result=%+v queue=%+v", result, queue.commands)
	}
}

func TestStartExternalWorkRunV0IncluyeProyectoYCambioEnRunRefDerivado(t *testing.T) {
	requestA := validExternalWorkRunRequestForTestV0()
	requestA.AppChangeRequest.AppRef = "opes"
	requestA.AppChangeRequest.ExternalWork.ProjectRef = "opes"
	requestA.AppChangeRequest.ChangeRef = "change-a"
	requestA.AppChangeRequest.ExternalWork.JobRef = "job-ref-001"
	requestB := validExternalWorkRunRequestForTestV0()
	requestB.AppChangeRequest.AppRef = "otra-app"
	requestB.AppChangeRequest.ExternalWork.ProjectRef = "otra-app"
	requestB.AppChangeRequest.ChangeRef = "change-b"
	requestB.AppChangeRequest.ExternalWork.JobRef = "job-ref-001"

	gotA := normalizeStartExternalWorkRunRequestV0(
		requestA,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	).RunRef
	gotB := normalizeStartExternalWorkRunRequestV0(
		requestB,
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	).RunRef
	if gotA == gotB {
		t.Fatalf("run_ref debe distinguir proyecto/cambio: %s", gotA)
	}
}

func TestStartExternalWorkRunV0RechazaRunExistenteDeOtroProyecto(t *testing.T) {
	request := validExternalWorkRunRequestForTestV0()
	request.RunRef = "run-compartido"
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0(orquestacoreRunForConflictTestV0("run-compartido"))

	result, err := StartExternalWorkRunV0(
		context.Background(),
		request,
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  &fakeExternalWorkRunQueueV0{},
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            orquestaappchange.NewInMemoryAppChangeStoreV0(),
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if err != nil {
		t.Fatalf("StartExternalWorkRunV0: %v", err)
	}
	if result.Status != ExternalWorkRunStatusInvalidV0 ||
		!reflect.DeepEqual(issueCodesV0(result.Issues), []string{ErrExternalWorkRunExistingRunConflictV0}) {
		t.Fatalf("result=%+v", result)
	}
}

func TestStartExternalWorkRunV0NoRegistraCambioSiFallaCola(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	changeStore := orquestaappchange.NewInMemoryAppChangeStoreV0()
	queueErr := errors.New("queue failed")
	_, err := StartExternalWorkRunV0(
		context.Background(),
		validExternalWorkRunRequestForTestV0(),
		StartExternalWorkRunPortsV0{
			RunStore:  store,
			EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
			RunQueue:  &fakeExternalWorkRunQueueV0{err: queueErr},
			AppChange: orquestaappchange.AppChangePortsV0{
				Store:            changeStore,
				DirectorNotifier: fakeExternalWorkRunNotifierV0{},
			},
		},
		StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"},
	)
	if !errors.Is(err, queueErr) {
		t.Fatalf("err=%v want %v", err, queueErr)
	}
	records, listErr := changeStore.ListAppChangeRecordsV0(
		context.Background(),
		orquestaappchange.AppChangeRecordFilterV0{},
	)
	if listErr != nil {
		t.Fatalf("ListAppChangeRecordsV0: %v", listErr)
	}
	if len(records) != 0 {
		t.Fatalf("no debe registrar cambio si cola falla: %+v", records)
	}
}

func TestStartExternalWorkRunV0RetryNoRenotificaCambioExistente(t *testing.T) {
	store := orquestacionnucleoapp.NewInMemoryRunStoreV0()
	changeStore := orquestaappchange.NewInMemoryAppChangeStoreV0()
	queue := &fakeExternalWorkRunQueueV0{}
	notifierCalls := 0
	ports := StartExternalWorkRunPortsV0{
		RunStore:  store,
		EventSink: orquestacionnucleoapp.NewInMemoryEventSinkV0(),
		RunQueue:  queue,
		AppChange: orquestaappchange.AppChangePortsV0{
			Store:            changeStore,
			DirectorNotifier: fakeExternalWorkRunNotifierV0{calls: &notifierCalls},
		},
	}
	request := validExternalWorkRunRequestForTestV0()
	config := StartExternalWorkRunConfigV0{OccurredAt: "2026-05-10T10:00:00Z"}
	if _, err := StartExternalWorkRunV0(context.Background(), request, ports, config); err != nil {
		t.Fatalf("first StartExternalWorkRunV0: %v", err)
	}
	if _, err := StartExternalWorkRunV0(context.Background(), request, ports, config); err != nil {
		t.Fatalf("retry StartExternalWorkRunV0: %v", err)
	}
	if notifierCalls != 1 {
		t.Fatalf("notifierCalls=%d want 1", notifierCalls)
	}
	if len(queue.commands) != 2 {
		t.Fatalf("queue debe permitir reenfile idempotente: %+v", queue.commands)
	}
}

func validExternalWorkRunRequestForTestV0() StartExternalWorkRunRequestV0 {
	return StartExternalWorkRunRequestV0{
		AppChangeRequest: orquestaappchange.AppChangeRequestV0{
			ChangeRef:  "change-ref-001",
			AppRef:     "opes",
			UserIntent: "Resolver trabajo externo de OPES.",
			AcceptanceCriteria: []string{
				"markdown valido",
			},
			ExternalWork: &orquestaappchange.AppChangeExternalWorkV0{
				ProjectRef: "opes",
				JobRef:     "job-ref-001",
				WorkKind:   "draft_content_block",
				InputFields: []orquestadomainwork.DomainWorkFieldV0{{
					Name:  "topic_id",
					Value: "topic-ref-001",
				}},
			},
		},
	}
}

func orquestacoreRunForConflictTestV0(runRef string) orquestacoreworkflow.OrchestrationRunV0 {
	return orquestacoreworkflow.OrchestrationRunV0{
		SchemaVersion: orquestacoreworkflow.OrchestrationRunSchemaVersionV0,
		RunID:         runRef,
		ProjectRef:    "otro-proyecto",
		AppSpecRef:    "app-spec-external-work-otro-proyecto",
		Status:        orquestacoreworkflow.OrchestrationRunStatusActiveV0,
		CurrentPhase:  orquestacoreworkflow.OrchestrationPhaseProgramacionV0,
		Phases:        orquestacoreworkflow.OrchestrationPhaseCatalogV0(),
	}
}

type fakeExternalWorkRunNotifierV0 struct {
	calls *int
}

func (fake fakeExternalWorkRunNotifierV0) NotifyAppChangeRequestedV0(
	context.Context,
	orquestaappchange.AppChangeRecordV0,
) (orquestaappchange.AppChangeDirectorNotificationV0, error) {
	if fake.calls != nil {
		(*fake.calls)++
	}
	return orquestaappchange.AppChangeDirectorNotificationV0{
		DirectorQuestionRef: "question-ref-app-change-change-ref-001",
		EvidenceRefs:        []string{"evidence-ref-app-change"},
	}, nil
}

type fakeExternalWorkRunQueueV0 struct {
	commands []orquestarunqueue.RunQueuePriorityCommandV0
	err      error
}

func (queue *fakeExternalWorkRunQueueV0) SetRunPriorityV0(
	_ context.Context,
	command orquestarunqueue.RunQueuePriorityCommandV0,
) (orquestarunqueue.RunSchedulingCandidateV0, error) {
	if queue.err != nil {
		return orquestarunqueue.RunSchedulingCandidateV0{}, queue.err
	}
	queue.commands = append(queue.commands, command)
	return orquestarunqueue.RunSchedulingCandidateV0{
		RunRef:        command.RunRef,
		AppRef:        command.AppRef,
		PriorityScore: command.PriorityScore,
		UpdatedAt:     command.UpdatedAt,
	}, nil
}

func issueCodesV0(issues []ExternalWorkRunIssueV0) []string {
	out := make([]string, 0, len(issues))
	for _, issue := range issues {
		out = append(out, issue.Code)
	}
	return out
}
