package agentmicrovm

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"testing"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

func (*dockerClientStub) ObservarContenedor(context.Context, string) (microvm.RespuestaContenedorV1, error) {
	return microvm.RespuestaContenedorV1{}, errors.New("unexpected Docker observation")
}

func (*dockerClientStub) LeerEventosSesionContenedor(context.Context, string, string, microvm.ConsultaEventosSesionContenedorV1) (microvm.RespuestaEventosSesionContenedorV1, error) {
	return microvm.RespuestaEventosSesionContenedorV1{}, errors.New("unexpected Docker event read")
}

type dockerObservationClientStub struct {
	*dockerClientStub
	physical               microvm.RespuestaContenedorV1
	pages                  map[uint64]microvm.RespuestaEventosSesionContenedorV1
	physicalErr, pageErr   error
	physicalHook, pageHook func()
	observed, sessionRefs  []string
	queries                []microvm.ConsultaEventosSesionContenedorV1
}

func (client *dockerObservationClientStub) ObservarContenedor(_ context.Context, reference string) (microvm.RespuestaContenedorV1, error) {
	if client.physicalHook != nil {
		client.physicalHook()
	}
	client.observed = append(client.observed, reference)
	return client.physical, client.physicalErr
}

func (client *dockerObservationClientStub) LeerEventosSesionContenedor(_ context.Context, _ string, sessionRef string,
	query microvm.ConsultaEventosSesionContenedorV1) (microvm.RespuestaEventosSesionContenedorV1, error) {
	if client.pageHook != nil {
		client.pageHook()
	}
	client.queries = append(client.queries, query)
	client.sessionRefs = append(client.sessionRefs, sessionRef)
	if client.pageErr != nil {
		return microvm.RespuestaEventosSesionContenedorV1{}, client.pageErr
	}
	return client.pages[query.DespuesDe], nil
}

func TestDockerObserverRebuildsTerminalOutputAcrossPagesReplayAndRestart(t *testing.T) {
	fixture, adapter, client, request := newDockerObserverFixture(t)
	client.pages = successfulDockerObservationPages(request, client.physical.Cerca)

	first, firstErr := adapter.ObserveAgent(context.Background(), request)
	second, secondErr := adapter.ObserveAgent(context.Background(), request)
	fixture.journal = fixture.journal.reopen()
	restarted := mustDockerObserverAdapter(t, fixture, client)
	third, thirdErr := restarted.ObserveAgent(context.Background(), request)
	if firstErr != nil || secondErr != nil || thirdErr != nil {
		t.Fatalf("errors=%v/%v/%v", firstErr, secondErr, thirdErr)
	}
	for index, observation := range []ports.AgentObservation{first, second, third} {
		if observation.Status != ports.AgentCompleted || observation.MediaType != request.ArtifactMediaType ||
			string(observation.Content) != "artifact-ok" || observation.ErrorCode != "" ||
			observation.ExecutionRef != request.ExecutionRef || observation.SpecHash != request.SpecHash ||
			observation.ObservedAt.IsZero() || ports.ValidateAgentObservation(observation, request.MaxOutputBytes) != nil {
			t.Fatalf("observation %d = %+v", index, observation)
		}
	}
	if !sameObservationEvidence(first, second) || !sameObservationEvidence(second, third) {
		t.Fatalf("replay changed evidence: %+v / %+v / %+v", first, second, third)
	}
	wantCursors := []uint64{0, 2, 0, 2, 0, 2}
	if len(client.queries) != len(wantCursors) {
		t.Fatalf("queries=%+v", client.queries)
	}
	for index, query := range client.queries {
		if query.DespuesDe != wantCursors[index] || query.RevisionEsperada != client.physical.Revision ||
			query.Cerca != client.physical.Cerca || query.MaximoEventos != microvm.MaximoEventosPaginaSesionV1 ||
			query.MaximoBytes != microvm.MaximoEventosSesionBytesV1 || client.sessionRefs[index] != request.SessionRef.String() {
			t.Fatalf("query %d=%+v session=%q", index, query, client.sessionRefs[index])
		}
	}
}

