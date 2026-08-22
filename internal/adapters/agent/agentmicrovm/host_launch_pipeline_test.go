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

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

type pipelineAuthorityRegistryStub struct {
	mu sync.Mutex

	authorities    map[ports.MicroVMHostLaunchAuthorityKey]ports.MicroVMHostLaunchAuthorityV1
	runtimeDigests map[ports.MicroVMHostLaunchAuthorityKey]ports.MicroVMHostLaunchRuntimeDigestsV1
	prepareErr     error
	bindErr        error
	mutateBind     func(*ports.MicroVMHostLaunchAuthorityV1)
	record         func(string)
	afterPrepare   func()

	prepareCalls        int
	bindCalls           int
	resolveCalls        int
	boundPrepareReplays int
	bindContextErrors   []error
}

func newPipelineAuthorityRegistryStub() *pipelineAuthorityRegistryStub {
	return &pipelineAuthorityRegistryStub{
		authorities:    make(map[ports.MicroVMHostLaunchAuthorityKey]ports.MicroVMHostLaunchAuthorityV1),
		runtimeDigests: make(map[ports.MicroVMHostLaunchAuthorityKey]ports.MicroVMHostLaunchRuntimeDigestsV1),
	}
}

func (registry *pipelineAuthorityRegistryStub) PrepareWithRuntime(
	ctx context.Context,
	authority ports.MicroVMHostLaunchAuthorityV1,
	runtime ports.MicroVMHostLaunchRuntimeDigestsV1,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.prepareCalls++
	if registry.record != nil {
		registry.record("prepare")
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if registry.prepareErr != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, registry.prepareErr
	}
	if existing, found := registry.runtimeDigests[runtime.Key]; found && existing != runtime {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("pipeline-registry: runtime conflict")
	}
	if persisted, found := registry.authorities[authority.Key]; found {
		prepared := ports.CloneMicroVMHostLaunchAuthorityV1(persisted)
		prepared.ExternalRef = ""
		if !reflect.DeepEqual(prepared, authority) {
			return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("pipeline-registry: prepare conflict")
		}
		if persisted.ExternalRef != "" {
			registry.boundPrepareReplays++
		}
		registry.runtimeDigests[runtime.Key] = runtime
		if registry.afterPrepare != nil {
			registry.afterPrepare()
		}
		return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
	}
	registry.authorities[authority.Key] = ports.CloneMicroVMHostLaunchAuthorityV1(authority)
	registry.runtimeDigests[runtime.Key] = runtime
	if registry.afterPrepare != nil {
		registry.afterPrepare()
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(authority), nil
}

func (registry *pipelineAuthorityRegistryStub) ResolveRuntime(
	ctx context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchRuntimeDigestsV1, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, err
	}
	value, found := registry.runtimeDigests[key]
	if !found {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, errors.New("pipeline-registry: runtime absent")
	}
	return value, nil
}

func (registry *pipelineAuthorityRegistryStub) Prepare(
	ctx context.Context,
	authority ports.MicroVMHostLaunchAuthorityV1,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.prepareCalls++
	if registry.record != nil {
		registry.record("prepare")
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	if registry.prepareErr != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, registry.prepareErr
	}
	if persisted, found := registry.authorities[authority.Key]; found {
		prepared := ports.CloneMicroVMHostLaunchAuthorityV1(persisted)
		prepared.ExternalRef = ""
		if !reflect.DeepEqual(prepared, authority) {
			return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("pipeline-registry: prepare conflict")
		}
		if persisted.ExternalRef != "" {
			registry.boundPrepareReplays++
		}
		if registry.afterPrepare != nil {
			registry.afterPrepare()
		}
		return ports.CloneMicroVMHostLaunchAuthorityV1(persisted), nil
	}
	registry.authorities[authority.Key] = ports.CloneMicroVMHostLaunchAuthorityV1(authority)
	if registry.afterPrepare != nil {
		registry.afterPrepare()
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(authority), nil
}

func (registry *pipelineAuthorityRegistryStub) BindExternal(
	ctx context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
	externalRef string,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.bindCalls++
	registry.bindContextErrors = append(registry.bindContextErrors, ctx.Err())
	if registry.record != nil {
		registry.record("bind")
	}
	if registry.bindErr != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, registry.bindErr
	}
	persisted, found := registry.authorities[key]
	if !found {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("pipeline-registry: authority absent")
	}
	if persisted.ExternalRef != "" && persisted.ExternalRef != externalRef {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("pipeline-registry: external conflict")
	}
	persisted.ExternalRef = externalRef
	registry.authorities[key] = ports.CloneMicroVMHostLaunchAuthorityV1(persisted)
	result := ports.CloneMicroVMHostLaunchAuthorityV1(persisted)
	if registry.mutateBind != nil {
		registry.mutateBind(&result)
	}
	return result, nil
}

