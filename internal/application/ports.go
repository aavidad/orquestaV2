package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

// AgentLauncher is the outbound launch port consumed by the orchestrator.
type AgentLauncher interface {
	Capabilities(context.Context) (ports.AgentCapabilities, error)
	// Launch is causally idempotent for an equal execution, key and request.
	Launch(context.Context, ports.AgentLaunchRequest) (ports.AgentLaunchReceipt, error)
}

// temporaryAgentError is implemented structurally by adapters when a launch
// can be retried safely. The application owns retry timing and durable queues.
type temporaryAgentError interface {
	error
	Temporary() bool
}

func isTemporaryAgentError(err error) bool {
	var temporary temporaryAgentError
	return errors.As(err, &temporary) && temporary.Temporary()
}

// definitelyNotAppliedAgentError is optional structural evidence that an
// adapter rejected an operation before crossing its external effect boundary.
// Temporary alone is deliberately insufficient: a timeout may hide success.
type definitelyNotAppliedAgentError interface {
	error
	DefinitelyNotApplied() bool
}

func isDefinitelyNotAppliedAgentError(err error) bool {
	var unapplied definitelyNotAppliedAgentError
	return errors.As(err, &unapplied) && unapplied.DefinitelyNotApplied()
}

// AgentObserver recovers observations, including terminal state, from durable
// accepted-execution identity.
type AgentObserver interface {
	ObserveAgent(context.Context, ports.AgentObserveRequest) (ports.AgentObservation, error)
}

// AgentController stops one exact execution. Global adapter shutdown remains a
// separate composition concern and is intentionally absent from this port.
type AgentController interface {
	ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error)
	Stop(context.Context, ports.AgentStopRequest) (ports.AgentStopReceipt, error)
}

var ErrAgentStopRecoveryUnsupported = errors.New("application.agent_stop_recovery_unsupported")

// AgentStopReconciler is an optional read-only recovery capability. It may
// inspect the exact durable stop but must never fall back to issuing Stop.
type AgentStopReconciler interface {
	ReconcileStop(context.Context, ports.AgentStopRequest) (ports.AgentStopReceipt, error)
}

// AgentStopReconcilerFrom keeps recovery structurally separate from ordinary
// controllers, so unsupported providers cannot receive a recovery call.
func AgentStopReconcilerFrom(controller AgentController) (AgentStopReconciler, error) {
	reconciler, ok := controller.(AgentStopReconciler)
	if controller == nil || !ok {
		return nil, ErrAgentStopRecoveryUnsupported
	}
	value := reflect.ValueOf(reconciler)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		if value.IsNil() {
			return nil, ErrAgentStopRecoveryUnsupported
		}
	}
	return reconciler, nil
}

// unsupportedAgentController is the null adapter used by compositions that
// have not opted into process control. It never performs an external effect;
// real compositions inject their provider-specific controller explicitly.
type unsupportedAgentController struct{}

func (unsupportedAgentController) ControlCapabilities(context.Context) (ports.AgentControlCapabilities, error) {
	return ports.AgentControlCapabilities{}, nil
}

func (unsupportedAgentController) Stop(
	_ context.Context,
	request ports.AgentStopRequest,
) (ports.AgentStopReceipt, error) {
	return ports.AgentStopReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, StopEffectAttemptRef: request.StopEffectAttemptRef,
		StopActionFence: request.StopActionFence, SpecHash: request.SpecHash,
		ProviderRef: request.ProviderRef, ModelRef: request.ModelRef, AgentRef: request.AgentRef,
		ExternalRef: request.ExternalRef, Mode: request.Mode, IdempotencyKey: request.IdempotencyKey,
		Status: ports.AgentStopUnsupported,
	}, nil
}

// ArtifactStore persists and reads immutable content-addressed blobs.
type ArtifactStore interface {
	Put(context.Context, ports.PutArtifactRequest) (ports.StoredArtifact, error)
	Get(context.Context, goal.ArtifactRef, int64) (ports.ArtifactContent, error)
}

type Clock interface {
	Now() time.Time
}

type ConfiguracionControladoresCuotaAgente struct {
	VigenciaObservacion, DemoraReconexion time.Duration
	Ahora                                 func() time.Time
	Sumidero                              func(context.Context, AgentQuotaObservation, []byte) error
}

type ControladorCuotaAgente interface {
	EsperarInicial(context.Context) error
	Cerrar(context.Context) error
}

type IniciadorControladoresCuotaAgente interface {
	IniciarControladoresCuota(context.Context, ConfiguracionControladoresCuotaAgente) ([]ControladorCuotaAgente, error)
}

type IDGenerator interface {
	// NewID returns an opaque unique value. Namespaces carrying authority or
	// secrets (for example *-token) require cryptographically unpredictable
	// output; adapters must not substitute counters or timestamps there.
	NewID(context.Context, string) (string, error)
}
