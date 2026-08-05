package agentmicrovm

import (
	"context"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

type staticSessionPromptRenderer string

func (renderer staticSessionPromptRenderer) RenderAgentPrompt(ports.AgentPrompt) (string, error) {
	return string(renderer), nil
}

type failedSessionPromptRenderer struct{ err error }

func (renderer failedSessionPromptRenderer) RenderAgentPrompt(ports.AgentPrompt) (string, error) {
	return "", renderer.err
}

// These defaults preserve launch-only tests while making their replay model
// explicit: an input receipt already exists, so no session mutation follows.
func (client *launchClientStub) RevisionTrabajo(
	context.Context,
	string,
) (microvm.RespuestaRevisionTrabajo, error) {
	return microvm.RespuestaRevisionTrabajo{Referencia: client.response.Referencia, RevisionTrabajo: 1}, nil
}

func (client *launchClientStub) IniciarSesion(
	context.Context,
	string,
	string,
	microvm.SolicitudIniciarSesionTrabajoV1,
) (microvm.RespuestaSesionTrabajoV1, error) {
	return microvm.RespuestaSesionTrabajoV1{}, errors.New("session start not configured")
}

func (client *launchClientStub) EnviarEntradaSesion(
	context.Context,
	string,
	string,
	string,
	microvm.SolicitudEntradaSesionTrabajoV1,
) (microvm.RespuestaSesionTrabajoV1, error) {
	return microvm.RespuestaSesionTrabajoV1{}, errors.New("session input not configured")
}

func (client *launchClientStub) ReconciliarEntradaSesion(
	_ context.Context,
	_ string,
	_ string,
	sessionRef string,
	fence uint64,
) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	receipt := microvm.RespuestaSesionTrabajoV1{
		EjecucionRef: client.response.Referencia, SesionRef: sessionRef,
		Estado: microvm.EstadoSesionActiva, Revision: 4, RevisionTrabajo: 2,
		RevisionSesion: 2, Cerca: fence,
	}
	return microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{
		EjecucionRef: client.response.Referencia, SesionRef: sessionRef, Cerca: fence,
		Estado: microvm.EstadoReconciliacionEntradaResuelta, Comprobante: &receipt,
	}, nil
}

func (*launchOnlyClientStub) RevisionTrabajo(context.Context, string) (microvm.RespuestaRevisionTrabajo, error) {
	return microvm.RespuestaRevisionTrabajo{}, nil
}
func (*launchOnlyClientStub) IniciarSesion(context.Context, string, string, microvm.SolicitudIniciarSesionTrabajoV1) (microvm.RespuestaSesionTrabajoV1, error) {
	return microvm.RespuestaSesionTrabajoV1{}, nil
}
func (*launchOnlyClientStub) EnviarEntradaSesion(context.Context, string, string, string, microvm.SolicitudEntradaSesionTrabajoV1) (microvm.RespuestaSesionTrabajoV1, error) {
	return microvm.RespuestaSesionTrabajoV1{}, nil
}
func (*launchOnlyClientStub) ReconciliarEntradaSesion(context.Context, string, string, string, uint64) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	return microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{}, nil
}

type reconcileReply struct {
	response microvm.RespuestaReconciliacionEntradaSesionTrabajoV1
	err      error
}

type sessionLaunchClientStub struct {
	mu sync.Mutex

	capabilities microvm.RespuestaCapacidades
	physical     microvm.RespuestaEjecucion
	work         microvm.RespuestaRevisionTrabajo
	start        microvm.RespuestaSesionTrabajoV1
	input        microvm.RespuestaSesionTrabajoV1
	page         microvm.PaginaEventosSesionTrabajoV1

	reconcileReplies []reconcileReply
	observeErr       error
	workErr          error
	startErr         error
	inputErr         error

	steps         []string
	startKeys     []string
	inputKeys     []string
	startRequests []microvm.SolicitudIniciarSesionTrabajoV1
	inputRequests []microvm.SolicitudEntradaSesionTrabajoV1
	reconcileKeys []string
	onCall        func(string)
}