func (registry *pipelineAuthorityRegistryStub) Resolve(
	ctx context.Context,
	key ports.MicroVMHostLaunchAuthorityKey,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.resolveCalls++
	if registry.record != nil {
		registry.record("resolve")
	}
	if err := ctx.Err(); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, err
	}
	authority, found := registry.authorities[key]
	if !found {
		return ports.MicroVMHostLaunchAuthorityV1{}, errors.New("pipeline-registry: authority absent")
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(authority), nil
}

func (registry *pipelineAuthorityRegistryStub) snapshot() (prepareCalls, bindCalls, resolveCalls, boundReplays int, bindContextErrors []error) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.prepareCalls, registry.bindCalls, registry.resolveCalls, registry.boundPrepareReplays,
		append([]error(nil), registry.bindContextErrors...)
}

func validPipelineClaimResolver(request ports.AgentLaunchRequest) *CredentialClaimResolver {
	credentialRef := credentials.CredentialRef("credential:provider_codex_primary")
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{
		credentialRef: 3,
	}}
	return pipelineClaimResolver(request, reader)
}

func pipelineClaimResolver(
	request ports.AgentLaunchRequest,
	reader *credentialClaimReaderStub,
) *CredentialClaimResolver {
	resolver, err := NewCredentialClaimResolver(reader, "provider:codex", []CredentialClaimBinding{{
		PlacementRef: request.ReferenciaColocacion, CredentialRef: "credential:provider_codex_primary",
	}})
	if err != nil {
		panic(err)
	}
	return resolver
}

type pipelineOrderRecorder struct {
	mu    sync.Mutex
	steps []string
}

func (recorder *pipelineOrderRecorder) add(step string) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.steps = append(recorder.steps, step)
}

func (recorder *pipelineOrderRecorder) snapshot() []string {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]string(nil), recorder.steps...)
}

type pipelineOrderedClient struct {
	*launchClientStub
	recorder *pipelineOrderRecorder
}

type pipelineAmbiguousFirstLaunchClient struct {
	*pipelineOrderedClient
	mu       sync.Mutex
	marker   error
	launches int
}

func (client *pipelineAmbiguousFirstLaunchClient) Lanzar(
	ctx context.Context,
	key string,
	request microvm.SolicitudLanzamiento,
) (microvm.RespuestaEjecucion, error) {
	response, err := client.pipelineOrderedClient.Lanzar(ctx, key, request)
	client.mu.Lock()
	defer client.mu.Unlock()
	client.launches++
	if client.launches == 1 {
		return response, client.marker
	}
	return response, err
}

func (client *pipelineOrderedClient) Lanzar(
	ctx context.Context,
	key string,
	request microvm.SolicitudLanzamiento,
) (microvm.RespuestaEjecucion, error) {
	client.recorder.add("launch")
	return client.launchClientStub.Lanzar(ctx, key, request)
}

func (client *pipelineOrderedClient) ReconciliarEntradaSesion(
	ctx context.Context,
	key string,
	externalRef string,
	sessionRef string,
	fence uint64,
) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error) {
	client.recorder.add("session")
	return client.launchClientStub.ReconciliarEntradaSesion(ctx, key, externalRef, sessionRef, fence)
}

