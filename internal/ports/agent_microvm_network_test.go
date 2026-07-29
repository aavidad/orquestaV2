package ports

import (
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func validAgentMicroVMNetworkPolicy(t *testing.T) AgentMicroVMNetworkPolicy {
	t.Helper()
	projectRef, _ := goal.NewProjectRef("project:network-test")
	goalRef, _ := goal.NewGoalRef("goal:network-test")
	workItemRef, _ := goal.NewWorkItemRef("work:network-test")
	executionRef, _ := goal.NewExecutionRef("execution:network-test")
	scope := AgentMicroVMNetworkScope{
		ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef, ExecutionRef: executionRef,
		PlanGeneration: 3, AppSpecGeneration: 2, ExecutionAttempt: 1,
		SpecHash: strings.Repeat("1", 64), AgentRef: "agent:codex:network-test",
	}
	proxy := AgentMicroVMVsockService{
		Port: 42002, IdentityRef: "service:controlled-egress-proxy",
		IdentityDigest: strings.Repeat("3", 64),
	}
	policy := AgentMicroVMNetworkPolicy{
		Ref: "policy:agent-network:test", Scope: scope, GuestCID: 42,
		LaunchIdentityRef: "launch-identity:network-test",
		LaunchCredential: AgentMicroVMLaunchCredentialBinding{
			Ref:        "credential:launch-network-test",
			OwnerRef:   scope.ProjectRef.String(),
			ScopeRef:   AgentMicroVMLaunchCredentialScopeRef(scope),
			PurposeRef: AgentMicroVMLaunchCredentialPurpose,
			Version:    1,
		},
		LaunchAttestationRef: mustAgentMicroVMAttestationRef(t, "attestation:launch:network-test"),
		EgressMode:           AgentMicroVMEgressControlledProxy,
		Broker: AgentMicroVMVsockService{
			Port: 42001, IdentityRef: "service:orquesta-broker",
			IdentityDigest: strings.Repeat("2", 64),
		},
		ControlledProxy:    &proxy,
		EgressPolicyRef:    "policy:egress:research",
		EgressPolicyDigest: strings.Repeat("4", 64),
	}
	policy.LaunchBindingDigest = AgentMicroVMLaunchBindingDigest(
		scope, policy.LaunchIdentityRef, policy.LaunchCredential, policy.LaunchAttestationRef,
	)
	return policy
}

func mustAgentMicroVMAttestationRef(t *testing.T, value string) goal.AttestationRef {
	t.Helper()
	ref, err := goal.NewAttestationRef(value)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}

func TestAgentMicroVMNetworkPolicyAcceptsOnlyExplicitBrokerAndProxy(t *testing.T) {
	policy := validAgentMicroVMNetworkPolicy(t)
	if err := ValidateAgentMicroVMNetworkPolicy(policy); err != nil {
		t.Fatalf("ValidateAgentMicroVMNetworkPolicy() error = %v", err)
	}
	digest, err := AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil || len(digest) != 64 {
		t.Fatalf("AgentMicroVMNetworkPolicyDigest() = %q, %v", digest, err)
	}

	brokerOnly := policy
	brokerOnly.EgressMode = AgentMicroVMEgressBrokerOnly
	brokerOnly.ControlledProxy = nil
	brokerOnly.EgressPolicyRef = ""
	brokerOnly.EgressPolicyDigest = ""
	if err := ValidateAgentMicroVMNetworkPolicy(brokerOnly); err != nil {
		t.Fatalf("broker-only policy rejected: %v", err)
	}
}

func TestAgentMicroVMNetworkPolicyFailsClosedOnAmbiguousOrUnsafeInputs(t *testing.T) {
	tests := map[string]struct {
		mutate   func(*AgentMicroVMNetworkPolicy)
		wantCode string
	}{
		"empty mode": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.EgressMode = "" },
			"agent_microvm_network.egress_mode_invalid",
		},
		"host CID as guest": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.GuestCID = AgentMicroVMVsockHostCID },
			"agent_microvm_network.guest_cid_invalid",
		},
		"reserved any CID": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.GuestCID = AgentMicroVMReservedAnyCID },
			"agent_microvm_network.guest_cid_invalid",
		},
		"identity detached from launch": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.ExecutionAttempt++ },
			"agent_microvm_network.launch_credential_invalid",
		},
		"identity replayed in another goal": {
			func(policy *AgentMicroVMNetworkPolicy) {
				policy.Scope.GoalRef, _ = goal.NewGoalRef("goal:spoofed")
			},
			"agent_microvm_network.launch_credential_invalid",
		},
		"missing credential ref": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.LaunchCredential.Ref = "" },
			"agent_microvm_network.launch_credential_invalid",
		},
		"credential scope replay": {
			func(policy *AgentMicroVMNetworkPolicy) {
				policy.LaunchCredential.ScopeRef = "scope:agent-microvm-launch:other"
			},
			"agent_microvm_network.launch_credential_invalid",
		},
		"missing attestation ref": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.LaunchAttestationRef = goal.AttestationRef{} },
			"agent_microvm_network.launch_attestation_ref_invalid",
		},
		"missing broker identity": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.Broker.IdentityDigest = "" },
			"agent_microvm_network.broker_invalid",
		},
		"proxy without egress policy": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.EgressPolicyDigest = "" },
			"agent_microvm_network.egress_policy_invalid",
		},
		"proxy shares broker port": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.ControlledProxy.Port = policy.Broker.Port },
			"agent_microvm_network.service_port_collision",
		},
		"broker-only keeps proxy": {
			func(policy *AgentMicroVMNetworkPolicy) { policy.EgressMode = AgentMicroVMEgressBrokerOnly },
			"agent_microvm_network.broker_only_scope_invalid",
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			policy := validAgentMicroVMNetworkPolicy(t)
			testCase.mutate(&policy)
			err := ValidateAgentMicroVMNetworkPolicy(policy)
			if code := AgentMicroVMNetworkContractErrorCode(err); code != testCase.wantCode {
				t.Fatalf("error code = %q, want %q (err=%v)", code, testCase.wantCode, err)
			}
		})
	}
}

