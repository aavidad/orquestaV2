package agentmicrovm

import (
	"context"
	"encoding/json"
	"errors"
	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"
	"net"
	"net/http"
	"orquesta/internal/ports"
	"strings"
	"testing"
)

type stopClientStub struct {
	*launchClientStub
	physical            microvm.RespuestaEjecucion
	observeErr, stopErr error
	stopReply           microvm.RespuestaDetencion
	stops               int
}

func (client *stopClientStub) Observar(context.Context, string) (microvm.RespuestaEjecucion, error) {
	return client.physical, client.observeErr
}
func (client *stopClientStub) Detener(context.Context, string, string, microvm.SolicitudDetencion) (microvm.RespuestaDetencion, error) {
	client.stops++
	return client.stopReply, client.stopErr
}
func TestTranslateStopReceiptKeepsModesAndPhysicalIdentityExact(t *testing.T) {
	launch := validLaunchRequest(t)
	client := validStopClient(launch, validRemoteCapabilities())
	cooperative := validStopRequest(launch, ports.AgentStopCooperative)
	forced := validStopRequest(launch, ports.AgentStopForced)
	assertStopTranslation(t, cooperative, client.physical, confirmedStopResponse(cooperative, client.physical, microvm.ModoDetencionEfectivoCooperativa), ports.AgentStopped, true)
	assertStopTranslation(t, cooperative, client.physical, confirmedStopResponse(cooperative, client.physical, microvm.ModoDetencionEfectivoForzada), "", false)
	assertStopTranslation(t, forced, client.physical, confirmedStopResponse(forced, client.physical, microvm.ModoDetencionEfectivoForzada), ports.AgentStopped, true)
	assertStopTranslation(t, cooperative, client.physical, confirmedStopResponse(cooperative, client.physical, microvm.ModoDetencionEfectivoYaAusente), ports.AgentStopAlreadyStopped, true)
	assertStopTranslation(t, cooperative, client.physical, pendingStopResponse(cooperative, client.physical), ports.AgentStopPending, true)
	base := confirmedStopResponse(forced, client.physical, microvm.ModoDetencionEfectivoForzada)
	for _, mutate := range []func(*microvm.RespuestaDetencion){
		func(value *microvm.RespuestaDetencion) {
			value.Ejecucion.Referencia = "ejecucion:" + strings.Repeat("b", 64)
		},
		func(value *microvm.RespuestaDetencion) { value.Ejecucion.Cerca++ },
		func(value *microvm.RespuestaDetencion) { value.ClaveIdempotencia += ":crossed" },
		func(value *microvm.RespuestaDetencion) { value.ModoSolicitado = microvm.ModoDetencionCooperativa },
	} {
		candidate := base
		mutate(&candidate)
		assertStopTranslation(t, forced, client.physical, candidate, "", false)
	}
}
func assertStopTranslation(t *testing.T, request ports.AgentStopRequest, physical microvm.RespuestaEjecucion, response microvm.RespuestaDetencion, status ports.AgentStopStatus, valid bool) {
	t.Helper()
	receipt, ok := translateStopReceipt(request, physical, response)
	if ok != valid || receipt.Status != status {
		t.Fatalf("receipt=%+v valid=%t", receipt, ok)
	}
}
func TestAdapterStopFailsClosedBeforeMutationAndNeverResends(t *testing.T) {
	assertStopFailure(t, CodeControlRequestInvalid, true, func(request *ports.AgentStopRequest, _ *stopClientStub) { request.ProviderRef = "provider:foreign" })
	assertStopFailure(t, CodeControlCanceledBeforeCall, true, func(_ *ports.AgentStopRequest, client *stopClientStub) { client.capabilitiesErr = context.Canceled })
	assertStopFailure(t, CodeControlObservationFailed, true, func(_ *ports.AgentStopRequest, client *stopClientStub) { client.observeErr = errors.New("observe") })
	assertStopFailure(t, CodeControlCanceledBeforeCall, true, func(_ *ports.AgentStopRequest, client *stopClientStub) { client.observeErr = context.Canceled })
	assertStopFailure(t, "", false, func(_ *ports.AgentStopRequest, client *stopClientStub) { client.stopErr = context.Canceled })
	launch := validLaunchRequest(t)
	request := validStopRequest(launch, ports.AgentStopCooperative)
	client := validStopClient(launch, validRemoteCapabilities())
	client.stopErr = errors.New("lost ack")
	if _, err := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false)).Stop(context.Background(), request); ErrorCode(err) != CodeControlUnavailable {
		t.Fatal(err)
	}
	client.physical.Estado, client.stopErr = "detenida", nil
	if _, err := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false)).Stop(context.Background(), request); ErrorCode(err) != CodeControlObservationInvalid || client.stops != 1 {
		t.Fatalf("restart err=%v calls=%d", err, client.stops)
	}
	remote := validRemoteCapabilities()
	remote.KVMDisponible, remote.FirecrackerConfigurado, remote.FirecrackerEjecutable, remote.MaximoEjecuciones = false, false, false, 0
	client = validStopClient(launch, remote)
	got, err := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false)).ControlCapabilities(context.Background())
	if err != nil || !got.CooperativeStop || !got.ForcedStop || client.capabilitiesCall != 1 {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}