func TestAdapterLaunchAuthorityResolveBuildAndPrepareFailuresStopBeforeLaunch(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	response := validPhysicalResponse(t, request, descriptor)

	t.Run("resolve", func(t *testing.T) {
		marker := errors.New("private-resolver-marker")
		reader := &credentialClaimReaderStub{
			versions: map[credentials.CredentialRef]credentials.Version{"credential:provider_codex_primary": 3},
			err:      marker,
		}
		client := &launchClientStub{capabilities: validRemoteCapabilities(), response: response}
		registry := newPipelineAuthorityRegistryStub()
		config := validAdapterConfig(client, validSigner(), request, descriptor)
		config.ClaimResolver = pipelineClaimResolver(request, reader)
		config.LaunchAuthorityRegistry = registry
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := adapter.Launch(context.Background(), request)
		assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityResolveFailed, marker, true)
		prepareCalls, bindCalls, _, _, _ := registry.snapshot()
		if len(client.launchRequests) != 0 || prepareCalls != 0 || bindCalls != 0 {
			t.Fatalf("launches=%d prepare=%d bind=%d", len(client.launchRequests), prepareCalls, bindCalls)
		}
	})

	t.Run("build", func(t *testing.T) {
		client := &launchClientStub{capabilities: validRemoteCapabilities(), response: response}
		registry := newPipelineAuthorityRegistryStub()
		signer := validSigner()
		signer.mutate = func(signed microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
			signed.Concesion = json.RawMessage(`{"private_marker":"build"}`)
			return signed
		}
		config := validAdapterConfig(client, signer, request, descriptor)
		config.LaunchAuthorityRegistry = registry
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := adapter.Launch(context.Background(), request)
		assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityBuildFailed, nil, true)
		prepareCalls, bindCalls, _, _, _ := registry.snapshot()
		if len(client.launchRequests) != 0 || prepareCalls != 0 || bindCalls != 0 {
			t.Fatalf("launches=%d prepare=%d bind=%d", len(client.launchRequests), prepareCalls, bindCalls)
		}
	})

	t.Run("prepare", func(t *testing.T) {
		marker := errors.New("private-prepare-marker")
		client := &launchClientStub{capabilities: validRemoteCapabilities(), response: response}
		registry := newPipelineAuthorityRegistryStub()
		registry.prepareErr = marker
		config := validAdapterConfig(client, validSigner(), request, descriptor)
		config.LaunchAuthorityRegistry = registry
		adapter, err := New(config)
		if err != nil {
			t.Fatal(err)
		}
		receipt, err := adapter.Launch(context.Background(), request)
		assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityPrepareFailed, marker, false)
		prepareCalls, bindCalls, _, _, _ := registry.snapshot()
		if len(client.launchRequests) != 0 || prepareCalls != 1 || bindCalls != 0 {
			t.Fatalf("launches=%d prepare=%d bind=%d", len(client.launchRequests), prepareCalls, bindCalls)
		}
	})
}

func TestAdapterLaunchAuthorityOrdersPrepareLaunchBindBeforeSession(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	recorder := &pipelineOrderRecorder{}
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	client := &pipelineOrderedClient{launchClientStub: base, recorder: recorder}
	registry := newPipelineAuthorityRegistryStub()
	registry.record = recorder.add
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if steps := recorder.snapshot(); !reflect.DeepEqual(steps, []string{"prepare", "launch", "bind", "session"}) {
		t.Fatalf("steps=%v", steps)
	}
	if receipt.ExternalRef != base.response.Referencia {
		t.Fatalf("receipt=%+v", receipt)
	}
}

func TestAdapterLaunchAuthorityBindsWithWithoutCancelAfterSuccessfulLaunch(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	recorder := &pipelineOrderRecorder{}
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
		onLaunch: cancel,
	}
	client := &pipelineOrderedClient{launchClientStub: base, recorder: recorder}
	registry := newPipelineAuthorityRegistryStub()
	registry.record = recorder.add
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(ctx, request)
	if receipt != (ports.AgentLaunchReceipt{}) || !errors.Is(err, context.Canceled) || ErrorCode(err) != "" {
		t.Fatalf("receipt=%+v err=%v code=%q", receipt, err, ErrorCode(err))
	}
	_, bindCalls, _, _, bindContextErrors := registry.snapshot()
	if bindCalls != 1 || len(bindContextErrors) != 1 || bindContextErrors[0] != nil ||
		!reflect.DeepEqual(recorder.snapshot(), []string{"prepare", "launch", "bind"}) {
		t.Fatalf("bind_calls=%d bind_context=%v steps=%v", bindCalls, bindContextErrors, recorder.snapshot())
	}
}

func TestAdapterLaunchCancellationAfterPrepareIsStructuredAndDefinitelyNotApplied(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	registry := newPipelineAuthorityRegistryStub()
	registry.afterPrepare = cancel
	config := validAdapterConfig(base, validSigner(), request, descriptor)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(ctx, request)
	assertPipelineAuthorityError(t, receipt, err, CodeLaunchCanceledBeforeSubmit, context.Canceled, true)
	prepareCalls, bindCalls, resolveCalls, _, _ := registry.snapshot()
	if prepareCalls != 1 || bindCalls != 0 || resolveCalls != 0 || len(base.launchRequests) != 0 {
		t.Fatalf("prepare=%d bind=%d resolve=%d launches=%d", prepareCalls, bindCalls, resolveCalls, len(base.launchRequests))
	}
}