func TestDockerAdapterRequiresReadOnlyObserverAtComposition(t *testing.T) {
	fixture := newDockerAdapterFixture(t)
	_, err := NewDockerAdapter(dockerObserverConfig(fixture, dockerLaunchOnlyClient{fixture.client}))
	if ErrorCode(err) != CodeObservationClientInvalid {
		t.Fatalf("NewDockerAdapter() = %v", err)
	}
}

func TestDockerObserverKeepsOnlyProvenNonterminalWorkNonterminal(t *testing.T) {
	tests := []struct {
		name       string
		missing    bool
		started    bool
		stopped    bool
		wantStatus ports.AgentStatus
		wantCode   string
	}{
		{name: "missing active session", missing: true, wantStatus: ports.AgentPending},
		{name: "empty active session", wantStatus: ports.AgentPending},
		{name: "started active session", started: true, wantStatus: ports.AgentRunning},
		{name: "stopped without terminal event", stopped: true, wantCode: CodeObservationResponseInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, adapter, client, request := newDockerObserverFixture(t)
			if test.missing {
				client.pageErr = &microvm.ErrorRespuesta{Estado: 404, Codigo: "api.sesion_ausente"}
			} else if test.started {
				client.pages = map[uint64]microvm.RespuestaEventosSesionContenedorV1{
					0: dockerObservationPage(request, client.physical.Cerca, 1, false,
						dockerObservationEvent(request, 1, microvm.TipoEventoSesionContenedorIniciada, nil)),
					1: dockerObservationPage(request, client.physical.Cerca, 1, false),
				}
			} else {
				client.pages = map[uint64]microvm.RespuestaEventosSesionContenedorV1{
					0: dockerObservationPage(request, client.physical.Cerca, 0, false),
				}
			}
			if test.stopped {
				client.physical.Estado = "detenido"
				alive, state := false, "removed"
				client.physical.RecursoVivo, client.physical.EstadoMotor = &alive, &state
			}
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if ErrorCode(err) != test.wantCode || (test.wantCode == "" &&
				(observation.Status != test.wantStatus || len(observation.Content) != 0 || observation.ErrorCode != "")) {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
		})
	}
}

