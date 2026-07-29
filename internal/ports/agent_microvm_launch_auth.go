package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

const (
	AgentMicroVMLaunchProofScheme                = "hmac-sha256-challenge.v2"
	AgentMicroVMLaunchAuthorizationReceiptSchema = "orquesta.agent-microvm-launch-authorization-receipt.v2"
	agentMicroVMChallengeBytes                   = 32
	agentMicroVMProofBytes                       = sha256.Size
)

// AgentMicroVMLaunchProof is callback-scoped proof material, not a durable
// credential. The host authorizer obtains the credential through its
// CredentialStore adapter and compares this proof without crossing that
// adapter boundary with secret material.
type AgentMicroVMLaunchProof struct {
	material []byte
}

func NewAgentMicroVMLaunchProof(material []byte) (AgentMicroVMLaunchProof, error) {
	if len(material) != agentMicroVMProofBytes {
		return AgentMicroVMLaunchProof{}, agentMicroVMNetworkError("launch_proof_invalid")
	}
	return AgentMicroVMLaunchProof{material: append([]byte(nil), material...)}, nil
}

func (proof AgentMicroVMLaunchProof) Bytes() []byte {
	return append([]byte(nil), proof.material...)
}

func (proof *AgentMicroVMLaunchProof) Destroy() {
	if proof != nil {
		clear(proof.material)
		proof.material = nil
	}
}

func (AgentMicroVMLaunchProof) String() string               { return "[REDACTED]" }
func (AgentMicroVMLaunchProof) GoString() string             { return "[REDACTED]" }
func (AgentMicroVMLaunchProof) MarshalJSON() ([]byte, error) { return json.Marshal("[REDACTED]") }

// AgentMicroVMLaunchProofRequest is transient. Raw proof bytes must never be
// logged, persisted or copied into a policy or receipt. Generic JSON
// serialization redacts them.
type AgentMicroVMLaunchProofRequest struct {
	Policy               AgentMicroVMNetworkPolicy
	ExpectedPolicyDigest string
	LaunchPlanDigest     string
	Scheme               string
	ChallengeRef         string
	Challenge            []byte
	Proof                AgentMicroVMLaunchProof
	RequestedAt          time.Time
}

// AgentMicroVMLaunchAuthorizationReceipt is audit evidence for an opening
// committed through AgentMicroVMLaunchAuthorizer. It is never an authority
// token and must not be accepted later to open broker or proxy access. It
// deliberately contains no challenge or proof material. ChallengeDigest
// supports audit without making the nonce reusable.
type AgentMicroVMLaunchAuthorizationReceipt struct {
	ProjectRef              string
	GoalRef                 string
	WorkItemRef             string
	ExecutionRef            string
	AgentRef                string
	PolicyRef               string
	PolicyDigest            string
	LaunchPlanDigest        string
	LaunchIdentityRef       string
	LaunchBindingDigest     string
	LaunchCredentialRef     string
	LaunchCredentialVersion uint64
	LaunchAttestationRef    string
	Scheme                  string
	ChallengeRef            string
	ChallengeDigest         string
	VerifierRef             string
	CredentialUseRequestRef string
	AuthorizedAt            time.Time
	ReceiptRef              string
}

// AgentMicroVMLaunchTransaction prepares one broker or proxy session without
// exposing it until Commit. Open may allocate partial internal resources, but
// Commit is the only operation allowed to make the service externally
// reachable and must do so atomically: an error means it remains unreachable.
// Rollback must clean any internal resource left by Begin, Open or a failed
// Commit and honor its bounded cleanup context.
type AgentMicroVMLaunchTransaction interface {
	Open(context.Context) error
	Commit(context.Context) error
	Rollback(context.Context) error
}

// AgentMicroVMLaunchTransactionFactory starts an isolated opening
// transaction. Begin must not make the service externally reachable and must
// either return a usable transaction or return an error with zero effects.
type AgentMicroVMLaunchTransactionFactory interface {
	Begin(context.Context) (AgentMicroVMLaunchTransaction, error)
}

// AgentMicroVMLaunchAuthorizer is the mandatory gate in front of both the
// Orquesta broker and the controlled egress proxy. It consumes the challenge
// before crossing an attestation or credential ledger, verifies the exact
// launch and proof, and owns the opening transaction through Commit or
// Rollback. The returned receipt is evidence only: possession of it never
// authorizes a later opening.
type AgentMicroVMLaunchAuthorizer interface {
	Authorize(
		context.Context,
		AgentMicroVMLaunchProofRequest,
		AgentMicroVMLaunchTransactionFactory,
	) (AgentMicroVMLaunchAuthorizationReceipt, error)
}

