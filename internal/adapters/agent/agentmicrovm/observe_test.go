package agentmicrovm

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

type observationClientStub struct {
	launchClientStub

	mu           sync.Mutex
	physical     microvm.RespuestaEjecucion
	physicalErr  error
	pages        map[uint64]microvm.PaginaEventosSesionTrabajoV1
	pageErr      error
	physicalHook func()
	pageHook     func()
	observedRefs []string
	queries      []microvm.ConsultaEventosSesionTrabajoV1
	sessionRefs  []string
}

func (client *observationClientStub) Observar(
	_ context.Context,
	reference string,
) (microvm.RespuestaEjecucion, error) {
	if client.physicalHook != nil {
		client.physicalHook()
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	client.observedRefs = append(client.observedRefs, reference)
	return client.physical, client.physicalErr
}

func (client *observationClientStub) LeerEventosSesion(
	_ context.Context,
	_ string,
	sessionRef string,
	query microvm.ConsultaEventosSesionTrabajoV1,
) (microvm.PaginaEventosSesionTrabajoV1, error) {
	if client.pageHook != nil {
		client.pageHook()
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	client.queries = append(client.queries, query)
	client.sessionRefs = append(client.sessionRefs, sessionRef)
	if client.pageErr != nil {
		return microvm.PaginaEventosSesionTrabajoV1{}, client.pageErr
	}
	return client.pages[query.DespuesDe], nil
}

func TestObserveAgentRebuildsTerminalArtifactAcrossPagesReplayAndRestart(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	client := validObservationClient(request)
	client.pages = successfulObservationPages(request, client.physical.Cerca)
	adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))

	first, err := adapter.ObserveAgent(context.Background(), request)
	if err != nil {
		t.Fatalf("ObserveAgent() first error = %v", err)
	}
	second, err := adapter.ObserveAgent(context.Background(), request)
	if err != nil {
		t.Fatalf("ObserveAgent() replay error = %v", err)
	}
	restarted := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
	third, err := restarted.ObserveAgent(context.Background(), request)
	if err != nil {
		t.Fatalf("ObserveAgent() after restart error = %v", err)
	}
	for index, observation := range []ports.AgentObservation{first, second, third} {
		if observation.Status != ports.AgentCompleted || observation.MediaType != request.ArtifactMediaType ||
			string(observation.Content) != "artifact-ok" || observation.ErrorCode != "" ||
			observation.ExecutionRef != request.ExecutionRef || observation.SpecHash != request.SpecHash ||
			observation.ObservedAt.IsZero() {
			t.Fatalf("observation %d = %+v", index, observation)
		}
		if err := ports.ValidateAgentObservation(observation, request.MaxOutputBytes); err != nil {
			t.Fatalf("ValidateAgentObservation(%d) = %v", index, err)
		}
	}
	if !sameObservationEvidence(first, second) || !sameObservationEvidence(second, third) {
		t.Fatalf("durable replay changed evidence: first=%+v second=%+v third=%+v", first, second, third)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	wantCursors := []uint64{0, 2, 0, 2, 0, 2}
	if len(client.queries) != len(wantCursors) {
		t.Fatalf("queries = %+v", client.queries)
	}
	for index, query := range client.queries {
		if query.DespuesDe != wantCursors[index] || query.Cerca != client.physical.Cerca ||
			query.MaximoEventos != microvm.MaximoEventosPaginaSesionV1 {
			t.Fatalf("query %d = %+v", index, query)
		}
		if client.sessionRefs[index] != request.SessionRef.String() {
			t.Fatalf("session ref %d = %q", index, client.sessionRefs[index])
		}
	}
}

func TestObserveAgentKeepsMissingAndAmbiguousSessionNonterminal(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	tests := []struct {
		name   string
		state  microvm.EstadoSesionTrabajoV1
		err    error
		status ports.AgentStatus
	}{
		{"missing", "", &microvm.ErrorRespuesta{Estado: 404, Codigo: "api.sesion_ausente"}, ports.AgentPending},
		{"admitted", microvm.EstadoSesionAdmitida, nil, ports.AgentPending},
		{"starting", microvm.EstadoSesionIniciando, nil, ports.AgentPending},
		{"active", microvm.EstadoSesionActiva, nil, ports.AgentRunning},
		{"ambiguous", microvm.EstadoSesionAmbigua, nil, ports.AgentRunning},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := validObservationClient(request)
			client.pageErr = test.err
			if test.err == nil {
				client.pages = map[uint64]microvm.PaginaEventosSesionTrabajoV1{
					0: emptyObservationPage(request, client.physical.Cerca, test.state),
				}
			}
			adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if err != nil || observation.Status != test.status || len(observation.Content) != 0 ||
				observation.ErrorCode != "" || observation.ObservedAt.IsZero() {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
		})
	}
}

