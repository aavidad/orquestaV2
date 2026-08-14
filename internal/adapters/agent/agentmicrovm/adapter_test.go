package agentmicrovm

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

type agentLaunchReconcilerContract interface {
	ReconcileLaunch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error)
}

var _ agentLaunchReconcilerContract = (*Adapter)(nil)

type launchClientStub struct {
	capabilities     microvm.RespuestaCapacidades
	capabilitiesErr  error
	response         microvm.RespuestaEjecucion
	launchErr        error
	capabilitiesCall int
	launchKeys       []string
	launchRequests   []microvm.SolicitudLanzamiento
	onCapabilities   func()
	onLaunch         func()
}

type blockingCapabilitiesClient struct {
	*launchClientStub
	mu           sync.Mutex
	calls        int
	firstStarted chan struct{}
	releaseFirst chan struct{}
}

func (client *blockingCapabilitiesClient) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	client.mu.Lock()
	client.calls++
	call := client.calls
	response, err := client.capabilities, client.capabilitiesErr
	client.mu.Unlock()
	if call == 1 {
		close(client.firstStarted)
		<-client.releaseFirst
	}
	return response, err
}

type gateWaitSignalContext struct {
	context.Context
	waitStarted chan struct{}
	once        sync.Once
}

func (ctx *gateWaitSignalContext) Done() <-chan struct{} {
	ctx.once.Do(func() { close(ctx.waitStarted) })
	return ctx.Context.Done()
}

func (client *launchClientStub) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	client.capabilitiesCall++
	if client.onCapabilities != nil {
		client.onCapabilities()
	}
	return client.capabilities, client.capabilitiesErr
}

func (client *launchClientStub) Lanzar(
	_ context.Context,
	key string,
	request microvm.SolicitudLanzamiento,
) (microvm.RespuestaEjecucion, error) {
	client.launchKeys = append(client.launchKeys, key)
	client.launchRequests = append(client.launchRequests, cloneSignedRequestUnchecked(request))
	if client.onLaunch != nil {
		client.onLaunch()
	}
	return client.response, client.launchErr
}

func (client *launchClientStub) Observar(
	context.Context,
	string,
) (microvm.RespuestaEjecucion, error) {
	return microvm.RespuestaEjecucion{}, errors.New("observation not configured")
}

func (client *launchClientStub) LeerEventosSesion(
	context.Context,
	string,
	string,
	microvm.ConsultaEventosSesionTrabajoV1,
) (microvm.PaginaEventosSesionTrabajoV1, error) {
	return microvm.PaginaEventosSesionTrabajoV1{}, errors.New("observation not configured")
}

type launchOnlyClientStub struct{}

func (*launchOnlyClientStub) Capacidades(context.Context) (microvm.RespuestaCapacidades, error) {
	return microvm.RespuestaCapacidades{}, nil
}

func (*launchOnlyClientStub) Lanzar(
	context.Context,
	string,
	microvm.SolicitudLanzamiento,
) (microvm.RespuestaEjecucion, error) {
	return microvm.RespuestaEjecucion{}, nil
}

type signerCall struct {
	ctx      context.Context
	request  ports.AgentLaunchRequest
	context  microvm.ContextoAutorizado
	plan     microvm.PlanLanzamiento
	issuedAt time.Time
	validity time.Duration
}

type launchGrantPreparer interface {
	Preparar(
		microvm.ContextoAutorizado,
		microvm.PlanLanzamiento,
		time.Time,
		time.Duration,
	) (microvm.SolicitudLanzamiento, error)
}

