package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

type agenteMicroVMDelegadoPrueba struct {
	mu                  sync.Mutex
	capacidad           agentmicrovm.NegotiatedPhysicalCapacity
	capacidadErr        error
	capabilitiesErr     error
	negociado           bool
	capabilitiesCalls   int
	launchCalls         int
	reconcileCalls      int
	observeCalls        int
	catalogoCalls       int
	capabilitiesEntered chan struct{}
	capabilitiesRelease chan struct{}
	stopCalls           int
	stopRequest         ports.AgentStopRequest
	reconcileRequest    ports.AgentLaunchRequest
	reconcileReceipt    ports.AgentLaunchReceipt
}

type agenteMicroVMPreservadorPrueba struct {
	*agenteMicroVMDelegadoPrueba
	autoridad ports.AgentHistoricalRuntimeAuthority
	solicitud ports.AgentPreserveRequest
	recibo    ports.AgentPreserveReceipt
}

func (adaptador *agenteMicroVMPreservadorPrueba) PreserveWithAuthority(
	_ context.Context,
	autoridad ports.AgentHistoricalRuntimeAuthority,
	solicitud ports.AgentPreserveRequest,
) (ports.AgentPreserveReceipt, error) {
	adaptador.autoridad = autoridad
	adaptador.solicitud = solicitud
	return adaptador.recibo, nil
}

func (adaptador *agenteMicroVMDelegadoPrueba) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	adaptador.mu.Lock()
	adaptador.capabilitiesCalls++
	adaptador.negociado = false
	err := adaptador.capabilitiesErr
	entrada, salida := adaptador.capabilitiesEntered, adaptador.capabilitiesRelease
	adaptador.mu.Unlock()
	if entrada != nil {
		close(entrada)
	}
	if salida != nil {
		select {
		case <-salida:
		case <-ctx.Done():
			return ports.AgentCapabilities{}, ctx.Err()
		}
	}
	if err != nil {
		return ports.AgentCapabilities{}, err
	}
	adaptador.mu.Lock()
	adaptador.negociado = true
	adaptador.mu.Unlock()
	return ports.AgentCapabilities{ProviderRef: "provider:codex", ModelRef: "model:codex", AgentRef: "agent:microvm"}, nil
}

func (adaptador *agenteMicroVMDelegadoPrueba) Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error) {
	adaptador.mu.Lock()
	defer adaptador.mu.Unlock()
	adaptador.launchCalls++
	return ports.AgentLaunchReceipt{}, nil
}

func (adaptador *agenteMicroVMDelegadoPrueba) ReconcileLaunch(
	_ context.Context,
	solicitud ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	adaptador.mu.Lock()
	defer adaptador.mu.Unlock()
	adaptador.reconcileCalls++
	adaptador.reconcileRequest = solicitud
	return adaptador.reconcileReceipt, nil
}

func (adaptador *agenteMicroVMDelegadoPrueba) ObserveAgent(context.Context, ports.AgentObserveRequest) (ports.AgentObservation, error) {
	adaptador.mu.Lock()
	defer adaptador.mu.Unlock()
	adaptador.observeCalls++
	return ports.AgentObservation{}, nil
}

func (adaptador *agenteMicroVMDelegadoPrueba) NegotiatedPhysicalCapacity() (agentmicrovm.NegotiatedPhysicalCapacity, error) {
	adaptador.mu.Lock()
	defer adaptador.mu.Unlock()
	adaptador.catalogoCalls++
	if !adaptador.negociado {
		return agentmicrovm.NegotiatedPhysicalCapacity{}, errors.New("capacidad no negociada")
	}
	return adaptador.capacidad, adaptador.capacidadErr
}

func (adaptador *agenteMicroVMDelegadoPrueba) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{CooperativeStop: true, ForcedStop: true}, nil
}

func (adaptador *agenteMicroVMDelegadoPrueba) Stop(_ context.Context, solicitud ports.AgentStopRequest) (ports.AgentStopReceipt, error) {
	adaptador.mu.Lock()
	defer adaptador.mu.Unlock()
	adaptador.stopCalls++
	adaptador.stopRequest = solicitud
	return ports.AgentStopReceipt{ExternalRef: solicitud.ExternalRef, Mode: solicitud.Mode, IdempotencyKey: solicitud.IdempotencyKey, Status: ports.AgentStopPending}, nil
}

