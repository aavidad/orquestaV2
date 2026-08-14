package agentmicrovm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/application"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

var _ application.AgentStopReconciler = (*DockerAdapter)(nil)

func TestDockerAdapterAdvertisesOnlyNegotiatedForcedStop(t *testing.T) {
	fixture := newDockerAdapterFixture(t)
	capabilities, err := fixture.adapter.ControlCapabilities(context.Background())
	if err != nil || capabilities != (ports.AgentControlCapabilities{ForcedStop: true}) ||
		fixture.client.negotiations != 1 ||
		!slices.Contains(fixture.client.capabilities.Operaciones, "detener_contenedor") {
		t.Fatalf("capabilities=%+v negotiations=%d err=%v", capabilities, fixture.client.negotiations, err)
	}
	fixture.client.capabilityErr = errors.New("unavailable")
	if got, err := fixture.adapter.ControlCapabilities(context.Background()); got != (ports.AgentControlCapabilities{}) || ErrorCode(err) != CodeCapabilitiesUnavailable {
		t.Fatalf("failed capabilities=%+v err=%v", got, err)
	}
}

func TestDockerAdapterStopPersistsExactBytesBeforeEffectAndReplays(t *testing.T) {
	fixture, request := newDockerStopFixture(t)
	first, firstErr := fixture.adapter.Stop(context.Background(), request)
	reopened := fixture.stopJournal.reopen()
	fixture.stopJournal, fixture.client.stopJournal = reopened, reopened
	fixture.adapter = fixture.mustAdapter(t, fixture.journal)
	second, secondErr := fixture.adapter.Stop(context.Background(), request)
	wantEvents := []string{"resolve:launch", "resolve:stop", "record:stop", "effect:stop",
		"resolve:launch", "resolve:stop", "effect:stop"}
	if firstErr != nil || secondErr != nil || !reflect.DeepEqual(first, second) ||
		!reflect.DeepEqual(fixture.events, wantEvents) || fixture.client.stopCalls != 2 ||
		!bytes.Equal(fixture.client.stopBodies[0], fixture.client.stopBodies[1]) ||
		ports.ValidateAgentStopReceipt(request, first) != nil || first.Status != ports.AgentStopped ||
		first.ConfirmedAt != fixture.now {
		t.Fatalf("errors=%v/%v receipts=%+v/%+v events=%v calls=%d", firstErr, secondErr,
			first, second, fixture.events, fixture.client.stopCalls)
	}
	stored := fixture.stopJournal.mustRecord(t, request)
	if stored.IdempotencyKey == request.IdempotencyKey || len(stored.IdempotencyKey) > 128 ||
		!bytes.Equal(stored.Body, fixture.client.stopBodies[0]) || stored.TargetRef != request.ExternalRef {
		t.Fatalf("stored=%+v application_key=%q", stored, request.IdempotencyKey)
	}
	var physical microvm.SolicitudDetenerContenedorV1
	if json.Unmarshal(stored.Body, &physical) != nil || physical.RevisionEsperada != stored.ExpectedRevision ||
		physical.Cerca != request.StopActionFence {
		t.Fatalf("physical=%+v stored=%+v", physical, stored)
	}
}