func TestAgentMicroVMNetworkPolicyDigestIsDeterministicAndCausal(t *testing.T) {
	baseline := validAgentMicroVMNetworkPolicy(t)
	want, err := AgentMicroVMNetworkPolicyDigest(baseline)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := AgentMicroVMNetworkPolicyDigest(baseline)
	if err != nil || repeated != want {
		t.Fatalf("digest is not deterministic: %q != %q (%v)", repeated, want, err)
	}

	mutations := map[string]func(*AgentMicroVMNetworkPolicy){
		"project": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.ProjectRef, _ = goal.NewProjectRef("project:other")
		},
		"goal": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.GoalRef, _ = goal.NewGoalRef("goal:other")
		},
		"work item": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.WorkItemRef, _ = goal.NewWorkItemRef("work:other")
		},
		"execution": func(policy *AgentMicroVMNetworkPolicy) {
			policy.Scope.ExecutionRef, _ = goal.NewExecutionRef("execution:other")
		},
		"plan generation":     func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.PlanGeneration++ },
		"app spec generation": func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.AppSpecGeneration++ },
		"attempt":             func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.ExecutionAttempt++ },
		"spec":                func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.SpecHash = strings.Repeat("9", 64) },
		"agent":               func(policy *AgentMicroVMNetworkPolicy) { policy.Scope.AgentRef = "agent:other" },
		"guest CID":           func(policy *AgentMicroVMNetworkPolicy) { policy.GuestCID++ },
		"broker port":         func(policy *AgentMicroVMNetworkPolicy) { policy.Broker.Port += 2 },
		"proxy identity": func(policy *AgentMicroVMNetworkPolicy) {
			policy.ControlledProxy.IdentityDigest = strings.Repeat("8", 64)
		},
		"egress policy": func(policy *AgentMicroVMNetworkPolicy) {
			policy.EgressPolicyDigest = strings.Repeat("7", 64)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := validAgentMicroVMNetworkPolicy(t)
			mutate(&candidate)
			candidate.LaunchCredential.OwnerRef = candidate.Scope.ProjectRef.String()
			candidate.LaunchCredential.ScopeRef = AgentMicroVMLaunchCredentialScopeRef(candidate.Scope)
			candidate.LaunchBindingDigest = AgentMicroVMLaunchBindingDigest(
				candidate.Scope, candidate.LaunchIdentityRef,
				candidate.LaunchCredential, candidate.LaunchAttestationRef,
			)
			got, digestErr := AgentMicroVMNetworkPolicyDigest(candidate)
			if digestErr != nil {
				t.Fatalf("valid mutation rejected: %v", digestErr)
			}
			if got == want {
				t.Fatalf("causal mutation did not change policy digest: %s", got)
			}
		})
	}
}