func TestAgentMicroVMCapacidadSoloTrasNegociacionYConservaMaximoExacto(t *testing.T) {
	colocacion, err := ports.NewAgentPlacementRef("placement:/home/cuenta-codex/perfil")
	if err != nil {
		t.Fatal(err)
	}
	delegado := &agenteMicroVMDelegadoPrueba{capacidad: agentmicrovm.NegotiatedPhysicalCapacity{
		PlacementRef: colocacion, Slots: ^uint32(0),
	}}
	agente := nuevoAgentMicroVMPrueba(t, delegado, func() error { return nil }, func() error { return nil })

	if descriptores, err := agente.DescribirCapacidadColocaciones(); err == nil || descriptores != nil {
		t.Fatalf("capacidad previa=%+v error=%v", descriptores, err)
	}
	if _, err := agente.Capabilities(context.Background()); err != nil {
		t.Fatalf("Capabilities() error=%v", err)
	}
	descriptores, err := agente.DescribirCapacidadColocaciones()
	if err != nil || len(descriptores) != 1 {
		t.Fatalf("descriptores=%+v error=%v", descriptores, err)
	}
	descriptor := descriptores[0]
	if descriptor.PlacementRef != colocacion || descriptor.SourceRef != "capacity-source:codex:microvm" ||
		descriptor.BaseMedicion != application.BaseMedicionCapacidadBruta || descriptor.Plazas != int64(^uint32(0)) {
		t.Fatalf("descriptor=%+v", descriptor)
	}
	pool := string(descriptor.PoolRef)
	if !strings.HasPrefix(pool, "capacity-pool:codex:microvm:") ||
		strings.Contains(pool, "home") || strings.Contains(pool, "cuenta") || strings.Contains(pool, "perfil") {
		t.Fatalf("pool filtra colocacion fisica: %q", pool)
	}
	repetido, err := agente.DescribirCapacidadColocaciones()
	if err != nil || repetido[0].PoolRef != descriptor.PoolRef {
		t.Fatalf("pool inestable: primero=%q repetido=%+v error=%v", descriptor.PoolRef, repetido, err)
	}
	otra, _ := ports.NewAgentPlacementRef("placement:/home/cuenta-codex/otro-perfil")
	if referenciaPoolCapacidadAgentMicroVM(otra) == descriptor.PoolRef {
		t.Fatal("colocaciones distintas colisionaron")
	}
}

func TestAgentMicroVMCapacidadFallaCerradoTrasNegociacionFallida(t *testing.T) {
	colocacion, _ := ports.NewAgentPlacementRef("placement:microvm")
	falloCatalogo := errors.New("fallo catalogo")
	falloNegociacion := errors.New("fallo remoto")
	delegado := &agenteMicroVMDelegadoPrueba{capacidad: agentmicrovm.NegotiatedPhysicalCapacity{PlacementRef: colocacion, Slots: 20}}
	agente := nuevoAgentMicroVMPrueba(t, delegado, func() error { return nil }, func() error { return nil })
	if _, err := agente.Capabilities(context.Background()); err != nil {
		t.Fatal(err)
	}
	delegado.mu.Lock()
	delegado.capacidadErr = falloCatalogo
	delegado.mu.Unlock()
	if descriptores, err := agente.DescribirCapacidadColocaciones(); !errors.Is(err, falloCatalogo) || descriptores != nil {
		t.Fatalf("fallo de catalogo ocultado: %+v error=%v", descriptores, err)
	}
	delegado.mu.Lock()
	delegado.capacidadErr = nil
	delegado.capabilitiesErr = falloNegociacion
	delegado.mu.Unlock()
	if _, err := agente.Capabilities(context.Background()); !errors.Is(err, falloNegociacion) {
		t.Fatalf("Capabilities() error=%v", err)
	}
	if descriptores, err := agente.DescribirCapacidadColocaciones(); err == nil || descriptores != nil {
		t.Fatalf("capacidad obsoleta publicada: %+v error=%v", descriptores, err)
	}
}

func TestAgentMicroVMRechazaDependenciasNulas(t *testing.T) {
	cierre := func() error { return nil }
	if _, err := newAgentMicroVM(nil, cierre, cierre); !errors.Is(err, errAgentMicroVMAdapterRequerido) {
		t.Fatalf("adapter nil error=%v", err)
	}
	var delegado *agenteMicroVMDelegadoPrueba
	if _, err := newAgentMicroVMConDelegado(delegado, cierre, cierre); !errors.Is(err, errAgentMicroVMAdapterRequerido) {
		t.Fatalf("adapter tipado nil error=%v", err)
	}
	valido := &agenteMicroVMDelegadoPrueba{}
	if _, err := newAgentMicroVMConDelegado(valido, nil, cierre); !errors.Is(err, errAgentMicroVMCierreConexiones) {
		t.Fatalf("cierre conexiones nil error=%v", err)
	}
	if _, err := newAgentMicroVMConDelegado(valido, cierre, nil); !errors.Is(err, errAgentMicroVMCierreCredenciales) {
		t.Fatalf("cierre credenciales nil error=%v", err)
	}
}