func TestDockerAdapterReconcileStopReadsExactJournalsWithoutRepeatingMutation(t *testing.T) {
	fixture, request := newDockerStopFixture(t)
	firstStop, err := fixture.adapter.Stop(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	stored := fixture.stopJournal.mustRecord(t, request)
	var physicalRequest microvm.SolicitudDetenerContenedorV1
	if json.Unmarshal(stored.Body, &physicalRequest) != nil {
		t.Fatal("invalid durable stop body")
	}
	client := &dockerObservationClientStub{dockerClientStub: fixture.client,
		physical: validDockerStoppedResponse(fixture.request, stored.TargetRef, physicalRequest)}
	adapter := mustDockerObserverAdapter(t, fixture, client)
	first, firstErr := adapter.ReconcileStop(context.Background(), request)

	fixture.journal, fixture.stopJournal = fixture.journal.reopen(), fixture.stopJournal.reopen()
	fixture.client.stopJournal = fixture.stopJournal
	restarted := mustDockerObserverAdapter(t, fixture, client)
	second, secondErr := restarted.ReconcileStop(context.Background(), request)
	if firstErr != nil || secondErr != nil || !reflect.DeepEqual(first, second) ||
		!reflect.DeepEqual(first, firstStop) || ports.ValidateAgentStopReceipt(request, first) != nil ||
		fixture.client.stopCalls != 1 || !reflect.DeepEqual(client.observed, []string{stored.TargetRef, stored.TargetRef}) {
		t.Fatalf("receipts=%+v/%+v stop=%+v errors=%v/%v stops=%d observed=%v",
			first, second, firstStop, firstErr, secondErr, fixture.client.stopCalls, client.observed)
	}
}

func TestDockerAdapterReconcileStopReportsExactPendingStateWithoutMutation(t *testing.T) {
	fixture, request := newDockerStopFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	fixture.stopJournal.afterRecord = cancel
	if _, err := fixture.adapter.Stop(ctx, request); ErrorCode(err) != CodeDockerStopCanceledBeforeSubmit {
		t.Fatalf("seed journal error=%v", err)
	}
	fixture.stopJournal.afterRecord = nil
	compiled := mustCompileDocker(t, fixture.request, fixture.binding)
	client := &dockerObservationClientStub{dockerClientStub: fixture.client,
		physical: validDockerContainerResponse(compiled.Plan, true)}
	adapter := mustDockerObserverAdapter(t, fixture, client)
	receipt, err := adapter.ReconcileStop(context.Background(), request)
	if err != nil || receipt.Status != ports.AgentStopPending ||
		ports.ValidateAgentStopReceipt(request, receipt) != nil || fixture.client.stopCalls != 0 ||
		!reflect.DeepEqual(client.observed, []string{request.ExternalRef}) {
		t.Fatalf("receipt=%+v err=%v stops=%d observed=%v", receipt, err, fixture.client.stopCalls, client.observed)
	}
}

func TestDockerAdapterReconcileStopFailsClosedBeforeAnySecondMutation(t *testing.T) {
	t.Run("missing durable stop", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		compiled := mustCompileDocker(t, fixture.request, fixture.binding)
		client := &dockerObservationClientStub{dockerClientStub: fixture.client,
			physical: validDockerContainerResponse(compiled.Plan, true)}
		_, err := mustDockerObserverAdapter(t, fixture, client).ReconcileStop(context.Background(), request)
		if ErrorCode(err) != CodeDockerStopNotRecorded || !isDefinitelyNotApplied(err) ||
			fixture.client.stopCalls != 0 || len(client.observed) != 0 {
			t.Fatalf("err=%v stops=%d observed=%v", err, fixture.client.stopCalls, client.observed)
		}
	})
	t.Run("crossed durable stop", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		if _, err := fixture.adapter.Stop(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		stored := fixture.stopJournal.mustRecord(t, request)
		stored.ExpectedRevision++
		fixture.stopJournal.records[stored.Key] = stored
		client := &dockerObservationClientStub{dockerClientStub: fixture.client}
		_, err := mustDockerObserverAdapter(t, fixture, client).ReconcileStop(context.Background(), request)
		if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.stopCalls != 1 || len(client.observed) != 0 {
			t.Fatalf("err=%v stops=%d observed=%v", err, fixture.client.stopCalls, client.observed)
		}
	})
	for _, test := range []struct {
		name   string
		mutate func(*microvm.RespuestaContenedorV1)
	}{
		{"crossed stopped target", func(value *microvm.RespuestaContenedorV1) {
			value.Referencia, value.Identidad.ExecutionLabel = "ejecucion:other", "ejecucion:other"
		}},
		{"crossed active fence", func(value *microvm.RespuestaContenedorV1) {
			makeDockerStopResponseActive(value)
			value.Cerca++
		}},
		{"crossed active generation", func(value *microvm.RespuestaContenedorV1) {
			makeDockerStopResponseActive(value)
			value.Identidad.Generation++
		}},
		{"coherent crossed active fence", func(value *microvm.RespuestaContenedorV1) {
			makeDockerStopResponseActive(value)
			value.Cerca++
			value.Identidad.Generation = value.Cerca
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := newDockerStopFixture(t)
			if _, err := fixture.adapter.Stop(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			stored := fixture.stopJournal.mustRecord(t, request)
			var physicalRequest microvm.SolicitudDetenerContenedorV1
			if json.Unmarshal(stored.Body, &physicalRequest) != nil {
				t.Fatal("invalid durable stop body")
			}
			physical := validDockerStoppedResponse(fixture.request, stored.TargetRef, physicalRequest)
			test.mutate(&physical)
			client := &dockerObservationClientStub{dockerClientStub: fixture.client, physical: physical}
			_, err := mustDockerObserverAdapter(t, fixture, client).ReconcileStop(context.Background(), request)
			if ErrorCode(err) != CodeDockerStopResponseInvalid || fixture.client.stopCalls != 1 || len(client.observed) != 1 {
				t.Fatalf("err=%v stops=%d observed=%v", err, fixture.client.stopCalls, client.observed)
			}
		})
	}
}

func makeDockerStopResponseActive(value *microvm.RespuestaContenedorV1) {
	value.Estado = "activo"
	value.Revision -= 2
	value.Cerca--
	value.Identidad.Generation--
	*value.RecursoVivo, *value.EstadoMotor = true, "running"
}

func TestDockerAdapterReconcileStopPreservesReadOnlyFailureAndCancellation(t *testing.T) {
	for _, test := range []struct {
		name     string
		physical error
		cancel   bool
	}{
		{"transport", errors.New("observation unavailable"), false},
		{"success after cancel", nil, true},
		{"error after cancel", errors.New("observation after cancel"), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := newDockerStopFixture(t)
			if _, err := fixture.adapter.Stop(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			stored := fixture.stopJournal.mustRecord(t, request)
			var physicalRequest microvm.SolicitudDetenerContenedorV1
			if json.Unmarshal(stored.Body, &physicalRequest) != nil {
				t.Fatal("invalid durable stop body")
			}
			client := &dockerObservationClientStub{dockerClientStub: fixture.client,
				physical:    validDockerStoppedResponse(fixture.request, stored.TargetRef, physicalRequest),
				physicalErr: test.physical}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test.cancel {
				client.physicalHook = cancel
			}
			receipt, err := mustDockerObserverAdapter(t, fixture, client).ReconcileStop(ctx, request)
			if test.cancel {
				if !errors.Is(err, context.Canceled) || ErrorCode(err) != "" {
					t.Fatalf("cancellation=%v code=%q", err, ErrorCode(err))
				}
			} else if ErrorCode(err) != CodeObservationUnavailable || !isTemporary(err) {
				t.Fatalf("transport=%v code=%q temporary=%t", err, ErrorCode(err), isTemporary(err))
			}
			if !reflect.DeepEqual(receipt, ports.AgentStopReceipt{}) || fixture.client.stopCalls != 1 || len(client.observed) != 1 {
				t.Fatalf("receipt=%+v stops=%d observed=%v", receipt, fixture.client.stopCalls, client.observed)
			}
		})
	}
}

func TestDockerAdapterStopRejectsCrossedAuthorityAndUnsupportedModeBeforeEffect(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*dockerAdapterFixture, *ports.AgentStopRequest)
	}{
		{"provider", func(_ *dockerAdapterFixture, request *ports.AgentStopRequest) { request.ProviderRef = "provider:other" }},
		{"model", func(_ *dockerAdapterFixture, request *ports.AgentStopRequest) { request.ModelRef = "model:other" }},
		{"agent", func(_ *dockerAdapterFixture, request *ports.AgentStopRequest) { request.AgentRef = "agent:other" }},
		{"target", func(_ *dockerAdapterFixture, request *ports.AgentStopRequest) {
			request.ExternalRef = "ejecucion:other"
		}},
		{"launch fence", func(_ *dockerAdapterFixture, request *ports.AgentStopRequest) { request.LaunchActionFence++ }},
		{"missing launch", func(fixture *dockerAdapterFixture, request *ports.AgentStopRequest) {
			delete(fixture.journal.records, ports.AgentProviderRequestKey{ExecutionRef: request.ExecutionRef,
				ActionFence: request.LaunchActionFence, Stage: ports.AgentProviderRequestLaunch})
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := newDockerStopFixture(t)
			test.mutate(fixture, &request)
			_, err := fixture.adapter.Stop(context.Background(), request)
			if ErrorCode(err) != CodeDockerStopRequestInvalid || fixture.client.stopCalls != 0 ||
				len(fixture.stopJournal.records) != 0 {
				t.Fatalf("err=%v calls=%d records=%d", err, fixture.client.stopCalls, len(fixture.stopJournal.records))
			}
		})
	}
	fixture, request := newDockerStopFixture(t)
	request.Mode = ports.AgentStopCooperative
	receipt, err := fixture.adapter.Stop(context.Background(), request)
	if err != nil || receipt.Status != ports.AgentStopUnsupported ||
		ports.ValidateAgentStopReceipt(request, receipt) != nil || fixture.client.stopCalls != 0 ||
		len(fixture.stopJournal.records) != 0 {
		t.Fatalf("unsupported receipt=%+v err=%v", receipt, err)
	}
}