func (client *sessionLaunchClientStub) record(step string) {
	client.mu.Lock()
	client.steps = append(client.steps, step)
	callback := client.onCall
	client.mu.Unlock()
	if callback != nil {
		callback(step)
	}
}

func (client *sessionLaunchClientStub) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	client.record("capabilities")
	return client.capabilities, nil
}

func (client *sessionLaunchClientStub) Lanzar(context.Context, string, microvm.SolicitudLanzamiento) (microvm.RespuestaEjecucion, error) {
	client.record("launch")
	return client.physical, nil
}

func (client *sessionLaunchClientStub) RevisionTrabajo(context.Context, string) (microvm.RespuestaRevisionTrabajo, error) {
	client.record("work-revision")
	return client.work, client.workErr
}

func (client *sessionLaunchClientStub) IniciarSesion(
	_ context.Context,
	key string,
	_ string,
	request microvm.SolicitudIniciarSesionTrabajoV1,
) (microvm.RespuestaSesionTrabajoV1, error) {
	client.mu.Lock()
	client.startKeys = append(client.startKeys, key)
	client.startRequests = append(client.startRequests, request)
	client.mu.Unlock()
	client.record("start")
	return client.start, client.startErr
}

func (client *sessionLaunchClientStub) EnviarEntradaSesion(
	_ context.Context,
	key string,
	_ string,
	_ string,
	request microvm.SolicitudEntradaSesionTrabajoV1,
) (microvm.RespuestaSesionTrabajoV1, error) {
	client.mu.Lock()
	client.inputKeys = append(client.inputKeys, key)
	client.inputRequests = append(client.inputRequests, request)
	client.mu.Unlock()
	client.record("input")
	return client.input, client.inputErr
}

func (client *sessionLaunchClientStub) ReconciliarEntradaSesion(
	_ context.Context,
	key string,
	_ string,
	_ string,
	_ uint64,
) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	client.mu.Lock()
	client.reconcileKeys = append(client.reconcileKeys, key)
	index := len(client.reconcileKeys) - 1
	var reply reconcileReply
	if index < len(client.reconcileReplies) {
		reply = client.reconcileReplies[index]
	}
	client.mu.Unlock()
	client.record("reconcile")
	return reply.response, reply.err
}

func (client *sessionLaunchClientStub) Observar(context.Context, string) (microvm.RespuestaEjecucion, error) {
	return client.physical, nil
}

func (client *sessionLaunchClientStub) LeerEventosSesion(
	context.Context,
	string,
	string,
	microvm.ConsultaEventosSesionTrabajoV1,
) (microvm.PaginaEventosSesionTrabajoV1, error) {
	client.record("observe-session")
	return client.page, client.observeErr
}

type orderedSessionRenderer struct {
	client *sessionLaunchClientStub
	output string
	cancel context.CancelFunc
}

func (renderer orderedSessionRenderer) RenderAgentPrompt(ports.AgentPrompt) (string, error) {
	renderer.client.record("render")
	if renderer.cancel != nil {
		renderer.cancel()
	}
	return renderer.output, nil
}

type orderedSessionSigner struct {
	client   *sessionLaunchClientStub
	delegate Signer
	cancel   context.CancelFunc
}

func (signer orderedSessionSigner) Preparar(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	authorized microvm.ContextoAutorizado,
	plan microvm.PlanLanzamiento,
	issuedAt time.Time,
	validity time.Duration,
) (microvm.SolicitudLanzamiento, error) {
	signer.client.record("sign")
	if signer.cancel != nil {
		signer.cancel()
	}
	return signer.delegate.Preparar(ctx, request, authorized, plan, issuedAt, validity)
}

