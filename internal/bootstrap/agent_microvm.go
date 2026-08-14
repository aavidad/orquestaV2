package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"reflect"
	"sync"

	"orquesta/internal/adapters/agent/agentmicrovm"
	"orquesta/internal/application"
	"orquesta/internal/ports"
)

const (
	referenciaFuenteCapacidadAgentMicroVM application.AgentCapacitySourceRef = "capacity-source:codex:microvm"
	dominioPoolCapacidadAgentMicroVM                                         = "orquesta.bootstrap.agent-microvm.capacity-pool.v1\x00"
)

var (
	errAgentMicroVMAdapterRequerido        = errors.New("bootstrap.agent_microvm_adapter_required")
	errAgentMicroVMCierreConexiones        = errors.New("bootstrap.agent_microvm_connection_cleanup_required")
	errAgentMicroVMCierreCredenciales      = errors.New("bootstrap.agent_microvm_credential_cleanup_required")
	errAgentMicroVMCerrado                 = errors.New("bootstrap.agent_microvm_closed")
	errAgentMicroVMCapacidadFisicaInvalida = errors.New("bootstrap.agent_microvm_physical_capacity_invalid")
)

// agenteMicroVMDelegado es la frontera cohesionada del adaptador físico. La
// factoría elige explícitamente la implementación; el wrapper solo coordina
// llamadas neutrales y recursos locales de composición.
type agenteMicroVMDelegado interface {
	application.AgentLauncher
	application.AgentLaunchReconciler
	application.AgentObserver
	NegotiatedPhysicalCapacity() (agentmicrovm.NegotiatedPhysicalCapacity, error)
}

// agenteMicroVM solo posee conexiones de composicion y credenciales. Nunca
// ordena detener ni cerrar una microVM; esos efectos pertenecen a B11/B12.
type agenteMicroVM struct {
	adaptador          agenteMicroVMDelegado
	cerrarConexiones   func() error
	cerrarCredenciales func() error
	compuerta          sync.RWMutex
	cerrado            bool
	shutdownOnce       sync.Once
	shutdownErr        error
}

func newAgentMicroVM(
	adaptador agenteMicroVMDelegado,
	cerrarConexiones func() error,
	cerrarCredenciales func() error,
) (*agenteMicroVM, error) {
	if interfazNulaAgentMicroVM(adaptador) {
		return nil, errAgentMicroVMAdapterRequerido
	}
	if cerrarConexiones == nil {
		return nil, errAgentMicroVMCierreConexiones
	}
	if cerrarCredenciales == nil {
		return nil, errAgentMicroVMCierreCredenciales
	}
	return &agenteMicroVM{
		adaptador:          adaptador,
		cerrarConexiones:   cerrarConexiones,
		cerrarCredenciales: cerrarCredenciales,
	}, nil
}

func (agente *agenteMicroVM) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	if agente == nil {
		return ports.AgentCapabilities{}, errAgentMicroVMAdapterRequerido
	}
	agente.compuerta.RLock()
	defer agente.compuerta.RUnlock()
	if agente.cerrado {
		return ports.AgentCapabilities{}, errAgentMicroVMCerrado
	}
	return agente.adaptador.Capabilities(ctx)
}

func (agente *agenteMicroVM) Launch(
	ctx context.Context,
	solicitud ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	if agente == nil {
		return ports.AgentLaunchReceipt{}, errAgentMicroVMAdapterRequerido
	}
	agente.compuerta.RLock()
	defer agente.compuerta.RUnlock()
	if agente.cerrado {
		return ports.AgentLaunchReceipt{}, errAgentMicroVMCerrado
	}
	return agente.adaptador.Launch(ctx, solicitud)
}

func (agente *agenteMicroVM) ReconcileLaunch(
	ctx context.Context,
	solicitud ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	if agente == nil {
		return ports.AgentLaunchReceipt{}, errAgentMicroVMAdapterRequerido
	}
	agente.compuerta.RLock()
	defer agente.compuerta.RUnlock()
	if agente.cerrado {
		return ports.AgentLaunchReceipt{}, errAgentMicroVMCerrado
	}
	return agente.adaptador.ReconcileLaunch(ctx, solicitud)
}