func TestDockerAdapterStopJournalFailureAndCancellationNeverReachProvider(t *testing.T) {
	t.Run("failure", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		fixture.stopJournal.recordErr = errors.New("journal unavailable")
		_, err := fixture.adapter.Stop(context.Background(), request)
		if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.stopCalls != 0 {
			t.Fatalf("err=%v calls=%d", err, fixture.client.stopCalls)
		}
	})
	t.Run("crossed record result", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		fixture.stopJournal.corruptResult = func(value *ports.AgentProviderStopRequest) { value.ExpectedRevision++ }
		_, err := fixture.adapter.Stop(context.Background(), request)
		if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.stopCalls != 0 {
			t.Fatalf("err=%v calls=%d", err, fixture.client.stopCalls)
		}
	})
	t.Run("crossed replay", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		if _, err := fixture.adapter.Stop(context.Background(), request); err != nil {
			t.Fatal(err)
		}
		fixture.client.stopCalls = 0
		request.IdempotencyKey += ":crossed"
		_, err := fixture.adapter.Stop(context.Background(), request)
		if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.stopCalls != 0 {
			t.Fatalf("err=%v calls=%d", err, fixture.client.stopCalls)
		}
	})
	t.Run("resolve failure", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		fixture.stopJournal.resolveErr = errors.New("journal unavailable")
		_, err := fixture.adapter.Stop(context.Background(), request)
		if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.stopCalls != 0 {
			t.Fatalf("err=%v calls=%d", err, fixture.client.stopCalls)
		}
	})
	t.Run("cancel after journal", func(t *testing.T) {
		fixture, request := newDockerStopFixture(t)
		ctx, cancel := context.WithCancel(context.Background())
		fixture.stopJournal.afterRecord = cancel
		_, err := fixture.adapter.Stop(ctx, request)
		if ErrorCode(err) != CodeDockerStopCanceledBeforeSubmit || !errors.Is(err, context.Canceled) ||
			!isDefinitelyNotApplied(err) || fixture.client.stopCalls != 0 || len(fixture.stopJournal.records) != 1 {
			t.Fatalf("err=%v calls=%d records=%d", err, fixture.client.stopCalls, len(fixture.stopJournal.records))
		}
	})
}

