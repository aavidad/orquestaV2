package agentmicrovm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestCredentialBrokerHappyPathProjectsThenPurges(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte(`{"tokens":{"access_token":"secret-marker"}}`))
	broker := fixture.broker(t, time.Second)

	err, header, material := runCredentialBrokerExchange(t, broker, fixture.aperture, fixture.request, exchangeBehavior{})
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if header.Estado != microvm.EstadoRespuestaAuthJSONCodexDisponible ||
		header.CodigoError != nil || string(material) != string(fixture.store.material) {
		t.Fatalf("unexpected response: %#v material=%q", header, material)
	}
	if fixture.store.callbackCalls() != 1 {
		t.Fatalf("callback calls = %d, want 1", fixture.store.callbackCalls())
	}
	if !fixture.store.callbackSecretWasCleared() {
		t.Fatal("broker did not destroy callback-scoped Secret")
	}
}

func TestCredentialBrokerRejectsEveryCrossedApertureBindingBeforeClaim(t *testing.T) {
	base := newCredentialBrokerFixture(t, []byte("secret-marker"))
	tests := map[string]func(*microvm.AperturaServicioHostV1){
		"role": func(value *microvm.AperturaServicioHostV1) {
			value.Papel = microvm.PapelServicioHostControlledEgressProxy
		},
		"external_ref":    func(value *microvm.AperturaServicioHostV1) { value.EjecucionRef = "ejecucion:crossed" },
		"run_ref":         func(value *microvm.AperturaServicioHostV1) { value.RunRef = "run:crossed" },
		"fence":           func(value *microvm.AperturaServicioHostV1) { value.Cerca++ },
		"plan":            func(value *microvm.AperturaServicioHostV1) { value.PlanSHA256 = strings.Repeat("d", 64) },
		"concession":      func(value *microvm.AperturaServicioHostV1) { value.ConcesionSHA256 = strings.Repeat("e", 64) },
		"service_ref":     func(value *microvm.AperturaServicioHostV1) { value.ServicioRef = "servicio:crossed" },
		"identity_ref":    func(value *microvm.AperturaServicioHostV1) { value.IdentidadRef = "identidad-servicio:crossed" },
		"identity_digest": func(value *microvm.AperturaServicioHostV1) { value.IdentidadSHA256 = strings.Repeat("f", 64) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			aperture := base.aperture
			mutate(&aperture)
			broker := base.broker(t, time.Second)
			server, client := net.Pipe()
			result := make(chan error, 1)
			go func() { result <- broker.Handle(context.Background(), server) }()
			writeAperture(t, client, aperture)
			_ = client.Close()
			err := awaitBroker(t, result)
			_ = server.Close()
			if ErrorCode(err) != CodeCredentialBrokerAuthorityDenied {
				t.Fatalf("error = %v", err)
			}
		})
	}
	if base.store.callbackCalls() != 0 {
		t.Fatalf("crossed apertures consumed credential %d times", base.store.callbackCalls())
	}
}

func TestCredentialBrokerRejectsPreparedAuthority(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	fixture.registry.authority.ExternalRef = ""
	broker := fixture.broker(t, time.Second)
	server, client := net.Pipe()
	result := make(chan error, 1)
	go func() { result <- broker.Handle(context.Background(), server) }()
	writeAperture(t, client, fixture.aperture)
	_ = client.Close()
	if err := awaitBroker(t, result); ErrorCode(err) != CodeCredentialBrokerAuthorityDenied {
		t.Fatalf("error = %v", err)
	}
	_ = server.Close()
	if fixture.store.callbackCalls() != 0 {
		t.Fatal("prepared authority reached credential store")
	}
}