func ValidateAgentMicroVMLaunchProofRequest(request AgentMicroVMLaunchProofRequest) error {
	if _, err := AgentMicroVMLaunchProofMessage(
		request.Policy,
		request.ExpectedPolicyDigest,
		request.LaunchPlanDigest,
		request.Scheme,
		request.ChallengeRef,
		request.Challenge,
	); err != nil {
		return err
	}
	proof := request.Proof.Bytes()
	defer clear(proof)
	if len(proof) != agentMicroVMProofBytes {
		return agentMicroVMNetworkError("launch_proof_invalid")
	}
	if request.RequestedAt.IsZero() {
		return agentMicroVMNetworkError("launch_proof_requested_at_required")
	}
	return nil
}

// AgentMicroVMLaunchProofMessage is the exact HMAC input shared by guest and
// verifier. It binds the single-use challenge to policy, launch identity,
// credential version, attestation, LaunchPlanDigest and the full causal scope.
// LaunchPlanDigest transitively binds the configured network AdapterRef.
func AgentMicroVMLaunchProofMessage(
	policy AgentMicroVMNetworkPolicy,
	expectedPolicyDigest string,
	launchPlanDigest string,
	scheme string,
	challengeRef string,
	challenge []byte,
) ([]byte, error) {
	if err := ValidateAgentMicroVMNetworkPolicy(policy); err != nil {
		return nil, agentMicroVMNetworkError("launch_proof_policy_invalid")
	}
	policyDigest, err := AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil || expectedPolicyDigest != policyDigest {
		return nil, agentMicroVMNetworkError("launch_proof_policy_digest_mismatch")
	}
	if !validWorkspaceDigest(launchPlanDigest) {
		return nil, agentMicroVMNetworkError("launch_proof_launch_plan_digest_invalid")
	}
	if scheme != AgentMicroVMLaunchProofScheme {
		return nil, agentMicroVMNetworkError("launch_proof_scheme_invalid")
	}
	if !validWorkspaceLogicalRef(challengeRef) || len(challenge) != agentMicroVMChallengeBytes {
		return nil, agentMicroVMNetworkError("launch_proof_challenge_invalid")
	}
	document := struct {
		Schema               string `json:"schema"`
		PolicyDigest         string `json:"policy_digest"`
		LaunchPlanDigest     string `json:"launch_plan_digest"`
		LaunchBindingDigest  string `json:"launch_binding_digest"`
		CredentialRef        string `json:"credential_ref"`
		CredentialVersion    uint64 `json:"credential_version"`
		LaunchAttestationRef string `json:"launch_attestation_ref"`
		ChallengeRef         string `json:"challenge_ref"`
		Challenge            []byte `json:"challenge"`
	}{
		Schema: AgentMicroVMLaunchProofScheme, PolicyDigest: policyDigest,
		LaunchPlanDigest:     launchPlanDigest,
		LaunchBindingDigest:  policy.LaunchBindingDigest,
		CredentialRef:        policy.LaunchCredential.Ref,
		CredentialVersion:    policy.LaunchCredential.Version,
		LaunchAttestationRef: policy.LaunchAttestationRef.String(),
		ChallengeRef:         challengeRef, Challenge: append([]byte(nil), challenge...),
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return nil, agentMicroVMNetworkError("launch_proof_message_invalid")
	}
	return payload, nil
}

func AgentMicroVMChallengeDigest(challenge []byte) string {
	digest := sha256.Sum256(challenge)
	return hex.EncodeToString(digest[:])
}

