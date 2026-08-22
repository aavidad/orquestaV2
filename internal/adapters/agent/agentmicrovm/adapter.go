package agentmicrovm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeConfigurationInvalid                  = "agentmicrovm.configuration_invalid"
	CodeCapabilitiesUnavailable               = "agentmicrovm.capabilities_unavailable"
	CodeCapabilitiesRejected                  = "agentmicrovm.capabilities_rejected"
	CodeNegotiatedPhysicalCapacityUnavailable = "agentmicrovm.negotiated_physical_capacity_unavailable"
	CodeProtocolIncompatible                  = "agentmicrovm.protocol_incompatible"
	CodePhysicalUnavailable                   = "agentmicrovm.physical_unavailable"
	CodeOperationUnsupported                  = "agentmicrovm.operation_unsupported"
	CodeCapabilityMismatch                    = "agentmicrovm.capability_mismatch"
	CodeSigningFailed                         = "agentmicrovm.signing_failed"
	CodeLaunchAuthorityResolveFailed          = "agentmicrovm.launch_authority_resolve_failed"
	CodeLaunchAuthorityBuildFailed            = "agentmicrovm.launch_authority_build_failed"
	CodeLaunchAuthorityLookupFailed           = "agentmicrovm.launch_authority_lookup_failed"
	CodeLaunchAuthorityReplayInvalid          = "agentmicrovm.launch_authority_replay_invalid"
	CodeLaunchAuthorityPrepareFailed          = "agentmicrovm.launch_authority_prepare_failed"
	CodeLaunchAuthorityBindFailed             = "agentmicrovm.launch_authority_bind_failed"
	CodeLaunchCanceledBeforeSubmit            = "agentmicrovm.launch_canceled_before_submit"
	CodeLaunchUnavailable                     = "agentmicrovm.launch_unavailable"
	CodeLaunchRejected                        = "agentmicrovm.launch_rejected"
	CodeLaunchResponseInvalid                 = "agentmicrovm.launch_response_invalid"
	CodeObservationUnavailable                = "agentmicrovm.observation_unavailable"
)

const (
	operationCreateExecution       = "crear_ejecucion"
	operationObserveExecution      = "consultar_ejecucion"
	operationReadWorkRevision      = "consultar_revision_trabajo"
	operationStartSession          = "iniciar_sesion"
	operationSendSessionInput      = "enviar_entrada_sesion"
	operationReadSessionEvents     = "leer_eventos_sesion"
	operationReconcileSessionInput = "reconciliar_entrada_sesion"
	operationStopExecution         = "detener_ejecucion"
	launchReceiptDomain            = "orquesta.agentmicrovm.launch-receipt.v1\x00"
	physicalExecutionPrefix        = "ejecucion:"
	maxPhysicalExecutionRefBytes   = 96
	maxDurableCounter              = uint64(1<<63 - 1)
)

var requiredRemoteOperations = [...]string{
	operationCreateExecution,
	operationObserveExecution,
	operationReadWorkRevision,
	operationStartSession,
	operationSendSessionInput,
	operationReadSessionEvents,
	operationReconcileSessionInput,
	operationStopExecution,
}

// Client is the public sibling boundary needed to admit physical capacity and
// deliver one durable agent work packet. Read-only observation remains split
// in observationClient so ObserveAgent cannot mutate the sibling.
type Client interface {
	Capacidades(context.Context) (microvm.RespuestaCapacidades, error)
	Lanzar(context.Context, string, microvm.SolicitudLanzamiento) (microvm.RespuestaEjecucion, error)
	RevisionTrabajo(context.Context, string) (microvm.RespuestaRevisionTrabajo, error)
	IniciarSesion(
		context.Context,
		string,
		string,
		microvm.SolicitudIniciarSesionTrabajoV1,
	) (microvm.RespuestaSesionTrabajoV1, error)
	EnviarEntradaSesion(
		context.Context,
		string,
		string,
		string,
		microvm.SolicitudEntradaSesionTrabajoV1,
	) (microvm.RespuestaSesionTrabajoV1, error)
	ReconciliarEntradaSesion(
		context.Context,
		string,
		string,
		string,
		uint64,
	) (microvm.RespuestaReconciliacionEntradaSesionTrabajoV1, error)
	Detener(context.Context, string, string, microvm.SolicitudDetencion) (microvm.RespuestaDetencion, error)
}