func TestCredentialBrokerRejectsCrossedRequestSessionAndFence(t *testing.T) {
	tests := map[string]func(*microvm.SolicitudAuthJSONCodexV1){
		"session": func(value *microvm.SolicitudAuthJSONCodexV1) { value.SesionRef = "execution-session:crossed" },
		"fence":   func(value *microvm.SolicitudAuthJSONCodexV1) { value.Cerca++ },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
			request := fixture.request
			mutate(&request)
			err, header, material := runCredentialBrokerExchange(t, fixture.broker(t, time.Second), fixture.aperture, request, exchangeBehavior{expectRejected: true})
			if ErrorCode(err) != CodeCredentialBrokerAuthorityDenied || len(material) != 0 ||
				header.CodigoError == nil || *header.CodigoError != microvm.CodigoErrorAuthJSONCodexAutoridadInvalida {
				t.Fatalf("error=%v header=%#v material=%q", err, header, material)
			}
			if fixture.store.callbackCalls() != 0 {
				t.Fatal("crossed request reached callback")
			}
		})
	}
}

func TestCredentialBrokerReplayNeverInvokesCallbackTwice(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	broker := fixture.broker(t, time.Second)
	if err, _, _ := runCredentialBrokerExchange(t, broker, fixture.aperture, fixture.request, exchangeBehavior{}); err != nil {
		t.Fatalf("first Handle: %v", err)
	}
	err, header, material := runCredentialBrokerExchange(t, broker, fixture.aperture, fixture.request, exchangeBehavior{expectRejected: true})
	if ErrorCode(err) != CodeCredentialBrokerReplayRejected || len(material) != 0 ||
		header.CodigoError == nil || *header.CodigoError != microvm.CodigoErrorAuthJSONCodexRepeticionRechazada {
		t.Fatalf("replay error=%v header=%#v material=%q", err, header, material)
	}
	if fixture.store.callbackCalls() != 1 {
		t.Fatalf("callback calls = %d, want 1", fixture.store.callbackCalls())
	}
}

func TestCredentialBrokerConcurrentClaimHasSingleWinner(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	broker := fixture.broker(t, 2*time.Second)
	start := make(chan struct{})
	results := make(chan error, 2)
	for index := 0; index < 2; index++ {
		go func() {
			<-start
			err, _, _ := runCredentialBrokerExchangeNoTestFailure(broker, fixture.aperture, fixture.request)
			results <- err
		}()
	}
	close(start)
	var success, replay int
	for index := 0; index < 2; index++ {
		switch code := ErrorCode(<-results); code {
		case "":
			success++
		case CodeCredentialBrokerReplayRejected:
			replay++
		default:
			t.Fatalf("unexpected result code %q", code)
		}
	}
	if success != 1 || replay != 1 || fixture.store.callbackCalls() != 1 {
		t.Fatalf("success=%d replay=%d callback=%d", success, replay, fixture.store.callbackCalls())
	}
}

func TestCredentialBrokerPostClaimFailuresAreAmbiguousAndConsumed(t *testing.T) {
	tests := map[string]exchangeBehavior{
		"projection_negative":        {projectionFailure: true},
		"projection_eof":             {closeAfterMaterial: true},
		"projection_crossed_session": {mutateProjection: func(ack *microvm.AcuseAuthJSONCodexV1) { ack.SesionRef = "execution-session:crossed" }},
		"projection_crossed_fence":   {mutateProjection: func(ack *microvm.AcuseAuthJSONCodexV1) { ack.Cerca++ }},
		"projection_crossed_stage":   {mutateProjection: func(ack *microvm.AcuseAuthJSONCodexV1) { ack.Etapa = microvm.EtapaAcuseAuthJSONCodexPurgada }},
		"purge_negative":             {purgeFailure: true},
		"purge_eof":                  {closeBeforePurge: true},
		"purge_crossed_session":      {mutatePurge: func(ack *microvm.AcuseAuthJSONCodexV1) { ack.SesionRef = "execution-session:crossed" }},
		"purge_crossed_fence":        {mutatePurge: func(ack *microvm.AcuseAuthJSONCodexV1) { ack.Cerca++ }},
		"purge_crossed_stage":        {mutatePurge: func(ack *microvm.AcuseAuthJSONCodexV1) { ack.Etapa = microvm.EtapaAcuseAuthJSONCodexProyectada }},
	}
	for name, behavior := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
			broker := fixture.broker(t, time.Second)
			err, _, _ := runCredentialBrokerExchange(t, broker, fixture.aperture, fixture.request, behavior)
			if ErrorCode(err) != CodeCredentialBrokerAmbiguous {
				t.Fatalf("error = %v", err)
			}
			if fixture.store.callbackCalls() != 1 {
				t.Fatalf("callback calls = %d", fixture.store.callbackCalls())
			}
			replayErr, replayHeader, _ := runCredentialBrokerExchange(t, broker, fixture.aperture, fixture.request, exchangeBehavior{expectRejected: true})
			if ErrorCode(replayErr) != CodeCredentialBrokerReplayRejected || replayHeader.CodigoError == nil ||
				*replayHeader.CodigoError != microvm.CodigoErrorAuthJSONCodexRepeticionRechazada {
				t.Fatalf("replay error=%v header=%#v", replayErr, replayHeader)
			}
			if fixture.store.callbackCalls() != 1 {
				t.Fatal("ambiguous claim was redelivered")
			}
		})
	}
}