type launchSignerStub struct {
	delegate    launchGrantPreparer
	err         error
	cancel      func()
	mutatePlan  func(*microvm.PlanLanzamiento)
	mutate      func(microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento
	calls       []signerCall
	delegateErr error
}

func (signer *launchSignerStub) Preparar(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	context microvm.ContextoAutorizado,
	plan microvm.PlanLanzamiento,
	issuedAt time.Time,
	validity time.Duration,
) (microvm.SolicitudLanzamiento, error) {
	if signer.mutatePlan != nil {
		signer.mutatePlan(&plan)
	}
	plan.Servicios = append([]microvm.ServicioVsock(nil), plan.Servicios...)
	signer.calls = append(signer.calls, signerCall{ctx, request, context, plan, issuedAt, validity})
	if signer.cancel != nil {
		signer.cancel()
	}
	if signer.err != nil {
		return microvm.SolicitudLanzamiento{}, signer.err
	}
	prepared, err := signer.delegate.Preparar(context, plan, issuedAt, validity)
	signer.delegateErr = err
	if err != nil {
		return microvm.SolicitudLanzamiento{}, err
	}
	if signer.mutate != nil {
		prepared = signer.mutate(cloneSignedRequestUnchecked(prepared))
	}
	return cloneSignedRequestUnchecked(prepared), nil
}

func TestAdapterLaunchAndReconcileSignAndReplayExactPhysicalRequest(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &launchClientStub{capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor)}
	signer := validSigner()
	adapter := mustNewAdapter(t, client, signer, request, descriptor)

	ctx := context.Background()
	first, err := adapter.Launch(ctx, request)
	if err != nil {
		t.Fatalf("Launch() first error = %v", err)
	}
	second, err := adapter.ReconcileLaunch(ctx, request)
	if err != nil {
		t.Fatalf("ReconcileLaunch() replay error = %v", err)
	}
	if first != second {
		t.Fatalf("replay receipt changed: first=%+v second=%+v", first, second)
	}
	if client.capabilitiesCall != 2 || len(client.launchKeys) != 2 ||
		client.launchKeys[0] != request.IdempotencyKey || client.launchKeys[1] != request.IdempotencyKey ||
		!reflect.DeepEqual(client.launchRequests[0], client.launchRequests[1]) {
		t.Fatalf("physical replay changed: capability_calls=%d keys=%v requests=%+v", client.capabilitiesCall, client.launchKeys, client.launchRequests)
	}
	if len(signer.calls) != 2 || !reflect.DeepEqual(signer.calls[0], signer.calls[1]) {
		t.Fatalf("signing material changed: %+v", signer.calls)
	}
	compiled := mustCompile(t, request, descriptor)
	wantCall := signerCall{ctx, request, compiled.Context, compiled.Plan, compiled.IssuedAt, compiled.Validity}
	if !reflect.DeepEqual(signer.calls[0], wantCall) {
		t.Fatalf("signing call = %+v, want %+v", signer.calls[0], wantCall)
	}
	if first.AcceptedAt != compiled.IssuedAt || first.ExternalRef != client.response.Referencia ||
		first.ReceiptRef != launchReceiptRef(client.launchRequests[0]) ||
		first.ProviderRef != adapter.capabilities.ProviderRef || first.ModelRef != adapter.capabilities.ModelRef ||
		first.AgentRef != adapter.capabilities.AgentRef {
		t.Fatalf("receipt = %+v", first)
	}
	if err := ports.ValidateAgentLaunchReceipt(request, first); err != nil {
		t.Fatalf("ValidateAgentLaunchReceipt() = %v", err)
	}

	client.launchErr = &microvm.ErrorRespuesta{Estado: 409, Codigo: "api.idempotencia_conflictiva"}
	mutated := request
	mutated.EffectAuthority.ActionFence++
	if _, err := adapter.ReconcileLaunch(ctx, mutated); ErrorCode(err) != CodeLaunchAuthorityLookupFailed ||
		isDefinitelyNotApplied(err) {
		t.Fatalf("ReconcileLaunch() divergent replay error=%v code=%q", err, ErrorCode(err))
	}
	if len(client.launchKeys) != 2 {
		t.Fatalf("divergent replay crossed sibling: keys=%v requests=%+v", client.launchKeys, client.launchRequests)
	}
}

func TestAdapterDurableEgressAuthorityCrossesOnlyTheSignedPlan(t *testing.T) {
	grant := validEgressGrant()
	grant.Referencia = "egreso:private_marker"
	grant.Destinos[0].Host = "private-marker.example"
	request := withEgressAuthority(t, validLaunchRequest(t), grant)
	descriptor := validDescriptor(t, true)
	client := &launchClientStub{
		capabilities: validRemoteCapabilities(),
		response:     validPhysicalResponse(t, request, descriptor),
	}
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if len(client.launchRequests) != 1 ||
		!strings.Contains(string(client.launchRequests[0].Plan), "private-marker.example") {
		t.Fatalf("grant did not cross its required signed-plan boundary: %+v", client.launchRequests)
	}
	receiptJSON, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"egreso:private_marker", "private-marker.example"} {
		if strings.Contains(string(receiptJSON), private) {
			t.Fatalf("egress grant leaked into receipt: %s", receiptJSON)
		}
	}

	invalidGrant := validEgressGrant()
	invalidGrant.Destinos[0].Host = "PRIVATE-MARKER.INVALID"
	invalidRequest := withEgressAuthority(t, request, invalidGrant)
	_, err = Compile(invalidRequest, config.ModelBinding.Profile, invalidRequest.EffectAuthority.ActionFence)
	if ErrorCode(err) != CodePlanInvalid || strings.Contains(err.Error(), "PRIVATE-MARKER") {
		t.Fatalf("invalid grant error leaked detail: code=%q error=%v", ErrorCode(err), err)
	}
}

