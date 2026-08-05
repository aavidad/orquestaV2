package agentmicrovm

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

const (
	CodeCredentialClaimResolverInvalid        = "agentmicrovm.credential_claim_resolver_invalid"
	CodeCredentialClaimRequestInvalid         = "agentmicrovm.credential_claim_request_invalid"
	CodeCredentialClaimPlacementUnknown       = "agentmicrovm.credential_claim_placement_unknown"
	CodeCredentialClaimDescriptionUnavailable = "agentmicrovm.credential_claim_description_unavailable"
	CodeCredentialClaimDescriptionInvalid     = "agentmicrovm.credential_claim_description_invalid"

	credentialClaimDescribeDomain = "orquesta.agentmicrovm.credential-claim.describe.v1"
	credentialClaimDescribePrefix = "request:agentmicrovm-credential-claim-describe:sha256:"
)

// CredentialClaimBinding selects one credential by exact opaque placement.
// It carries references only and deliberately has no fallback account.
type CredentialClaimBinding struct {
	PlacementRef  ports.AgentPlacementRef
	CredentialRef credentials.CredentialRef
}

// CredentialClaimResolver pins the currently authorized credential version
// before a physical launch can claim it through UseOnce.
type CredentialClaimResolver struct {
	reader   credentials.UseAuthorityReader
	purpose  credentials.PurposeRef
	bindings map[ports.AgentPlacementRef]credentials.CredentialRef
}

func NewCredentialClaimResolver(
	reader credentials.UseAuthorityReader,
	purpose credentials.PurposeRef,
	bindings []CredentialClaimBinding,
) (*CredentialClaimResolver, error) {
	if nilInterface(reader) || credentials.ValidatePurposeRef(purpose) != nil || len(bindings) == 0 {
		return nil, fail(CodeCredentialClaimResolverInvalid, nil)
	}
	owned := make(map[ports.AgentPlacementRef]credentials.CredentialRef, len(bindings))
	for _, binding := range bindings {
		canonicalPlacement, err := ports.NewAgentPlacementRef(binding.PlacementRef.String())
		if err != nil || canonicalPlacement != binding.PlacementRef ||
			credentials.ValidateCredentialRef(binding.CredentialRef) != nil {
			return nil, fail(CodeCredentialClaimResolverInvalid, err)
		}
		if _, duplicate := owned[binding.PlacementRef]; duplicate {
			return nil, fail(CodeCredentialClaimResolverInvalid, nil)
		}
		owned[binding.PlacementRef] = binding.CredentialRef
	}
	return &CredentialClaimResolver{reader: reader, purpose: purpose, bindings: owned}, nil
}

func (resolver *CredentialClaimResolver) Resolve(
	ctx context.Context,
	request ports.AgentLaunchRequest,
) (credentials.OneShotUseRequest, error) {
	if resolver == nil || nilInterface(resolver.reader) || nilInterface(ctx) {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimResolverInvalid, nil)
	}
	if err := ctx.Err(); err != nil {
		return credentials.OneShotUseRequest{}, err
	}
	if err := ports.ValidateAgentLaunchRequest(request); err != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	if err := ports.ValidateAgentLaunchEffectAuthority(request.EffectAuthority); err != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	placement, err := ports.NewAgentPlacementRef(request.ReferenciaColocacion.String())
	if err != nil || placement != request.ReferenciaColocacion {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	credentialRef, found := resolver.bindings[placement]
	if !found {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimPlacementUnknown, nil)
	}
	actor, err := goal.NewActorRef(request.ActorRef.String())
	if err != nil || actor != request.ActorRef {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	project, err := goal.NewProjectRef(request.ProjectRef.String())
	if err != nil || project != request.ProjectRef {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	ownerRef := credentials.OwnerRef(actor.String())
	scopeRef := credentials.ScopeRef(project.String())
	if credentials.ValidateOwnerRef(ownerRef) != nil || credentials.ValidateScopeRef(scopeRef) != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, nil)
	}
	oneShotRequestRef, err := ports.BuildMicroVMHostLaunchOneShotRequestRefV1(
		ports.MicroVMHostLaunchAuthorityKey{
			RunRef: request.ExecutionRef, ActionFence: request.EffectAuthority.ActionFence,
		},
		request.EffectAuthority.EffectAttemptRef,
		request.SessionRef,
	)
	if err != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	describeRequest := credentials.DescribeUseAuthorityRequest{
		ActorRef: actor.String(),
		RequestRef: buildCredentialClaimDescribeRequestRef(
			oneShotRequestRef, placement.String(), credentialRef.String(),
			actor.String(), project.String(), resolver.purpose.String(),
		),
		CredentialRef: credentialRef,
		OwnerRef:      ownerRef,
		ScopeRef:      scopeRef,
		PurposeRef:    resolver.purpose,
	}
	if err := credentials.ValidateDescribeUseAuthorityRequest(describeRequest); err != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimRequestInvalid, err)
	}
	described, err := resolver.reader.DescribeUseAuthority(ctx, describeRequest)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return credentials.OneShotUseRequest{}, err
		}
		if contextErr := ctx.Err(); contextErr != nil {
			return credentials.OneShotUseRequest{}, contextErr
		}
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimDescriptionUnavailable, err)
	}
	if err := ctx.Err(); err != nil {
		return credentials.OneShotUseRequest{}, err
	}
	if err := credentials.ValidateDescribedUseAuthority(describeRequest, described); err != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimDescriptionInvalid, err)
	}
	claim := credentials.OneShotUseRequest{
		ActorRef: actor.String(), RequestRef: oneShotRequestRef,
		CredentialRef: described.CredentialRef, OwnerRef: described.OwnerRef,
		ScopeRef: described.ScopeRef, PurposeRef: described.PurposeRef, Version: described.Version,
	}
	if err := credentials.ValidateOneShotUseRequest(claim); err != nil {
		return credentials.OneShotUseRequest{}, fail(CodeCredentialClaimDescriptionInvalid, err)
	}
	return claim, nil
}

func buildCredentialClaimDescribeRequestRef(fields ...string) string {
	digest := sha256.New()
	credentialClaimDigestField(digest, credentialClaimDescribeDomain)
	for _, field := range fields {
		credentialClaimDigestField(digest, field)
	}
	return credentialClaimDescribePrefix + hex.EncodeToString(digest.Sum(nil))
}

type credentialClaimDigestWriter interface{ Write([]byte) (int, error) }

func credentialClaimDigestField(digest credentialClaimDigestWriter, field string) {
	_, _ = digest.Write(binary.BigEndian.AppendUint64(nil, uint64(len(field))))
	_, _ = digest.Write([]byte(field))
}
