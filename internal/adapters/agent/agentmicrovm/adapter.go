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
	"strings"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeConfigurationInvalid    = "agentmicrovm.configuration_invalid"
	CodeCapabilitiesUnavailable = "agentmicrovm.capabilities_unavailable"
	CodeCapabilitiesRejected    = "agentmicrovm.capabilities_rejected"
	CodeProtocolIncompatible    = "agentmicrovm.protocol_incompatible"
	CodePhysicalUnavailable     = "agentmicrovm.physical_unavailable"
	CodeOperationUnsupported    = "agentmicrovm.operation_unsupported"
	CodeCapabilityMismatch      = "agentmicrovm.capability_mismatch"
	CodeSigningFailed           = "agentmicrovm.signing_failed"
	CodeLaunchUnavailable       = "agentmicrovm.launch_unavailable"
	CodeLaunchRejected          = "agentmicrovm.launch_rejected"
	CodeLaunchResponseInvalid   = "agentmicrovm.launch_response_invalid"
)

const (
	operationCreateExecution       = "crear_ejecucion"
	operationStartSession          = "iniciar_sesion"
	operationSendSessionInput      = "enviar_entrada_sesion"
	operationReadSessionEvents     = "leer_eventos_sesion"
	operationReconcileSessionInput = "reconciliar_entrada_sesion"
	launchReceiptDomain            = "orquesta.agentmicrovm.launch-receipt.v1\x00"
	physicalExecutionPrefix        = "ejecucion:"
	maxPhysicalExecutionRefBytes   = 96
	maxDurableCounter              = uint64(1<<63 - 1)
)

var requiredRemoteOperations = [...]string{
	operationCreateExecution,
	operationStartSession,
	operationSendSessionInput,
	operationReadSessionEvents,
	operationReconcileSessionInput,
}

// Client is the narrow public boundary needed by the launch half of B10.3.
// Session, observation and control methods are deliberately absent here.
type Client interface {
	Capacidades(context.Context) (microvm.RespuestaCapacidades, error)
	Lanzar(context.Context, string, microvm.SolicitudLanzamiento) (microvm.RespuestaEjecucion, error)
}

// Signer receives the exact durable compilation. Key material remains owned by
// composition and never enters the adapter configuration as bytes.
type Signer interface {
	Preparar(
		microvm.ContextoAutorizado,
		microvm.PlanLanzamiento,
		time.Time,
		time.Duration,
	) (microvm.SolicitudLanzamiento, error)
}

// Config is fully resolved by composition. Capabilities describe the provider
// hosted inside the microVM; ProfileBinding describes only its physical image.
type Config struct {
	Client       Client
	Signer       Signer
	Profile      ProfileBinding
	Capabilities ports.AgentCapabilities
}

// Adapter translates launch authority without owning a lifecycle or retaining
// replay state. Repetition always crosses the sibling idempotency boundary with
// the same key and signed bytes.
type Adapter struct {
	client       Client
	signer       Signer
	profile      ProfileBinding
	capabilities ports.AgentCapabilities
}