func TestDockerObserverMapsOnlyExplicitTerminalOutcome(t *testing.T) {
	zero, one, signal := int32(0), int32(1), int32(9)
	remote := "executor.failed"
	tests := []struct {
		name       string
		stdout     string
		truncated  bool
		exit       *int32
		signal     *int32
		exhausted  bool
		remoteCode *string
		maxOutput  int64
		wantStatus ports.AgentStatus
		wantCode   string
	}{
		{name: "success", stdout: "artifact", exit: &zero, maxOutput: 32, wantStatus: ports.AgentCompleted},
		{name: "exact output boundary", stdout: "art", exit: &zero, maxOutput: 3, wantStatus: ports.AgentCompleted},
		{name: "nonzero", stdout: "partial", exit: &one, maxOutput: 32, wantStatus: ports.AgentFailed, wantCode: CodeSessionExitNonzero},
		{name: "signal", stdout: "partial", signal: &signal, maxOutput: 32, wantStatus: ports.AgentFailed, wantCode: CodeSessionSignaled},
		{name: "exhausted", stdout: "partial", exhausted: true, maxOutput: 32, wantStatus: ports.AgentFailed, wantCode: CodeSessionExhausted},
		{name: "remote failure", stdout: "partial", remoteCode: &remote, maxOutput: 32, wantStatus: ports.AgentFailed, wantCode: CodeSessionFailed},
		{name: "truncated", stdout: "partial", truncated: true, exit: &zero, maxOutput: 32, wantStatus: ports.AgentFailed, wantCode: CodeSessionOutputTruncated},
		{name: "overflow", stdout: "artifact", exit: &zero, maxOutput: 3, wantStatus: ports.AgentFailed, wantCode: CodeSessionOutputTooLarge},
		{name: "missing output", exit: &zero, maxOutput: 32, wantStatus: ports.AgentFailed, wantCode: CodeSessionOutputMissing},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, adapter, client, request := newDockerObserverFixture(t)
			request.MaxOutputBytes = test.maxOutput
			events := []microvm.EventoSesionContenedorV1{}
			if test.stdout != "" {
				events = append(events, dockerObservationEvent(request, 1, microvm.TipoEventoSesionContenedorStdout, []byte(test.stdout)))
			}
			if test.truncated {
				truncated := dockerObservationEvent(request, uint64(len(events)+1), microvm.TipoEventoSesionContenedorSalidaTruncada, nil)
				code := "output.truncated"
				truncated.CodigoError = &code
				events = append(events, truncated)
			}
			terminal := dockerObservationEvent(request, uint64(len(events)+1), microvm.TipoEventoSesionContenedorFinalizada, nil)
			terminal.CodigoSalida, terminal.Senal, terminal.Agotada, terminal.CodigoError =
				test.exit, test.signal, test.exhausted, test.remoteCode
			events = append(events, terminal)
			client.pages = map[uint64]microvm.RespuestaEventosSesionContenedorV1{
				0: dockerObservationPage(request, client.physical.Cerca, uint64(len(events)), true, events...),
			}
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if err != nil || observation.Status != test.wantStatus || observation.ErrorCode != test.wantCode ||
				(test.wantStatus == ports.AgentCompleted && string(observation.Content) != test.stdout) ||
				(test.wantStatus == ports.AgentFailed && len(observation.Content) != 0) {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
		})
	}
}

func TestDockerObserverRejectsCrossedIdentityAndEventCausality(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ports.AgentObserveRequest, *dockerObservationClientStub)
	}{
		{name: "provider", mutate: func(request *ports.AgentObserveRequest, _ *dockerObservationClientStub) {
			request.ProviderRef = "provider:foreign"
		}},
		{name: "physical ref", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			client.physical.Referencia = "ejecucion:foreign"
		}},
		{name: "physical fence", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) { client.physical.Cerca++ }},
		{name: "coherent crossed physical fence", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			client.physical.Cerca++
			client.physical.Identidad.Generation++
			for cursor, page := range client.pages {
				page.Cerca++
				client.pages[cursor] = page
			}
		}},
		{name: "foreign run label", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			client.physical.Identidad.RunLabel = "execution:foreign"
		}},
		{name: "physical revision", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) { client.physical.Revision = 0 }},
		{name: "physical resource", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) { client.physical.MemoriaMiB++ }},
		{name: "page ref", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[0]
			page.Referencia = "ejecucion:foreign"
			client.pages[0] = page
		}},
		{name: "page cursor", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[0]
			page.SiguienteCursor++
			client.pages[0] = page
		}},
		{name: "event sequence", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[0]
			page.Eventos[0].Secuencia++
			client.pages[0] = page
		}},
		{name: "invalid base64", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[0]
			page.Eventos[0].ContenidoBase64 = "%"
			client.pages[0] = page
		}},
		{name: "noncanonical base64", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[0]
			page.Eventos[1].ContenidoBase64 = "Zg==\n"
			client.pages[0] = page
		}},
		{name: "oversized event", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[0]
			page.Eventos[1].ContenidoBase64 = base64.StdEncoding.EncodeToString(make([]byte, 8193))
			client.pages[0] = page
		}},
		{name: "event after terminal", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[2]
			zero := int32(0)
			page.Eventos[0].Tipo, page.Eventos[0].CodigoSalida = microvm.TipoEventoSesionContenedorFinalizada, &zero
			page.Eventos[0].ContenidoBase64 = ""
			client.pages[2] = page
		}},
		{name: "incomplete terminal", mutate: func(_ *ports.AgentObserveRequest, client *dockerObservationClientStub) {
			page := client.pages[2]
			page.Eventos[len(page.Eventos)-1].CodigoSalida = nil
			client.pages[2] = page
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, adapter, client, request := newDockerObserverFixture(t)
			client.pages = successfulDockerObservationPages(request, client.physical.Cerca)
			test.mutate(&request, client)
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if ErrorCode(err) != map[bool]string{true: CodeObservationRequestInvalid, false: CodeObservationResponseInvalid}[test.name == "provider"] ||
				!reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
			if test.name == "provider" && len(client.observed) != 0 {
				t.Fatalf("invalid request reached client: %v", client.observed)
			}
		})
	}
}