func TestAdapterRejectsSignerPlanMutationBeforeSocket(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	mutatedProfile := strings.Repeat("f", 64)
	tests := map[string]func(microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento{
		"null": func(value microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
			value.Plan = json.RawMessage("null")
			return value
		},
		"trailing": func(value microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
			value.Plan = append(value.Plan, []byte(` {}`)...)
			return value
		},
		"unknown": func(value microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
			var object map[string]any
			if err := json.Unmarshal(value.Plan, &object); err != nil {
				panic(err)
			}
			object["campo_desconocido"] = true
			value.Plan, _ = json.Marshal(object)
			return value
		},
		"schema":   mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.Esquema = "agentmicrovm.plan-lanzamiento.v0" }),
		"plan ref": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.PlanRef = "plan:" + strings.Repeat("0", 64) }),
		"run ref":  mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.RunRef = "execution:other" }),
		"fence":    mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.Cerca++ }),
		"vcpu":     mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.VCPU++ }),
		"memory":   mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.MemoriaMiB++ }),
		"kernel":   mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.KernelSHA256 = strings.Repeat("0", 64) }),
		"initramfs": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) {
			plan.InitramfsSHA256 = strings.Repeat("0", 64)
		}),
		"profile": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.PerfilSHA256 = &mutatedProfile }),
		"time limit": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) {
			plan.LimiteTiempoMS++
		}),
		"ram limit": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) {
			plan.LimiteRAMPicoBytes++
		}),
		"disk limit": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) {
			plan.LimiteDiscoPicoBytes++
		}),
		"token limit": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) {
			plan.LimiteTokensAgente++
		}),
		"services": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) { plan.Servicios[0].Puerto++ }),
		"egress": mutateSignedPlan(func(plan *microvm.PlanLanzamiento) {
			plan.Egreso = &microvm.ConcesionEgreso{Esquema: microvm.EsquemaConcesionEgreso}
		}),
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			client := &launchClientStub{
				capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
			}
			signer := validSigner()
			signer.mutate = mutate
			adapter := mustNewAdapter(t, client, signer, request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != CodeSigningFailed || !isDefinitelyNotApplied(err) || len(client.launchRequests) != 0 {
				t.Fatalf("Launch() error=%v code=%q definite=%v launches=%d", err, ErrorCode(err), isDefinitelyNotApplied(err), len(client.launchRequests))
			}
		})
	}
}

func TestAdapterRejectsInPlaceSignerMutationWithoutAliasingCompiledPlan(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	signer := validSigner()
	signer.mutatePlan = func(plan *microvm.PlanLanzamiento) {
		plan.Servicios[0].Puerto++
		plan.Servicios = append(plan.Servicios, microvm.ServicioVsock{
			Papel: "controlled_egress_proxy", ServicioRef: "servicio:egreso", Puerto: 10_003,
			IdentidadRef: "identidad-servicio:egreso", IdentidadSHA256: strings.Repeat("8", 64),
		})
		plan.Egreso = &microvm.ConcesionEgreso{
			Esquema: microvm.EsquemaConcesionEgreso, Referencia: "egreso:test",
			Destinos:         []microvm.DestinoEgreso{{Host: "example.com", Puertos: []uint16{80}}},
			MaximoConexiones: 1, LimiteTiempoMS: plan.LimiteTiempoMS,
			LimiteSubidaBytes: 1, LimiteBajadaBytes: 1,
		}
		plan.Egreso.Destinos[0].Puertos[0] = 443
	}
	adapter := mustNewAdapter(t, client, signer, request, descriptor)

	_, err := adapter.Launch(context.Background(), request)
	if ErrorCode(err) != CodeSigningFailed || signer.delegateErr != nil || len(client.launchRequests) != 0 {
		t.Fatalf("Launch() error=%v code=%q signer=%v launches=%d", err, ErrorCode(err), signer.delegateErr, len(client.launchRequests))
	}
	compiled := mustCompile(t, request, descriptor)
	if compiled.Plan.Servicios[0].Puerto != 10_001 || len(compiled.Plan.Servicios) != 1 || compiled.Plan.Egreso != nil {
		t.Fatalf("canonical compilation was changed: %+v", compiled.Plan)
	}
}

func TestCloneLaunchPlanOwnsEveryMutableLevel(t *testing.T) {
	profileSHA := strings.Repeat("a", 64)
	original := microvm.PlanLanzamiento{
		PerfilSHA256: &profileSHA,
		Servicios:    []microvm.ServicioVsock{{Puerto: 10_001}},
		Egreso: &microvm.ConcesionEgreso{Destinos: []microvm.DestinoEgreso{
			{Host: "example.com", Puertos: []uint16{80, 443}},
		}},
	}
	cloned := cloneLaunchPlan(original)
	*cloned.PerfilSHA256 = strings.Repeat("b", 64)
	cloned.Servicios[0].Puerto++
	cloned.Egreso.Destinos[0].Host = "other.example"
	cloned.Egreso.Destinos[0].Puertos[0] = 8080

	if *original.PerfilSHA256 != strings.Repeat("a", 64) || original.Servicios[0].Puerto != 10_001 ||
		original.Egreso.Destinos[0].Host != "example.com" || original.Egreso.Destinos[0].Puertos[0] != 80 {
		t.Fatalf("clone retained mutable aliases: original=%+v clone=%+v", original, cloned)
	}
}

func TestCloneProfileBindingOwnsDescriptorServices(t *testing.T) {
	request := validLaunchRequest(t)
	original := profileBinding(request, validDescriptor(t, true))
	cloned := cloneProfileBinding(original)

	cloned.Descriptor.ServiciosDisponibles[0].Puerto++
	if original.Descriptor.ServiciosDisponibles[0].Puerto != 10_001 {
		t.Fatalf("profile clone retained mutable aliases: original=%+v clone=%+v", original, cloned)
	}
}