func TestAgentMicroVMConservaReconciliacionOptInSinRelanzar(t *testing.T) {
	solicitud := ports.AgentLaunchRequest{
		SpecHash:       "sha256:solicitud-exacta",
		IdempotencyKey: "idempotency:reconcile-launch",
		SkillRefs:      []string{"skill:uno", "skill:dos"},
		WriteSet:       []string{"internal/bootstrap/agent_microvm.go"},
		EffectAuthority: ports.AgentLaunchEffectAuthority{
			EffectAttemptRef: "effect-attempt:historical",
			ActionFence:      17,
		},
	}
	reciboEsperado := ports.AgentLaunchReceipt{
		SpecHash:       solicitud.SpecHash,
		ExternalRef:    "microvm:reconciliada",
		IdempotencyKey: solicitud.IdempotencyKey,
		ReceiptRef:     "receipt:reconcile-launch",
	}
	delegado := &agenteMicroVMDelegadoPrueba{reconcileReceipt: reciboEsperado}
	agente := nuevoAgentMicroVMPrueba(t, delegado, func() error { return nil }, func() error { return nil })

	reconciliador, err := application.AgentLaunchReconcilerFrom(agente)
	if err != nil {
		t.Fatalf("AgentLaunchReconcilerFrom() error=%v", err)
	}
	recibo, err := reconciliador.ReconcileLaunch(context.Background(), solicitud)
	if err != nil {
		t.Fatalf("ReconcileLaunch() error=%v", err)
	}
	if !reflect.DeepEqual(recibo, reciboEsperado) {
		t.Fatalf("recibo=%+v esperado=%+v", recibo, reciboEsperado)
	}

	delegado.mu.Lock()
	defer delegado.mu.Unlock()
	if delegado.reconcileCalls != 1 || delegado.launchCalls != 0 {
		t.Fatalf("reconcile=%d launch=%d", delegado.reconcileCalls, delegado.launchCalls)
	}
	if !reflect.DeepEqual(delegado.reconcileRequest, solicitud) {
		t.Fatalf("solicitud=%+v esperada=%+v", delegado.reconcileRequest, solicitud)
	}
}

func TestAgentMicroVMShutdownCanceladoUneErroresUnaVezYNoDetieneVM(t *testing.T) {
	delegado := &agenteMicroVMDelegadoPrueba{}
	falloConexiones := errors.New("fallo conexiones")
	falloCredenciales := errors.New("fallo credenciales")
	var cierresConexiones, cierresCredenciales atomic.Int64
	agente := nuevoAgentMicroVMPrueba(t, delegado, func() error {
		cierresConexiones.Add(1)
		return falloConexiones
	}, func() error {
		cierresCredenciales.Add(1)
		return falloCredenciales
	})
	solicitud := ports.AgentStopRequest{ExternalRef: "ejecucion:microvm-exacta", Mode: ports.AgentStopCooperative, IdempotencyKey: "stop:microvm-exacta"}
	if _, err := agente.Stop(context.Background(), solicitud); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	const llamadas = 32
	errores := make(chan error, llamadas)
	var grupo sync.WaitGroup
	for indice := 0; indice < llamadas; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			errores <- agente.Shutdown(ctx)
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if !errors.Is(err, falloConexiones) || !errors.Is(err, falloCredenciales) {
			t.Fatalf("Shutdown() error=%v", err)
		}
	}
	if cierresConexiones.Load() != 1 || cierresCredenciales.Load() != 1 || delegado.stopCalls != 1 || delegado.stopRequest != solicitud {
		t.Fatalf("cierres conexiones=%d credenciales=%d stop=%d request=%+v", cierresConexiones.Load(), cierresCredenciales.Load(), delegado.stopCalls, delegado.stopRequest)
	}
}