func TestAdapterLaunchBuildsPacketBeforeMutationAndDeliversExactInput(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := newNormalSessionClient(t, request, descriptor)
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.PromptRenderer = orderedSessionRenderer{client: client, output: "perform exact work"}
	config.Signer = orderedSessionSigner{client: client, delegate: validSigner()}
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatalf("Launch() = %v", err)
	}
	wantSteps := []string{"render", "capabilities", "sign", "launch", "reconcile", "observe-session", "work-revision", "start", "reconcile", "input"}
	if !reflect.DeepEqual(client.steps, wantSteps) {
		t.Fatalf("steps = %v, want %v", client.steps, wantSteps)
	}
	if receipt.ExternalRef != client.physical.Referencia {
		t.Fatalf("receipt = %+v", receipt)
	}
	if len(client.startRequests) != 1 || len(client.inputRequests) != 1 {
		t.Fatalf("start=%d input=%d", len(client.startRequests), len(client.inputRequests))
	}
	if len(client.startKeys) != 1 || len(client.inputKeys) != 1 || len(client.reconcileKeys) != 2 ||
		client.startKeys[0] != sessionIdempotencyKey(sessionStartKeyPrefix, sessionStartStage, request) ||
		client.inputKeys[0] != sessionIdempotencyKey(sessionInputKeyPrefix, sessionInputStage, request) ||
		client.reconcileKeys[0] != client.inputKeys[0] || client.reconcileKeys[1] != client.inputKeys[0] {
		t.Fatalf("keys start=%v input=%v reconcile=%v", client.startKeys, client.inputKeys, client.reconcileKeys)
	}
	start := client.startRequests[0]
	if start.SesionRef != request.SessionRef.String() || start.RevisionEsperada != client.physical.Revision ||
		start.RevisionTrabajoEsperada != client.work.RevisionTrabajo || start.Cerca != client.physical.Cerca ||
		start.EjecutorRef != descriptor.EjecutorRef || start.DirectorioTrabajo != "." ||
		start.PlazoTotalMilisegundos != 90_000 || start.MaximoEventosBytes != microvm.MaximoEventosSesionBytesV1 {
		t.Fatalf("start request = %+v", start)
	}
	input := client.inputRequests[0]
	if input.RevisionEsperada != client.start.Revision || input.RevisionTrabajoEsperada != client.start.RevisionTrabajo ||
		input.RevisionSesionEsperada != client.start.RevisionSesion || input.Cerca != client.physical.Cerca || !input.CerrarStdin {
		t.Fatalf("input request = %+v", input)
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(input.ContenidoBase64)
	if err != nil {
		t.Fatal(err)
	}
	compiled := mustCompile(t, request, descriptor)
	_, wantPacket, err := BuildWorkPacketV1(
		request, config.ModelBinding.Profile, compiled, config.ModelBinding.ProviderModel,
		staticSessionPromptRenderer("perform exact work"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(wantPacket) {
		t.Fatalf("stdin changed: got=%s want=%s", decoded, wantPacket)
	}
}

func TestAdapterLaunchPrevalidationMakesZeroTransportCalls(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := newNormalSessionClient(t, request, descriptor)
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.PromptRenderer = failedSessionPromptRenderer{err: errors.New("private-render-detail")}
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	_, err = adapter.Launch(context.Background(), request)
	if ErrorCode(err) != CodeWorkPacketRenderFailed || len(client.steps) != 0 || !isDefinitelyNotApplied(err) ||
		strings.Contains(err.Error(), "private-render-detail") {
		t.Fatalf("error=%v steps=%v definite=%v", err, client.steps, isDefinitelyNotApplied(err))
	}
}

func TestAdapterRejectsUnboundLogicalOrProviderModel(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	for _, test := range []struct {
		name   string
		mutate func(*Config)
	}{
		{"logical selector mismatch", func(config *Config) { config.ModelBinding.ModelRef = "model:other" }},
		{"provider model empty", func(config *Config) { config.ModelBinding.ProviderModel = "" }},
		{"provider model control", func(config *Config) { config.ModelBinding.ProviderModel = "gpt-5.6\n" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			config := validAdapterConfig(client, validSigner(), request, descriptor)
			test.mutate(&config)
			adapter, err := New(config)
			if adapter != nil || ErrorCode(err) != CodeConfigurationInvalid || len(client.steps) != 0 {
				t.Fatalf("adapter=%v error=%v steps=%v", adapter, err, client.steps)
			}
		})
	}
}

func TestAdapterLaunchReconciliationResolvedAndPendingNeverResend(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	for _, test := range []struct {
		name     string
		reply    reconcileReply
		wantCode string
	}{
		{"resolved", resolvedReconciliation(request, validPhysicalResponse(t, request, descriptor), 4, 2, 2), ""},
		{"pending", pendingReconciliation(request, validPhysicalResponse(t, request, descriptor)), CodeSessionInputPending},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			client.reconcileReplies = []reconcileReply{test.reply}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.wantCode || len(client.startRequests) != 0 || len(client.inputRequests) != 0 ||
				len(client.reconcileKeys) != 1 {
				t.Fatalf("error=%v steps=%v starts=%d inputs=%d", err, client.steps, len(client.startRequests), len(client.inputRequests))
			}
			if test.wantCode != "" && !isTemporary(err) {
				t.Fatalf("pending not temporary: %v", err)
			}
		})
	}
}

