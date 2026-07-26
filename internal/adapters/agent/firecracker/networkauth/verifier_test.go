package networkauth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"orquesta/internal/credentials"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

var errAttestationDenied = errors.New("attestation denied")

type credentialStoreFake struct {
	credentials.Store
	material []byte
	at       time.Time
	useCount atomic.Int32
	deny     bool
}

func (store *credentialStoreFake) Use(
	_ context.Context,
	request credentials.UseRequest,
	consume func(credentials.Secret) error,
) (credentials.Receipt, error) {
	store.useCount.Add(1)
	if store.deny {
		return credentials.Receipt{}, credentials.NewError(credentials.ErrorRevoked, "credential_ref")
	}
	secret, err := credentials.NewSecret(store.material)
	if err != nil {
		return credentials.Receipt{}, err
	}
	defer secret.Destroy()
	if err := consume(secret); err != nil {
		return credentials.Receipt{}, credentials.WrapError(credentials.ErrorConsumerFailed, "consumer", err)
	}
	return credentials.Receipt{
		CredentialRef: request.CredentialRef, OwnerRef: request.OwnerRef,
		ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: request.Version,
		RequestRef: request.RequestRef, ActorRef: request.ActorRef, Operation: "use",
		OccurredAt: store.at,
	}, nil
}

type attestationVerifierFake struct {
	want      LaunchAttestationCheck
	deny      bool
	callCount atomic.Int32
}

func (verifier *attestationVerifierFake) VerifyLaunchAttestation(
	_ context.Context,
	check LaunchAttestationCheck,
) error {
	verifier.callCount.Add(1)
	if verifier.deny || check != verifier.want {
		return errAttestationDenied
	}
	return nil
}

func validNetworkPolicy(t *testing.T) ports.AgentMicroVMNetworkPolicy {
	t.Helper()
	projectRef, _ := goal.NewProjectRef("project:network-auth")
	goalRef, _ := goal.NewGoalRef("goal:network-auth")
	workItemRef, _ := goal.NewWorkItemRef("work:network-auth")
	executionRef, _ := goal.NewExecutionRef("execution:network-auth")
	attestationRef, _ := goal.NewAttestationRef("attestation:network-auth")
	scope := ports.AgentMicroVMNetworkScope{
		ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		PlanGeneration: 2, AppSpecGeneration: 3, ExecutionAttempt: 1,
		SpecHash: bytesToDigest([]byte("spec")), AgentRef: "agent:codex",
	}
	policy := ports.AgentMicroVMNetworkPolicy{
		Ref: "policy:agent-network:auth", Scope: scope, GuestCID: 44,
		LaunchIdentityRef: "launch-identity:network-auth",
		LaunchCredential: ports.AgentMicroVMLaunchCredentialBinding{
			Ref: "credential:network-auth", OwnerRef: projectRef.String(),
			ScopeRef:   ports.AgentMicroVMLaunchCredentialScopeRef(scope),
			PurposeRef: ports.AgentMicroVMLaunchCredentialPurpose, Version: 1,
		},
		LaunchAttestationRef: attestationRef,
		EgressMode:           ports.AgentMicroVMEgressBrokerOnly,
		Broker: ports.AgentMicroVMVsockService{
			Port: 62001, IdentityRef: "service:orquesta-broker",
			IdentityDigest: bytesToDigest([]byte("broker")),
		},
	}
	policy.LaunchBindingDigest = ports.AgentMicroVMLaunchBindingDigest(
		scope, policy.LaunchIdentityRef, policy.LaunchCredential, policy.LaunchAttestationRef,
	)
	return policy
}

type verifierFixture struct {
	verifier     *Verifier
	challenges   *MemoryChallengeStore
	credentials  *credentialStoreFake
	attestations *attestationVerifierFake
	now          time.Time
	key          []byte
}

func newVerifierFixture(t *testing.T) verifierFixture {
	t.Helper()
	now := time.Unix(500, 0).UTC()
	policy := validNetworkPolicy(t)
	policyDigest, err := ports.AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	challenges, err := NewMemoryChallengeStore(ChallengeStoreOptions{
		Now: func() time.Time { return now }, Random: bytes.NewReader(bytes.Repeat([]byte{0x42}, 4096)),
		TTL: time.Minute, MaxActive: 8,
	})
	if err != nil {
		t.Fatal(err)
	}
	credentialStore := &credentialStoreFake{material: []byte("launch-proof-key"), at: now}
	attestations := &attestationVerifierFake{want: LaunchAttestationCheck{
		AttestationRef: policy.LaunchAttestationRef.String(),
		PolicyDigest:   policyDigest, LaunchBindingDigest: policy.LaunchBindingDigest,
		ProjectRef: policy.Scope.ProjectRef.String(), GoalRef: policy.Scope.GoalRef.String(),
		WorkItemRef: policy.Scope.WorkItemRef.String(), ExecutionRef: policy.Scope.ExecutionRef.String(),
		AgentRef: policy.Scope.AgentRef, ExecutionAttempt: policy.Scope.ExecutionAttempt,
	}}
	verifier, err := New(Config{
		CredentialStore: credentialStore, Attestations: attestations, Challenges: challenges,
		Now: func() time.Time { return now }, VerifierRef: "verifier:agent-firecracker-network",
		CredentialActorRef: "actor:agent-firecracker-network",
	})
	if err != nil {
		t.Fatal(err)
	}
	return verifierFixture{
		verifier: verifier, challenges: challenges, credentials: credentialStore,
		attestations: attestations, now: now, key: append([]byte(nil), credentialStore.material...),
	}
}