func TestAdapterCapabilitiesNegotiatesCompleteRuntimeBeforeReturningNeutralFacts(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &launchClientStub{capabilities: validRemoteCapabilities()}
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)

	first, err := adapter.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() error = %v", err)
	}
	want := validAdapterCapabilities()
	if !reflect.DeepEqual(first, want) {
		t.Fatalf("Capabilities() = %+v, want %+v", first, want)
	}
	first.RoleKeys[0] = "role:mutated"
	second, err := adapter.Capabilities(context.Background())
	if err != nil || !reflect.DeepEqual(second, want) {
		t.Fatalf("capability storage was aliased: second=%+v err=%v", second, err)
	}
}

func TestAdapterNegotiatedPhysicalCapacityFailsClosedThenReturnsImmutableProfilePlacement(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	remote := validRemoteCapabilities()
	remote.MaximoEjecuciones = 5
	client := &launchClientStub{capabilities: remote}
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	wantPlacement := config.ModelBinding.Profile.PlacementRef
	adapter, err := New(config)
	if err != nil {
		t.Fatalf("New() = %v", err)
	}

	before, err := adapter.NegotiatedPhysicalCapacity()
	if before != (NegotiatedPhysicalCapacity{}) || ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable ||
		!isTemporary(err) || !isDefinitelyNotApplied(err) {
		t.Fatalf("capacity before negotiation=%+v error=%v code=%q", before, err, ErrorCode(err))
	}
	otherPlacement, _ := ports.NewAgentPlacementRef("placement:mutated-after-new")
	config.ModelBinding.Profile.PlacementRef = otherPlacement
	if _, err := adapter.Capabilities(context.Background()); err != nil {
		t.Fatalf("Capabilities() = %v", err)
	}

	capacity, err := adapter.NegotiatedPhysicalCapacity()
	if err != nil || capacity.PlacementRef != wantPlacement || capacity.Slots != 5 {
		t.Fatalf("NegotiatedPhysicalCapacity()=%+v error=%v", capacity, err)
	}
	capacity.PlacementRef = otherPlacement
	capacity.Slots = 999
	again, err := adapter.NegotiatedPhysicalCapacity()
	if err != nil || again.PlacementRef != wantPlacement || again.Slots != 5 {
		t.Fatalf("capacity storage was aliased: %+v error=%v", again, err)
	}
}

func TestAdapterNegotiatedPhysicalCapacityClearsPriorOnFailedNegotiation(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	tests := []struct {
		name   string
		mutate func(*launchClientStub)
	}{
		{
			name: "incomplete operations",
			mutate: func(client *launchClientStub) {
				removeOperation(&client.capabilities, operationReadSessionEvents)
			},
		},
		{
			name: "invalid physical maximum",
			mutate: func(client *launchClientStub) {
				client.capabilities.MaximoEjecuciones = 0
			},
		},
		{
			name: "transport failure",
			mutate: func(client *launchClientStub) {
				client.capabilitiesErr = errors.New("socket unavailable")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &launchClientStub{capabilities: validRemoteCapabilities()}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			if _, err := adapter.Capabilities(context.Background()); err != nil {
				t.Fatalf("initial Capabilities() = %v", err)
			}
			test.mutate(client)
			if _, err := adapter.Capabilities(context.Background()); err == nil {
				t.Fatal("failed negotiation returned success")
			}
			capacity, err := adapter.NegotiatedPhysicalCapacity()
			if capacity != (NegotiatedPhysicalCapacity{}) || ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable {
				t.Fatalf("stale capacity survived: %+v error=%v code=%q", capacity, err, ErrorCode(err))
			}
		})
	}
}

func TestAdapterNegotiatedPhysicalCapacityClearsBeforeQueryAndPublishesRemoteChange(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	remote := validRemoteCapabilities()
	remote.MaximoEjecuciones = 5
	client := &launchClientStub{capabilities: remote}
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
	if _, err := adapter.Capabilities(context.Background()); err != nil {
		t.Fatalf("initial Capabilities() = %v", err)
	}

	codeDuringQuery := ""
	client.capabilities.MaximoEjecuciones = 20
	client.onCapabilities = func() {
		_, err := adapter.NegotiatedPhysicalCapacity()
		codeDuringQuery = ErrorCode(err)
	}
	if _, err := adapter.Capabilities(context.Background()); err != nil {
		t.Fatalf("updated Capabilities() = %v", err)
	}
	capacity, err := adapter.NegotiatedPhysicalCapacity()
	if codeDuringQuery != CodeNegotiatedPhysicalCapacityUnavailable || err != nil || capacity.Slots != 20 ||
		capacity.PlacementRef != request.ReferenciaColocacion {
		t.Fatalf("during=%q capacity=%+v error=%v", codeDuringQuery, capacity, err)
	}
}