func TestDockerAdapterStopRejectsEveryCrossedDurableReplayBeforeEffect(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ports.AgentProviderStopRequest)
	}{
		{"execution key", func(value *ports.AgentProviderStopRequest) {
			other, err := goal.NewExecutionRef("execution:docker-stop-other")
			if err != nil {
				t.Fatal(err)
			}
			value.Key.ExecutionRef = other
		}},
		{"launch fence key", func(value *ports.AgentProviderStopRequest) { value.Key.LaunchActionFence++ }},
		{"stop fence key", func(value *ports.AgentProviderStopRequest) { value.Key.StopActionFence++ }},
		{"attempt", func(value *ports.AgentProviderStopRequest) { value.StopEffectAttemptRef = "effect-attempt:other" }},
		{"provider", func(value *ports.AgentProviderStopRequest) { value.ProviderRef = "provider:other" }},
		{"physical key", func(value *ports.AgentProviderStopRequest) { value.IdempotencyKey += "x" }},
		{"target", func(value *ports.AgentProviderStopRequest) { value.TargetRef = "ejecucion:other" }},
		{"revision", func(value *ports.AgentProviderStopRequest) { value.ExpectedRevision++ }},
		{"body sha", func(value *ports.AgentProviderStopRequest) { value.BodySHA256 = strings.Repeat("0", 64) }},
		{"body revision", func(value *ports.AgentProviderStopRequest) {
			mutateDockerStopPhysical(t, value, func(physical *microvm.SolicitudDetenerContenedorV1) {
				physical.RevisionEsperada++
			})
		}},
		{"body fence", func(value *ports.AgentProviderStopRequest) {
			mutateDockerStopPhysical(t, value, func(physical *microvm.SolicitudDetenerContenedorV1) {
				physical.Cerca++
			})
		}},
		{"non canonical body", func(value *ports.AgentProviderStopRequest) {
			value.Body = append(value.Body, ' ')
			value.BodySHA256 = ports.AgentProviderRequestBodySHA256(value.Body)
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := newDockerStopFixture(t)
			if _, err := fixture.adapter.Stop(context.Background(), request); err != nil {
				t.Fatal(err)
			}
			stored := fixture.stopJournal.mustRecord(t, request)
			test.mutate(&stored)
			key := ports.AgentProviderStopRequestKey{ExecutionRef: request.ExecutionRef,
				LaunchActionFence: request.LaunchActionFence, StopActionFence: request.StopActionFence}
			fixture.stopJournal.records[key] = ports.CloneAgentProviderStopRequest(stored)
			fixture.stopJournal.byPhysicalKey[stored.IdempotencyKey] = ports.CloneAgentProviderStopRequest(stored)
			fixture.client.stopCalls = 0
			_, err := fixture.adapter.Stop(context.Background(), request)
			if ErrorCode(err) != CodeDockerJournalFailed || fixture.client.stopCalls != 0 {
				t.Fatalf("err=%v calls=%d stored=%+v", err, fixture.client.stopCalls, stored)
			}
		})
	}
}