func TestAdapterLaunchFailsClosedForAmbiguousAndTerminalSessions(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	for _, test := range []struct {
		name     string
		state    microvm.EstadoSesionTrabajoV1
		terminal bool
		wantCode string
	}{
		{"ambiguous", microvm.EstadoSesionAmbigua, false, CodeSessionAmbiguous},
		{"terminal", microvm.EstadoSesionFinalizada, true, CodeSessionInconsistent},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			client.page = sessionPage(request, client.physical, test.state, test.terminal)
			client.observeErr = nil
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.wantCode || len(client.startRequests) != 0 || len(client.inputRequests) != 0 {
				t.Fatalf("error=%v steps=%v", err, client.steps)
			}
		})
	}
}

func TestAdapterLaunchContinuesActiveObservedSessionWithoutRestart(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := newNormalSessionClient(t, request, descriptor)
	client.page = sessionPage(request, client.physical, microvm.EstadoSesionActiva, false)
	client.observeErr = nil
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
	if _, err := adapter.Launch(context.Background(), request); err != nil {
		t.Fatalf("Launch() = %v", err)
	}
	if len(client.startRequests) != 0 || len(client.inputRequests) != 1 {
		t.Fatalf("steps=%v starts=%d inputs=%d", client.steps, len(client.startRequests), len(client.inputRequests))
	}
	input := client.inputRequests[0]
	if input.RevisionEsperada != client.page.Revision ||
		input.RevisionTrabajoEsperada != client.page.RevisionTrabajo ||
		input.RevisionSesionEsperada != client.page.RevisionSesion {
		t.Fatalf("input revisions = %+v, page=%+v", input, client.page)
	}
}

func TestAdapterLaunchWaitsForAdmittedOrStartingSessionWithoutResend(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	for _, state := range []microvm.EstadoSesionTrabajoV1{
		microvm.EstadoSesionAdmitida,
		microvm.EstadoSesionIniciando,
	} {
		t.Run(string(state), func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			client.page = sessionPage(request, client.physical, state, false)
			client.observeErr = nil
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != CodeSessionPending || !isTemporary(err) ||
				len(client.startRequests) != 0 || len(client.inputRequests) != 0 {
				t.Fatalf("error=%v temporary=%v steps=%v", err, isTemporary(err), client.steps)
			}
		})
	}
}