func TestAdapterNegotiationCancellationSupersedesBlockedGenerationWithoutStalePublication(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &blockingCapabilitiesClient{
		launchClientStub: &launchClientStub{capabilities: validRemoteCapabilities()},
		firstStarted:     make(chan struct{}),
		releaseFirst:     make(chan struct{}),
	}
	defer func() {
		select {
		case <-client.releaseFirst:
		default:
			close(client.releaseFirst)
		}
	}()
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)

	firstResult := make(chan error, 1)
	go func() {
		_, err := adapter.Capabilities(context.Background())
		firstResult <- err
	}()
	<-client.firstStarted
	if capacity, err := adapter.NegotiatedPhysicalCapacity(); capacity != (NegotiatedPhysicalCapacity{}) ||
		ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable {
		t.Fatalf("blocked first negotiation exposed capacity=%+v error=%v", capacity, err)
	}

	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	waitStarted := make(chan struct{})
	secondContext := &gateWaitSignalContext{Context: base, waitStarted: waitStarted}
	secondResult := make(chan error, 1)
	go func() {
		_, err := adapter.Capabilities(secondContext)
		secondResult <- err
	}()
	<-waitStarted
	if capacity, err := adapter.NegotiatedPhysicalCapacity(); capacity != (NegotiatedPhysicalCapacity{}) ||
		ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable {
		t.Fatalf("queued generation exposed capacity=%+v error=%v", capacity, err)
	}
	cancel()
	if err := <-secondResult; err != context.Canceled {
		t.Fatalf("canceled queued negotiation = %v, want literal context.Canceled", err)
	}

	close(client.releaseFirst)
	if err := <-firstResult; err != nil {
		t.Fatalf("stale but otherwise valid first negotiation = %v", err)
	}
	if capacity, err := adapter.NegotiatedPhysicalCapacity(); capacity != (NegotiatedPhysicalCapacity{}) ||
		ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable {
		t.Fatalf("stale first negotiation republished capacity=%+v error=%v", capacity, err)
	}

	if _, err := adapter.Capabilities(context.Background()); err != nil {
		t.Fatalf("third Capabilities() = %v", err)
	}
	capacity, err := adapter.NegotiatedPhysicalCapacity()
	if err != nil || capacity.PlacementRef != request.ReferenciaColocacion || capacity.Slots != 16 {
		t.Fatalf("third capacity=%+v error=%v", capacity, err)
	}
}

func TestAdapterNegotiationGenerationExhaustionFailsClosedWithoutWrapping(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &launchClientStub{capabilities: validRemoteCapabilities()}
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
	adapter.physicalCapacity.generation = ^uint64(0)
	adapter.physicalCapacity.capacity = NegotiatedPhysicalCapacity{
		PlacementRef: request.ReferenciaColocacion,
		Slots:        999,
	}
	adapter.physicalCapacity.available = true

	if _, err := adapter.Capabilities(context.Background()); ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable {
		t.Fatalf("Capabilities() = %v code=%q", err, ErrorCode(err))
	}
	capacity, err := adapter.NegotiatedPhysicalCapacity()
	if adapter.physicalCapacity.generation != ^uint64(0) ||
		capacity != (NegotiatedPhysicalCapacity{}) ||
		ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable || client.capabilitiesCall != 0 {
		t.Fatalf("generation=%d capacity=%+v error=%v calls=%d",
			adapter.physicalCapacity.generation, capacity, err, client.capabilitiesCall)
	}
}

func TestAdapterNegotiatedPhysicalCapacityConcurrentCapabilitiesLaunchAndRead(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &concurrentResolvedSessionClient{
		physical: validPhysicalResponse(t, request, descriptor),
		request:  request,
	}
	signer := concurrentSessionSigner{delegate: validSigner().delegate}
	adapter := mustNewAdapter(t, client, signer, request, descriptor)
	if _, err := adapter.Capabilities(context.Background()); err != nil {
		t.Fatalf("initial Capabilities() = %v", err)
	}

	const workers = 12
	start := make(chan struct{})
	errorsFound := make(chan error, workers*3)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(3)
		go func() {
			defer wait.Done()
			<-start
			_, err := adapter.Capabilities(context.Background())
			if err != nil {
				errorsFound <- err
			}
		}()
		go func() {
			defer wait.Done()
			<-start
			_, err := adapter.Launch(context.Background(), request)
			if err != nil {
				errorsFound <- err
			}
		}()
		go func() {
			defer wait.Done()
			<-start
			capacity, err := adapter.NegotiatedPhysicalCapacity()
			if err != nil {
				if capacity != (NegotiatedPhysicalCapacity{}) || ErrorCode(err) != CodeNegotiatedPhysicalCapacityUnavailable {
					errorsFound <- errors.New("capacity read did not fail closed")
				}
				return
			}
			if capacity.PlacementRef != request.ReferenciaColocacion || capacity.Slots != 16 {
				errorsFound <- errors.New("capacity read returned mixed snapshot")
			}
		}()
	}
	close(start)
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		t.Fatalf("concurrent operation = %v", err)
	}
	if _, err := adapter.Capabilities(context.Background()); err != nil {
		t.Fatalf("final Capabilities() = %v", err)
	}
	capacity, err := adapter.NegotiatedPhysicalCapacity()
	if err != nil || capacity.PlacementRef != request.ReferenciaColocacion || capacity.Slots != 16 {
		t.Fatalf("final capacity=%+v error=%v", capacity, err)
	}
}

