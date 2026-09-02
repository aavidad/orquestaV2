package agentmicrovm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	microvm "github.com/aavidad/agente_microvm/conectores/orquesta"

	"orquesta/internal/application"
	"orquesta/internal/ports"
)

func (adapter *Adapter) PrepareExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	binding application.ExpiredAgentLaunchContinuationCausalBindingV41,
) (application.ExpiredAgentLaunchContinuationPreparationV41, error) {
	prepared, err := adapter.prepareExpiredAgentLaunchContinuationV41(ctx, request, binding)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationPreparationV41{}, err
	}
	manifest := prepared.Manifest
	return application.ExpiredAgentLaunchContinuationPreparationV41{
		Binding: binding, ManifestSHA256: prepared.ManifestSHA256,
		RequestKeySHA256: manifest.RequestKeySHA256, OriginalRequestSHA256: manifest.OriginalRequestSHA256,
		AMVLaunchRef: manifest.LaunchRef, AMVExecutionRef: manifest.ExecutionRef, AMVRunRef: manifest.RunRef,
		AMVFence: manifest.Fence, AMVGeneration: manifest.Generation, AMVCID: manifest.CID,
		AMVIdentitySHA256: manifest.IdentitySHA256,
	}, nil
}

func (adapter *Adapter) IssueExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	binding application.ExpiredAgentLaunchContinuationCausalBindingV41,
	issuance application.ExpiredAgentLaunchContinuationIssuanceV41,
) (application.ExpiredAgentLaunchContinuationRecordV41, error) {
	if adapter == nil || adapter.expiredContinuationSigner == nil ||
		issuance.ActorRef == "" || issuance.ProjectRef != binding.ProjectRef ||
		issuance.ExpectedManifestSHA256 == "" || issuance.IssuedAt.IsZero() {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, fail(CodeConfigurationInvalid, nil)
	}
	prepared, err := adapter.prepareExpiredAgentLaunchContinuationV41(ctx, request, binding)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, err
	}
	if prepared.ManifestSHA256 != issuance.ExpectedManifestSHA256 {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, ErrExpiredLaunchContinuationDivergent
	}
	issued, err := adapter.expiredContinuationSigner.Issue(
		ctx, adapter.expiredContinuation, prepared, issuance.ActorRef, issuance.ProjectRef,
		issuance.CredentialRequestRef, issuance.AuthorityRef, issuance.IssuedAt,
	)
	if err != nil {
		return application.ExpiredAgentLaunchContinuationRecordV41{}, err
	}
	admittedUnixMS := uint64(issuance.IssuedAt.UTC().UnixMilli())
	return BuildExpiredAgentLaunchContinuationRecordV41(
		issuance.SubjectRef, issued, issuance.IssuedAt, admittedUnixMS,
	)
}

// prepareExpiredAgentLaunchContinuationV41 reconstructs and proves the exact
// original signed request without preparing a new host authority. The only
// sibling call is the observational V41 preparation endpoint.
func (adapter *Adapter) prepareExpiredAgentLaunchContinuationV41(
	ctx context.Context,
	request ports.AgentLaunchRequest,
	binding application.ExpiredAgentLaunchContinuationCausalBindingV41,
) (PreparedExpiredLaunchContinuationV41, error) {
	if adapter == nil || adapter.expiredContinuation == nil || nilInterface(adapter.signer) ||
		nilInterface(adapter.launchAuthorityRegistry) || ctx == nil ||
		validateExpiredContinuationRequestBinding(request, application.ExpiredAgentLaunchContinuationRecordV41{
			ProjectRef: binding.ProjectRef, GoalRef: binding.GoalRef, WorkItemRef: binding.WorkItemRef,
			ExecutionRef: binding.ExecutionRef, EffectAttemptRef: binding.EffectAttemptRef,
			PlanGeneration: binding.PlanGeneration, ActionFence: binding.ActionFence, AMVFence: binding.ActionFence,
		}) != nil {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeConfigurationInvalid, ErrExpiredLaunchContinuationInvalid)
	}
	if err := ctx.Err(); err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	if err := adapter.validateRequirements(request); err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	key := ports.MicroVMHostLaunchAuthorityKey{RunRef: request.ExecutionRef, ActionFence: request.EffectAuthority.ActionFence}
	historical, err := adapter.launchAuthorityRegistry.Resolve(ctx, key)
	if err != nil || !adapter.validHistoricalLaunchAuthority(request, historical) {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeLaunchAuthorityReplayInvalid, err)
	}
	compiled, err := Compile(request, adapter.profile, request.EffectAuthority.ActionFence)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	signed, err := adapter.signer.Preparar(
		ctx, request, compiled.Context, cloneLaunchPlan(compiled.Plan), compiled.IssuedAt, compiled.Validity,
	)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return PreparedExpiredLaunchContinuationV41{}, err
		}
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeSigningFailed, err)
	}
	if !validSignedPlan(compiled, signed.Plan) {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeLaunchAuthorityReplayInvalid, nil)
	}
	profileJSON, err := json.Marshal(adapter.profile.Descriptor)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeDescriptorInvalid, err)
	}
	signed.Perfil = profileJSON
	signed, ok := cloneSignedRequest(signed)
	if !ok {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeSigningFailed, nil)
	}
	resolved := ResolvedCredentialClaim{placement: request.ReferenciaColocacion, claim: historical.OneShotClaim}
	authority, err := BuildHostLaunchAuthorityV1(request, compiled, signed, resolved)
	if err != nil || !validPreparedLaunchAuthorityReplay(authority, historical) {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeLaunchAuthorityReplayInvalid, err)
	}
	runtimeDigests, err := BuildMicroVMHostLaunchRuntimeDigestsV1(request, compiled, signed)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	historicalDigests, err := adapter.launchAuthorityRegistry.ResolveRuntime(ctx, authority.Key)
	if err != nil || historicalDigests != runtimeDigests {
		return PreparedExpiredLaunchContinuationV41{}, fail(CodeLaunchAuthorityReplayInvalid, err)
	}
	subject := ExpiredLaunchContinuationSubjectV41{
		Binding: binding, IdempotencyKey: request.IdempotencyKey,
		Original: microvm.SolicitudLanzamiento{
			Perfil: append(json.RawMessage(nil), signed.Perfil...), Plan: append(json.RawMessage(nil), signed.Plan...),
			Concesion: append(json.RawMessage(nil), signed.Concesion...),
		},
		ProfileSHA256: bytesSHA256(signed.Perfil), PlanSHA256: bytesSHA256(signed.Plan),
		ConcessionSHA256: bytesSHA256(signed.Concesion),
	}
	prepared, err := adapter.expiredContinuation.PrepareObserved(ctx, subject)
	if err != nil {
		return PreparedExpiredLaunchContinuationV41{}, err
	}
	if !bytes.Equal(prepared.Subject.Original.Perfil, signed.Perfil) ||
		!bytes.Equal(prepared.Subject.Original.Plan, signed.Plan) ||
		!bytes.Equal(prepared.Subject.Original.Concesion, signed.Concesion) {
		return PreparedExpiredLaunchContinuationV41{}, ErrExpiredLaunchContinuationDivergent
	}
	return prepared, nil
}

var _ application.ExpiredAgentLaunchContinuationWriterV41 = (*Adapter)(nil)