func mutateDockerStopPhysical(t *testing.T, value *ports.AgentProviderStopRequest,
	mutate func(*microvm.SolicitudDetenerContenedorV1)) {
	t.Helper()
	var physical microvm.SolicitudDetenerContenedorV1
	if json.Unmarshal(value.Body, &physical) != nil {
		t.Fatal("invalid fixture stop body")
	}
	mutate(&physical)
	var err error
	value.Body, err = microvm.CodificarSolicitudDetenerContenedorV1(physical)
	if err != nil {
		t.Fatalf("invalid mutated stop body: %v", err)
	}
	value.BodySHA256 = ports.AgentProviderRequestBodySHA256(value.Body)
}

func TestDockerAdapterStopRejectsEveryCrossedPhysicalReceipt(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*microvm.RespuestaContenedorV1)
	}{
		{"target", func(value *microvm.RespuestaContenedorV1) { value.Referencia = "ejecucion:other" }},
		{"state", func(value *microvm.RespuestaContenedorV1) { value.Estado = "activo" }},
		{"revision", func(value *microvm.RespuestaContenedorV1) { value.Revision++ }},
		{"stop fence", func(value *microvm.RespuestaContenedorV1) { value.Cerca++ }},
		{"vcpu", func(value *microvm.RespuestaContenedorV1) { value.VCPU++ }},
		{"memory", func(value *microvm.RespuestaContenedorV1) { value.MemoriaMiB++ }},
		{"missing liveness", func(value *microvm.RespuestaContenedorV1) { value.RecursoVivo = nil }},
		{"liveness", func(value *microvm.RespuestaContenedorV1) { *value.RecursoVivo = true }},
		{"missing engine", func(value *microvm.RespuestaContenedorV1) { value.EstadoMotor = nil }},
		{"engine", func(value *microvm.RespuestaContenedorV1) { *value.EstadoMotor = "running" }},
		{"pid", func(value *microvm.RespuestaContenedorV1) { value.Identidad.PIDObservado = 0 }},
		{"execution", func(value *microvm.RespuestaContenedorV1) { value.Identidad.ExecutionLabel = "ejecucion:other" }},
		{"run", func(value *microvm.RespuestaContenedorV1) { value.Identidad.RunLabel = "execution:other" }},
		{"generation", func(value *microvm.RespuestaContenedorV1) { value.Identidad.Generation++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := newDockerStopFixture(t)
			fixture.client.mutateStop = test.mutate
			_, err := fixture.adapter.Stop(context.Background(), request)
			if ErrorCode(err) != CodeDockerStopResponseInvalid || fixture.client.stopCalls != 1 {
				t.Fatalf("err=%v calls=%d", err, fixture.client.stopCalls)
			}
		})
	}
}

