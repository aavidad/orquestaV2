package networkauth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/ports"
)

type LaunchAttestationCheck struct {
	AttestationRef      string
	PolicyDigest        string
	LaunchBindingDigest string
	ProjectRef          string
	GoalRef             string
	WorkItemRef         string
	ExecutionRef        string
	AgentRef            string
	ExecutionAttempt    uint64
}

type LaunchAttestationVerifier interface {
	VerifyLaunchAttestation(context.Context, LaunchAttestationCheck) error
}

type Config struct {
	CredentialStore    credentials.Store
	Attestations       LaunchAttestationVerifier
	Challenges         ChallengeConsumer
	Now                func() time.Time
	VerifierRef        string
	CredentialActorRef string
}

type Verifier struct {
	config Config
}

type Error struct {
	Code string
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var authErr *Error
	if errors.As(err, &authErr) {
		return authErr.Code
	}
	return ""
}

func New(config Config) (*Verifier, error) {
	if config.CredentialStore == nil || config.Attestations == nil || config.Challenges == nil ||
		config.Now == nil || !validRef(config.VerifierRef) || !validRef(config.CredentialActorRef) {
		return nil, authError("config_invalid")
	}
	return &Verifier{config: config}, nil
}

func (verifier *Verifier) Verify(
	ctx context.Context,
	request ports.AgentMicroVMLaunchProofRequest,
) (ports.AgentMicroVMLaunchAuthorizationReceipt, error) {
	if verifier == nil || ctx == nil || ctx.Err() != nil {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("unavailable")
	}
	if err := ports.ValidateAgentMicroVMLaunchProofRequest(request); err != nil {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("request_invalid")
	}
	policy := request.Policy
	check := LaunchAttestationCheck{
		AttestationRef: policy.LaunchAttestationRef.String(),
		PolicyDigest:   request.ExpectedPolicyDigest, LaunchBindingDigest: policy.LaunchBindingDigest,
		ProjectRef: policy.Scope.ProjectRef.String(), GoalRef: policy.Scope.GoalRef.String(),
		WorkItemRef: policy.Scope.WorkItemRef.String(), ExecutionRef: policy.Scope.ExecutionRef.String(),
		AgentRef: policy.Scope.AgentRef, ExecutionAttempt: policy.Scope.ExecutionAttempt,
	}
	if err := verifier.config.Attestations.VerifyLaunchAttestation(ctx, check); err != nil {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("attestation_denied")
	}
	message, err := ports.AgentMicroVMLaunchProofMessage(
		policy, request.ExpectedPolicyDigest, request.Scheme, request.ChallengeRef, request.Challenge,
	)
	if err != nil {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("request_invalid")
	}
	defer clear(message)
	proof := request.Proof.Bytes()
	defer clear(proof)

	useRequest := credentials.UseRequest{
		ActorRef:      verifier.config.CredentialActorRef,
		RequestRef:    credentialUseRequestRef(request),
		CredentialRef: credentials.CredentialRef(policy.LaunchCredential.Ref),
		OwnerRef:      credentials.OwnerRef(policy.LaunchCredential.OwnerRef),
		ScopeRef:      credentials.ScopeRef(policy.LaunchCredential.ScopeRef),
		PurposeRef:    credentials.PurposeRef(policy.LaunchCredential.PurposeRef),
		Version:       credentials.Version(policy.LaunchCredential.Version),
	}
	callbackCalled, proofMatched := false, false
	useReceipt, err := verifier.config.CredentialStore.Use(
		ctx,
		useRequest,
		func(secret credentials.Secret) error {
			callbackCalled = true
			key := secret.Bytes()
			defer clear(key)
			mac := hmac.New(sha256.New, key)
			_, _ = mac.Write(message)
			expected := mac.Sum(nil)
			defer clear(expected)
			proofMatched = hmac.Equal(expected, proof)
			if !proofMatched {
				return authError("proof_invalid")
			}
			return nil
		},
	)
	if err != nil {
		if callbackCalled && !proofMatched {
			return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("proof_invalid")
		}
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("credential_denied")
	}
	if !callbackCalled || !proofMatched || !validCredentialUseReceipt(useRequest, useReceipt) {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("credential_receipt_invalid")
	}
	if err := verifier.config.Challenges.Consume(ctx, ChallengeConsumeRequest{
		Ref: request.ChallengeRef, Value: request.Challenge,
		PolicyDigest: request.ExpectedPolicyDigest, LaunchBindingDigest: policy.LaunchBindingDigest,
		ExecutionRef: policy.Scope.ExecutionRef.String(), AgentRef: policy.Scope.AgentRef,
	}); err != nil {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("challenge_denied")
	}
	authorizedAt := verifier.config.Now().UTC()
	if authorizedAt.IsZero() || authorizedAt.Before(request.RequestedAt) {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("clock_invalid")
	}
	receipt := ports.AgentMicroVMLaunchAuthorizationReceipt{
		ProjectRef: policy.Scope.ProjectRef.String(), GoalRef: policy.Scope.GoalRef.String(),
		WorkItemRef: policy.Scope.WorkItemRef.String(), ExecutionRef: policy.Scope.ExecutionRef.String(),
		AgentRef: policy.Scope.AgentRef, PolicyRef: policy.Ref,
		PolicyDigest: request.ExpectedPolicyDigest, LaunchIdentityRef: policy.LaunchIdentityRef,
		LaunchBindingDigest:     policy.LaunchBindingDigest,
		LaunchCredentialRef:     policy.LaunchCredential.Ref,
		LaunchCredentialVersion: policy.LaunchCredential.Version,
		LaunchAttestationRef:    policy.LaunchAttestationRef.String(), Scheme: request.Scheme,
		ChallengeRef:    request.ChallengeRef,
		ChallengeDigest: ports.AgentMicroVMChallengeDigest(request.Challenge),
		VerifierRef:     verifier.config.VerifierRef, CredentialUseRequestRef: useReceipt.RequestRef,
		AuthorizedAt: authorizedAt,
	}
	receipt.ReceiptRef = ports.AgentMicroVMLaunchAuthorizationReceiptRef(receipt)
	if err := ports.ValidateAgentMicroVMLaunchAuthorizationReceipt(request, receipt); err != nil {
		return ports.AgentMicroVMLaunchAuthorizationReceipt{}, authError("receipt_invalid")
	}
	return receipt, nil
}

func credentialUseRequestRef(request ports.AgentMicroVMLaunchProofRequest) string {
	digest := sha256.New()
	_, _ = digest.Write([]byte(request.ExpectedPolicyDigest))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(request.Policy.LaunchBindingDigest))
	_, _ = digest.Write([]byte{0})
	_, _ = digest.Write([]byte(request.ChallengeRef))
	return "request:agent-microvm-launch-proof:" + hex.EncodeToString(digest.Sum(nil))
}

func validCredentialUseReceipt(request credentials.UseRequest, receipt credentials.Receipt) bool {
	return receipt.CredentialRef == request.CredentialRef && receipt.OwnerRef == request.OwnerRef &&
		receipt.ScopeRef == request.ScopeRef && receipt.PurposeRef == request.PurposeRef &&
		receipt.Version == request.Version && receipt.RequestRef == request.RequestRef &&
		receipt.ActorRef == request.ActorRef && receipt.Operation == "use" && !receipt.OccurredAt.IsZero()
}

func validRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\x00\r\n")
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func authError(suffix string) error {
	return &Error{Code: "agent_firecracker_network_auth." + suffix}
}

var _ ports.AgentMicroVMLaunchProofVerifier = (*Verifier)(nil)
