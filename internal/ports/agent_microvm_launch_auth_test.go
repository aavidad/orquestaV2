package ports

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func validAgentMicroVMLaunchProofRequest(t *testing.T) AgentMicroVMLaunchProofRequest {
	t.Helper()
	policy := validAgentMicroVMNetworkPolicy(t)
	policyDigest, err := AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	challenge := bytes.Repeat([]byte{0x5a}, agentMicroVMChallengeBytes)
	message, err := AgentMicroVMLaunchProofMessage(
		policy, policyDigest, AgentMicroVMLaunchProofScheme, "challenge:launch:test", challenge,
	)
	if err != nil {
		t.Fatal(err)
	}
	mac := hmac.New(sha256.New, []byte("test-only-launch-key"))
	_, _ = mac.Write(message)
	proof, err := NewAgentMicroVMLaunchProof(mac.Sum(nil))
	if err != nil {
		t.Fatal(err)
	}
	return AgentMicroVMLaunchProofRequest{
		Policy: policy, ExpectedPolicyDigest: policyDigest, Scheme: AgentMicroVMLaunchProofScheme,
		ChallengeRef: "challenge:launch:test", Challenge: challenge, Proof: proof,
		RequestedAt: time.Unix(100, 0).UTC(),
	}
}

func validAgentMicroVMLaunchAuthorizationReceipt(
	request AgentMicroVMLaunchProofRequest,
) AgentMicroVMLaunchAuthorizationReceipt {
	policy := request.Policy
	receipt := AgentMicroVMLaunchAuthorizationReceipt{
		ProjectRef: policy.Scope.ProjectRef.String(), GoalRef: policy.Scope.GoalRef.String(),
		WorkItemRef: policy.Scope.WorkItemRef.String(), ExecutionRef: policy.Scope.ExecutionRef.String(),
		AgentRef: policy.Scope.AgentRef, PolicyRef: policy.Ref,
		PolicyDigest: request.ExpectedPolicyDigest, LaunchIdentityRef: policy.LaunchIdentityRef,
		LaunchBindingDigest:     policy.LaunchBindingDigest,
		LaunchCredentialRef:     policy.LaunchCredential.Ref,
		LaunchCredentialVersion: policy.LaunchCredential.Version,
		LaunchAttestationRef:    policy.LaunchAttestationRef.String(), Scheme: request.Scheme,
		ChallengeRef: request.ChallengeRef, ChallengeDigest: AgentMicroVMChallengeDigest(request.Challenge),
		VerifierRef:             "verifier:agent-microvm-launch",
		CredentialUseRequestRef: "request:credential-use:launch-test",
		AuthorizedAt:            request.RequestedAt.Add(time.Second),
	}
	receipt.ReceiptRef = AgentMicroVMLaunchAuthorizationReceiptRef(receipt)
	return receipt
}

func TestAgentMicroVMLaunchProofContractBindsCredentialAndAttestationWithoutLeakingProof(t *testing.T) {
	request := validAgentMicroVMLaunchProofRequest(t)
	defer request.Proof.Destroy()
	if err := ValidateAgentMicroVMLaunchProofRequest(request); err != nil {
		t.Fatal(err)
	}
	receipt := validAgentMicroVMLaunchAuthorizationReceipt(request)
	if err := ValidateAgentMicroVMLaunchAuthorizationReceipt(request, receipt); err != nil {
		t.Fatal(err)
	}

	payload, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(payload, []byte("test-only-launch-key")) ||
		!bytes.Contains(payload, []byte(`"Proof":"[REDACTED]"`)) {
		t.Fatalf("transient proof was not redacted: %s", payload)
	}
	receiptPayload, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(receiptPayload, []byte("Proof")) || bytes.Contains(receiptPayload, request.Challenge) {
		t.Fatalf("receipt contains proof or raw challenge: %s", receiptPayload)
	}
}

func TestAgentMicroVMLaunchProofRejectsReplayAcrossAttemptAndGoal(t *testing.T) {
	original := validAgentMicroVMLaunchProofRequest(t)
	defer original.Proof.Destroy()
	originalMessage, err := AgentMicroVMLaunchProofMessage(
		original.Policy, original.ExpectedPolicyDigest, original.Scheme,
		original.ChallengeRef, original.Challenge,
	)
	if err != nil {
		t.Fatal(err)
	}
	originalProof := hmacSHA256([]byte("test-only-launch-key"), originalMessage)

	tests := map[string]func(*AgentMicroVMNetworkPolicy){
		"attempt": func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.ExecutionAttempt++ },
		"goal": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.GoalRef, _ = goal.NewGoalRef("goal:replay-target")
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			replayed := original.Policy
			mutate(&replayed)
			replayed.LaunchCredential.OwnerRef = replayed.Scope.ProjectRef.String()
			replayed.LaunchCredential.ScopeRef = AgentMicroVMLaunchCredentialScopeRef(replayed.Scope)
			replayed.LaunchBindingDigest = AgentMicroVMLaunchBindingDigest(
				replayed.Scope, replayed.LaunchIdentityRef,
				replayed.LaunchCredential, replayed.LaunchAttestationRef,
			)
			replayedDigest, digestErr := AgentMicroVMNetworkPolicyDigest(replayed)
			if digestErr != nil {
				t.Fatal(digestErr)
			}
			replayedMessage, messageErr := AgentMicroVMLaunchProofMessage(
				replayed, replayedDigest, original.Scheme, original.ChallengeRef, original.Challenge,
			)
			if messageErr != nil {
				t.Fatal(messageErr)
			}
			if hmac.Equal(originalProof, hmacSHA256([]byte("test-only-launch-key"), replayedMessage)) {
				t.Fatal("proof replay survived causal scope change")
			}
		})
	}
}

func TestAgentMicroVMLaunchProofAndReceiptFailClosed(t *testing.T) {
	tests := map[string]func(*AgentMicroVMLaunchProofRequest){
		"wrong scheme":    func(request *AgentMicroVMLaunchProofRequest) { request.Scheme = "none" },
		"short challenge": func(request *AgentMicroVMLaunchProofRequest) { request.Challenge = []byte("short") },
		"missing time":    func(request *AgentMicroVMLaunchProofRequest) { request.RequestedAt = time.Time{} },
		"wrong policy": func(request *AgentMicroVMLaunchProofRequest) {
			request.ExpectedPolicyDigest = strings.Repeat("0", 64)
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := validAgentMicroVMLaunchProofRequest(t)
			defer request.Proof.Destroy()
			mutate(&request)
			if ValidateAgentMicroVMLaunchProofRequest(request) == nil {
				t.Fatal("unsafe proof request accepted")
			}
		})
	}

	request := validAgentMicroVMLaunchProofRequest(t)
	defer request.Proof.Destroy()
	receipt := validAgentMicroVMLaunchAuthorizationReceipt(request)
	receipt.GoalRef = "goal:other"
	if code := AgentMicroVMNetworkContractErrorCode(
		ValidateAgentMicroVMLaunchAuthorizationReceipt(request, receipt),
	); code != "agent_microvm_network.launch_authorization_receipt_mismatch" {
		t.Fatalf("tampered receipt code = %q", code)
	}
}

func hmacSHA256(key, payload []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}