func (agente *agenteMicroVM) ObserveAgent(
	ctx context.Context,
	solicitud ports.AgentObserveRequest,
) (ports.AgentObservation, error) {
	if agente == nil {
		return ports.AgentObservation{}, errAgentMicroVMAdapterRequerido
	}
	agente.compuerta.RLock()
	defer agente.compuerta.RUnlock()
	if agente.cerrado {
		return ports.AgentObservation{}, errAgentMicroVMCerrado
	}
	return agente.adaptador.ObserveAgent(ctx, solicitud)
}

func (agente *agenteMicroVM) DescribirCapacidadColocaciones() ([]application.DescriptorCapacidadColocacionAgente, error) {
	if agente == nil {
		return nil, errAgentMicroVMAdapterRequerido
	}
	agente.compuerta.RLock()
	defer agente.compuerta.RUnlock()
	if agente.cerrado {
		return nil, errAgentMicroVMCerrado
	}
	capacidad, err := agente.adaptador.NegotiatedPhysicalCapacity()
	if err != nil {
		return nil, err
	}
	if capacidad.PlacementRef.String() == "" || capacidad.Slots == 0 {
		return nil, errAgentMicroVMCapacidadFisicaInvalida
	}
	return []application.DescriptorCapacidadColocacionAgente{{
		PlacementRef: capacidad.PlacementRef,
		SourceRef:    referenciaFuenteCapacidadAgentMicroVM,
		PoolRef:      referenciaPoolCapacidadAgentMicroVM(capacidad.PlacementRef),
		BaseMedicion: application.BaseMedicionCapacidadBruta,
		Plazas:       int64(capacidad.Slots),
	}}, nil
}

// Shutdown drena operaciones que ya cruzaron la compuerta, la cierra para
// nuevas llamadas y libera solo recursos locales de composicion. El contexto
// no cancela cleanup: una parada solicitada debe cerrar ambos owners una vez.
func (agente *agenteMicroVM) Shutdown(_ context.Context) error {
	if agente == nil {
		return errAgentMicroVMAdapterRequerido
	}
	agente.shutdownOnce.Do(func() {
		agente.compuerta.Lock()
		agente.cerrado = true
		agente.compuerta.Unlock()
		agente.shutdownErr = errors.Join(
			agente.cerrarConexiones(),
			agente.cerrarCredenciales(),
		)
	})
	return agente.shutdownErr
}

func referenciaPoolCapacidadAgentMicroVM(colocacion ports.AgentPlacementRef) application.AgentCapacityPoolRef {
	texto := colocacion.String()
	hash := sha256.New()
	_, _ = hash.Write([]byte(dominioPoolCapacidadAgentMicroVM))
	var longitud [8]byte
	binary.BigEndian.PutUint64(longitud[:], uint64(len(texto)))
	_, _ = hash.Write(longitud[:])
	_, _ = hash.Write([]byte(texto))
	return application.AgentCapacityPoolRef("capacity-pool:codex:microvm:" + hex.EncodeToString(hash.Sum(nil)))
}

func interfazNulaAgentMicroVM(valor any) bool {
	if valor == nil {
		return true
	}
	reflejo := reflect.ValueOf(valor)
	switch reflejo.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflejo.IsNil()
	default:
		return false
	}
}

var _ agenteMicroVMDelegado = (*agentmicrovm.Adapter)(nil)
var _ agenteMicroVMDelegado = (*agentmicrovm.DockerAdapter)(nil)
var _ AgentAdapter = (*agenteMicroVM)(nil)
var _ application.AgentLaunchReconciler = (*agenteMicroVM)(nil)
var _ catalogoCapacidadColocacionAgente = (*agenteMicroVM)(nil)
