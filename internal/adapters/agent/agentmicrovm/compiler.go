// Package agentmicrovm translates Orquesta launch authority into the public
// Agente MicroVM contract. It does not own transport, signing or lifecycle.
package agentmicrovm

import (
	"errors"
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
	services, ok := servicesWithoutEgress(descriptor)
	if !ok {
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
	return Compilation{Plan: plan, Context: context, IssuedAt: issuedAt, Validity: validity}, nil
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

func servicesWithoutEgress(descriptor microvm.DescriptorPerfilLanzamientoV1) ([]microvm.ServicioVsock, bool) {
	for _, service := range descriptor.ServiciosDisponibles {
		if service.Papel == "control_broker" {
			return []microvm.ServicioVsock{service}, true
		}
	}
	return nil, false
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