func TestAdapterLaunchRedactsRemoteSessionErrors(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := newNormalSessionClient(t, request, descriptor)
	client.startErr = &microvm.ErrorRespuesta{
		Estado: 422, Codigo: "sesion.rechazada", Detalle: "credential=private-marker",
	}
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
	_, err := adapter.Launch(context.Background(), request)
	if ErrorCode(err) != CodeSessionStartRejected || err.Error() != CodeSessionStartRejected ||
		strings.Contains(err.Error(), "private-marker") {
		t.Fatalf("error=%q code=%q", err, ErrorCode(err))
	}
}

func TestAdapterLaunchMapsOnlyTypedExactRemotePending(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	for _, test := range []struct {
		name     string
		code     string
		wantCode string
	}{
		{"exact", remoteSessionPending, CodeSessionPending},
		{"missing", "", CodeSessionStartRejected},
		{"other", "sesiones.otro_conflicto", CodeSessionStartRejected},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			client.startErr = &microvm.ErrorRespuesta{Estado: 409, Codigo: test.code, Detalle: "private-marker"}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.wantCode || len(client.startRequests) != 1 || len(client.inputRequests) != 0 ||
				strings.Contains(err.Error(), "private-marker") {
				t.Fatalf("error=%v steps=%v", err, client.steps)
			}
			if (test.wantCode == CodeSessionPending) != isTemporary(err) {
				t.Fatalf("error=%v temporary=%v", err, isTemporary(err))
			}
		})
	}
}

func TestAdapterLaunchSecondReconcileToInputPendingRaceNeverResends(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := newNormalSessionClient(t, request, descriptor)
	client.inputErr = &microvm.ErrorRespuesta{
		Estado: 409, Codigo: remoteSessionPending, Detalle: "input may already be applied",
	}
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
	_, err := adapter.Launch(context.Background(), request)
	if ErrorCode(err) != CodeSessionInputPending || !isTemporary(err) ||
		len(client.reconcileKeys) != 2 || len(client.inputRequests) != 1 {
		t.Fatalf("error=%v steps=%v reconciles=%d inputs=%d", err, client.steps, len(client.reconcileKeys), len(client.inputRequests))
	}
}

func TestAdapterLaunchRejectsAlteredSessionBindings(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	tests := []struct {
		name     string
		mutate   func(*sessionLaunchClientStub)
		wantCode string
	}{
		{"work ref", func(c *sessionLaunchClientStub) { c.work.Referencia = "ejecucion:foreign" }, CodeWorkRevisionInvalid},
		{"start ref", func(c *sessionLaunchClientStub) { c.start.SesionRef = "session:foreign" }, CodeSessionResponseInvalid},
		{"start fence", func(c *sessionLaunchClientStub) { c.start.Cerca++ }, CodeSessionResponseInvalid},
		{"start revision", func(c *sessionLaunchClientStub) { c.start.Revision = 1 }, CodeSessionResponseInvalid},
		{"input ref", func(c *sessionLaunchClientStub) { c.input.EjecucionRef = "ejecucion:foreign" }, CodeSessionResponseInvalid},
		{"input revision", func(c *sessionLaunchClientStub) { c.input.RevisionSesion = 1 }, CodeSessionResponseInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			test.mutate(client)
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.wantCode {
				t.Fatalf("error=%v code=%q", err, ErrorCode(err))
			}
		})
	}
}

func TestAdapterLaunchRejectsStaleOrCrossReconciliationReceipt(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	for _, test := range []struct {
		name   string
		mutate func(*reconcileReply, microvm.RespuestaEjecucion)
	}{
		{"stale physical revision", func(reply *reconcileReply, physical microvm.RespuestaEjecucion) {
			reply.response.Comprobante.Revision = physical.Revision - 1
		}},
		{"cross receipt execution", func(reply *reconcileReply, _ microvm.RespuestaEjecucion) {
			reply.response.Comprobante.EjecucionRef = "ejecucion:foreign"
		}},
		{"cross envelope session", func(reply *reconcileReply, _ microvm.RespuestaEjecucion) {
			reply.response.SesionRef = "session:foreign"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			reply := resolvedReconciliation(request, client.physical, 4, 2, 2)
			test.mutate(&reply, client.physical)
			client.reconcileReplies = []reconcileReply{reply}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != CodeSessionReconcileInvalid || len(client.startRequests) != 0 || len(client.inputRequests) != 0 {
				t.Fatalf("error=%v steps=%v", err, client.steps)
			}
		})
	}
}