func TestObserveAgentClassifiesReadOnlyFailuresWithoutInventingTerminalWork(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	tests := []struct {
		name      string
		physical  error
		session   error
		code      string
		temporary bool
	}{
		{"socket", errors.New("unix socket unavailable"), nil, CodeObservationUnavailable, true},
		{"physical missing", &microvm.ErrorRespuesta{Estado: 404, Codigo: "api.ejecucion_ausente"}, nil, CodeObservationUnavailable, true},
		{"stale fence", nil, &microvm.ErrorRespuesta{Estado: 409, Codigo: "api.cerca_obsoleta"}, CodeObservationUnavailable, true},
		{"protocol", &microvm.ErrorProtocolo{Recibido: "agentmicrovm.local.v0"}, nil, CodeProtocolIncompatible, false},
		{"invalid event", nil, &microvm.ErrorSesionTrabajoV1{Codigo: "sesion_trabajo.cursor_invalido"}, CodeObservationResponseInvalid, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := validObservationClient(request)
			client.physicalErr, client.pageErr = test.physical, test.session
			adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if ErrorCode(err) != test.code || isTemporary(err) != test.temporary ||
				!reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("observation=%+v err=%v code=%q temporary=%v", observation, err, ErrorCode(err), isTemporary(err))
			}
		})
	}
}

func TestObserveAgentReturnsContextCancellationBeforeTransportClassification(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	for _, stage := range []string{"physical", "events"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			client := validObservationClient(request)
			if stage == "physical" {
				client.physicalHook = cancel
				client.physicalErr = errors.New("transport returned after cancellation")
			} else {
				client.pageHook = cancel
				client.pageErr = errors.New("transport returned after cancellation")
			}
			adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
			observation, err := adapter.ObserveAgent(ctx, request)
			if !errors.Is(err, context.Canceled) || ErrorCode(err) != "" ||
				!reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("observation=%+v err=%v code=%q", observation, err, ErrorCode(err))
			}
		})
	}
}