func TestCredentialBrokerProjectionTimeoutIsBoundedAmbiguousAndConsumed(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	broker := fixture.broker(t, 40*time.Millisecond)
	server, client := net.Pipe()
	result := make(chan error, 1)
	go func() { result <- broker.Handle(context.Background(), server) }()
	writeAperture(t, client, fixture.aperture)
	writeRequest(t, client, fixture.request)
	header, material := readBrokerResponse(t, client)
	if header.Estado != microvm.EstadoRespuestaAuthJSONCodexDisponible || string(material) != "secret-marker" {
		t.Fatalf("response=%#v material=%q", header, material)
	}
	started := time.Now()
	err := awaitBrokerWithin(t, result, time.Second)
	_ = client.Close()
	_ = server.Close()
	if ErrorCode(err) != CodeCredentialBrokerAmbiguous || time.Since(started) > 500*time.Millisecond {
		t.Fatalf("error=%v elapsed=%s", err, time.Since(started))
	}
	if fixture.store.callbackCalls() != 1 {
		t.Fatal("timeout did not retain one-shot claim")
	}
}

func TestCredentialBrokerRejectsOversizeMaterialWithoutWritingOrRetainingIt(t *testing.T) {
	material := bytes.Repeat([]byte("secret-marker"), microvm.MaximoMaterialAuthJSONCodexV1/len("secret-marker")+1)
	fixture := newCredentialBrokerFixture(t, material)
	broker := fixture.broker(t, time.Second)
	server, client := net.Pipe()
	result := make(chan error, 1)
	go func() { result <- broker.Handle(context.Background(), server) }()
	writeAperture(t, client, fixture.aperture)
	writeRequest(t, client, fixture.request)
	err := awaitBroker(t, result)
	_ = client.Close()
	_ = server.Close()
	if ErrorCode(err) != CodeCredentialBrokerAmbiguous || strings.Contains(err.Error(), "secret-marker") {
		t.Fatalf("unsafe error = %v", err)
	}
	brokerType := reflect.TypeOf(*broker)
	for index := 0; index < brokerType.NumField(); index++ {
		field := brokerType.Field(index)
		if field.Type == reflect.TypeOf([]byte(nil)) || strings.Contains(strings.ToLower(field.Name), "material") || strings.Contains(strings.ToLower(field.Name), "secret") {
			t.Fatalf("broker retains credential-shaped field %s %v", field.Name, field.Type)
		}
	}
}

func TestCredentialBrokerHandlesFragmentedConnectionIO(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	broker := fixture.broker(t, time.Second)
	server, client := net.Pipe()
	fragmented := &fragmentedConn{Conn: server, maximum: 1}
	result := make(chan error, 1)
	go func() { result <- broker.Handle(context.Background(), fragmented) }()
	writeAperture(t, client, fixture.aperture)
	writeRequest(t, client, fixture.request)
	header, material := readBrokerResponse(t, client)
	if header.Estado != microvm.EstadoRespuestaAuthJSONCodexDisponible || string(material) != "secret-marker" {
		t.Fatalf("response=%#v material=%q", header, material)
	}
	writeAck(t, client, fixture.ack(microvm.EtapaAcuseAuthJSONCodexProyectada, true))
	writeAck(t, client, fixture.ack(microvm.EtapaAcuseAuthJSONCodexPurgada, true))
	if err := awaitBroker(t, result); err != nil {
		t.Fatalf("Handle fragmented: %v", err)
	}
	_ = client.Close()
	_ = server.Close()
}