// Signer receives the exact durable compilation. Key material remains owned by
// composition and never enters the adapter configuration as bytes.
type Signer interface {
	Preparar(
		context.Context,
		ports.AgentLaunchRequest,
		microvm.ContextoAutorizado,
		microvm.PlanLanzamiento,
		time.Time,
		time.Duration,
	) (microvm.SolicitudLanzamiento, error)
}

// ProviderModelBinding is one immutable composition choice. ModelRef is the
// logical scheduler selector; ProviderModel is the exact Codex runtime model;
// Profile binds both to one published physical image.
type ProviderModelBinding struct {
	ModelRef      string
	ProviderModel string
	Profile       ProfileBinding
}

// Config is fully resolved by composition. Capabilities describe the provider
// hosted inside the microVM; ModelBinding prevents a logical selector, runtime
// model and physical executor from drifting independently.
type Config struct {
	Client                  Client
	Signer                  Signer
	ClaimResolver           *CredentialClaimResolver
	LaunchAuthorityRegistry ports.MicroVMHostLaunchPreparationRegistry
	Capabilities            ports.AgentCapabilities
	ModelBinding            ProviderModelBinding
	PromptRenderer          PromptRenderer
}

// NegotiatedPhysicalCapacity is the last complete physical-capacity proof
// obtained from Agente MicroVM. PlacementRef remains opaque and Slots is the
// exact remote maximum; neither value is inferred from logical capacity.
type NegotiatedPhysicalCapacity struct {
	PlacementRef ports.AgentPlacementRef
	Slots        uint32
}

// Adapter translates launch authority without owning a lifecycle or retaining
// replay state. Repetition always crosses the sibling idempotency boundary with
// the same key and signed bytes.
type Adapter struct {
	client                  Client
	observer                observationClient
	signer                  Signer
	claimResolver           *CredentialClaimResolver
	launchAuthorityRegistry ports.MicroVMHostLaunchPreparationRegistry
	profile                 ProfileBinding
	capabilities            ports.AgentCapabilities
	model                   string
	renderer                PromptRenderer

	negotiationGate                     chan struct{}
	physicalCapacityMu                  sync.RWMutex
	physicalCapacityGeneration          uint64
	physicalCapacityGenerationExhausted bool
	physicalCapacity                    NegotiatedPhysicalCapacity
	physicalCapacityAvailable           bool
}

// New validates only local, immutable composition facts. Remote readiness is
// negotiated by Capabilities and again immediately before every launch.
func New(config Config) (*Adapter, error) {
	if nilInterface(config.Client) || nilInterface(config.Signer) ||
		nilInterface(config.ClaimResolver) || nilInterface(config.LaunchAuthorityRegistry) ||
		nilInterface(config.PromptRenderer) ||
		config.ModelBinding.ModelRef != config.Capabilities.ModelRef ||
		!validWorkPacketToken(config.ModelBinding.ProviderModel, 128) {
		return nil, fail(CodeConfigurationInvalid, nil)
	}
	observer, ok := config.Client.(observationClient)
	if !ok || nilInterface(observer) {
		return nil, fail(CodeObservationClientInvalid, nil)
	}
	if config.ModelBinding.Profile.PlacementRef.String() == "" {
		return nil, fail(CodeProfileBindingInvalid, nil)
	}
	if err := microvm.ValidarDescriptorPerfilLanzamientoV1(config.ModelBinding.Profile.Descriptor); err != nil {
		return nil, fail(CodeDescriptorInvalid, err)
	}
	if err := ports.ValidateAgentCapabilities(config.Capabilities); err != nil ||
		!config.Capabilities.RequierePreservacionEntorno {
		return nil, fail(CodeConfigurationInvalid, err)
	}
	return &Adapter{
		client:                  config.Client,
		observer:                observer,
		signer:                  config.Signer,
		claimResolver:           config.ClaimResolver,
		launchAuthorityRegistry: config.LaunchAuthorityRegistry,
		profile:                 cloneProfileBinding(config.ModelBinding.Profile),
		capabilities:            cloneCapabilities(config.Capabilities),
		model:                   config.ModelBinding.ProviderModel,
		renderer:                config.PromptRenderer,
		negotiationGate:         make(chan struct{}, 1),
	}, nil
}