func (fixture verifierFixture) proofRequest(t *testing.T) ports.AgentMicroVMLaunchProofRequest {
	t.Helper()
	policy := validNetworkPolicy(t)
	policyDigest, err := ports.AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := fixture.challenges.Issue(context.Background(), ChallengeIssueRequest{
		PolicyDigest: policyDigest, LaunchBindingDigest: policy.LaunchBindingDigest,
		ExecutionRef: policy.Scope.ExecutionRef.String(), AgentRef: policy.Scope.AgentRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	message, err := ports.AgentMicroVMLaunchProofMessage(
		policy, policyDigest, ports.AgentMicroVMLaunchProofScheme, challenge.Ref, challenge.Value,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(message)
	mac := hmac.New(sha256.New, fixture.key)
	_, _ = mac.Write(message)
	proof, err := ports.NewAgentMicroVMLaunchProof(mac.Sum(nil))
	if err != nil {
		t.Fatal(err)
	}
	return ports.AgentMicroVMLaunchProofRequest{
		Policy: policy, ExpectedPolicyDigest: policyDigest, Scheme: ports.AgentMicroVMLaunchProofScheme,
		ChallengeRef: challenge.Ref, Challenge: challenge.Value, Proof: proof,
		RequestedAt: fixture.now.Add(-time.Second),
	}
}

func TestVerifierAuthorizesOnceAndRejectsIdenticalReplay(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	receipt, err := fixture.verifier.Verify(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if err := ports.ValidateAgentMicroVMLaunchAuthorizationReceipt(request, receipt); err != nil {
		t.Fatalf("receipt invalid: %v", err)
	}
	if _, err := fixture.verifier.Verify(context.Background(), request); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_denied" {
		t.Fatalf("identical replay error = %v", err)
	}
}

func TestVerifierDoesNotConsumeChallengeOnBadProof(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	badProof, _ := ports.NewAgentMicroVMLaunchProof(bytes.Repeat([]byte{0xff}, sha256.Size))
	bad := request
	bad.Proof = badProof
	defer bad.Proof.Destroy()
	if _, err := fixture.verifier.Verify(context.Background(), bad); ErrorCode(err) !=
		"agent_firecracker_network_auth.proof_invalid" {
		t.Fatalf("bad proof error = %v", err)
	}
	if _, err := fixture.verifier.Verify(context.Background(), request); err != nil {
		t.Fatalf("valid proof after rejection failed: %v", err)
	}
}

func TestVerifierRejectsAttestationBeforeCredentialUse(t *testing.T) {
	fixture := newVerifierFixture(t)
	fixture.attestations.deny = true
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	if _, err := fixture.verifier.Verify(context.Background(), request); ErrorCode(err) !=
		"agent_firecracker_network_auth.attestation_denied" {
		t.Fatalf("attestation error = %v", err)
	}
	if got := fixture.credentials.useCount.Load(); got != 0 {
		t.Fatalf("credential used before attestation: %d", got)
	}
}

func TestVerifierConcurrentReplayHasSingleWinner(t *testing.T) {
	fixture := newVerifierFixture(t)
	request := fixture.proofRequest(t)
	defer request.Proof.Destroy()
	var wait sync.WaitGroup
	var successes atomic.Int32
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			if _, err := fixture.verifier.Verify(context.Background(), request); err == nil {
				successes.Add(1)
			}
		}()
	}
	wait.Wait()
	if got := successes.Load(); got != 1 {
		t.Fatalf("concurrent winners = %d, want 1", got)
	}
}

func TestMemoryChallengeStoreBindsScopeAndExpires(t *testing.T) {
	now := time.Unix(700, 0).UTC()
	store, err := NewMemoryChallengeStore(ChallengeStoreOptions{
		Now: func() time.Time { return now }, Random: bytes.NewReader(bytes.Repeat([]byte{0x33}, 64)),
		TTL: time.Second, MaxActive: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := ChallengeIssueRequest{
		PolicyDigest:        bytesToDigest([]byte("policy")),
		LaunchBindingDigest: bytesToDigest([]byte("binding")),
		ExecutionRef:        "execution:challenge", AgentRef: "agent:codex",
	}
	challenge, err := store.Issue(context.Background(), issue)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Issue(context.Background(), issue); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_capacity_reached" {
		t.Fatalf("capacity error = %v", err)
	}
	mismatched := ChallengeConsumeRequest{
		Ref: challenge.Ref, Value: challenge.Value, PolicyDigest: issue.PolicyDigest,
		LaunchBindingDigest: issue.LaunchBindingDigest, ExecutionRef: "execution:other",
		AgentRef: issue.AgentRef,
	}
	if err := store.Consume(context.Background(), mismatched); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_binding_mismatch" {
		t.Fatalf("binding error = %v", err)
	}
	now = now.Add(2 * time.Second)
	valid := mismatched
	valid.ExecutionRef = issue.ExecutionRef
	if err := store.Consume(context.Background(), valid); ErrorCode(err) !=
		"agent_firecracker_network_auth.challenge_expired" {
		t.Fatalf("expiry error = %v", err)
	}
}

func bytesToDigest(value []byte) string {
	digest := sha256.Sum256(value)
	return hexString(digest[:])
}

func hexString(value []byte) string {
	const digits = "0123456789abcdef"
	result := make([]byte, len(value)*2)
	for index, octet := range value {
		result[index*2] = digits[octet>>4]
		result[index*2+1] = digits[octet&0x0f]
	}
	return string(result)
}