func ValidateAgentMicroVMLaunchAuthorizationReceipt(
	request AgentMicroVMLaunchProofRequest,
	receipt AgentMicroVMLaunchAuthorizationReceipt,
) error {
	if err := ValidateAgentMicroVMLaunchProofRequest(request); err != nil {
		return err
	}
	policy := request.Policy
	checks := []bool{
		receipt.ProjectRef == policy.Scope.ProjectRef.String(),
		receipt.GoalRef == policy.Scope.GoalRef.String(),
		receipt.WorkItemRef == policy.Scope.WorkItemRef.String(),
		receipt.ExecutionRef == policy.Scope.ExecutionRef.String(),
		receipt.AgentRef == policy.Scope.AgentRef,
		receipt.PolicyRef == policy.Ref,
		receipt.PolicyDigest == request.ExpectedPolicyDigest,
		receipt.LaunchPlanDigest == request.LaunchPlanDigest,
		receipt.LaunchIdentityRef == policy.LaunchIdentityRef,
		receipt.LaunchBindingDigest == policy.LaunchBindingDigest,
		receipt.LaunchCredentialRef == policy.LaunchCredential.Ref,
		receipt.LaunchCredentialVersion == policy.LaunchCredential.Version,
		receipt.LaunchAttestationRef == policy.LaunchAttestationRef.String(),
		receipt.Scheme == AgentMicroVMLaunchProofScheme,
		receipt.ChallengeRef == request.ChallengeRef,
		receipt.ChallengeDigest == AgentMicroVMChallengeDigest(request.Challenge),
	}
	for _, matches := range checks {
		if !matches {
			return agentMicroVMNetworkError("launch_authorization_receipt_mismatch")
		}
	}
	if !validWorkspaceLogicalRef(receipt.VerifierRef) ||
		!validWorkspaceLogicalRef(receipt.CredentialUseRequestRef) ||
		!strings.HasPrefix(receipt.CredentialUseRequestRef, "request:") ||
		receipt.AuthorizedAt.IsZero() || receipt.AuthorizedAt.Before(request.RequestedAt) ||
		receipt.ReceiptRef != AgentMicroVMLaunchAuthorizationReceiptRef(receipt) {
		return agentMicroVMNetworkError("launch_authorization_receipt_invalid")
	}
	return nil
}

func AgentMicroVMLaunchAuthorizationReceiptRef(
	receipt AgentMicroVMLaunchAuthorizationReceipt,
) string {
	return agentMicroVMLaunchAuthorizationReceiptRef(
		AgentMicroVMLaunchAuthorizationReceiptSchema,
		receipt,
	)
}

func agentMicroVMLaunchAuthorizationReceiptRef(
	schema string,
	receipt AgentMicroVMLaunchAuthorizationReceipt,
) string {
	document := struct {
		Schema                  string `json:"schema"`
		ProjectRef              string `json:"project_ref"`
		GoalRef                 string `json:"goal_ref"`
		WorkItemRef             string `json:"work_item_ref"`
		ExecutionRef            string `json:"execution_ref"`
		AgentRef                string `json:"agent_ref"`
		PolicyRef               string `json:"policy_ref"`
		PolicyDigest            string `json:"policy_digest"`
		LaunchPlanDigest        string `json:"launch_plan_digest"`
		LaunchIdentityRef       string `json:"launch_identity_ref"`
		LaunchBindingDigest     string `json:"launch_binding_digest"`
		LaunchCredentialRef     string `json:"launch_credential_ref"`
		LaunchCredentialVersion uint64 `json:"launch_credential_version"`
		LaunchAttestationRef    string `json:"launch_attestation_ref"`
		Scheme                  string `json:"scheme"`
		ChallengeRef            string `json:"challenge_ref"`
		ChallengeDigest         string `json:"challenge_digest"`
		VerifierRef             string `json:"verifier_ref"`
		CredentialUseRequestRef string `json:"credential_use_request_ref"`
		AuthorizedAt            string `json:"authorized_at"`
	}{
		Schema:     schema,
		ProjectRef: receipt.ProjectRef, GoalRef: receipt.GoalRef,
		WorkItemRef: receipt.WorkItemRef, ExecutionRef: receipt.ExecutionRef,
		AgentRef: receipt.AgentRef, PolicyRef: receipt.PolicyRef, PolicyDigest: receipt.PolicyDigest,
		LaunchPlanDigest:  receipt.LaunchPlanDigest,
		LaunchIdentityRef: receipt.LaunchIdentityRef, LaunchBindingDigest: receipt.LaunchBindingDigest,
		LaunchCredentialRef:     receipt.LaunchCredentialRef,
		LaunchCredentialVersion: receipt.LaunchCredentialVersion,
		LaunchAttestationRef:    receipt.LaunchAttestationRef, Scheme: receipt.Scheme,
		ChallengeRef: receipt.ChallengeRef, ChallengeDigest: receipt.ChallengeDigest,
		VerifierRef: receipt.VerifierRef, CredentialUseRequestRef: receipt.CredentialUseRequestRef,
		AuthorizedAt: receipt.AuthorizedAt.UTC().Format(time.RFC3339Nano),
	}
	return "agent-microvm-launch-authorization-receipt:" + agentMicroVMNetworkDocumentDigest(document)
}