// Capabilities returns the exact neutral provider facts configured by the
// composition only after the independent runtime proves the complete B10
// protocol surface is currently usable.
func (adapter *Adapter) Capabilities(ctx context.Context) (ports.AgentCapabilities, error) {
	if adapter == nil || adapter.client == nil {
		return ports.AgentCapabilities{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := adapter.negotiate(ctx); err != nil {
		return ports.AgentCapabilities{}, err
	}
	return cloneCapabilities(adapter.capabilities), nil
}

// NegotiatedPhysicalCapacity returns only a capacity published by a complete
// successful negotiation. A negotiation in progress or any later failed
// negotiation leaves this read closed instead of retaining stale capacity.
func (adapter *Adapter) NegotiatedPhysicalCapacity() (NegotiatedPhysicalCapacity, error) {
	if adapter == nil {
		return NegotiatedPhysicalCapacity{}, fail(CodeNegotiatedPhysicalCapacityUnavailable, nil)
	}
	adapter.physicalCapacityMu.RLock()
	capacity, available := adapter.physicalCapacity, adapter.physicalCapacityAvailable
	adapter.physicalCapacityMu.RUnlock()
	if !available {
		return NegotiatedPhysicalCapacity{}, fail(CodeNegotiatedPhysicalCapacityUnavailable, nil)
	}
	return capacity, nil
}

// Launch signs and submits one exact physical launch. A physically available
// microVM is only an accepted launch; it is not evidence that agent work ran.
func (adapter *Adapter) Launch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	return adapter.launch(ctx, request, false)
}

func (adapter *Adapter) launch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	reconcile bool,
) (ports.AgentLaunchReceipt, error) {
	if adapter == nil || nilInterface(adapter.client) || nilInterface(adapter.signer) ||
		nilInterface(adapter.claimResolver) || nilInterface(adapter.launchAuthorityRegistry) ||
		nilInterface(adapter.renderer) || nilInterface(ctx) {
		return ports.AgentLaunchReceipt{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := adapter.validateRequirements(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	var historical ports.MicroVMHostLaunchAuthorityV1
	if reconcile {
		key := ports.MicroVMHostLaunchAuthorityKey{
			RunRef: request.ExecutionRef, ActionFence: request.EffectAuthority.ActionFence,
		}
		var err error
		historical, err = adapter.launchAuthorityRegistry.Resolve(ctx, key)
		if err != nil {
			return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityLookupFailed, err)
		}
		if !adapter.validHistoricalLaunchAuthority(request, historical) {
			return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityReplayInvalid, nil)
		}
	}
	compiled, err := Compile(request, adapter.profile, request.EffectAuthority.ActionFence)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	packet, packetJSON, err := BuildWorkPacketV1(
		request, adapter.profile, compiled, adapter.model, adapter.renderer,
	)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := adapter.negotiate(ctx); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	signed, err := adapter.signer.Preparar(
		ctx,
		request,
		compiled.Context,
		cloneLaunchPlan(compiled.Plan),
		compiled.IssuedAt,
		compiled.Validity,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ports.AgentLaunchReceipt{}, err
		}
		if contextErr := ctx.Err(); contextErr != nil {
			return ports.AgentLaunchReceipt{}, contextErr
		}
		return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, err)
	}
	if !validSignedPlan(compiled, signed.Plan) {
		return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	signed, ok := cloneSignedRequest(signed)
	if !ok {
		return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, nil)
	}
	var resolved ResolvedCredentialClaim
	if reconcile {
		resolved = ResolvedCredentialClaim{
			placement: request.ReferenciaColocacion,
			claim:     historical.OneShotClaim,
		}
	} else {
		resolved, err = adapter.claimResolver.Resolve(ctx, request)
		if err != nil {
			return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityResolveFailed, err)
		}
	}
	authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved)
	if err != nil {
		if reconcile {
			return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityReplayInvalid, err)
		}
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityBuildFailed, err)
	}
	runtimeDigests, err := BuildMicroVMHostLaunchRuntimeDigestsV1(request, compiled, signed)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if reconcile {
		historicalDigests, resolveErr := adapter.launchAuthorityRegistry.ResolveRuntime(ctx, authority.Key)
		if resolveErr != nil || historicalDigests != runtimeDigests {
			return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityReplayInvalid, resolveErr)
		}
	}
	if reconcile && !validPreparedLaunchAuthorityReplay(authority, historical) {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityReplayInvalid, nil)
	}
	prepared, err := adapter.launchAuthorityRegistry.PrepareWithRuntime(ctx, authority, runtimeDigests)
	if err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityPrepareFailed, err)
	}
	if !validPreparedLaunchAuthorityReplay(authority, prepared) {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityPrepareFailed, nil)
	}
	if err := ctx.Err(); err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchCanceledBeforeSubmit, err)
	}
	receiptRef := launchReceiptRef(signed)
	response, err := adapter.client.Lanzar(ctx, request.IdempotencyKey, cloneSignedRequestUnchecked(signed))
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return ports.AgentLaunchReceipt{}, contextErr
		}
		return ports.AgentLaunchReceipt{}, classifyLaunchError(err)
	}
	if !validLaunchResponse(response, compiled.Plan) {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchResponseInvalid, nil)
	}
	bound, err := adapter.launchAuthorityRegistry.BindExternal(
		context.WithoutCancel(ctx), authority.Key, response.Referencia,
	)
	if err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityBindFailed, err)
	}
	if !validBoundLaunchAuthority(authority, prepared, bound, response.Referencia) {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchAuthorityBindFailed, nil)
	}
	if err := adapter.launchSession(ctx, request, response, packet.TimeBudgetMS, packetJSON); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: adapter.capabilities.ProviderRef, ModelRef: adapter.capabilities.ModelRef,
		AgentRef: adapter.capabilities.AgentRef, ExternalRef: bound.ExternalRef,
		IdempotencyKey: request.IdempotencyKey, ReceiptRef: receiptRef,
		AcceptedAt: compiled.IssuedAt, RequierePreservacionEntorno: request.RequierePreservacionEntorno,
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchResponseInvalid, err)
	}
	return receipt, nil
}