func TestAdapterRejectsMutatedPhysicalLaunchResponseAsAmbiguous(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	workRevision := uint64(1)
	processDead := false
	wrongMotor := "Stopped"
	tests := map[string]func(*microvm.RespuestaEjecucion){
		"state":         func(value *microvm.RespuestaEjecucion) { value.Estado = "iniciando" },
		"reference":     func(value *microvm.RespuestaEjecucion) { value.Referencia = "execution:foreign" },
		"fence":         func(value *microvm.RespuestaEjecucion) { value.Cerca++ },
		"vcpu":          func(value *microvm.RespuestaEjecucion) { value.VCPU++ },
		"memory":        func(value *microvm.RespuestaEjecucion) { value.MemoriaMiB++ },
		"revision":      func(value *microvm.RespuestaEjecucion) { value.Revision = 0 },
		"work revision": func(value *microvm.RespuestaEjecucion) { value.RevisionTrabajo = &workRevision },
		"identity":      func(value *microvm.RespuestaEjecucion) { value.Identidad = nil },
		"dead process":  func(value *microvm.RespuestaEjecucion) { value.ProcesoVivo = &processDead },
		"wrong motor":   func(value *microvm.RespuestaEjecucion) { value.EstadoMotor = &wrongMotor },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			response := validPhysicalResponse(t, request, descriptor)
			mutate(&response)
			client := &launchClientStub{capabilities: validRemoteCapabilities(), response: response}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != CodeLaunchResponseInvalid || isDefinitelyNotApplied(err) {
				t.Fatalf("Launch() error=%v code=%q definitely_not_applied=%v", err, ErrorCode(err), isDefinitelyNotApplied(err))
			}
		})
	}
}

func TestAdapterRejectsIncompleteRemoteCapabilitiesBeforeLaunch(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	tests := []struct {
		name      string
		mutate    func(*microvm.RespuestaCapacidades)
		code      string
		temporary bool
	}{
		{"protocol", func(value *microvm.RespuestaCapacidades) { value.Protocolo = "agentmicrovm.local.v0" }, CodeProtocolIncompatible, false},
		{"version", func(value *microvm.RespuestaCapacidades) { value.Version = "" }, CodeProtocolIncompatible, false},
		{"kvm", func(value *microvm.RespuestaCapacidades) { value.KVMDisponible = false }, CodePhysicalUnavailable, true},
		{"configuration", func(value *microvm.RespuestaCapacidades) { value.FirecrackerConfigurado = false }, CodePhysicalUnavailable, true},
		{"executable", func(value *microvm.RespuestaCapacidades) { value.FirecrackerEjecutable = false }, CodePhysicalUnavailable, true},
		{"capacity", func(value *microvm.RespuestaCapacidades) { value.MaximoEjecuciones = 0 }, CodePhysicalUnavailable, true},
		{"create", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationCreateExecution) }, CodeOperationUnsupported, false},
		{"observe execution", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationObserveExecution) }, CodeOperationUnsupported, false},
		{"work revision", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationReadWorkRevision) }, CodeOperationUnsupported, false},
		{"start", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationStartSession) }, CodeOperationUnsupported, false},
		{"input", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationSendSessionInput) }, CodeOperationUnsupported, false},
		{"events", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationReadSessionEvents) }, CodeOperationUnsupported, false},
		{"reconcile", func(value *microvm.RespuestaCapacidades) { removeOperation(value, operationReconcileSessionInput) }, CodeOperationUnsupported, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			remote := validRemoteCapabilities()
			test.mutate(&remote)
			client := &launchClientStub{capabilities: remote, response: validPhysicalResponse(t, request, descriptor)}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.code || isTemporary(err) != test.temporary || !isDefinitelyNotApplied(err) || len(client.launchRequests) != 0 {
				t.Fatalf("Launch() error=%v code=%q temporary=%v definite=%v launches=%d", err, ErrorCode(err), isTemporary(err), isDefinitelyNotApplied(err), len(client.launchRequests))
			}
		})
	}
}

func TestAdapterKeepsClientLaunchErrorsAmbiguous(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	tests := []struct {
		name      string
		err       error
		code      string
		temporary bool
	}{
		{"transport", errors.New("socket closed after write"), CodeLaunchUnavailable, true},
		{"server rejection", &microvm.ErrorRespuesta{Estado: 422, Codigo: "api.entrada_invalida"}, CodeLaunchRejected, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &launchClientStub{
				capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor), launchErr: test.err,
			}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(context.Background(), request)
			if ErrorCode(err) != test.code || isTemporary(err) != test.temporary || isDefinitelyNotApplied(err) {
				t.Fatalf("Launch() error=%v code=%q temporary=%v definite=%v", err, ErrorCode(err), isTemporary(err), isDefinitelyNotApplied(err))
			}
		})
	}
}