func TestSessionIdempotencyKeysAreDeterministicFramedAndSecretFree(t *testing.T) {
	request := validLaunchRequest(t)
	request.IdempotencyKey = "original-secret-marker"
	start := sessionIdempotencyKey(sessionStartKeyPrefix, sessionStartStage, request)
	input := sessionIdempotencyKey(sessionInputKeyPrefix, sessionInputStage, request)
	if start != sessionIdempotencyKey(sessionStartKeyPrefix, sessionStartStage, request) || start == input ||
		len(start) > 128 || len(input) > 128 || !strings.HasPrefix(start, sessionStartKeyPrefix) ||
		!strings.HasPrefix(input, sessionInputKeyPrefix) {
		t.Fatalf("keys start=%q input=%q", start, input)
	}
	for _, secret := range []string{
		request.IdempotencyKey, request.ExecutionRef.String(), request.SessionRef.String(),
		request.SpecHash, request.EffectAuthority.EffectAttemptRef,
	} {
		if strings.Contains(start, secret) || strings.Contains(input, secret) {
			t.Fatalf("key leaked field %q", secret)
		}
	}
	left, right := request, request
	left.IdempotencyKey, left.EffectAuthority.EffectAttemptRef = "ab", "c"
	right.IdempotencyKey, right.EffectAuthority.EffectAttemptRef = "a", "bc"
	if sessionIdempotencyKey(sessionInputKeyPrefix, sessionInputStage, left) ==
		sessionIdempotencyKey(sessionInputKeyPrefix, sessionInputStage, right) {
		t.Fatal("length framing did not separate adjacent fields")
	}
}

func TestAdapterLaunchPreservesCancellationAtEverySessionBoundary(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	boundaries := []string{"render", "capabilities", "sign", "launch", "reconcile", "observe-session", "work-revision", "start", "second-reconcile", "input"}
	for _, boundary := range boundaries {
		t.Run(boundary, func(t *testing.T) {
			client := newNormalSessionClient(t, request, descriptor)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			seenReconcile := 0
			client.onCall = func(step string) {
				if step == "reconcile" {
					seenReconcile++
					if boundary == "second-reconcile" && seenReconcile == 2 {
						cancel()
					}
				}
				if step == boundary {
					cancel()
				}
			}
			config := validAdapterConfig(client, validSigner(), request, descriptor)
			config.PromptRenderer = orderedSessionRenderer{client: client, output: "work"}
			config.Signer = orderedSessionSigner{client: client, delegate: validSigner()}
			if boundary == "render" {
				config.PromptRenderer = orderedSessionRenderer{client: client, output: "work", cancel: cancel}
			}
			if boundary == "sign" {
				config.Signer = orderedSessionSigner{client: client, delegate: validSigner(), cancel: cancel}
			}
			adapter, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			_, err = adapter.Launch(ctx, request)
			if !errors.Is(err, context.Canceled) || ErrorCode(err) != "" {
				t.Fatalf("error=%v code=%q steps=%v", err, ErrorCode(err), client.steps)
			}
		})
	}
}