// validHistoricalLaunchAuthority accepts only the exact historical key and
// causal claim selected by the current placement configuration. It never asks
// the credential store for the current version: recovery must retain the
// version prepared before the original physical launch.
func (adapter *Adapter) validHistoricalLaunchAuthority(
	request ports.AgentLaunchRequest,
	authority ports.MicroVMHostLaunchAuthorityV1,
) bool {
	if ports.ValidateMicroVMHostLaunchAuthorityV1(authority) != nil ||
		authority.Key.RunRef != request.ExecutionRef ||
		authority.Key.ActionFence != request.EffectAuthority.ActionFence ||
		authority.EffectAttemptRef != request.EffectAuthority.EffectAttemptRef ||
		authority.SessionRef != request.SessionRef || adapter.claimResolver == nil {
		return false
	}
	credentialRef, found := adapter.claimResolver.bindings[request.ReferenciaColocacion]
	claim := authority.OneShotClaim
	return found && credentialRef == claim.CredentialRef &&
		adapter.claimResolver.purpose == claim.PurposeRef &&
		claim.ActorRef == request.ActorRef.String() &&
		claim.OwnerRef.String() == request.ActorRef.String() &&
		claim.ScopeRef.String() == request.ProjectRef.String()
}

func validPreparedLaunchAuthorityReplay(
	want ports.MicroVMHostLaunchAuthorityV1,
	got ports.MicroVMHostLaunchAuthorityV1,
) bool {
	prepared := ports.CloneMicroVMHostLaunchAuthorityV1(got)
	prepared.ExternalRef = ""
	if !reflect.DeepEqual(prepared, want) {
		return false
	}
	if got.ExternalRef == "" {
		return ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(got) == nil
	}
	return ports.ValidateMicroVMHostLaunchAuthorityBoundV1(got) == nil
}

func validBoundLaunchAuthority(
	want ports.MicroVMHostLaunchAuthorityV1,
	prepared ports.MicroVMHostLaunchAuthorityV1,
	bound ports.MicroVMHostLaunchAuthorityV1,
	externalRef string,
) bool {
	want = ports.CloneMicroVMHostLaunchAuthorityV1(want)
	want.ExternalRef = externalRef
	return (prepared.ExternalRef == "" || prepared.ExternalRef == externalRef) &&
		ports.ValidateMicroVMHostLaunchAuthorityBoundV1(bound) == nil &&
		reflect.DeepEqual(bound, want)
}

// ReconcileLaunch repeats the exact durable launch through the sibling's
// idempotency boundary. It deliberately shares the complete Launch pipeline so
// signing, session delivery and receipt derivation cannot drift during host
// recovery.
func (adapter *Adapter) ReconcileLaunch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	return adapter.launch(ctx, request, true)
}

