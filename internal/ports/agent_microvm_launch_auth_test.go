package ports

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	launchPlanDigest := hmacDigestText([]byte("launch-plan"))
	message, err := AgentMicroVMLaunchProofMessage(
		policy, policyDigest, launchPlanDigest,
		AgentMicroVMLaunchProofScheme, "challenge:launch:test", challenge,
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
		Policy: policy, ExpectedPolicyDigest: policyDigest, LaunchPlanDigest: launchPlanDigest,
		Scheme:       AgentMicroVMLaunchProofScheme,
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
		PolicyDigest: request.ExpectedPolicyDigest, LaunchPlanDigest: request.LaunchPlanDigest,
		LaunchIdentityRef:       policy.LaunchIdentityRef,
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

func TestAgentMicroVMLaunchProofV2Golden(t *testing.T) {
	request := validAgentMicroVMLaunchProofRequest(t)
	defer request.Proof.Destroy()
	message, err := AgentMicroVMLaunchProofMessage(
		request.Policy,
		request.ExpectedPolicyDigest,
		request.LaunchPlanDigest,
		request.Scheme,
		request.ChallengeRef,
		request.Challenge,
	)
	if err != nil {
		t.Fatal(err)
	}
	const wantMessage = `{"schema":"hmac-sha256-challenge.v2",` +
		`"policy_digest":"3b1d00726284e8c77078e55b470ca8928c729c43df81d448bdb30d14baf3b57d",` +
		`"launch_plan_digest":"54a9532223630f8d509c4a80a9856365a99c4c4879463c045d2610405c60ecf3",` +
		`"launch_binding_digest":"5c69c392cfbedd9ea8a0241dd2957c400dad0bb4aeff3c577fedb59d34f06d5b",` +
		`"credential_ref":"credential:launch-network-test","credential_version":1,` +
		`"launch_attestation_ref":"attestation:launch:network-test",` +
		`"challenge_ref":"challenge:launch:test",` +
		`"challenge":"WlpaWlpaWlpaWlpaWlpaWlpaWlpaWlpaWlpaWlpaWlo="}`
	const wantHMAC = "13417b339c8c188e476c7d156f9cda37debe5c73a57f0169cce63e266cfe8c86"
	gotHMAC := hex.EncodeToString(hmacSHA256([]byte("test-only-launch-key"), message))
	if string(message) != wantMessage || gotHMAC != wantHMAC {
		t.Fatalf("v2 golden drifted:\nmessage=%q\nhmac=%s", message, gotHMAC)
	}
}

func TestAgentMicroVMLaunchProtocolRejectsV1ProofAndReceipt(t *testing.T) {
	v1ProofRequest := validAgentMicroVMLaunchProofRequest(t)
	defer v1ProofRequest.Proof.Destroy()
	v1ProofRequest.Scheme = "hmac-sha256-challenge.v1"
	if code := AgentMicroVMNetworkContractErrorCode(
		ValidateAgentMicroVMLaunchProofRequest(v1ProofRequest),
	); code != "agent_microvm_network.launch_proof_scheme_invalid" {
		t.Fatalf("v1 proof scheme code = %q", code)
	}

	receiptRequest := validAgentMicroVMLaunchProofRequest(t)
	defer receiptRequest.Proof.Destroy()
	receipt := validAgentMicroVMLaunchAuthorizationReceipt(receiptRequest)
	receipt.ReceiptRef = agentMicroVMLaunchAuthorizationReceiptRef(
		"orquesta.agent-microvm-launch-authorization-receipt.v1",
		receipt,
	)
	if code := AgentMicroVMNetworkContractErrorCode(
		ValidateAgentMicroVMLaunchAuthorizationReceipt(receiptRequest, receipt),
	); code != "agent_microvm_network.launch_authorization_receipt_invalid" {
		t.Fatalf("v1 receipt code = %q", code)
	}
}

func TestAgentMicroVMLaunchProofRejectsReplayAcrossAttemptAndGoal(t *testing.T) {
	original := validAgentMicroVMLaunchProofRequest(t)
	defer original.Proof.Destroy()
	originalMessage, err := AgentMicroVMLaunchProofMessage(
		original.Policy, original.ExpectedPolicyDigest, original.LaunchPlanDigest, original.Scheme,
		original.ChallengeRef, original.Challenge,
	)
	if err != nil {
		t.Fatal(err)
	}
	originalProof := hmacSHA256([]byte("test-only-launch-key"), originalMessage)

	tests := map[string]func(*AgentMicroVMNetworkPolicy){
		"project": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.ProjectRef, _ = goal.NewProjectRef("project:replay-target")
		},
		"plan generation": func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.PlanGeneration++ },
		"app spec generation": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.AppSpecGeneration++
		},
		"spec": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.SpecHash = strings.Repeat("9", 64)
		},
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
				replayed, replayedDigest, original.LaunchPlanDigest,
				original.Scheme, original.ChallengeRef, original.Challenge,
			)
			if messageErr != nil {
				t.Fatal(messageErr)
			}
			if hmac.Equal(originalProof, hmacSHA256([]byte("test-only-launch-key"), replayedMessage)) {
				t.Fatal("proof replay survived causal scope change")
			}
		})
	}

	changedLaunchPlanMessage, err := AgentMicroVMLaunchProofMessage(
		original.Policy,
		original.ExpectedPolicyDigest,
		hmacDigestText([]byte("substituted launch plan")),
		original.Scheme,
		original.ChallengeRef,
		original.Challenge,
	)
	if err != nil {
		t.Fatal(err)
	}
	if hmac.Equal(
		originalProof,
		hmacSHA256([]byte("test-only-launch-key"), changedLaunchPlanMessage),
	) {
		t.Fatal("proof replay survived LaunchPlanDigest substitution")
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
		"missing launch plan": func(request *AgentMicroVMLaunchProofRequest) {
			request.LaunchPlanDigest = ""
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

func hmacDigestText(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