func TestDockerObserverEnforcesAggregateOutputAndPaginationBoundaries(t *testing.T) {
	t.Run("aggregate output plus one across pages", func(t *testing.T) {
		_, adapter, client, request := newDockerObserverFixture(t)
		request.MaxOutputBytes = 3
		zero := int32(0)
		terminal := dockerObservationEvent(request, 3, microvm.TipoEventoSesionContenedorFinalizada, nil)
		terminal.CodigoSalida = &zero
		client.pages = map[uint64]microvm.RespuestaEventosSesionContenedorV1{
			0: dockerObservationPage(request, client.physical.Cerca, 1, false,
				dockerObservationEvent(request, 1, microvm.TipoEventoSesionContenedorStdout, []byte("ab"))),
			1: dockerObservationPage(request, client.physical.Cerca, 3, true,
				dockerObservationEvent(request, 2, microvm.TipoEventoSesionContenedorStdout, []byte("cd")), terminal),
		}
		observation, err := adapter.ObserveAgent(context.Background(), request)
		if err != nil || observation.Status != ports.AgentFailed || observation.ErrorCode != CodeSessionOutputTooLarge {
			t.Fatalf("observation=%+v err=%v", observation, err)
		}
	})

	t.Run("page cap", func(t *testing.T) {
		_, adapter, client, request := newDockerObserverFixture(t)
		client.pages = make(map[uint64]microvm.RespuestaEventosSesionContenedorV1, maxDockerObservationPages)
		for cursor := uint64(0); cursor < maxDockerObservationPages; cursor++ {
			client.pages[cursor] = dockerObservationPage(request, client.physical.Cerca, cursor+1, false,
				dockerObservationEvent(request, cursor+1, microvm.TipoEventoSesionContenedorStderr, nil))
		}
		observation, err := adapter.ObserveAgent(context.Background(), request)
		if ErrorCode(err) != CodeObservationResponseInvalid || len(client.queries) != maxDockerObservationPages ||
			!reflect.DeepEqual(observation, ports.AgentObservation{}) {
			t.Fatalf("observation=%+v queries=%d err=%v", observation, len(client.queries), err)
		}
	})
}

func TestDockerObserverRequiresExactDurableLaunchBinding(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*dockerAdapterFixture, *ports.AgentObserveRequest)
	}{
		{name: "missing", mutate: func(fixture *dockerAdapterFixture, request *ports.AgentObserveRequest) {
			delete(fixture.journal.records, dockerJournalKey(fixture.request, ports.AgentProviderRequestLaunch))
		}},
		{name: "crossed fence", mutate: func(_ *dockerAdapterFixture, request *ports.AgentObserveRequest) {
			request.LaunchActionFence++
		}},
		{name: "crossed binding", mutate: func(fixture *dockerAdapterFixture, request *ports.AgentObserveRequest) {
			key := dockerJournalKey(fixture.request, ports.AgentProviderRequestLaunch)
			stored := fixture.journal.records[key]
			stored.LaunchBindingRef = "ejecucion:foreign"
			fixture.journal.records[key] = stored
		}},
		{name: "crossed revision", mutate: func(fixture *dockerAdapterFixture, request *ports.AgentObserveRequest) {
			key := dockerJournalKey(fixture.request, ports.AgentProviderRequestLaunch)
			stored := fixture.journal.records[key]
			stored.LaunchBindingRevision++
			fixture.journal.records[key] = stored
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, adapter, client, request := newDockerObserverFixture(t)
			test.mutate(fixture, &request)
			observation, err := adapter.ObserveAgent(context.Background(), request)
			wantObserved := map[string]int{"crossed revision": 1}[test.name]
			if ErrorCode(err) != CodeObservationResponseInvalid || !reflect.DeepEqual(observation, ports.AgentObservation{}) || len(client.observed) != wantObserved || len(client.queries) != 0 {
				t.Fatalf("observation=%+v err=%v observed=%v queries=%v", observation, err, client.observed, client.queries)
			}
		})
	}
}