func TestObserveAgentRejectsForeignIdentityAndBrokenCursor(t *testing.T) {
	launch := validLaunchRequest(t)
	valid := validObserveRequest(t, launch)
	t.Run("configured identity", func(t *testing.T) {
		request := valid
		request.ProviderRef = "provider:foreign"
		client := validObservationClient(valid)
		adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
		_, err := adapter.ObserveAgent(context.Background(), request)
		if ErrorCode(err) != CodeObservationRequestInvalid || len(client.observedRefs) != 0 {
			t.Fatalf("err=%v physical calls=%v", err, client.observedRefs)
		}
	})
	t.Run("physical ref", func(t *testing.T) {
		client := validObservationClient(valid)
		client.physical.Referencia = "ejecucion:foreign"
		adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
		_, err := adapter.ObserveAgent(context.Background(), valid)
		if ErrorCode(err) != CodeObservationResponseInvalid {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("session ref", func(t *testing.T) {
		client := validObservationClient(valid)
		page := emptyObservationPage(valid, client.physical.Cerca, microvm.EstadoSesionActiva)
		page.SesionRef = "execution-session:foreign"
		client.pages = map[uint64]microvm.PaginaEventosSesionTrabajoV1{0: page}
		adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
		_, err := adapter.ObserveAgent(context.Background(), valid)
		if ErrorCode(err) != CodeObservationResponseInvalid {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("cursor", func(t *testing.T) {
		client := validObservationClient(valid)
		pages := successfulObservationPages(valid, client.physical.Cerca)
		page := pages[2]
		page.DespuesDe = 1
		pages[2] = page
		client.pages = pages
		adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
		_, err := adapter.ObserveAgent(context.Background(), valid)
		if ErrorCode(err) != CodeObservationResponseInvalid {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestObserveAgentMapsOnlyDurableTerminalResult(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	zero, one, signal := int32(0), int32(1), int32(9)
	remoteCode := "sesion.motor_fallido"
	tests := []struct {
		name      string
		state     microvm.EstadoSesionTrabajoV1
		result    microvm.ResultadoTerminalSesionTrabajoV1
		content   string
		maxOutput int64
		status    ports.AgentStatus
		failure   string
	}{
		{"success", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero}, "ok", 64, ports.AgentCompleted, ""},
		{"remote failure", microvm.EstadoSesionFallida, microvm.ResultadoTerminalSesionTrabajoV1{CodigoError: &remoteCode}, "ignored", 64, ports.AgentFailed, CodeSessionFailed},
		{"truncated", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero, SalidaTruncada: true}, "partial", 64, ports.AgentFailed, CodeSessionOutputTruncated},
		{"exhausted", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero, Agotada: true}, "partial", 64, ports.AgentFailed, CodeSessionExhausted},
		{"signal", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{Senal: &signal}, "partial", 64, ports.AgentFailed, CodeSessionSignaled},
		{"exit", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &one}, "partial", 64, ports.AgentFailed, CodeSessionExitNonzero},
		{"oversize", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero}, "too-big", 3, ports.AgentFailed, CodeSessionOutputTooLarge},
		{"missing", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero}, "", 64, ports.AgentFailed, CodeSessionOutputMissing},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := request
			candidate.MaxOutputBytes = test.maxOutput
			client := validObservationClient(candidate)
			client.pages = terminalPage(candidate, client.physical.Cerca, test.state, test.result, test.content)
			adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
			observation, err := adapter.ObserveAgent(context.Background(), candidate)
			if err != nil || observation.Status != test.status || observation.ErrorCode != test.failure {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
			if test.status == ports.AgentFailed && len(observation.Content) != 0 {
				t.Fatalf("failed observation leaked output: %q", observation.Content)
			}
		})
	}
}

func TestObserveAgentRejectsIncompleteOrContradictoryTerminalResult(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	zero, signal := int32(0), int32(9)
	remoteCode := "sesion.motor_fallido"
	tests := []struct {
		name   string
		state  microvm.EstadoSesionTrabajoV1
		result microvm.ResultadoTerminalSesionTrabajoV1
	}{
		{"all nil", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{}},
		{"exit and signal", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero, Senal: &signal}},
		{"successful state with error", microvm.EstadoSesionFinalizada, microvm.ResultadoTerminalSesionTrabajoV1{CodigoError: &remoteCode}},
		{"failed state without error", microvm.EstadoSesionFallida, microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := validObservationClient(request)
			client.pages = terminalPage(request, client.physical.Cerca, test.state, test.result, "partial")
			adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if ErrorCode(err) != CodeObservationResponseInvalid ||
				!reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
		})
	}
}

func TestObserveAgentRejectsTruncationEventContradictedByTerminalReceiptAcrossReplay(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	zero := int32(0)
	client := validObservationClient(request)
	client.pages = map[uint64]microvm.PaginaEventosSesionTrabajoV1{
		0: observationPage(request, client.physical.Cerca, microvm.EstadoSesionActiva, 0, false, []microvm.EventoSesionTrabajoV1{
			observationEvent(request, client.physical.Cerca, 1, microvm.TipoEventoSesionIniciada, microvm.EstadoSesionActiva, "", nil),
			observationEvent(request, client.physical.Cerca, 2, microvm.TipoEventoSesionSalidaTruncada, microvm.EstadoSesionActiva, "", nil),
		}),
		2: observationPage(request, client.physical.Cerca, microvm.EstadoSesionFinalizada, 2, true, []microvm.EventoSesionTrabajoV1{
			observationEvent(request, client.physical.Cerca, 3, microvm.TipoEventoSesionStdout, microvm.EstadoSesionActiva, "partial", nil),
			observationEvent(request, client.physical.Cerca, 4, microvm.TipoEventoSesionFinalizada, microvm.EstadoSesionFinalizada, "", &microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero}),
		}),
	}
	adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))
	for attempt := 0; attempt < 2; attempt++ {
		observation, err := adapter.ObserveAgent(context.Background(), request)
		if ErrorCode(err) != CodeObservationResponseInvalid ||
			!reflect.DeepEqual(observation, ports.AgentObservation{}) {
			t.Fatalf("attempt %d observation=%+v err=%v", attempt, observation, err)
		}
	}
}