func TestNewCredentialBrokerRequiresAllDependenciesAndTimeout(t *testing.T) {
	fixture := newCredentialBrokerFixture(t, []byte("secret-marker"))
	var nilRegistry *credentialBrokerRegistryStub
	var nilStore *credentialBrokerStoreStub
	for name, test := range map[string]struct {
		registry ports.MicroVMHostLaunchAuthorityRegistry
		store    credentials.OneShotStore
		timeout  time.Duration
	}{
		"nil_registry":   {nil, fixture.store, time.Second},
		"typed_registry": {nilRegistry, fixture.store, time.Second},
		"nil_store":      {fixture.registry, nil, time.Second},
		"typed_store":    {fixture.registry, nilStore, time.Second},
		"zero_timeout":   {fixture.registry, fixture.store, 0},
		"negative":       {fixture.registry, fixture.store, -time.Second},
	} {
		t.Run(name, func(t *testing.T) {
			broker, err := NewCredentialBroker(test.registry, test.store, test.timeout)
			if broker != nil || ErrorCode(err) != CodeCredentialBrokerInvalid {
				t.Fatalf("broker=%#v error=%v", broker, err)
			}
		})
	}
}

type credentialBrokerFixture struct {
	authority ports.MicroVMHostLaunchAuthorityV1
	aperture  microvm.AperturaServicioHostV1
	request   microvm.SolicitudAuthJSONCodexV1
	registry  *credentialBrokerRegistryStub
	store     *credentialBrokerStoreStub
}

func newCredentialBrokerFixture(t *testing.T, material []byte) credentialBrokerFixture {
	t.Helper()
	runRef, err := goal.NewExecutionRef("run:credential-broker")
	if err != nil {
		t.Fatal(err)
	}
	sessionRef, err := ports.NewExecutionSessionRef("execution-session:credential-broker")
	if err != nil {
		t.Fatal(err)
	}
	key := ports.MicroVMHostLaunchAuthorityKey{RunRef: runRef, ActionFence: 17}
	requestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(key, "effect-attempt:credential-broker", sessionRef)
	if err != nil {
		t.Fatal(err)
	}
	authority := ports.MicroVMHostLaunchAuthorityV1{
		Key: key, EffectAttemptRef: "effect-attempt:credential-broker", SessionRef: sessionRef,
		OneShotClaim: credentials.OneShotUseRequest{
			ActorRef: "actor:credential-broker", RequestRef: requestRef,
			CredentialRef: "credential:codex7", OwnerRef: "actor:credential-broker",
			ScopeRef: "project:credential-broker", PurposeRef: "codex_auth_json", Version: 3,
		},
		PlanSHA256: strings.Repeat("a", 64), ConcessionSHA256: strings.Repeat("b", 64),
		Services: []ports.MicroVMHostServiceAuthorityV1{{
			Role: ports.MicroVMHostServiceControlBroker, ServiceRef: "servicio:control", Port: 10_001,
			IdentityRef: "identidad-servicio:control", IdentitySHA256: strings.Repeat("c", 64),
		}},
		ExternalRef: "ejecucion:credential-broker",
	}
	if err := ports.ValidateMicroVMHostLaunchAuthorityBoundV1(authority); err != nil {
		t.Fatalf("fixture authority: %v", err)
	}
	aperture := microvm.AperturaServicioHostV1{
		Protocolo: microvm.ProtocoloAperturaServicioHostV1, EjecucionRef: authority.ExternalRef,
		RunRef: runRef.String(), CIDVsock: 3, PlanSHA256: authority.PlanSHA256,
		ConcesionSHA256: authority.ConcessionSHA256, Cerca: key.ActionFence,
		Papel: microvm.PapelServicioHostControlBroker, ServicioRef: authority.Services[0].ServiceRef,
		IdentidadRef: authority.Services[0].IdentityRef, IdentidadSHA256: authority.Services[0].IdentitySHA256,
		FirecrackerPID: 1234, FirecrackerUID: 987,
	}
	request := microvm.SolicitudAuthJSONCodexV1{
		Protocolo: microvm.ProtocoloAuthJSONCodexOneShotV1,
		Operacion: microvm.OperacionAuthJSONCodexAdquirir,
		SesionRef: sessionRef.String(), Cerca: key.ActionFence,
	}
	registry := &credentialBrokerRegistryStub{authority: ports.CloneMicroVMHostLaunchAuthorityV1(authority)}
	store := &credentialBrokerStoreStub{material: append([]byte(nil), material...)}
	return credentialBrokerFixture{authority: authority, aperture: aperture, request: request, registry: registry, store: store}
}