func assertStopFailure(t *testing.T, code string, before bool, mutate func(*ports.AgentStopRequest, *stopClientStub)) {
	t.Helper()
	launch := validLaunchRequest(t)
	request := validStopRequest(launch, ports.AgentStopForced)
	client := validStopClient(launch, validRemoteCapabilities())
	client.stopReply = confirmedStopResponse(request, client.physical, microvm.ModoDetencionEfectivoForzada)
	mutate(&request, client)
	_, err := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false)).Stop(context.Background(), request)
	if ErrorCode(err) != code || isDefinitelyNotApplied(err) != before || before && client.stops != 0 || !before && client.stops != 1 {
		t.Fatalf("err=%v definite=%t calls=%d", err, isDefinitelyNotApplied(err), client.stops)
	}
}
func TestAdapterStopFailsClosedOnPublicUnixCooperativeEscalation(t *testing.T) {
	launch := validLaunchRequest(t)
	request := validStopRequest(launch, ports.AgentStopCooperative)
	physical := microvm.RespuestaEjecucion{Referencia: request.ExternalRef, Estado: "disponible", Revision: 11, Cerca: launch.EffectAuthority.ActionFence}
	want := microvm.SolicitudDetencion{RevisionEsperada: 11, Cerca: physical.Cerca, Modo: microvm.ModoDetencionCooperativa}
	stopResponse := confirmedStopResponse(request, physical, microvm.ModoDetencionEfectivoForzada)
	handler := http.HandlerFunc(func(response http.ResponseWriter, incoming *http.Request) {
		response.Header().Set(microvm.CabeceraProtocolo, microvm.ProtocoloLocal)
		if incoming.Header.Get(microvm.CabeceraProtocolo) != microvm.ProtocoloLocal {
			t.Errorf("protocol header=%q", incoming.Header.Get(microvm.CabeceraProtocolo))
		}
		var payload any
		switch incoming.Method + " " + incoming.URL.Path {
		case "GET /v1/capacidades":
			payload = validRemoteCapabilities()
		case "GET /v1/ejecuciones/" + request.ExternalRef:
			payload = physical
		case "POST /v1/ejecuciones/" + request.ExternalRef + "/detencion":
			var got microvm.SolicitudDetencion
			if err := json.NewDecoder(incoming.Body).Decode(&got); err != nil || got != want ||
				incoming.Header.Get(microvm.CabeceraIdempotencia) != request.IdempotencyKey {
				t.Errorf("physical request=%+v err=%v key=%q", got, err, incoming.Header.Get(microvm.CabeceraIdempotencia))
			}
			payload = stopResponse
		default:
			http.NotFound(response, incoming)
			return
		}
		if err := json.NewEncoder(response).Encode(payload); err != nil {
			t.Error(err)
		}
	})
	socket := t.TempDir() + "/agentmicrovm.sock"
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	client, err := microvm.Nuevo(socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(client.LiberarConexiones)
	receipt, err := mustNewAdapter(t, client, validSigner(), launch, validDescriptor(t, false)).Stop(context.Background(), request)
	var contractErr *microvm.ErrorDetencionV1
	if ErrorCode(err) != CodeControlUnavailable || isDefinitelyNotApplied(err) || !errors.As(err, &contractErr) || receipt != (ports.AgentStopReceipt{}) {
		t.Fatalf("receipt=%+v err=%v", receipt, err)
	}
}
func validStopRequest(launch ports.AgentLaunchRequest, mode ports.AgentStopMode) ports.AgentStopRequest {
	return ports.AgentStopRequest{ExecutionRef: launch.ExecutionRef, GoalRef: launch.GoalRef, WorkItemRef: launch.WorkItemRef, PlanGeneration: launch.PlanGeneration, AppSpecGeneration: launch.AppSpecGeneration, ExecutionAttempt: launch.ExecutionAttempt, SpecHash: launch.SpecHash, ProviderRef: "provider:codex", ModelRef: "model:codex-microvm", AgentRef: "agent:codex-microvm", ExternalRef: "ejecucion:" + strings.Repeat("a", 64), Mode: mode, IdempotencyKey: "stop:control:microvm-1"}
}
func validStopClient(launch ports.AgentLaunchRequest, remote microvm.RespuestaCapacidades) *stopClientStub {
	request := validStopRequest(launch, ports.AgentStopCooperative)
	physical := microvm.RespuestaEjecucion{Referencia: request.ExternalRef, Estado: "disponible", Revision: 11, Cerca: launch.EffectAuthority.ActionFence}
	client := &stopClientStub{launchClientStub: &launchClientStub{capabilities: remote}, physical: physical}
	client.stopReply = pendingStopResponse(request, physical)
	return client
}
func pendingStopResponse(request ports.AgentStopRequest, physical microvm.RespuestaEjecucion) microvm.RespuestaDetencion {
	mode, _ := physicalStopMode(request.Mode)
	physical.Estado, physical.Revision = "deteniendo", physical.Revision+1
	return microvm.RespuestaDetencion{Ejecucion: physical, ClaveIdempotencia: request.IdempotencyKey, Estado: microvm.EstadoDetencionPendiente, ModoSolicitado: mode}
}
func confirmedStopResponse(request ports.AgentStopRequest, physical microvm.RespuestaEjecucion, effective microvm.ModoDetencionEfectivoV1) microvm.RespuestaDetencion {
	mode, _ := physicalStopMode(request.Mode)
	receipt := "detencion:" + strings.Repeat("c", 64)
	at := uint64(1_786_905_600_000)
	physical.Estado, physical.Revision = "detenida", physical.Revision+1
	return microvm.RespuestaDetencion{Ejecucion: physical, ClaveIdempotencia: request.IdempotencyKey, Estado: microvm.EstadoDetencionConfirmada, ModoSolicitado: mode, ModoEfectivo: &effective, ReceiptRef: &receipt, ConfirmadaUnixMS: &at}
}
