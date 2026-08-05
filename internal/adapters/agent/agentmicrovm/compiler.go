// Package agentmicrovm translates Orquesta launch authority into the public
// Agente MicroVM contract. It does not own transport, signing or lifecycle.
package agentmicrovm

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"strings"
	"time"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/ports"
)

const (
	CodeLaunchRequestInvalid    = "agentmicrovm.launch_request_invalid"
	CodeSessionRequired         = "agentmicrovm.session_required"
	CodeAccessAuthorityRequired = "agentmicrovm.access_authority_required"
	CodeEffectAuthorityInvalid  = "agentmicrovm.effect_authority_invalid"
	CodeFenceInvalid            = "agentmicrovm.fence_invalid"
	CodeFenceMismatch           = "agentmicrovm.fence_mismatch"
	CodeProfileBindingInvalid   = "agentmicrovm.profile_binding_invalid"
	CodeDescriptorInvalid       = "agentmicrovm.profile_descriptor_invalid"
	CodeControlBrokerRequired   = "agentmicrovm.control_broker_required"
	CodeEgressAuthorityInvalid  = "agentmicrovm.egress_authority_invalid"
	CodeEgressAuthorityMismatch = "agentmicrovm.egress_authority_mismatch"
	CodeEgressProxyRequired     = "agentmicrovm.egress_proxy_required"
	CodeGrantWindowInvalid      = "agentmicrovm.grant_window_invalid"
	CodeLimitsInvalid           = "agentmicrovm.limits_invalid"
	CodeLimitOverflow           = "agentmicrovm.limit_overflow"
	CodeContextInvalid          = "agentmicrovm.authorized_context_invalid"
	CodePlanInvalid             = "agentmicrovm.launch_plan_invalid"
)

const bytesPerMiB = uint64(1024 * 1024)

// Compilation is deterministic launch material. A later boundary signs it and
// the sibling connector replaces PlanRef with the context-bound canonical ref.
type Compilation struct {
	Plan     microvm.PlanLanzamiento
	Context  microvm.ContextoAutorizado
	IssuedAt time.Time
	Validity time.Duration

	// sourceFingerprint seals every request and profile fact consumed by this
	// package. Being unexported prevents another package from manufacturing a
	// Compilation that can be crossed with a different request.
	sourceFingerprint [sha256.Size]byte
}

// ProfileBinding is the composition-owned mapping from an opaque placement to
// one immutable physical descriptor. Compile never chooses a profile itself.
type ProfileBinding struct {
	PlacementRef ports.AgentPlacementRef
	Descriptor   microvm.DescriptorPerfilLanzamientoV1
}