// New validates only local, immutable composition facts. Remote readiness is
// negotiated by Capabilities and again immediately before every launch.
func New(config Config) (*Adapter, error) {
	if nilInterface(config.Client) || nilInterface(config.Signer) {
		return nil, fail(CodeConfigurationInvalid, nil)
	}
	if config.Profile.PlacementRef.String() == "" {
		return nil, fail(CodeProfileBindingInvalid, nil)
	}
	if err := microvm.ValidarDescriptorPerfilLanzamientoV1(config.Profile.Descriptor); err != nil {
		return nil, fail(CodeDescriptorInvalid, err)
	}
	if err := ports.ValidateAgentCapabilities(config.Capabilities); err != nil ||
		!config.Capabilities.RequierePreservacionEntorno {
		return nil, fail(CodeConfigurationInvalid, err)
	}
	return &Adapter{
		client:       config.Client,
		signer:       config.Signer,
		profile:      cloneProfileBinding(config.Profile),
		capabilities: cloneCapabilities(config.Capabilities),
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

// Launch signs and submits one exact physical launch. A physically available
// microVM is only an accepted launch; it is not evidence that agent work ran.
func (adapter *Adapter) Launch(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (ports.AgentLaunchReceipt, error) {
	if adapter == nil || adapter.client == nil || adapter.signer == nil {
		return ports.AgentLaunchReceipt{}, fail(CodeConfigurationInvalid, nil)
	}
	if err := adapter.validateRequirements(request); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	compiled, err := Compile(request, adapter.profile, request.EffectAuthority.ActionFence)
	if err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	if err := adapter.negotiate(ctx); err != nil {
		return ports.AgentLaunchReceipt{}, err
	}
	signed, err := adapter.signer.Preparar(
		compiled.Context,
		compiled.Plan,
		compiled.IssuedAt,
		compiled.Validity,
	)
	if err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, err)
	}
	if !validSignedPlan(compiled, signed.Plan) {
		return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, nil)
	}
	signed, ok := cloneSignedRequest(signed)
	if !ok {
		return ports.AgentLaunchReceipt{}, fail(CodeSigningFailed, nil)
	}
	receiptRef := launchReceiptRef(signed)
	response, err := adapter.client.Lanzar(ctx, request.IdempotencyKey, cloneSignedRequestUnchecked(signed))
	if err != nil {
		return ports.AgentLaunchReceipt{}, classifyLaunchError(err)
	}
	if !validLaunchResponse(response, compiled.Plan) {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchResponseInvalid, nil)
	}
	receipt := ports.AgentLaunchReceipt{
		ExecutionRef: request.ExecutionRef, GoalRef: request.GoalRef, WorkItemRef: request.WorkItemRef,
		PlanGeneration: request.PlanGeneration, AppSpecGeneration: request.AppSpecGeneration,
		ExecutionAttempt: request.ExecutionAttempt, SpecHash: request.SpecHash,
		ProviderRef: adapter.capabilities.ProviderRef, ModelRef: adapter.capabilities.ModelRef,
		AgentRef: adapter.capabilities.AgentRef, ExternalRef: response.Referencia,
		IdempotencyKey: request.IdempotencyKey, ReceiptRef: receiptRef,
		AcceptedAt: compiled.IssuedAt, RequierePreservacionEntorno: request.RequierePreservacionEntorno,
	}
	if err := ports.ValidateAgentLaunchReceipt(request, receipt); err != nil {
		return ports.AgentLaunchReceipt{}, fail(CodeLaunchResponseInvalid, err)
	}
	return receipt, nil
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
	response, err := adapter.client.Capacidades(ctx)
	if err != nil {
		return classifyCapabilitiesError(err)
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
	operations := make(map[string]struct{}, len(response.Operaciones))
	for _, operation := range response.Operaciones {
		operations[operation] = struct{}{}
	}
	for _, required := range requiredRemoteOperations {
		if _, available := operations[required]; !available {
			return fail(CodeOperationUnsupported, nil)
		}
	}
	return nil
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
	case CodeCapabilitiesUnavailable, CodePhysicalUnavailable, CodeLaunchUnavailable:
		return true
	default:
		return false
	}
}

// DefinitelyNotApplied is true only while no launch mutation crossed the Unix
// socket. A client launch error and an invalid success response stay ambiguous.
func (err *Error) DefinitelyNotApplied() bool {
	if err == nil {
		return false
	}
	switch err.Code {
	case CodeConfigurationInvalid, CodeCapabilitiesUnavailable, CodeCapabilitiesRejected,
		CodeProtocolIncompatible, CodePhysicalUnavailable, CodeOperationUnsupported,
		CodeCapabilityMismatch, CodeSigningFailed,
		CodeLaunchRequestInvalid, CodeSessionRequired, CodeAccessAuthorityRequired,
		CodeEffectAuthorityInvalid, CodeFenceInvalid, CodeFenceMismatch,
		CodeProfileBindingInvalid, CodeDescriptorInvalid, CodeControlBrokerRequired,
		CodeGrantWindowInvalid, CodeLimitsInvalid, CodeLimitOverflow,
		CodeContextInvalid, CodePlanInvalid:
		return true
	default:
		return false
	}
}