func TestDockerAdapterStopClassifiesProviderFailureConservatively(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{"rejected", &microvm.ErrorRespuesta{Estado: 409, Codigo: "conflict"}, CodeDockerStopRejected},
		{"unavailable", errors.New("lost acknowledgement"), CodeDockerStopUnavailable},
		{"canceled", context.Canceled, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, request := newDockerStopFixture(t)
			fixture.client.stopErr = test.err
			_, err := fixture.adapter.Stop(context.Background(), request)
			if ErrorCode(err) != test.code || isDefinitelyNotApplied(err) ||
				(test.code == CodeDockerStopUnavailable) != isTemporary(err) {
				t.Fatalf("err=%v temporary=%t definite=%t", err, isTemporary(err), isDefinitelyNotApplied(err))
			}
			if test.code == "" && !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation lost: %v", err)
			}
		})
	}
}

func newDockerStopFixture(t *testing.T) (*dockerAdapterFixture, ports.AgentStopRequest) {
	t.Helper()
	fixture := newDockerAdapterFixture(t)
	launch, err := fixture.adapter.Launch(context.Background(), fixture.request)
	if err != nil {
		t.Fatal(err)
	}
	fixture.events = nil
	return fixture, ports.AgentStopRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, LaunchActionFence: launch.LaunchActionFence,
		StopEffectAttemptRef: "effect-attempt:docker-stop", StopActionFence: launch.LaunchActionFence + 1,
		SpecHash: launch.SpecHash, ProviderRef: launch.ProviderRef, ModelRef: launch.ModelRef,
		AgentRef: launch.AgentRef, ExternalRef: launch.ExternalRef, Mode: ports.AgentStopForced,
		IdempotencyKey: "stop-application:" + strings.Repeat("opaque/", 32),
	}
}

type dockerStopJournalStub struct {
	records       map[ports.AgentProviderStopRequestKey]ports.AgentProviderStopRequest
	byPhysicalKey map[string]ports.AgentProviderStopRequest
	events        *[]string
	recordErr     error
	resolveErr    error
	afterRecord   func()
	corruptResult func(*ports.AgentProviderStopRequest)
}