// Error exposes a stable code while retaining the lower-level contract error
// for local diagnosis.
type Error struct {
	Code  string
	Cause error
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func ErrorCode(err error) string {
	var adapterError *Error
	if errors.As(err, &adapterError) {
		return adapterError.Code
	}
	return ""
}

// Compile translates only durable facts. The supplied fence must be the exact
// action fence already carried by the effect authority; it is never inferred
// from process or Firecracker state.
func Compile(
	request ports.AgentLaunchRequest,
	binding ProfileBinding,
	fence uint64,
) (Compilation, error) {
	if request.SessionRef.String() == "" {
		return Compilation{}, fail(CodeSessionRequired, nil)
	}
	if !completeAccessAuthority(request.AccessAuthority) {
		return Compilation{}, fail(CodeAccessAuthorityRequired, nil)
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return Compilation{}, fail(CodeLaunchRequestInvalid, err)
	}
	if err := ports.ValidateAgentLaunchEffectAuthority(request.EffectAuthority); err != nil {
		return Compilation{}, fail(CodeEffectAuthorityInvalid, err)
	}
	if fence == 0 {
		return Compilation{}, fail(CodeFenceInvalid, nil)
	}
	if fence != request.EffectAuthority.ActionFence {
		return Compilation{}, fail(CodeFenceMismatch, nil)
	}
	if binding.PlacementRef.String() == "" || binding.PlacementRef != request.ReferenciaColocacion {
		return Compilation{}, fail(CodeProfileBindingInvalid, nil)
	}
	descriptor := binding.Descriptor
	if err := microvm.ValidarDescriptorPerfilLanzamientoV1(descriptor); err != nil {
		return Compilation{}, fail(CodeDescriptorInvalid, err)
	}
	egress, egressCode, err := decodeEgressAuthority(request.EgressAuthority)
	if err != nil {
		return Compilation{}, fail(egressCode, err)
	}
	services, serviceCode := authorizedServices(descriptor, egress != nil)
	if serviceCode != "" {
		return Compilation{}, fail(serviceCode, nil)
	}
	if len(services) == 0 {
		return Compilation{}, fail(CodeControlBrokerRequired, nil)
	}
	issuedAt, validity, ok := grantWindow(request.EffectAuthority)
	if !ok {
		return Compilation{}, fail(CodeGrantWindowInvalid, nil)
	}

	timeLimitMS := request.BudgetDemand.Resources.ActiveTimeNS / int64(time.Millisecond)
	if timeLimitMS <= 0 || request.BudgetDemand.Resources.Tokens <= 0 ||
		request.BudgetDemand.Resources.DiskBytes <= 0 {
		return Compilation{}, fail(CodeLimitsInvalid, nil)
	}
	ramLimitBytes, ok := checkedMultiply(uint64(descriptor.MemoriaMiB), bytesPerMiB)
	if !ok {
		return Compilation{}, fail(CodeLimitOverflow, nil)
	}

	profileSHA := descriptor.PerfilSHA256
	plan := microvm.PlanLanzamiento{
		Esquema:              microvm.EsquemaPlanLanzamiento,
		PlanRef:              "plan:" + request.SpecHash,
		RunRef:               request.ExecutionRef.String(),
		Cerca:                fence,
		VCPU:                 descriptor.VCPU,
		MemoriaMiB:           descriptor.MemoriaMiB,
		KernelSHA256:         descriptor.KernelSHA256,
		InitramfsSHA256:      descriptor.InitramfsSHA256,
		PerfilSHA256:         &profileSHA,
		LimiteTiempoMS:       uint64(timeLimitMS),
		LimiteRAMPicoBytes:   ramLimitBytes,
		LimiteDiscoPicoBytes: uint64(request.BudgetDemand.Resources.DiskBytes),
		LimiteTokensAgente:   uint64(request.BudgetDemand.Resources.Tokens),
		Servicios:            services,
		Egreso:               egress,
	}
	if err := microvm.ValidarPlanConDescriptorPerfilLanzamientoV1(descriptor, plan); err != nil {
		return Compilation{}, fail(CodePlanInvalid, err)
	}

	context := microvm.ContextoAutorizado{
		ProyectoRef:          request.ProjectRef.String(),
		GoalRef:              request.GoalRef.String(),
		WorkItemRef:          request.WorkItemRef.String(),
		EjecucionRef:         request.ExecutionRef.String(),
		AutorizacionRef:      request.EffectAuthority.AuthorizationReceiptRef,
		AprobacionEfectoRef:  request.EffectAuthority.EffectApprovalRef,
		IntentoEfectoRef:     request.EffectAuthority.EffectAttemptRef,
		SesionRef:            request.SessionRef.String(),
		AccesoArtefactosRef:  request.AccessAuthority.ArtifactAccessRef.String(),
		AccesoMCPRef:         request.AccessAuthority.MCPAccessRef.String(),
		EndpointBuzonRef:     request.AccessAuthority.MailboxEndpointRef.String(),
		DescriptorPerfilRef:  descriptor.DescriptorRef,
		EspecificacionSHA256: request.SpecHash,
	}
	if !validContextBindings(context) {
		return Compilation{}, fail(CodeContextInvalid, nil)
	}
	return Compilation{
		Plan: plan, Context: context, IssuedAt: issuedAt, Validity: validity,
		sourceFingerprint: compilationSourceFingerprint(request, binding),
	}, nil
}

const compilationFingerprintDomain = "orquesta.agentmicrovm.compilation-source.v1\x00"

func compilationSourceFingerprint(request ports.AgentLaunchRequest, binding ProfileBinding) [sha256.Size]byte {
	digest := sha256.New()
	fingerprintString(digest, compilationFingerprintDomain)

	// AgentLaunchRequest: fixed field order is the v1 canonical encoding.
	fingerprintString(digest, request.ExecutionRef.String())
	fingerprintString(digest, request.ReferenciaColocacion.String())
	fingerprintString(digest, request.SessionRef.String())
	fingerprintString(digest, request.AccessAuthority.ArtifactAccessRef.String())
	fingerprintString(digest, request.AccessAuthority.MCPAccessRef.String())
	fingerprintString(digest, request.AccessAuthority.MailboxEndpointRef.String())
	fingerprintString(digest, request.ExecutionWorkspaceRef.String())
	fingerprintString(digest, request.GoalRef.String())
	fingerprintString(digest, request.WorkItemRef.String())
	fingerprintUint64(digest, uint64(request.PlanGeneration))
	fingerprintUint64(digest, uint64(request.AppSpecGeneration))
	fingerprintUint64(digest, request.ExecutionAttempt)
	fingerprintString(digest, request.SpecHash)
	fingerprintString(digest, request.ActorRef.String())
	fingerprintString(digest, request.ProjectRef.String())
	fingerprintString(digest, request.Objective)
	fingerprintString(digest, request.PhaseRef)
	fingerprintString(digest, request.PhaseKey)
	fingerprintString(digest, request.PhaseTemplateRef)
	fingerprintStrings(digest, request.PhaseInputRefs)
	fingerprintStrings(digest, request.PhaseCriterionRefs)
	fingerprintString(digest, request.RoleKey)
	fingerprintStrings(digest, request.SkillRefs)
	fingerprintStrings(digest, request.ToolRefs)
	fingerprintStrings(digest, request.CapabilityRefs)
	fingerprintStrings(digest, request.WriteSet)
	fingerprintString(digest, request.OutputContract)
	fingerprintString(digest, request.ArtifactMediaType)
	fingerprintString(digest, request.IdempotencyKey)
	fingerprintInt64(digest, request.MaxOutputBytes)
	fingerprintString(digest, request.BudgetDemand.Ref)
	fingerprintInt64(digest, request.BudgetDemand.Resources.Tokens)
	fingerprintInt64(digest, request.BudgetDemand.Resources.MoneyMicros)
	fingerprintString(digest, string(request.BudgetDemand.Resources.Currency))
	fingerprintInt64(digest, request.BudgetDemand.Resources.ActiveTimeNS)
	fingerprintInt64(digest, request.BudgetDemand.Resources.ProcessSlots)
	fingerprintInt64(digest, request.BudgetDemand.Resources.DiskBytes)
	fingerprintString(digest, string(request.SecurityCriticality))
	fingerprintString(digest, string(request.ReasoningEffort))
	fingerprintBool(digest, request.RequierePreservacionEntorno)
	fingerprintString(digest, request.EffectAuthority.AuthorizationReceiptRef)
	fingerprintString(digest, request.EffectAuthority.EffectApprovalRef)
	fingerprintString(digest, request.EffectAuthority.EffectAttemptRef)
	fingerprintUint64(digest, request.EffectAuthority.ActionFence)
	fingerprintTime(digest, request.EffectAuthority.StartedAt)
	fingerprintTime(digest, request.EffectAuthority.ClaimLeaseUntil)
	fingerprintTime(digest, request.EffectAuthority.ApprovalExpiresAt)
	// Empty authority deliberately writes no byte: launches without egress keep
	// the exact pre-egress v1 fingerprint. A present tuple gets its own domain
	// and seals the byte-exact durable authority, not only its parsed semantics.
	if !request.EgressAuthority.IsEmpty() {
		fingerprintString(digest, compilationEgressFingerprintDomain)
		fingerprintString(digest, request.EgressAuthority.PolicyRef)
		fingerprintString(digest, request.EgressAuthority.PayloadSHA256)
		fingerprintString(digest, string(request.EgressAuthority.CanonicalPayload))
	}

	// ProfileBinding preserves declared service order. The sibling descriptor
	// digest sorts services for identity; this seal must still detect aliasing or
	// a crossed composition binding byte-for-byte at the semantic field level.
	descriptor := binding.Descriptor
	fingerprintString(digest, binding.PlacementRef.String())
	fingerprintString(digest, descriptor.Esquema)
	fingerprintString(digest, descriptor.DescriptorRef)
	fingerprintString(digest, descriptor.EjecutorRef)
	fingerprintUint64(digest, uint64(descriptor.VCPU))
	fingerprintUint64(digest, uint64(descriptor.MemoriaMiB))
	fingerprintString(digest, descriptor.KernelSHA256)
	fingerprintString(digest, descriptor.InitramfsSHA256)
	fingerprintString(digest, descriptor.PerfilSHA256)
	fingerprintUint64(digest, uint64(len(descriptor.ServiciosDisponibles)))
	for _, service := range descriptor.ServiciosDisponibles {
		fingerprintString(digest, service.Papel)
		fingerprintString(digest, service.ServicioRef)
		fingerprintUint64(digest, uint64(service.Puerto))
		fingerprintString(digest, service.IdentidadRef)
		fingerprintString(digest, service.IdentidadSHA256)
	}
	var result [sha256.Size]byte
	copy(result[:], digest.Sum(nil))
	return result
}

func fingerprintString(digest hash.Hash, value string) {
	fingerprintUint64(digest, uint64(len(value)))
	_, _ = io.WriteString(digest, value)
}

func fingerprintStrings(digest hash.Hash, values []string) {
	fingerprintUint64(digest, uint64(len(values)))
	for _, value := range values {
		fingerprintString(digest, value)
	}
}

func fingerprintUint64(digest hash.Hash, value uint64) {
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, _ = digest.Write(encoded[:])
}

func fingerprintInt64(digest hash.Hash, value int64) {
	fingerprintUint64(digest, uint64(value))
}

func fingerprintBool(digest hash.Hash, value bool) {
	encoded := byte(0)
	if value {
		encoded = 1
	}
	_, _ = digest.Write([]byte{encoded})
}

func fingerprintTime(digest hash.Hash, value time.Time) {
	if value.IsZero() {
		fingerprintBool(digest, false)
		return
	}
	fingerprintBool(digest, true)
	utc := value.UTC()
	fingerprintInt64(digest, utc.Unix())
	fingerprintUint64(digest, uint64(utc.Nanosecond()))
}

func grantWindow(authority ports.AgentLaunchEffectAuthority) (time.Time, time.Duration, bool) {
	issuedAt := authority.StartedAt.UTC()
	if issuedAt.UnixMilli() <= 0 {
		return time.Time{}, 0, false
	}
	expiresAt := authority.ClaimLeaseUntil.UTC()
	if !authority.ApprovalExpiresAt.IsZero() && authority.ApprovalExpiresAt.UTC().Before(expiresAt) {
		expiresAt = authority.ApprovalExpiresAt.UTC()
	}
	protocolExpiry := issuedAt.Add(microvm.VigenciaMaximaConcesion)
	if protocolExpiry.Before(expiresAt) {
		expiresAt = protocolExpiry
	}
	validity := expiresAt.Sub(issuedAt)
	return issuedAt, validity, validity > 0 && validity <= microvm.VigenciaMaximaConcesion &&
		expiresAt.UnixMilli() > issuedAt.UnixMilli()
}

func completeAccessAuthority(authority ports.AgentLaunchAccessAuthority) bool {
	return authority.ArtifactAccessRef.String() != "" && authority.MCPAccessRef.String() != "" &&
		authority.MailboxEndpointRef.String() != ""
}

func authorizedServices(
	descriptor microvm.DescriptorPerfilLanzamientoV1,
	egressAuthorized bool,
) ([]microvm.ServicioVsock, string) {
	services := make([]microvm.ServicioVsock, 0, len(descriptor.ServiciosDisponibles))
	hasBroker := false
	hasProxy := false
	for _, service := range descriptor.ServiciosDisponibles {
		switch service.Papel {
		case "control_broker":
			hasBroker = true
			services = append(services, service)
		case "controlled_egress_proxy":
			hasProxy = true
			if egressAuthorized {
				services = append(services, service)
			}
		}
	}
	if !hasBroker {
		return nil, CodeControlBrokerRequired
	}
	if !egressAuthorized {
		return services, ""
	}
	if !hasProxy {
		return nil, CodeEgressProxyRequired
	}
	return services, ""
}

const compilationEgressFingerprintDomain = "orquesta.agentmicrovm.egress-authority.v1\x00"

func decodeEgressAuthority(
	authority ports.AgentLaunchEgressAuthority,
) (*microvm.ConcesionEgreso, string, error) {
	if authority.IsEmpty() {
		return nil, "", nil
	}
	grant, err := microvm.DecodificarConcesionEgresoV1(authority.CanonicalPayload)
	if err != nil {
		return nil, CodeEgressAuthorityInvalid, err
	}
	if grant.Referencia != authority.PolicyRef {
		return nil, CodeEgressAuthorityMismatch, fmt.Errorf("egress policy reference mismatch")
	}
	return &grant, "", nil
}

func checkedMultiply(left, right uint64) (uint64, bool) {
	if left != 0 && right > math.MaxUint64/left {
		return 0, false
	}
	return left * right, true
}

func validContextBindings(context microvm.ContextoAutorizado) bool {
	for _, value := range []string{
		context.ProyectoRef, context.GoalRef, context.WorkItemRef, context.EjecucionRef,
		context.AutorizacionRef, context.AprobacionEfectoRef, context.IntentoEfectoRef,
		context.SesionRef, context.AccesoArtefactosRef, context.AccesoMCPRef, context.EndpointBuzonRef,
	} {
		if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\x00\r\n") {
			return false
		}
	}
	return true
}

func fail(code string, cause error) error {
	return &Error{Code: code, Cause: cause}
}