func TestAdapterLaunchAuthorityBindFailureStopsSessionAndReceipt(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	marker := errors.New("private-bind-marker")
	recorder := &pipelineOrderRecorder{}
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	client := &pipelineOrderedClient{launchClientStub: base, recorder: recorder}
	registry := newPipelineAuthorityRegistryStub()
	registry.bindErr = marker
	registry.record = recorder.add
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(context.Background(), request)
	assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityBindFailed, marker, false)
	if !reflect.DeepEqual(recorder.snapshot(), []string{"prepare", "launch", "bind"}) {
		t.Fatalf("steps=%v", recorder.snapshot())
	}
}

func TestAdapterReconcileLaunchAcceptsExactBoundReplayAndRejectsCrossedResponse(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	recorder := &pipelineOrderRecorder{}
	client := &pipelineOrderedClient{launchClientStub: base, recorder: recorder}
	registry := newPipelineAuthorityRegistryStub()
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	first, err := adapter.Launch(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.ReconcileLaunch(context.Background(), request)
	if err != nil || second != first {
		t.Fatalf("exact replay first=%+v second=%+v err=%v", first, second, err)
	}
	prepareCalls, bindCalls, resolveCalls, boundReplays, _ := registry.snapshot()
	if prepareCalls != 2 || bindCalls != 2 || resolveCalls != 1 || boundReplays != 1 {
		t.Fatalf("prepare=%d bind=%d resolve=%d bound_replays=%d", prepareCalls, bindCalls, resolveCalls, boundReplays)
	}

	base.response.Referencia = "ejecucion:" + strings.Repeat("b", 64)
	receipt, err := adapter.ReconcileLaunch(context.Background(), request)
	assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityBindFailed, nil, false)
	if steps := recorder.snapshot(); !reflect.DeepEqual(steps, []string{
		"launch", "session", "launch", "session", "launch",
	}) {
		t.Fatalf("session count changed after crossed response: steps=%v", steps)
	}
}

func TestAdapterReconcilePreparedLaunchUsesHistoricalCredentialVersion(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	marker := errors.New("private-ambiguous-launch-marker")
	recorder := &pipelineOrderRecorder{}
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	client := &pipelineAmbiguousFirstLaunchClient{
		pipelineOrderedClient: &pipelineOrderedClient{launchClientStub: base, recorder: recorder},
		marker:                marker,
	}
	reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{
		"credential:provider_codex_primary": 3,
	}}
	registry := newPipelineAuthorityRegistryStub()
	registry.record = recorder.add
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.ClaimResolver = pipelineClaimResolver(request, reader)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}

	first, err := adapter.Launch(context.Background(), request)
	assertPipelineAuthorityError(t, first, err, CodeLaunchUnavailable, marker, false)
	reader.mu.Lock()
	reader.versions["credential:provider_codex_primary"] = 4
	reader.mu.Unlock()

	receipt, err := adapter.ReconcileLaunch(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ExternalRef != base.response.Referencia {
		t.Fatalf("receipt=%+v", receipt)
	}
	reader.mu.Lock()
	describeCalls := len(reader.requests)
	reader.mu.Unlock()
	prepareCalls, bindCalls, resolveCalls, _, _ := registry.snapshot()
	if describeCalls != 1 || prepareCalls != 2 || bindCalls != 1 || resolveCalls != 1 ||
		len(base.launchRequests) != 2 || !reflect.DeepEqual(base.launchRequests[0], base.launchRequests[1]) {
		t.Fatalf("describe=%d prepare=%d bind=%d resolve=%d launches=%d equal=%t",
			describeCalls, prepareCalls, bindCalls, resolveCalls, len(base.launchRequests),
			len(base.launchRequests) == 2 && reflect.DeepEqual(base.launchRequests[0], base.launchRequests[1]))
	}
	if steps := recorder.snapshot(); !reflect.DeepEqual(steps, []string{
		"prepare", "launch", "resolve", "prepare", "launch", "bind", "session",
	}) {
		t.Fatalf("steps=%v", steps)
	}
}