func TestAgentMicroVMShutdownDrenaEnVueloYCierraTodasLasEntradas(t *testing.T) {
	entrada := make(chan struct{})
	salida := make(chan struct{})
	delegado := &agenteMicroVMDelegadoPrueba{capabilitiesEntered: entrada, capabilitiesRelease: salida}
	var cierres atomic.Int64
	agente := nuevoAgentMicroVMPrueba(t, delegado, func() error { cierres.Add(1); return nil }, func() error { cierres.Add(1); return nil })
	if _, err := agente.Launch(context.Background(), ports.AgentLaunchRequest{}); err != nil {
		t.Fatalf("Launch() delegado error=%v", err)
	}
	if _, err := agente.ObserveAgent(context.Background(), ports.AgentObserveRequest{}); err != nil {
		t.Fatalf("ObserveAgent() delegado error=%v", err)
	}

	capacidadTerminada := make(chan error, 1)
	go func() {
		_, err := agente.Capabilities(context.Background())
		capacidadTerminada <- err
	}()
	<-entrada
	shutdownTerminado := make(chan error, 1)
	go func() { shutdownTerminado <- agente.Shutdown(context.Background()) }()
	if cierres.Load() != 0 {
		t.Fatal("shutdown no dreno llamada en vuelo")
	}
	close(salida)
	if err := <-capacidadTerminada; err != nil {
		t.Fatalf("llamada en vuelo error=%v", err)
	}
	if err := <-shutdownTerminado; err != nil || cierres.Load() != 2 {
		t.Fatalf("Shutdown() error=%v cierres=%d", err, cierres.Load())
	}

	if _, err := agente.Capabilities(context.Background()); !errors.Is(err, errAgentMicroVMCerrado) {
		t.Fatalf("Capabilities posterior error=%v", err)
	}
	if _, err := agente.Launch(context.Background(), ports.AgentLaunchRequest{}); !errors.Is(err, errAgentMicroVMCerrado) {
		t.Fatalf("Launch posterior error=%v", err)
	}
	if _, err := agente.ObserveAgent(context.Background(), ports.AgentObserveRequest{}); !errors.Is(err, errAgentMicroVMCerrado) {
		t.Fatalf("ObserveAgent posterior error=%v", err)
	}
	if _, err := agente.DescribirCapacidadColocaciones(); !errors.Is(err, errAgentMicroVMCerrado) {
		t.Fatalf("catalogo posterior error=%v", err)
	}
	delegado.mu.Lock()
	defer delegado.mu.Unlock()
	if delegado.capabilitiesCalls != 1 || delegado.launchCalls != 1 || delegado.observeCalls != 1 || delegado.catalogoCalls != 0 {
		t.Fatalf("llamadas cruzaron tras shutdown: capabilities=%d launch=%d observe=%d catalogo=%d",
			delegado.capabilitiesCalls, delegado.launchCalls, delegado.observeCalls, delegado.catalogoCalls)
	}
}

func TestAgentMicroVMPreserveConservaAutoridadHistoricaYFallaCerrado(t *testing.T) {
	autoridad := ports.AgentHistoricalRuntimeAuthority{
		Key:     ports.AgentHistoricalRuntimeAuthorityKey{ActionFence: 41},
		Digests: ports.AgentHistoricalRuntimeDigests{PlanSHA256: strings.Repeat("a", 64)},
	}
	solicitud := ports.AgentPreserveRequest{IdempotencyKey: "idempotency:preserve:historical"}
	recibo := ports.AgentPreserveReceipt{IdempotencyKey: solicitud.IdempotencyKey, ReceiptRef: "receipt:preserve"}
	delegado := &agenteMicroVMPreservadorPrueba{
		agenteMicroVMDelegadoPrueba: &agenteMicroVMDelegadoPrueba{},
		recibo:                      recibo,
	}
	agente := nuevoAgentMicroVMPrueba(t, delegado, func() error { return nil }, func() error { return nil })

	obtenido, err := agente.PreserveWithAuthority(context.Background(), autoridad, solicitud)
	if err != nil || !reflect.DeepEqual(obtenido, recibo) || delegado.autoridad != autoridad || delegado.solicitud != solicitud {
		t.Fatalf("PreserveWithAuthority() recibo=%+v error=%v autoridad=%+v solicitud=%+v", obtenido, err, delegado.autoridad, delegado.solicitud)
	}

	sinAutoridad := nuevoAgentMicroVMPrueba(t, &agenteMicroVMDelegadoPrueba{}, func() error { return nil }, func() error { return nil })
	if obtenido, err := sinAutoridad.PreserveWithAuthority(context.Background(), autoridad, solicitud); !errors.Is(err, errAgentMicroVMAutoridadHistoricaRequerida) || !reflect.DeepEqual(obtenido, ports.AgentPreserveReceipt{}) {
		t.Fatalf("delegado sin autoridad recibo=%+v error=%v", obtenido, err)
	}
	if err := agente.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
	if obtenido, err := agente.PreserveWithAuthority(context.Background(), autoridad, solicitud); !errors.Is(err, errAgentMicroVMCerrado) || !reflect.DeepEqual(obtenido, ports.AgentPreserveReceipt{}) {
		t.Fatalf("preserve posterior a shutdown recibo=%+v error=%v", obtenido, err)
	}
}

func nuevoAgentMicroVMPrueba(
	t *testing.T,
	adaptador agenteMicroVMDelegado,
	cerrarConexiones func() error,
	cerrarCredenciales func() error,
) *agenteMicroVM {
	t.Helper()
	agente, err := newAgentMicroVMConDelegado(adaptador, cerrarConexiones, cerrarCredenciales)
	if err != nil {
		t.Fatal(err)
	}
	return agente
}