func (fixture credentialBrokerFixture) broker(t *testing.T, timeout time.Duration) *CredentialBroker {
	t.Helper()
	broker, err := NewCredentialBroker(fixture.registry, fixture.store, timeout)
	if err != nil {
		t.Fatal(err)
	}
	return broker
}

func (fixture credentialBrokerFixture) ack(stage microvm.EtapaAcuseAuthJSONCodexV1, success bool) microvm.AcuseAuthJSONCodexV1 {
	ack := microvm.AcuseAuthJSONCodexV1{
		Protocolo: microvm.ProtocoloAuthJSONCodexOneShotV1,
		SesionRef: fixture.authority.SessionRef.String(), Cerca: fixture.authority.Key.ActionFence,
		Etapa: stage, Correcto: success,
	}
	if !success {
		code := microvm.CodigoErrorAuthJSONCodexProyeccionFallida
		if stage == microvm.EtapaAcuseAuthJSONCodexPurgada {
			code = microvm.CodigoErrorAuthJSONCodexPurgaFallida
		}
		ack.CodigoError = &code
	}
	return ack
}

type credentialBrokerRegistryStub struct {
	authority ports.MicroVMHostLaunchAuthorityV1
	err       error
}

func (registry *credentialBrokerRegistryStub) Prepare(context.Context, ports.MicroVMHostLaunchAuthorityV1) (ports.MicroVMHostLaunchAuthorityV1, error) {
	return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("unexpected Prepare")
}

func (registry *credentialBrokerRegistryStub) BindExternal(context.Context, ports.MicroVMHostLaunchAuthorityKey, string) (ports.MicroVMHostLaunchAuthorityV1, error) {
	return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("unexpected BindExternal")
}

func (registry *credentialBrokerRegistryStub) Resolve(_ context.Context, key ports.MicroVMHostLaunchAuthorityKey) (ports.MicroVMHostLaunchAuthorityV1, error) {
	if registry.err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, registry.err
	}
	if key.RunRef.String() != registry.authority.Key.RunRef.String() || key.ActionFence != registry.authority.Key.ActionFence {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("not found")
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(registry.authority), nil
}

type credentialBrokerStoreStub struct {
	mu       sync.Mutex
	material []byte
	claimed  bool
	receipt  credentials.Receipt
	calls    int
	cleared  bool
}

func (store *credentialBrokerStoreStub) UseOnce(
	_ context.Context,
	request credentials.OneShotUseRequest,
	consume func(credentials.Secret) error,
) (credentials.OneShotUseResult, error) {
	store.mu.Lock()
	if store.claimed {
		result := credentials.OneShotUseResult{Receipt: store.receipt, Replayed: true}
		store.mu.Unlock()
		return result, credentials.NewError(credentials.ErrorAlreadyConsumed, "request_ref")
	}
	store.claimed = true
	store.calls++
	store.receipt = credentials.Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef, ScopeRef: request.ScopeRef,
		PurposeRef: request.PurposeRef, Version: request.Version, RequestRef: request.RequestRef,
		ActorRef: request.ActorRef, Operation: credentials.OperationUseOnce, OccurredAt: time.Now().UTC(),
	}
	receipt := store.receipt
	material := append([]byte(nil), store.material...)
	store.mu.Unlock()

	secret, err := credentials.NewSecret(material)
	clear(material)
	if err != nil {
		return credentials.OneShotUseResult{Receipt: receipt}, credentials.NewError(credentials.ErrorConsumerFailed, "consume")
	}
	consumeErr := consume(secret)
	remaining := secret.Bytes()
	store.mu.Lock()
	store.cleared = len(remaining) > 0
	for _, value := range remaining {
		store.cleared = store.cleared && value == 0
	}
	store.mu.Unlock()
	clear(remaining)
	secret.Destroy()
	if consumeErr != nil {
		return credentials.OneShotUseResult{Receipt: receipt}, credentials.NewError(credentials.ErrorConsumerFailed, "consume")
	}
	return credentials.OneShotUseResult{Receipt: receipt}, nil
}