func validSignedPlan(compiled Compilation, raw json.RawMessage) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) < 2 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' {
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var actual microvm.PlanLanzamiento
	if err := decoder.Decode(&actual); err != nil {
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return false
	}
	contextJSON, err := json.Marshal(compiled.Context)
	if err != nil {
		return false
	}
	contextDigest := sha256.Sum256(contextJSON)
	expected := compiled.Plan
	expected.Esquema = microvm.EsquemaPlanLanzamiento
	expected.PlanRef = "plan:" + hex.EncodeToString(contextDigest[:])
	expected.RunRef = compiled.Context.EjecucionRef
	return reflect.DeepEqual(actual, expected)
}

func (adapter *Adapter) validateRequirements(request ports.AgentLaunchRequest) error {
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return fail(CodeLaunchRequestInvalid, err)
	}
	requirements := ports.AgentRequirements{
		RoleKey: request.RoleKey, SkillRefs: request.SkillRefs,
		ToolRefs: request.ToolRefs, CapabilityRefs: request.CapabilityRefs,
	}
	if !ports.MatchAgentCapabilities(adapter.capabilities, requirements) ||
		request.RequierePreservacionEntorno && !adapter.capabilities.RequierePreservacionEntorno {
		return fail(CodeCapabilityMismatch, nil)
	}
	return nil
}