func TestDockerObserverClassifiesReadOnlyFailureAndCancellation(t *testing.T) {
	tests := []struct {
		name, wantCode            string
		journal, physical, events error
		temporary                 bool
	}{
		{name: "journal", journal: errors.New("state unavailable"), wantCode: CodeObservationUnavailable, temporary: true},
		{name: "transport", physical: errors.New("socket unavailable"), wantCode: CodeObservationUnavailable, temporary: true},
		{name: "invalid container response", physical: &microvm.ErrorContenedorV1{Codigo: "contenedores.respuesta_invalida"}, wantCode: CodeObservationResponseInvalid},
		{name: "stale revision", events: &microvm.ErrorRespuesta{Estado: 409, Codigo: "contenedores.revision_obsoleta"}, wantCode: CodeObservationUnavailable, temporary: true},
		{name: "protocol", events: &microvm.ErrorProtocolo{Recibido: "agentmicrovm.local.v0"}, wantCode: CodeProtocolIncompatible},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, adapter, client, request := newDockerObserverFixture(t)
			fixture.journal.resolveErr = test.journal
			client.physicalErr, client.pageErr = test.physical, test.events
			observation, err := adapter.ObserveAgent(context.Background(), request)
			if ErrorCode(err) != test.wantCode || isTemporary(err) != test.temporary || !reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
		})
	}
	for _, stage := range []string{"journal", "journal error", "physical", "events"} {
		t.Run("cancel "+stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			fixture, adapter, client, request := newDockerObserverFixture(t)
			if stage == "journal" || stage == "journal error" {
				fixture.journal.resolveHook = cancel
				if stage == "journal error" {
					fixture.journal.resolveErr = errors.New("state after cancel")
				}
			} else if stage == "physical" {
				client.physicalHook, client.physicalErr = cancel, errors.New("transport after cancel")
			} else {
				client.pageHook, client.pageErr = cancel, errors.New("transport after cancel")
			}
			observation, err := adapter.ObserveAgent(ctx, request)
			if !errors.Is(err, context.Canceled) || ErrorCode(err) != "" || (stage == "journal" || stage == "journal error") && len(client.observed) != 0 || !reflect.DeepEqual(observation, ports.AgentObservation{}) {
				t.Fatalf("observation=%+v err=%v", observation, err)
			}
		})
	}
}

func newDockerObserverFixture(t *testing.T) (*dockerAdapterFixture, *DockerAdapter, *dockerObservationClientStub, ports.AgentObserveRequest) {
	t.Helper()
	fixture := newDockerAdapterFixture(t)
	compiled := mustCompileDocker(t, fixture.request, fixture.binding)
	if _, err := fixture.adapter.Launch(context.Background(), fixture.request); err != nil {
		t.Fatalf("seed durable Docker launch: %v", err)
	}
	client := &dockerObservationClientStub{dockerClientStub: fixture.client,
		physical: validDockerContainerResponse(compiled.Plan, true), pages: make(map[uint64]microvm.RespuestaEventosSesionContenedorV1)}
	adapter := mustDockerObserverAdapter(t, fixture, client)
	request := ports.AgentObserveRequest{
		ExecutionRef: fixture.request.ExecutionRef, GoalRef: fixture.request.GoalRef,
		WorkItemRef: fixture.request.WorkItemRef, PlanGeneration: fixture.request.PlanGeneration,
		AppSpecGeneration: fixture.request.AppSpecGeneration, ExecutionAttempt: fixture.request.ExecutionAttempt,
		LaunchActionFence: fixture.request.EffectAuthority.ActionFence,
		SpecHash:          fixture.request.SpecHash, ProviderRef: adapter.capabilities.ProviderRef,
		ModelRef: adapter.capabilities.ModelRef, AgentRef: adapter.capabilities.AgentRef,
		ExternalRef: compiled.Plan.EjecucionRef, SessionRef: fixture.request.SessionRef,
		ArtifactMediaType: fixture.request.ArtifactMediaType, MaxOutputBytes: fixture.request.MaxOutputBytes,
	}
	return fixture, adapter, client, request
}