func TestObserveAgentConcurrentReplayIsRaceFree(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validObserveRequest(t, launch)
	client := validObservationClient(request)
	client.pages = successfulObservationPages(request, client.physical.Cerca)
	adapter := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false))

	const workers = 32
	var wait sync.WaitGroup
	errorsCh := make(chan error, workers)
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if err == nil && (observation.Status != ports.AgentCompleted || string(observation.Content) != "artifact-ok") {
				err = errors.New("unexpected observation")
			}
			errorsCh <- err
		}()
	}
	wait.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func validObserveRequest(t *testing.T, launch ports.AgentLaunchRequest) ports.AgentObserveRequest {
	t.Helper()
	return ports.AgentObserveRequest{
		ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef,
		PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration,
		ExecutionAttempt: launch.ExecutionAttempt, SpecHash: launch.SpecHash,
		ProviderRef: "provider:codex", ModelRef: "model:codex-microvm", AgentRef: "agent:codex-microvm",
		ExternalRef: "ejecucion:" + strings.Repeat("a", 64), SessionRef: launch.SessionRef,
		ArtifactMediaType: launch.ArtifactMediaType, MaxOutputBytes: launch.MaxOutputBytes,
	}
}

func validObservationClient(request ports.AgentObserveRequest) *observationClientStub {
	physical := microvm.RespuestaEjecucion{
		Referencia: request.ExternalRef, Estado: "disponible", Revision: 8, Cerca: 7,
	}
	return &observationClientStub{
		launchClientStub: launchClientStub{capabilities: validRemoteCapabilities()},
		physical:         physical, pages: make(map[uint64]microvm.PaginaEventosSesionTrabajoV1),
	}
}

func successfulObservationPages(
	request ports.AgentObserveRequest,
	fence uint64,
) map[uint64]microvm.PaginaEventosSesionTrabajoV1 {
	zero := int32(0)
	return map[uint64]microvm.PaginaEventosSesionTrabajoV1{
		0: observationPage(request, fence, microvm.EstadoSesionActiva, 0, false, []microvm.EventoSesionTrabajoV1{
			observationEvent(request, fence, 1, microvm.TipoEventoSesionIniciada, microvm.EstadoSesionActiva, "", nil),
			observationEvent(request, fence, 2, microvm.TipoEventoSesionStdout, microvm.EstadoSesionActiva, "artifact-", nil),
		}),
		2: observationPage(request, fence, microvm.EstadoSesionFinalizada, 2, true, []microvm.EventoSesionTrabajoV1{
			observationEvent(request, fence, 3, microvm.TipoEventoSesionStdout, microvm.EstadoSesionActiva, "ok", nil),
			observationEvent(request, fence, 4, microvm.TipoEventoSesionFinalizada, microvm.EstadoSesionFinalizada, "", &microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &zero}),
		}),
	}
}