func (store *credentialBrokerStoreStub) callbackCalls() int {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.calls
}

func (store *credentialBrokerStoreStub) callbackSecretWasCleared() bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	return store.cleared
}

type exchangeBehavior struct {
	expectRejected     bool
	projectionFailure  bool
	closeAfterMaterial bool
	purgeFailure       bool
	closeBeforePurge   bool
	mutateProjection   func(*microvm.AcuseAuthJSONCodexV1)
	mutatePurge        func(*microvm.AcuseAuthJSONCodexV1)
}

func runCredentialBrokerExchange(
	t *testing.T,
	broker *CredentialBroker,
	aperture microvm.AperturaServicioHostV1,
	request microvm.SolicitudAuthJSONCodexV1,
	behavior exchangeBehavior,
) (error, microvm.CabeceraRespuestaAuthJSONCodexV1, []byte) {
	t.Helper()
	server, client := net.Pipe()
	result := make(chan error, 1)
	go func() { result <- broker.Handle(context.Background(), server) }()
	writeAperture(t, client, aperture)
	writeRequest(t, client, request)
	header, material := readBrokerResponse(t, client)
	if behavior.expectRejected {
		err := awaitBroker(t, result)
		_ = client.Close()
		_ = server.Close()
		return err, header, material
	}
	if behavior.closeAfterMaterial {
		_ = client.Close()
		err := awaitBroker(t, result)
		_ = server.Close()
		return err, header, material
	}
	fixtureAck := func(stage microvm.EtapaAcuseAuthJSONCodexV1, success bool) microvm.AcuseAuthJSONCodexV1 {
		ack := microvm.AcuseAuthJSONCodexV1{
			Protocolo: microvm.ProtocoloAuthJSONCodexOneShotV1,
			SesionRef: request.SesionRef, Cerca: request.Cerca, Etapa: stage, Correcto: success,
		}
		if !success {
			code := microvm.CodigoErrorAuthJSONCodexProyeccionFallida
			if stage == microvm.EtapaAcuseAuthJSONCodexPurgada {
				code = microvm.CodigoErrorAuthJSONCodexPurgaFallida
			}
			ack.CodigoError = &code
		}
		return ack
	}
	projectionAck := fixtureAck(microvm.EtapaAcuseAuthJSONCodexProyectada, !behavior.projectionFailure)
	if behavior.mutateProjection != nil {
		behavior.mutateProjection(&projectionAck)
	}
	writeAck(t, client, projectionAck)
	if behavior.projectionFailure || behavior.mutateProjection != nil {
		err := awaitBroker(t, result)
		_ = client.Close()
		_ = server.Close()
		return err, header, material
	}
	if behavior.closeBeforePurge {
		_ = client.Close()
		err := awaitBroker(t, result)
		_ = server.Close()
		return err, header, material
	}
	purgeAck := fixtureAck(microvm.EtapaAcuseAuthJSONCodexPurgada, !behavior.purgeFailure)
	if behavior.mutatePurge != nil {
		behavior.mutatePurge(&purgeAck)
	}
	writeAck(t, client, purgeAck)
	err := awaitBroker(t, result)
	_ = client.Close()
	_ = server.Close()
	return err, header, material
}