func TestAdapterLaunchConcurrentResolvedReplayIsRaceFree(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	physical := validPhysicalResponse(t, request, descriptor)
	client := &concurrentResolvedSessionClient{physical: physical, request: request}
	config := validAdapterConfig(client, concurrentSessionSigner{delegate: validSigner().delegate}, request, descriptor)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	const workers = 32
	var wg sync.WaitGroup
	errorsFound := make(chan error, workers)
	receipts := make(chan ports.AgentLaunchReceipt, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			receipt, launchErr := adapter.Launch(context.Background(), request)
			if launchErr != nil {
				errorsFound <- launchErr
				return
			}
			receipts <- receipt
		}()
	}
	wg.Wait()
	close(errorsFound)
	close(receipts)
	for err := range errorsFound {
		t.Fatalf("concurrent Launch() = %v", err)
	}
	var first ports.AgentLaunchReceipt
	for receipt := range receipts {
		if first == (ports.AgentLaunchReceipt{}) {
			first = receipt
		} else if receipt != first {
			t.Fatalf("receipt changed: first=%+v got=%+v", first, receipt)
		}
	}
	if client.launches.Load() != workers || client.reconciles.Load() != workers || client.sessionMutations.Load() != 0 {
		t.Fatalf("launches=%d reconciles=%d session_mutations=%d", client.launches.Load(), client.reconciles.Load(), client.sessionMutations.Load())
	}
}

func newNormalSessionClient(
	t *testing.T,
	request ports.AgentLaunchRequest,
	descriptor microvm.DescriptorPerfilLanzamientoV1,
) *sessionLaunchClientStub {
	t.Helper()
	physical := validPhysicalResponse(t, request, descriptor)
	return &sessionLaunchClientStub{
		capabilities: validRemoteCapabilities(), physical: physical,
		work: microvm.RespuestaRevisionTrabajo{Referencia: physical.Referencia, RevisionTrabajo: 1},
		start: microvm.RespuestaSesionTrabajoV1{
			EjecucionRef: physical.Referencia, SesionRef: request.SessionRef.String(), Estado: microvm.EstadoSesionActiva,
			Revision: 4, RevisionTrabajo: 2, RevisionSesion: 1, Cerca: physical.Cerca,
		},
		input: microvm.RespuestaSesionTrabajoV1{
			EjecucionRef: physical.Referencia, SesionRef: request.SessionRef.String(), Estado: microvm.EstadoSesionActiva,
			Revision: 5, RevisionTrabajo: 3, RevisionSesion: 2, Cerca: physical.Cerca,
		},
		reconcileReplies: []reconcileReply{
			{err: &microvm.ErrorRespuesta{Estado: 404, Codigo: "sesion.ausente"}},
			{err: &microvm.ErrorRespuesta{Estado: 404, Codigo: "entrada.ausente"}},
		},
		observeErr: &microvm.ErrorRespuesta{Estado: 404, Codigo: "sesion.ausente"},
	}
}

func resolvedReconciliation(
	request ports.AgentLaunchRequest,
	physical microvm.RespuestaEjecucion,
	revision uint64,
	work uint64,
	session uint64,
) reconcileReply {
	receipt := microvm.RespuestaSesionTrabajoV1{
		EjecucionRef: physical.Referencia, SesionRef: request.SessionRef.String(), Estado: microvm.EstadoSesionActiva,
		Revision: revision, RevisionTrabajo: work, RevisionSesion: session, Cerca: physical.Cerca,
	}
	return reconcileReply{response: microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{
		EjecucionRef: physical.Referencia, SesionRef: request.SessionRef.String(), Cerca: physical.Cerca,
		Estado: microvm.EstadoReconciliacionEntradaResuelta, Comprobante: &receipt,
	}}
}

func pendingReconciliation(request ports.AgentLaunchRequest, physical microvm.RespuestaEjecucion) reconcileReply {
	return reconcileReply{response: microvm.RespuestaReconciliacionEntradaSesionTrabajoV1{
		EjecucionRef: physical.Referencia, SesionRef: request.SessionRef.String(), Cerca: physical.Cerca,
		Estado: microvm.EstadoReconciliacionEntradaPendiente,
	}}
}