func TestAdapterDoesNotInventDefinitelyUnappliedForFutureErrors(t *testing.T) {
	unknown := &Error{Code: "agentmicrovm.future_external_error"}
	if unknown.DefinitelyNotApplied() {
		t.Fatal("an unclassified future error was treated as definitely unapplied")
	}
}

func TestAdapterRejectsInvalidConfigurationAndSigningBeforeSocketMutation(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	config := validAdapterConfig(&launchClientStub{}, validSigner(), request, descriptor)
	config.Client = nil
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeConfigurationInvalid || !isDefinitelyNotApplied(err) {
		t.Fatalf("New() adapter=%v error=%v", adapter, err)
	}
	var nilSigner *launchSignerStub
	config = validAdapterConfig(&launchClientStub{}, nilSigner, request, descriptor)
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeConfigurationInvalid {
		t.Fatalf("New() accepted typed nil signer: adapter=%v error=%v", adapter, err)
	}
	config = validAdapterConfig(&launchClientStub{}, validSigner(), request, descriptor)
	config.Capabilities.RequierePreservacionEntorno = false
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeConfigurationInvalid {
		t.Fatalf("New() accepted disposable microVM: adapter=%v error=%v", adapter, err)
	}
	config = validAdapterConfig(&launchOnlyClientStub{}, validSigner(), request, descriptor)
	if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeObservationClientInvalid {
		t.Fatalf("New() accepted client without read-only observation: adapter=%v error=%v", adapter, err)
	}

	client := &launchClientStub{capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor)}
	signer := validSigner()
	signer.err = errors.New("key unavailable")
	adapter := mustNewAdapter(t, client, signer, request, descriptor)
	if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeSigningFailed ||
		!isDefinitelyNotApplied(err) || len(client.launchRequests) != 0 {
		t.Fatalf("signing error=%v launches=%d", err, len(client.launchRequests))
	}
}

func TestAdapterPreservesContextCanceledDuringSigningBeforeSocketMutation(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	signer := validSigner()
	ctx, cancel := context.WithCancel(context.Background())
	signer.cancel = cancel
	adapter := mustNewAdapter(t, client, signer, request, descriptor)

	_, err := adapter.Launch(ctx, request)
	if !errors.Is(err, context.Canceled) || ErrorCode(err) != "" || len(client.launchRequests) != 0 {
		t.Fatalf("Launch() error=%v code=%q launches=%d", err, ErrorCode(err), len(client.launchRequests))
	}
}

func TestAdapterRejectsNilAndPreCanceledContextsBeforeTransport(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	client := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
	methods := map[string]func(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error){
		"launch":    adapter.Launch,
		"reconcile": adapter.ReconcileLaunch,
	}
	for name, method := range methods {
		t.Run(name, func(t *testing.T) {
			var nilContext context.Context
			if _, err := method(nilContext, request); ErrorCode(err) != CodeConfigurationInvalid {
				t.Fatalf("nil context error=%v code=%q", err, ErrorCode(err))
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := method(ctx, request); !errors.Is(err, context.Canceled) || ErrorCode(err) != "" {
				t.Fatalf("canceled context error=%v code=%q", err, ErrorCode(err))
			}
		})
	}
	if client.capabilitiesCall != 0 || len(client.launchRequests) != 0 {
		t.Fatalf("context rejection touched transport: capabilities=%d launches=%d", client.capabilitiesCall, len(client.launchRequests))
	}
}

func TestAdapterPreservesContextErrorsFromBothTransports(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	tests := []struct {
		name      string
		build     func(context.CancelFunc) *launchClientStub
		wantError error
	}{
		{
			name: "capabilities deadline",
			build: func(context.CancelFunc) *launchClientStub {
				return &launchClientStub{capabilitiesErr: context.DeadlineExceeded}
			},
			wantError: context.DeadlineExceeded,
		},
		{
			name: "capabilities canceled with generic transport error",
			build: func(cancel context.CancelFunc) *launchClientStub {
				return &launchClientStub{onCapabilities: cancel, capabilitiesErr: errors.New("transport stopped")}
			},
			wantError: context.Canceled,
		},
		{
			name: "launch canceled",
			build: func(context.CancelFunc) *launchClientStub {
				return &launchClientStub{
					capabilities: validRemoteCapabilities(), launchErr: context.Canceled,
				}
			},
			wantError: context.Canceled,
		},
		{
			name: "launch canceled with generic transport error",
			build: func(cancel context.CancelFunc) *launchClientStub {
				return &launchClientStub{
					capabilities: validRemoteCapabilities(), onLaunch: cancel,
					launchErr: errors.New("socket stopped"),
				}
			},
			wantError: context.Canceled,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := test.build(cancel)
			if client.response.Referencia == "" {
				client.response = validPhysicalResponse(t, request, descriptor)
			}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Launch(ctx, request)
			if !errors.Is(err, test.wantError) || ErrorCode(err) != "" {
				t.Fatalf("Launch() error=%v code=%q, want %v", err, ErrorCode(err), test.wantError)
			}
		})
	}
}