func TestAgentMicroVMNetworkPlanContractBindsAdapterReceiptAndCausalScope(t *testing.T) {
	policy := validAgentMicroVMNetworkPolicy(t)
	policyDigest, err := AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	planDigest := strings.Repeat("d", 64)
	contract, err := NewAgentMicroVMNetworkPlanContract(
		"adapter:firecracker-network-plan",
		policy.Scope,
		policyDigest,
		planDigest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAgentMicroVMNetworkPlanContract(
		contract.AdapterRef,
		policy.Scope,
		contract,
	); err != nil {
		t.Fatal(err)
	}
	substitute, err := NewAgentMicroVMNetworkPlanContract(
		"adapter:substitute",
		policy.Scope,
		policyDigest,
		planDigest,
	)
	if err != nil {
		t.Fatal(err)
	}
	if substitute.ReceiptRef == contract.ReceiptRef {
		t.Fatal("joint adapter and receipt substitution preserved ReceiptRef")
	}
	if code := AgentMicroVMNetworkContractErrorCode(
		ValidateAgentMicroVMNetworkPlanContract(contract.AdapterRef, policy.Scope, substitute),
	); code != "agent_microvm_network.network_plan_adapter_mismatch" {
		t.Fatalf("adapter substitution code = %q", code)
	}
}

func TestAgentMicroVMNetworkPlanContractRejectsProjectGenerationAndSpecReplay(t *testing.T) {
	policy := validAgentMicroVMNetworkPolicy(t)
	policyDigest, err := AgentMicroVMNetworkPolicyDigest(policy)
	if err != nil {
		t.Fatal(err)
	}
	contract, err := NewAgentMicroVMNetworkPlanContract(
		"adapter:firecracker-network-plan",
		policy.Scope,
		policyDigest,
		strings.Repeat("e", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*AgentMicroVMNetworkScope){
		"project": func(scope *AgentMicroVMNetworkScope) {
			scope.ProjectRef, _ = goal.NewProjectRef("project:replay")
		},
		"plan generation": func(scope *AgentMicroVMNetworkScope) { scope.PlanGeneration++ },
		"app spec generation": func(scope *AgentMicroVMNetworkScope) {
			scope.AppSpecGeneration++
		},
		"spec": func(scope *AgentMicroVMNetworkScope) { scope.SpecHash = strings.Repeat("f", 64) },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			replayedScope := policy.Scope
			mutate(&replayedScope)
			if code := AgentMicroVMNetworkContractErrorCode(
				ValidateAgentMicroVMNetworkPlanContract(contract.AdapterRef, replayedScope, contract),
			); code != "agent_microvm_network.network_plan_scope_mismatch" {
				t.Fatalf("scope replay code = %q", code)
			}
		})
	}
}