func runCredentialBrokerExchangeNoTestFailure(
	broker *CredentialBroker,
	aperture microvm.AperturaServicioHostV1,
	request microvm.SolicitudAuthJSONCodexV1,
) (error, microvm.CabeceraRespuestaAuthJSONCodexV1, []byte) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	result := make(chan error, 1)
	go func() { result <- broker.Handle(context.Background(), server) }()
	apertureFrame, err := microvm.CodificarAperturaServicioHostV1(aperture)
	if err != nil {
		return err, microvm.CabeceraRespuestaAuthJSONCodexV1{}, nil
	}
	requestFrame, err := microvm.CodificarSolicitudAuthJSONCodexV1(request)
	if err != nil {
		return err, microvm.CabeceraRespuestaAuthJSONCodexV1{}, nil
	}
	if _, err = client.Write(apertureFrame); err != nil {
		return err, microvm.CabeceraRespuestaAuthJSONCodexV1{}, nil
	}
	if _, err = client.Write(requestFrame); err != nil {
		return err, microvm.CabeceraRespuestaAuthJSONCodexV1{}, nil
	}
	header, err := microvm.DecodificarCabeceraRespuestaAuthJSONCodexV1(client)
	if err != nil {
		return err, microvm.CabeceraRespuestaAuthJSONCodexV1{}, nil
	}
	material := make([]byte, int(header.LongitudMaterial))
	if _, err = io.ReadFull(client, material); err != nil {
		return err, header, nil
	}
	if header.Estado == microvm.EstadoRespuestaAuthJSONCodexDisponible {
		for _, stage := range []microvm.EtapaAcuseAuthJSONCodexV1{microvm.EtapaAcuseAuthJSONCodexProyectada, microvm.EtapaAcuseAuthJSONCodexPurgada} {
			frame, encodeErr := microvm.CodificarAcuseAuthJSONCodexV1(microvm.AcuseAuthJSONCodexV1{
				Protocolo: microvm.ProtocoloAuthJSONCodexOneShotV1,
				SesionRef: request.SesionRef, Cerca: request.Cerca, Etapa: stage, Correcto: true,
			})
			if encodeErr != nil {
				return encodeErr, header, material
			}
			if _, err = client.Write(frame); err != nil {
				return err, header, material
			}
		}
	}
	return <-result, header, material
}

func writeAperture(t *testing.T, writer io.Writer, aperture microvm.AperturaServicioHostV1) {
	t.Helper()
	frame, err := microvm.CodificarAperturaServicioHostV1(aperture)
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(t, writer, frame)
}

func writeRequest(t *testing.T, writer io.Writer, request microvm.SolicitudAuthJSONCodexV1) {
	t.Helper()
	frame, err := microvm.CodificarSolicitudAuthJSONCodexV1(request)
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(t, writer, frame)
}

func writeAck(t *testing.T, writer io.Writer, ack microvm.AcuseAuthJSONCodexV1) {
	t.Helper()
	frame, err := microvm.CodificarAcuseAuthJSONCodexV1(ack)
	if err != nil {
		t.Fatal(err)
	}
	writeFrame(t, writer, frame)
}

func writeFrame(t *testing.T, writer io.Writer, frame []byte) {
	t.Helper()
	for len(frame) > 0 {
		written, err := writer.Write(frame)
		if err != nil || written <= 0 || written > len(frame) {
			if err == nil {
				err = io.ErrShortWrite
			}
			t.Fatal(err)
		}
		frame = frame[written:]
	}
}

func readBrokerResponse(t *testing.T, reader io.Reader) (microvm.CabeceraRespuestaAuthJSONCodexV1, []byte) {
	t.Helper()
	header, err := microvm.DecodificarCabeceraRespuestaAuthJSONCodexV1(reader)
	if err != nil {
		t.Fatal(err)
	}
	material := make([]byte, int(header.LongitudMaterial))
	if _, err := io.ReadFull(reader, material); err != nil {
		t.Fatal(err)
	}
	return header, material
}

func awaitBroker(t *testing.T, result <-chan error) error {
	t.Helper()
	return awaitBrokerWithin(t, result, 2*time.Second)
}

func awaitBrokerWithin(t *testing.T, result <-chan error, timeout time.Duration) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(timeout):
		t.Fatal("broker did not finish")
		return nil
	}
}

type fragmentedConn struct {
	net.Conn
	maximum int
}

func (conn *fragmentedConn) Read(buffer []byte) (int, error) {
	if len(buffer) > conn.maximum {
		buffer = buffer[:conn.maximum]
	}
	return conn.Conn.Read(buffer)
}

func (conn *fragmentedConn) Write(buffer []byte) (int, error) {
	if len(buffer) > conn.maximum {
		buffer = buffer[:conn.maximum]
	}
	return conn.Conn.Write(buffer)
}