func mustDockerObserverAdapter(t *testing.T, fixture *dockerAdapterFixture, client DockerClient) *DockerAdapter {
	t.Helper()
	adapter, err := NewDockerAdapter(dockerObserverConfig(fixture, client))
	if err != nil {
		t.Fatalf("NewDockerAdapter() observer = %v", err)
	}
	return adapter
}

type dockerLaunchOnlyClient struct{ DockerClient }

func dockerObserverConfig(fixture *dockerAdapterFixture, client DockerClient) DockerConfig {
	return DockerConfig{
		Client: client, Signer: fixture.signer, Journal: fixture.journal, StopJournal: fixture.stopJournal,
		Capabilities: cloneCapabilities(fixture.adapter.capabilities), PlacementRef: fixture.adapter.placement,
		ProviderModel: fixture.adapter.model, PromptRenderer: fixture.adapter.renderer,
		ImageRef: fixture.adapter.imageRef, ExecutorRef: fixture.adapter.executorRef,
		VCPU: fixture.adapter.vcpu, MemoryMiB: fixture.adapter.memoryMiB, MaxPIDs: fixture.adapter.maxPIDs,
		Now: fixture.adapter.now,
	}
}

func successfulDockerObservationPages(request ports.AgentObserveRequest, fence uint64) map[uint64]microvm.RespuestaEventosSesionContenedorV1 {
	zero := int32(0)
	terminal := dockerObservationEvent(request, 4, microvm.TipoEventoSesionContenedorFinalizada, nil)
	terminal.CodigoSalida = &zero
	return map[uint64]microvm.RespuestaEventosSesionContenedorV1{
		0: dockerObservationPage(request, fence, 2, false,
			dockerObservationEvent(request, 1, microvm.TipoEventoSesionContenedorIniciada, nil),
			dockerObservationEvent(request, 2, microvm.TipoEventoSesionContenedorStdout, []byte("artifact-"))),
		2: dockerObservationPage(request, fence, 4, true,
			dockerObservationEvent(request, 3, microvm.TipoEventoSesionContenedorStdout, []byte("ok")), terminal),
	}
}

func dockerObservationPage(request ports.AgentObserveRequest, fence, next uint64, terminal bool,
	events ...microvm.EventoSesionContenedorV1) microvm.RespuestaEventosSesionContenedorV1 {
	if events == nil {
		events = []microvm.EventoSesionContenedorV1{}
	}
	return microvm.RespuestaEventosSesionContenedorV1{
		Referencia: request.ExternalRef, SesionRef: request.SessionRef.String(), Cerca: fence,
		Eventos: events, SiguienteCursor: next, Terminal: terminal,
	}
}

func dockerObservationEvent(request ports.AgentObserveRequest, sequence uint64, typeName microvm.TipoEventoSesionContenedorV1,
	content []byte) microvm.EventoSesionContenedorV1 {
	return microvm.EventoSesionContenedorV1{
		SesionRef: request.SessionRef.String(), Secuencia: sequence, Tipo: typeName,
		ContenidoBase64: base64.StdEncoding.EncodeToString(content),
	}
}