func (adapter *Adapter) negotiate(ctx context.Context) error {
	if nilInterface(ctx) {
		return fail(CodeConfigurationInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	generation, ok := adapter.beginPhysicalCapacityNegotiation()
	if !ok || adapter.negotiationGate == nil {
		return fail(CodeNegotiatedPhysicalCapacityUnavailable, nil)
	}
	select {
	case adapter.negotiationGate <- struct{}{}:
		defer func() { <-adapter.negotiationGate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	response, err := adapter.client.Capacidades(ctx)
	if err != nil {
		if contextErr := preserveContextError(ctx, err); contextErr != nil {
			return contextErr
		}
		return classifyCapabilitiesError(err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if response.Protocolo != microvm.ProtocoloLocal {
		return fail(CodeProtocolIncompatible, nil)
	}
	if !validRemoteVersion(response.Version) {
		return fail(CodeProtocolIncompatible, nil)
	}
	if !response.KVMDisponible || !response.FirecrackerConfigurado ||
		!response.FirecrackerEjecutable || response.MaximoEjecuciones == 0 {
		return fail(CodePhysicalUnavailable, nil)
	}
	if !supportsRemoteOperations(response.Operaciones, requiredRemoteOperations[:]...) {
		return fail(CodeOperationUnsupported, nil)
	}
	adapter.publishNegotiatedPhysicalCapacity(generation, NegotiatedPhysicalCapacity{
		PlacementRef: adapter.profile.PlacementRef,
		Slots:        response.MaximoEjecuciones,
	})
	return nil
}

func supportsRemoteOperations(available []string, required ...string) bool {
	for _, operation := range required {
		if !slices.Contains(available, operation) {
			return false
		}
	}
	return true
}

func (adapter *Adapter) beginPhysicalCapacityNegotiation() (uint64, bool) {
	adapter.physicalCapacityMu.Lock()
	defer adapter.physicalCapacityMu.Unlock()
	adapter.physicalCapacity = NegotiatedPhysicalCapacity{}
	adapter.physicalCapacityAvailable = false
	if adapter.physicalCapacityGeneration == ^uint64(0) {
		adapter.physicalCapacityGenerationExhausted = true
		return 0, false
	}
	adapter.physicalCapacityGeneration++
	return adapter.physicalCapacityGeneration, true
}

func (adapter *Adapter) publishNegotiatedPhysicalCapacity(
	generation uint64,
	capacity NegotiatedPhysicalCapacity,
) bool {
	adapter.physicalCapacityMu.Lock()
	defer adapter.physicalCapacityMu.Unlock()
	if adapter.physicalCapacityGenerationExhausted || adapter.physicalCapacityGeneration != generation {
		return false
	}
	adapter.physicalCapacity = capacity
	adapter.physicalCapacityAvailable = true
	return true
}

func validLaunchResponse(response microvm.RespuestaEjecucion, plan microvm.PlanLanzamiento) bool {
	if response.Estado != "disponible" || !validPhysicalExecutionRef(response.Referencia) ||
		response.Revision == 0 || response.Revision > maxDurableCounter || response.Cerca != plan.Cerca ||
		response.VCPU != plan.VCPU || response.MemoriaMiB != plan.MemoriaMiB ||
		response.RevisionTrabajo != nil || response.Identidad == nil || response.Identidad.PID == 0 ||
		response.Identidad.InicioTicks == 0 {
		return false
	}
	if response.ProcesoVivo != nil && !*response.ProcesoVivo {
		return false
	}
	return response.EstadoMotor == nil || *response.EstadoMotor == "Running"
}

func validPhysicalExecutionRef(value string) bool {
	suffix := strings.TrimPrefix(value, physicalExecutionPrefix)
	if suffix == value || suffix == "" || len(value) > maxPhysicalExecutionRefBytes {
		return false
	}
	for index := 0; index < len(suffix); index++ {
		character := suffix[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') &&
			character != '-' && character != '_' {
			return false
		}
	}
	return true
}

func validRemoteVersion(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func launchReceiptRef(request microvm.SolicitudLanzamiento) string {
	digest := sha256.New()
	digest.Write([]byte(launchReceiptDomain))
	appendDigestField(digest, request.Plan)
	appendDigestField(digest, request.Concesion)
	return "agentmicrovm-launch:sha256:" + hex.EncodeToString(digest.Sum(nil))
}

type digestWriter interface{ Write([]byte) (int, error) }

func appendDigestField(digest digestWriter, value []byte) {
	_, _ = digest.Write(binary.BigEndian.AppendUint64(nil, uint64(len(value))))
	_, _ = digest.Write(value)
}

func cloneSignedRequest(request microvm.SolicitudLanzamiento) (microvm.SolicitudLanzamiento, bool) {
	if len(request.Plan) == 0 || len(request.Concesion) == 0 ||
		!json.Valid(request.Plan) || !json.Valid(request.Concesion) {
		return microvm.SolicitudLanzamiento{}, false
	}
	return cloneSignedRequestUnchecked(request), true
}

func cloneSignedRequestUnchecked(request microvm.SolicitudLanzamiento) microvm.SolicitudLanzamiento {
	return microvm.SolicitudLanzamiento{
		Plan:      append(json.RawMessage(nil), request.Plan...),
		Concesion: append(json.RawMessage(nil), request.Concesion...),
	}
}

func cloneProfileBinding(binding ProfileBinding) ProfileBinding {
	binding.Descriptor.ServiciosDisponibles = append(
		[]microvm.ServicioVsock(nil), binding.Descriptor.ServiciosDisponibles...,
	)
	return binding
}

func cloneLaunchPlan(plan microvm.PlanLanzamiento) microvm.PlanLanzamiento {
	if plan.PerfilSHA256 != nil {
		profileSHA256 := *plan.PerfilSHA256
		plan.PerfilSHA256 = &profileSHA256
	}
	plan.Servicios = append([]microvm.ServicioVsock(nil), plan.Servicios...)
	if plan.Egreso != nil {
		egress := *plan.Egreso
		egress.Destinos = append([]microvm.DestinoEgreso(nil), plan.Egreso.Destinos...)
		for index := range egress.Destinos {
			egress.Destinos[index].Puertos = append([]uint16(nil), egress.Destinos[index].Puertos...)
		}
		plan.Egreso = &egress
	}
	return plan
}

func cloneCapabilities(capabilities ports.AgentCapabilities) ports.AgentCapabilities {
	capabilities.RoleKeys = append([]string(nil), capabilities.RoleKeys...)
	capabilities.SkillRefs = append([]string(nil), capabilities.SkillRefs...)
	capabilities.ToolRefs = append([]string(nil), capabilities.ToolRefs...)
	capabilities.CapabilityRefs = append([]string(nil), capabilities.CapabilityRefs...)
	return capabilities
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func preserveContextError(ctx context.Context, err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if !nilInterface(ctx) {
		return ctx.Err()
	}
	return nil
}

func classifyCapabilitiesError(err error) error {
	var protocolError *microvm.ErrorProtocolo
	if errors.As(err, &protocolError) {
		return fail(CodeProtocolIncompatible, err)
	}
	var configurationError *microvm.ErrorConfiguracion
	var responseTooLarge *microvm.ErrorRespuestaGrande
	if errors.As(err, &configurationError) {
		return fail(CodeConfigurationInvalid, err)
	}
	if errors.As(err, &responseTooLarge) {
		return fail(CodeCapabilitiesRejected, err)
	}
	var responseError *microvm.ErrorRespuesta
	if errors.As(err, &responseError) && responseError.Estado != http.StatusRequestTimeout &&
		responseError.Estado != http.StatusTooManyRequests && responseError.Estado < http.StatusInternalServerError {
		return fail(CodeCapabilitiesRejected, err)
	}
	return fail(CodeCapabilitiesUnavailable, err)
}

func classifyLaunchError(err error) error {
	var responseError *microvm.ErrorRespuesta
	if errors.As(err, &responseError) && responseError.Estado != http.StatusRequestTimeout &&
		responseError.Estado != http.StatusTooManyRequests && responseError.Estado < http.StatusInternalServerError {
		return fail(CodeLaunchRejected, err)
	}
	var protocolError *microvm.ErrorProtocolo
	var configurationError *microvm.ErrorConfiguracion
	var responseTooLarge *microvm.ErrorRespuestaGrande
	if errors.As(err, &protocolError) || errors.As(err, &configurationError) ||
		errors.As(err, &responseTooLarge) {
		return fail(CodeLaunchRejected, err)
	}
	var networkError net.Error
	if errors.As(err, &networkError) || responseError != nil {
		return fail(CodeLaunchUnavailable, err)
	}
	// An arbitrary client error may hide a submitted request. It is retryable
	// only through the same idempotency key and is never definitely unapplied.
	return fail(CodeLaunchUnavailable, err)
}

// Temporary reports only conditions that may clear without changing the
// request. A retry remains governed by the application's durable attempt.
func (err *Error) Temporary() bool {
	if err == nil {
		return false
	}
	switch err.Code {
	case CodeCapabilitiesUnavailable, CodePhysicalUnavailable, CodeLaunchUnavailable,
		CodeNegotiatedPhysicalCapacityUnavailable,
		CodeObservationUnavailable, CodeWorkRevisionUnavailable,
		CodeControlObservationFailed, CodeControlUnavailable,
		CodeSessionStartUnavailable, CodeSessionPending,
		CodeSessionInputUnavailable, CodeSessionInputPending,
		CodeSessionObserveUnavailable:
		return true
	default:
		return false
	}
}

// DefinitelyNotApplied is true only when the exact failure stage proves that
// this call did not invoke client.Lanzar. The generic Prepare code also covers
// invalid or conflicting durable replays and stays conservative; cancellation
// observed after a successful Prepare has an explicit pre-submit code instead.
func (err *Error) DefinitelyNotApplied() bool {
	if err == nil {
		return false
	}
	switch err.Code {
	case CodeConfigurationInvalid, CodeCapabilitiesUnavailable, CodeCapabilitiesRejected,
		CodeNegotiatedPhysicalCapacityUnavailable,
		CodeProtocolIncompatible, CodePhysicalUnavailable, CodeOperationUnsupported,
		CodeControlRequestInvalid, CodeControlObservationInvalid, CodeControlObservationFailed,
		CodeControlCanceledBeforeCall,
		CodeCapabilityMismatch, CodeSigningFailed,
		CodeLaunchAuthorityResolveFailed, CodeLaunchAuthorityBuildFailed,
		CodeLaunchCanceledBeforeSubmit,
		CodeLaunchRequestInvalid, CodeSessionRequired, CodeAccessAuthorityRequired,
		CodeEffectAuthorityInvalid, CodeFenceInvalid, CodeFenceMismatch,
		CodeProfileBindingInvalid, CodeDescriptorInvalid, CodeControlBrokerRequired,
		CodeGrantWindowInvalid, CodeLimitsInvalid, CodeLimitOverflow,
		CodeContextInvalid, CodePlanInvalid,
		CodeWorkPacketRequestInvalid, CodeWorkPacketCompilationMismatch,
		CodeWorkPacketModelInvalid, CodeWorkPacketEffortInvalid,
		CodeWorkPacketBudgetInvalid, CodeWorkPacketOutputInvalid,
		CodeWorkPacketRendererInvalid, CodeWorkPacketRenderFailed,
		CodeWorkPacketPromptInvalid, CodeWorkPacketEncodeFailed,
		CodeWorkPacketTooLarge, CodeWorkPacketRoundTripInvalid:
		return true
	default:
		return false
	}
}