func newDockerStopJournalStub(events *[]string) *dockerStopJournalStub {
	return &dockerStopJournalStub{records: make(map[ports.AgentProviderStopRequestKey]ports.AgentProviderStopRequest),
		byPhysicalKey: make(map[string]ports.AgentProviderStopRequest), events: events}
}

func (journal *dockerStopJournalStub) RecordAgentProviderStopRequest(
	_ context.Context,
	request ports.AgentProviderStopRequest,
) (ports.AgentProviderStopRequest, error) {
	*journal.events = append(*journal.events, "record:stop")
	if journal.recordErr != nil {
		return ports.AgentProviderStopRequest{}, journal.recordErr
	}
	if ports.ValidateAgentProviderStopRequest(request) != nil {
		return ports.AgentProviderStopRequest{}, errors.New("invalid stop request")
	}
	if existing, found := journal.records[request.Key]; found {
		if !reflect.DeepEqual(existing, request) {
			return ports.AgentProviderStopRequest{}, errors.New("divergent stop replay")
		}
		return ports.CloneAgentProviderStopRequest(existing), nil
	}
	stored := ports.CloneAgentProviderStopRequest(request)
	journal.records[request.Key], journal.byPhysicalKey[request.IdempotencyKey] = stored, stored
	if journal.afterRecord != nil {
		journal.afterRecord()
	}
	result := ports.CloneAgentProviderStopRequest(stored)
	if journal.corruptResult != nil {
		journal.corruptResult(&result)
	}
	return result, nil
}

func (journal *dockerStopJournalStub) ResolveAgentProviderStopRequest(
	_ context.Context,
	key ports.AgentProviderStopRequestKey,
) (ports.AgentProviderStopRequest, bool, error) {
	*journal.events = append(*journal.events, "resolve:stop")
	if journal.resolveErr != nil {
		return ports.AgentProviderStopRequest{}, false, journal.resolveErr
	}
	request, found := journal.records[key]
	return ports.CloneAgentProviderStopRequest(request), found, nil
}

func (journal *dockerStopJournalStub) mustRecord(
	t *testing.T,
	request ports.AgentStopRequest,
) ports.AgentProviderStopRequest {
	t.Helper()
	key := ports.AgentProviderStopRequestKey{ExecutionRef: request.ExecutionRef,
		LaunchActionFence: request.LaunchActionFence, StopActionFence: request.StopActionFence}
	stored, found := journal.records[key]
	if !found || ports.ValidateAgentProviderStopRequest(stored) != nil {
		t.Fatalf("missing stop journal: %+v", stored)
	}
	return ports.CloneAgentProviderStopRequest(stored)
}

func (journal *dockerStopJournalStub) reopen() *dockerStopJournalStub {
	reopened := newDockerStopJournalStub(journal.events)
	for key, request := range journal.records {
		stored := ports.CloneAgentProviderStopRequest(request)
		reopened.records[key], reopened.byPhysicalKey[stored.IdempotencyKey] = stored, stored
	}
	return reopened
}

func validDockerStoppedResponse(
	launch ports.AgentLaunchRequest,
	target string,
	request microvm.SolicitudDetenerContenedorV1,
) microvm.RespuestaContenedorV1 {
	binding := validDockerPhysicalBinding(launch)
	alive, state := false, "removed"
	return microvm.RespuestaContenedorV1{
		Referencia: target, Estado: "detenido", Revision: request.RevisionEsperada + 2,
		Cerca: request.Cerca, VCPU: binding.VCPU, MemoriaMiB: binding.MemoryMiB,
		Identidad: &microvm.IdentidadContenedorV1{ContainerID: strings.Repeat("1", 64), PIDObservado: 1234,
			OwnerLabel: "v1", ExecutionLabel: target, RunLabel: launch.ExecutionRef.String(), Generation: request.Cerca},
		RecursoVivo: &alive, EstadoMotor: &state,
	}
}