func TestAdapterClassifiesCapabilityQueryWithoutInventingLaunchEffect(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	tests := []struct {
		name      string
		err       error
		code      string
		temporary bool
	}{
		{"transport", errors.New("socket unavailable"), CodeCapabilitiesUnavailable, true},
		{"protocol", &microvm.ErrorProtocolo{Recibido: "agentmicrovm.local.v0"}, CodeProtocolIncompatible, false},
		{"rejected", &microvm.ErrorRespuesta{Estado: 404, Codigo: "api.ruta_ausente"}, CodeCapabilitiesRejected, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &launchClientStub{capabilitiesErr: test.err}
			adapter := mustNewAdapter(t, client, validSigner(), request, descriptor)
			_, err := adapter.Capabilities(context.Background())
			if ErrorCode(err) != test.code || isTemporary(err) != test.temporary || !isDefinitelyNotApplied(err) {
				t.Fatalf("Capabilities() error=%v code=%q temporary=%v definite=%v", err, ErrorCode(err), isTemporary(err), isDefinitelyNotApplied(err))
			}
		})
	}
}

func validAdapterConfig(
	client Client,
	signer Signer,
	request ports.AgentLaunchRequest,
	descriptor microvm.DescriptorPerfilLanzamientoV1,
) Config {
	return Config{
		Client:                  client,
		Signer:                  signer,
		ClaimResolver:           validPipelineClaimResolver(request),
		LaunchAuthorityRegistry: newPipelineAuthorityRegistryStub(),
		Capabilities:            validAdapterCapabilities(),
		ModelBinding: ProviderModelBinding{
			ModelRef: "model:codex-microvm", ProviderModel: "gpt-5.6",
			Profile: profileBinding(request, descriptor),
		},
		PromptRenderer: staticSessionPromptRenderer("work"),
	}
}

func mustNewAdapter(
	t *testing.T,
	client Client,
	signer Signer,
	request ports.AgentLaunchRequest,
	descriptor microvm.DescriptorPerfilLanzamientoV1,
) *Adapter {
	t.Helper()
	adapter, err := New(validAdapterConfig(client, signer, request, descriptor))
	if err != nil {
		t.Fatalf("New() = %v", err)
	}
	return adapter
}

func validAdapterCapabilities() ports.AgentCapabilities {
	return ports.AgentCapabilities{
		ProviderRef: "provider:codex", ModelRef: "model:codex-microvm", AgentRef: "agent:codex-microvm",
		RequierePreservacionEntorno: true,
		RoleKeys:                    []string{"role:worker"}, SkillRefs: []string{"skill:go"},
		ToolRefs: []string{"tool:test"}, CapabilityRefs: []string{"capability:code"},
	}
}

func validRemoteCapabilities() microvm.RespuestaCapacidades {
	return microvm.RespuestaCapacidades{
		Protocolo: microvm.ProtocoloLocal, Version: "0.1.0",
		Operaciones: []string{
			"salud", "capacidades", operationCreateExecution, operationObserveExecution,
			operationReadWorkRevision, operationStartSession,
			operationSendSessionInput, operationReadSessionEvents, operationReconcileSessionInput,
		},
		KVMDisponible: true, FirecrackerConfigurado: true, FirecrackerEjecutable: true,
		MaximoEjecuciones: 16,
	}
}

func validSigner() *launchSignerStub {
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	signer, err := microvm.NuevoFirmanteConcesiones("clave-publica:test", privateKey)
	if err != nil {
		panic(err)
	}
	return &launchSignerStub{delegate: signer}
}

func mutateSignedPlan(
	mutate func(*microvm.PlanLanzamiento),
) func(microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
	return func(request microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
		var plan microvm.PlanLanzamiento
		if err := json.Unmarshal(request.Plan, &plan); err != nil {
			panic(err)
		}
		mutate(&plan)
		request.Plan, _ = json.Marshal(plan)
		return request
	}
}

func validPhysicalResponse(
	t *testing.T,
	request ports.AgentLaunchRequest,
	descriptor microvm.DescriptorPerfilLanzamientoV1,
) microvm.RespuestaEjecucion {
	t.Helper()
	return microvm.RespuestaEjecucion{
		Referencia: "ejecucion:" + strings.Repeat("a", 64), Estado: "disponible", Revision: 3,
		Cerca: request.EffectAuthority.ActionFence, VCPU: descriptor.VCPU, MemoriaMiB: descriptor.MemoriaMiB,
		Identidad: &microvm.IdentidadProceso{PID: 1234, InicioTicks: 5678},
	}
}

func removeOperation(capabilities *microvm.RespuestaCapacidades, unwanted string) {
	filtered := capabilities.Operaciones[:0]
	for _, operation := range capabilities.Operaciones {
		if operation != unwanted {
			filtered = append(filtered, operation)
		}
	}
	capabilities.Operaciones = filtered
}

func isTemporary(err error) bool {
	var temporary interface{ Temporary() bool }
	return errors.As(err, &temporary) && temporary.Temporary()
}

func isDefinitelyNotApplied(err error) bool {
	var definite interface{ DefinitelyNotApplied() bool }
	return errors.As(err, &definite) && definite.DefinitelyNotApplied()
}