func terminalPage(
	request ports.AgentObserveRequest,
	fence uint64,
	state microvm.EstadoSesionTrabajoV1,
	result microvm.ResultadoTerminalSesionTrabajoV1,
	content string,
) map[uint64]microvm.PaginaEventosSesionTrabajoV1 {
	events := make([]microvm.EventoSesionTrabajoV1, 0, 2)
	if content != "" {
		events = append(events, observationEvent(request, fence, 1, microvm.TipoEventoSesionStdout, microvm.EstadoSesionActiva, content, nil))
	}
	if result.SalidaTruncada {
		events = append(events, observationEvent(request, fence, uint64(len(events)+1), microvm.TipoEventoSesionSalidaTruncada, microvm.EstadoSesionActiva, "", nil))
	}
	events = append(events, observationEvent(request, fence, uint64(len(events)+1), microvm.TipoEventoSesionFinalizada, state, "", &result))
	return map[uint64]microvm.PaginaEventosSesionTrabajoV1{
		0: observationPage(request, fence, state, 0, true, events),
	}
}

func emptyObservationPage(
	request ports.AgentObserveRequest,
	fence uint64,
	state microvm.EstadoSesionTrabajoV1,
) microvm.PaginaEventosSesionTrabajoV1 {
	return microvm.PaginaEventosSesionTrabajoV1{
		EjecucionRef: request.ExternalRef, SesionRef: request.SessionRef.String(), Estado: state,
		Revision: 3, RevisionTrabajo: 1, RevisionSesion: 1, Cerca: fence,
		DespuesDe: 0, Eventos: []microvm.EventoSesionTrabajoV1{}, SiguienteCursor: 0,
	}
}

func observationPage(
	request ports.AgentObserveRequest,
	fence uint64,
	state microvm.EstadoSesionTrabajoV1,
	after uint64,
	terminal bool,
	events []microvm.EventoSesionTrabajoV1,
) microvm.PaginaEventosSesionTrabajoV1 {
	lastRevision, lastWork, lastSession := uint64(3), uint64(1), uint64(1)
	if len(events) > 0 {
		last := events[len(events)-1]
		lastRevision, lastWork, lastSession = last.Revision, last.RevisionTrabajo, last.RevisionSesion
	}
	return microvm.PaginaEventosSesionTrabajoV1{
		EjecucionRef: request.ExternalRef, SesionRef: request.SessionRef.String(), Estado: state,
		Revision: lastRevision, RevisionTrabajo: lastWork, RevisionSesion: lastSession, Cerca: fence,
		DespuesDe: after, Eventos: events, SiguienteCursor: after + uint64(len(events)), Terminal: terminal,
	}
}

func observationEvent(
	request ports.AgentObserveRequest,
	fence uint64,
	sequence uint64,
	kind microvm.TipoEventoSesionTrabajoV1,
	state microvm.EstadoSesionTrabajoV1,
	content string,
	result *microvm.ResultadoTerminalSesionTrabajoV1,
) microvm.EventoSesionTrabajoV1 {
	return microvm.EventoSesionTrabajoV1{
		SesionRef: request.SessionRef.String(), Secuencia: sequence, Tipo: kind, Estado: state,
		Revision: sequence + 2, RevisionTrabajo: 1 + sequence/4, RevisionSesion: sequence,
		Cerca: fence, ContenidoBase64: base64.StdEncoding.EncodeToString([]byte(content)),
		Terminal: kind == microvm.TipoEventoSesionFinalizada, Resultado: result,
	}
}

func sameObservationEvidence(left ports.AgentObservation, right ports.AgentObservation) bool {
	right.ObservedAt = left.ObservedAt
	return reflect.DeepEqual(left, right)
}