func TestAdapterReconcileRejectsUnaccreditedPlacementAndDifferentSignedAuthority(t *testing.T) {
	tests := map[string]func(*Adapter){
		"credential binding": func(adapter *Adapter) {
			adapter.claimResolver.bindings[adapter.profile.PlacementRef] = "credential:another-account"
		},
		"purpose": func(adapter *Adapter) {
			adapter.claimResolver.purpose = "provider:another"
		},
		"signed authority": func(adapter *Adapter) {
			seed := make([]byte, ed25519.SeedSize)
			seed[0] = 1
			signer, err := microvm.NuevoFirmanteConcesiones("clave-publica:alternate", ed25519.NewKeyFromSeed(seed))
			if err != nil {
				panic(err)
			}
			adapter.signer = &launchSignerStub{delegate: signer}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := validLaunchRequest(t)
			descriptor := validDescriptor(t, false)
			base := &launchClientStub{
				capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
			}
			client := &pipelineAmbiguousFirstLaunchClient{
				pipelineOrderedClient: &pipelineOrderedClient{launchClientStub: base, recorder: &pipelineOrderRecorder{}},
				marker:                errors.New("private-ambiguous-launch-marker"),
			}
			reader := &credentialClaimReaderStub{versions: map[credentials.CredentialRef]credentials.Version{
				"credential:provider_codex_primary": 3,
			}}
			registry := newPipelineAuthorityRegistryStub()
			config := validAdapterConfig(client, validSigner(), request, descriptor)
			config.ClaimResolver = pipelineClaimResolver(request, reader)
			config.LaunchAuthorityRegistry = registry
			adapter, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := adapter.Launch(context.Background(), request); ErrorCode(err) != CodeLaunchUnavailable {
				t.Fatalf("initial error=%v", err)
			}
			mutate(adapter)

			receipt, err := adapter.ReconcileLaunch(context.Background(), request)
			assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityReplayInvalid, nil, false)
			reader.mu.Lock()
			describeCalls := len(reader.requests)
			reader.mu.Unlock()
			prepareCalls, bindCalls, resolveCalls, _, _ := registry.snapshot()
			if describeCalls != 1 || prepareCalls != 1 || bindCalls != 0 || resolveCalls != 1 || len(base.launchRequests) != 1 {
				t.Fatalf("describe=%d prepare=%d bind=%d resolve=%d launches=%d",
					describeCalls, prepareCalls, bindCalls, resolveCalls, len(base.launchRequests))
			}
		})
	}
}

func TestAdapterLaunchAuthorityRejectsBoundDifferentFromResponse(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	recorder := &pipelineOrderRecorder{}
	base := &launchClientStub{
		capabilities: validRemoteCapabilities(), response: validPhysicalResponse(t, request, descriptor),
	}
	client := &pipelineOrderedClient{launchClientStub: base, recorder: recorder}
	registry := newPipelineAuthorityRegistryStub()
	registry.mutateBind = func(authority *ports.MicroVMHostLaunchAuthorityV1) {
		authority.ExternalRef = "ejecucion:" + strings.Repeat("c", 64)
	}
	config := validAdapterConfig(client, validSigner(), request, descriptor)
	config.LaunchAuthorityRegistry = registry
	adapter, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := adapter.Launch(context.Background(), request)
	assertPipelineAuthorityError(t, receipt, err, CodeLaunchAuthorityBindFailed, nil, false)
	if steps := recorder.snapshot(); !reflect.DeepEqual(steps, []string{"launch"}) {
		t.Fatalf("session ran after crossed bound: steps=%v", recorder.snapshot())
	}
}

func TestAdapterRejectsNilAndTypedNilLaunchAuthorityDependencies(t *testing.T) {
	request := validLaunchRequest(t)
	descriptor := validDescriptor(t, false)
	base := validAdapterConfig(&launchClientStub{}, validSigner(), request, descriptor)
	var typedNilRegistry *pipelineAuthorityRegistryStub
	tests := map[string]func(*Config){
		"typed nil resolver": func(config *Config) { config.ClaimResolver = nil },
		"nil registry":       func(config *Config) { config.LaunchAuthorityRegistry = nil },
		"typed nil registry": func(config *Config) { config.LaunchAuthorityRegistry = typedNilRegistry },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			config := base
			mutate(&config)
			if adapter, err := New(config); adapter != nil || ErrorCode(err) != CodeConfigurationInvalid {
				t.Fatalf("adapter=%v err=%v", adapter, err)
			}
		})
	}
}

func assertPipelineAuthorityError(
	t *testing.T,
	receipt ports.AgentLaunchReceipt,
	err error,
	code string,
	cause error,
	definitelyNotApplied bool,
) {
	t.Helper()
	if receipt != (ports.AgentLaunchReceipt{}) || ErrorCode(err) != code || err.Error() != code ||
		isDefinitelyNotApplied(err) != definitelyNotApplied ||
		(cause != nil && (!errors.Is(err, cause) || strings.Contains(err.Error(), cause.Error()))) {
		t.Fatalf("receipt=%+v err=%v code=%q definite=%v", receipt, err, ErrorCode(err), isDefinitelyNotApplied(err))
	}
}
