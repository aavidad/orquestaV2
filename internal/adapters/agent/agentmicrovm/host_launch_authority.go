package agentmicrovm

import (
	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

const CodeLaunchAuthorityInvalid = "agentmicrovm.launch_authority_invalid"

// BuildHostLaunchAuthorityV1 derives the material-free authority that must be
// persisted before crossing client.Lanzar. Signed hashes and services come
// exclusively from Agente MicroVM's canonical signed-request extractor.
func BuildHostLaunchAuthorityV1(
	request ports.AgentLaunchRequest,
	compiled Compilation,
	signed microvm.SolicitudLanzamiento,
	resolved ResolvedCredentialClaim,
) (ports.MicroVMHostLaunchAuthorityV1, error) {
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, err)
	}
	if err := ports.ValidateAgentLaunchEffectAuthority(request.EffectAuthority); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, err)
	}
	if !validHostLaunchCompilation(request, compiled) {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, nil)
	}
	claim := resolved.OneShotUseRequest()
	if err := credentials.ValidateOneShotUseRequest(claim); err != nil ||
		resolved.placement != request.ReferenciaColocacion ||
		claim.ActorRef != request.ActorRef.String() ||
		claim.OwnerRef.String() != request.ActorRef.String() ||
		claim.ScopeRef.String() != request.ProjectRef.String() {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, err)
	}
	if !validSignedPlan(compiled, signed.Plan) {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, nil)
	}

	extracted, err := microvm.ExtraerAutoridadServiciosHostLanzamientoV1(signed)
	if err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, err)
	}
	if extracted.RunRef != request.ExecutionRef.String() ||
		extracted.Cerca != request.EffectAuthority.ActionFence {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, nil)
	}

	services := make([]ports.MicroVMHostServiceAuthorityV1, len(extracted.Servicios))
	for index, service := range extracted.Servicios {
		var role ports.MicroVMHostServiceRole
		switch service.Papel {
		case string(ports.MicroVMHostServiceControlBroker):
			role = ports.MicroVMHostServiceControlBroker
		case string(ports.MicroVMHostServiceControlledEgressProxy):
			role = ports.MicroVMHostServiceControlledEgressProxy
		default:
			return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, nil)
		}
		services[index] = ports.MicroVMHostServiceAuthorityV1{
			Role: role, ServiceRef: service.ServicioRef, Port: service.Puerto,
			IdentityRef: service.IdentidadRef, IdentitySHA256: service.IdentidadSHA256,
		}
	}

	authority := ports.MicroVMHostLaunchAuthorityV1{
		Key: ports.MicroVMHostLaunchAuthorityKey{
			RunRef: request.ExecutionRef, ActionFence: request.EffectAuthority.ActionFence,
		},
		EffectAttemptRef: request.EffectAuthority.EffectAttemptRef,
		SessionRef:       request.SessionRef,
		OneShotClaim:     claim,
		PlanSHA256:       extracted.PlanSHA256,
		ConcessionSHA256: extracted.ConcesionSHA256,
		Services:         services,
	}
	if err := ports.ValidateMicroVMHostLaunchAuthorityPreparedV1(authority); err != nil {
		return ports.MicroVMHostLaunchAuthorityV1{}, fail(CodeLaunchAuthorityInvalid, err)
	}
	return ports.CloneMicroVMHostLaunchAuthorityV1(authority), nil
}

// BuildMicroVMHostLaunchRuntimeDigestsV1 derives the five immutable launch
// digests from the canonical signed plan already bound to request.
func BuildMicroVMHostLaunchRuntimeDigestsV1(
	request ports.AgentLaunchRequest,
	compiled Compilation,
	signed microvm.SolicitudLanzamiento,
) (ports.MicroVMHostLaunchRuntimeDigestsV1, error) {
	if !validHostLaunchCompilation(request, compiled) || compiled.Plan.PerfilSHA256 == nil || !validSignedPlan(compiled, signed.Plan) {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, fail(CodeLaunchAuthorityInvalid, nil)
	}
	extracted, err := microvm.ExtraerAutoridadServiciosHostLanzamientoV1(signed)
	if err != nil || extracted.RunRef != request.ExecutionRef.String() || extracted.Cerca != request.EffectAuthority.ActionFence {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, fail(CodeLaunchAuthorityInvalid, err)
	}
	value := ports.MicroVMHostLaunchRuntimeDigestsV1{
		Key:        ports.MicroVMHostLaunchAuthorityKey{RunRef: request.ExecutionRef, ActionFence: request.EffectAuthority.ActionFence},
		PlanSHA256: extracted.PlanSHA256, ConcessionSHA256: extracted.ConcesionSHA256,
		KernelSHA256: compiled.Plan.KernelSHA256, InitramfsSHA256: compiled.Plan.InitramfsSHA256,
		ProfileSHA256: *compiled.Plan.PerfilSHA256,
	}
	if ports.ValidateMicroVMHostLaunchRuntimeDigestsV1(value) != nil {
		return ports.MicroVMHostLaunchRuntimeDigestsV1{}, fail(CodeLaunchAuthorityInvalid, nil)
	}
	return value, nil
}

func validHostLaunchCompilation(request ports.AgentLaunchRequest, compiled Compilation) bool {
	context := compiled.Context
	issuedAt, validity, validWindow := grantWindow(request.EffectAuthority)
	return compiled.sourceFingerprint != ([32]byte{}) && validWindow &&
		compiled.IssuedAt.Equal(issuedAt) && compiled.Validity == validity &&
		context.ProyectoRef == request.ProjectRef.String() &&
		context.GoalRef == request.GoalRef.String() &&
		context.WorkItemRef == request.WorkItemRef.String() &&
		context.EjecucionRef == request.ExecutionRef.String() &&
		context.AutorizacionRef == request.EffectAuthority.AuthorizationReceiptRef &&
		context.AprobacionEfectoRef == request.EffectAuthority.EffectApprovalRef &&
		context.IntentoEfectoRef == request.EffectAuthority.EffectAttemptRef &&
		context.SesionRef == request.SessionRef.String() &&
		context.AccesoArtefactosRef == request.AccessAuthority.ArtifactAccessRef.String() &&
		context.AccesoMCPRef == request.AccessAuthority.MCPAccessRef.String() &&
		context.EndpointBuzonRef == request.AccessAuthority.MailboxEndpointRef.String() &&
		context.EspecificacionSHA256 == request.SpecHash
}