func sessionPage(
	request ports.AgentLaunchRequest,
	physical microvm.RespuestaEjecucion,
	state microvm.EstadoSesionTrabajoV1,
	terminal bool,
) microvm.PaginaEventosSesionTrabajoV1 {
	page := microvm.PaginaEventosSesionTrabajoV1{
		EjecucionRef: physical.Referencia, SesionRef: request.SessionRef.String(), Estado: state,
		Revision: 4, RevisionTrabajo: 2, RevisionSesion: 1, Cerca: physical.Cerca,
		Eventos: []microvm.EventoSesionTrabajoV1{}, Terminal: terminal,
	}
	if terminal {
		exitCode := int32(0)
		result := &microvm.ResultadoTerminalSesionTrabajoV1{CodigoSalida: &exitCode}
		page.Eventos = []microvm.EventoSesionTrabajoV1{{
			SesionRef: request.SessionRef.String(), Secuencia: 1,
			Tipo: microvm.TipoEventoSesionFinalizada, Estado: state,
			Revision: 4, RevisionTrabajo: 2, RevisionSesion: 1, Cerca: physical.Cerca,
			Terminal: true, Resultado: result,
		}}
		page.SiguienteCursor = 1
	}
	return page
}

type concurrentSessionSigner struct{ delegate launchGrantPreparer }

func (signer concurrentSessionSigner) Preparar(
	_ context.Context,
	_ ports.AgentLaunchRequest,
	authorized microvm.ContextoAutorizado,
	plan microvm.PlanLanzamiento,
	issuedAt time.Time,
	validity time.Duration,
) (microvm.SolicitudLanzamiento, error) {
	return signer.delegate.Preparar(authorized, plan, issuedAt, validity)
}

type concurrentResolvedSessionClient struct {
	physical         microvm.RespuestaEjecucion
	request          ports.AgentLaunchRequest
	launches         atomic.Int64
	reconciles       atomic.Int64
	sessionMutations atomic.Int64
}

func (client *concurrentResolvedSessionClient) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	return validRemoteCapabilities(), nil
}
func (client *concurrentResolvedSessionClient) Lanzar(context.Context, string, microvm.SolicitudLanzamiento) (microvm.RespuestaEjecucion, error) {
	client.launches.Add(1)
	return client.physical, nil
}
func (client *concurrentResolvedSessionClient) RevisionTrabajo(context.Context, string) (microvm.RespuestaRevisionTrabajo, error) {
	client.sessionMutations.Add(1)
	return microvm.RespuestaRevisionTrabajo{}, nil
}
func (client *concurrentResolvedSessionClient) IniciarSesion(context.Context, string, string, microvm.SolicitudIniciarSesionTrabajoV1) (microvm.RespuestaSesionTrabajoV1, error) {
	client.sessionMutations.Add(1)
	return microvm.RespuestaSesionTrabajoV1{}, nil
}
func (client *concurrentResolvedSessionClient) EnviarEntradaSesion(context.Context, string, string, string, microvm.SolicitudEntradaSesionTrabajoV1) (microvm.RespuestaSesionTrabajoV1, error) {
	client.sessionMutations.Add(1)
	return microvm.RespuestaSesionTrabajoV1{}, nil
}
func (client *concurrentResolvedSessionClient) ReconciliarEntradaSesion(context.Context, string, string, string, uint64) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	client.reconciles.Add(1)
	return resolvedReconciliation(client.request, client.physical, 4, 2, 2).response, nil
}
func (client *concurrentResolvedSessionClient) Observar(context.Context, string) (microvm.RespuestaEjecucion, error) {
	return client.physical, nil
}
func (client *concurrentResolvedSessionClient) LeerEventosSesion(context.Context, string, string, microvm.ConsultaEventosSesionTrabajoV1) (microvm.PaginaEventosSesionTrabajoV1, error) {
	client.sessionMutations.Add(1)
	return microvm.PaginaEventosSesionTrabajoV1{}, nil
}
