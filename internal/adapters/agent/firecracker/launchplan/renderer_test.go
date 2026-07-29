package launchplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func validLaunchPlanFixture(t *testing.T) (Options, RenderRequest) {
	t.Helper()
	projectRef, _ := goal.NewProjectRef("project:launch-plan")
	goalRef, _ := goal.NewGoalRef("goal:launch-plan")
	workItemRef, _ := goal.NewWorkItemRef("work:launch-plan")
	executionRef, _ := goal.NewExecutionRef("execution:launch-plan")
	scope := ports.AgentMicroVMNetworkScope{
		ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef,
		ExecutionRef: executionRef, PlanGeneration: 1, AppSpecGeneration: 1,
		ExecutionAttempt: 3, SpecHash: strings.Repeat("1", 64), AgentRef: "agent:codex",
	}
	networkPlan, err := ports.NewAgentMicroVMNetworkPlanContract(
		"adapter:firecracker-network-plan",
		scope,
		digestText("network-policy"),
		digestText("network-plan:adapter:firecracker-network-plan"),
	)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("bundle descriptor fixture")
	artifactDigest := sha256.Sum256(content)
	artifactDigestText := hex.EncodeToString(artifactDigest[:])
	artifactRef, _ := goal.NewArtifactRef("artifact:sha256:" + artifactDigestText)
	bundleAdapterRef := "adapter:bundle-descriptor"
	bundle, err := ports.NewAgentMicroVMBundleDescriptorBinding(
		bundleAdapterRef,
		ports.StoredArtifact{
			Ref: artifactRef, Digest: artifactDigestText, Size: int64(len(content)),
			MediaType: ports.AgentMicroVMGitBundleMediaType,
		},
		scope,
	)
	if err != nil {
		t.Fatal(err)
	}
	return Options{
			AdapterRef:                        "adapter:firecracker-launch-plan",
			NetworkPlanAdapterRef:             networkPlan.AdapterRef,
			BundleDescriptorBindingAdapterRef: bundleAdapterRef,
		},
		RenderRequest{
			NetworkPlan: networkPlan, InputBundle: bundle,
			IdempotencyKey: "launch-plan:execution:3",
		}
}

func TestRenderComposesOnlyNeutralContractsAsPlannedRequirements(t *testing.T) {
	options, request := validLaunchPlanFixture(t)
	first, err := Render(options, request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Render(options, request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Document, second.Document) || first.Receipt != second.Receipt {
		t.Fatal("launch plan is not deterministic")
	}
	if err := ValidateReceipt(options, request, first); err != nil {
		t.Fatal(err)
	}
	var document planDocument
	if err := json.Unmarshal(first.Document, &document); err != nil {
		t.Fatal(err)
	}
	if document.Schema != PlanSchema || document.Status != ReceiptStatus ||
		document.AdapterRef != options.AdapterRef ||
		document.Network.AdapterRef != request.NetworkPlan.AdapterRef ||
		document.Network.PlanDigest != request.NetworkPlan.PlanDigest ||
		document.Network.ReceiptRef != request.NetworkPlan.ReceiptRef ||
		document.Input.ArtifactRef != request.InputBundle.Artifact.Ref.String() ||
		document.Input.Digest != request.InputBundle.Artifact.Digest ||
		document.Input.Size != request.InputBundle.Artifact.Size ||
		document.Input.MediaType != request.InputBundle.Artifact.MediaType ||
		document.Input.BindingRef != request.InputBundle.Binding.BindingRef {
		t.Fatalf("launch plan binding incomplete: %+v", document)
	}
	if len(document.Requirements) != len(launchRequirements) {
		t.Fatalf("requirements = %+v", document.Requirements)
	}
	foundCASVerification := false
	for _, requirement := range document.Requirements {
		if requirement.State != RequirementState {
			t.Fatalf("requirement asserted as applied: %+v", requirement)
		}
		if requirement.Name == ports.AgentMicroVMRequirementVerifyBundleBytesFromCAS {
			foundCASVerification = true
		}
	}
	if !foundCASVerification {
		t.Fatal("CAS byte verification is not required before apply")
	}
	for _, forbidden := range []string{
		"host_path", "host_mount", `"nic"`, `"ip"`, `"tap"`, `"bridge"`,
		`"nat"`, `"inbound"`, `"east_west"`, `"direct_internet"`,
		`"content_verified"`, `"bytes_sealed"`,
	} {
		if bytes.Contains(first.Document, []byte(forbidden)) {
			t.Fatalf("launch plan exposed forbidden claim %q: %s", forbidden, first.Document)
		}
	}
}

func TestValidateReceiptRejectsJointAdapterConfigAndReceiptSubstitution(t *testing.T) {
	options, request := validLaunchPlanFixture(t)
	original, err := Render(options, request)
	if err != nil {
		t.Fatal(err)
	}
	substituteRequest := request
	substituteRequest.NetworkPlan, err = ports.NewAgentMicroVMNetworkPlanContract(
		"adapter:substitute-network-plan",
		request.InputBundle.Scope,
		request.NetworkPlan.PolicyDigest,
		digestText("network-plan:adapter:substitute-network-plan"),
	)
	if err != nil {
		t.Fatal(err)
	}
	substituteOptions := options
	substituteOptions.NetworkPlanAdapterRef = substituteRequest.NetworkPlan.AdapterRef
	substitute, err := Render(substituteOptions, substituteRequest)
	if err != nil {
		t.Fatal(err)
	}
	if substitute.Receipt.NetworkPlanAdapterRef != substituteOptions.NetworkPlanAdapterRef ||
		substitute.Receipt.NetworkPlanReceiptRef != substituteRequest.NetworkPlan.ReceiptRef {
		t.Fatal("fixture did not jointly substitute adapter, config and receipt")
	}
	if substitute.Receipt.PlanDigest == original.Receipt.PlanDigest {
		t.Fatal("joint network substitution preserved LaunchPlanDigest")
	}
	if code := ErrorCode(ValidateReceipt(options, substituteRequest, substitute)); code !=
		"agent_firecracker_launch_plan.network_plan_contract_invalid" {
		t.Fatalf("joint substitution code = %q", code)
	}
}

func TestRenderRejectsDescriptorBindingTamperAndCausalReplay(t *testing.T) {
	options, request := validLaunchPlanFixture(t)
	tests := map[string]struct {
		mutate func(*RenderRequest)
		code   string
	}{
		"descriptor digest": {
			func(value *RenderRequest) {
				value.InputBundle.Binding.BindingDigest = strings.Repeat("4", 64)
			},
			"agent_firecracker_launch_plan.input_descriptor_binding_invalid",
		},
		"network receipt": {
			func(value *RenderRequest) {
				value.NetworkPlan.ReceiptRef = "agent-microvm-network-plan-receipt:" +
					strings.Repeat("5", 64)
			},
			"agent_firecracker_launch_plan.network_plan_contract_invalid",
		},
		"project replay": {
			func(value *RenderRequest) {
				value.NetworkPlan.Scope.ProjectRef, _ = goal.NewProjectRef("project:replay")
			},
			"agent_firecracker_launch_plan.network_plan_contract_invalid",
		},
		"plan generation replay": {
			func(value *RenderRequest) { value.NetworkPlan.Scope.PlanGeneration++ },
			"agent_firecracker_launch_plan.network_plan_contract_invalid",
		},
		"app spec generation replay": {
			func(value *RenderRequest) { value.NetworkPlan.Scope.AppSpecGeneration++ },
			"agent_firecracker_launch_plan.network_plan_contract_invalid",
		},
		"spec replay": {
			func(value *RenderRequest) { value.NetworkPlan.Scope.SpecHash = strings.Repeat("6", 64) },
			"agent_firecracker_launch_plan.network_plan_contract_invalid",
		},
	}
	for name, testCase := range tests {
		t.Run(name, func(t *testing.T) {
			candidate := request
			testCase.mutate(&candidate)
			if code := ErrorCode(mustRenderError(options, candidate)); code != testCase.code {
				t.Fatalf("code = %q", code)
			}
		})
	}
}

func TestCausalScopeReplayChangesLaunchPlanDigest(t *testing.T) {
	options, request := validLaunchPlanFixture(t)
	baseline, err := Render(options, request)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*ports.AgentMicroVMNetworkScope){
		"project": func(scope *ports.AgentMicroVMNetworkScope) {
			scope.ProjectRef, _ = goal.NewProjectRef("project:other")
		},
		"plan generation": func(scope *ports.AgentMicroVMNetworkScope) { scope.PlanGeneration++ },
		"app spec generation": func(scope *ports.AgentMicroVMNetworkScope) {
			scope.AppSpecGeneration++
		},
		"spec": func(scope *ports.AgentMicroVMNetworkScope) { scope.SpecHash = strings.Repeat("7", 64) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			replayedScope := request.InputBundle.Scope
			mutate(&replayedScope)
			replayedRequest := rebuildLaunchPlanRequest(t, options, request, replayedScope)
			replayed, renderErr := Render(options, replayedRequest)
			if renderErr != nil {
				t.Fatal(renderErr)
			}
			if replayed.Receipt.PlanDigest == baseline.Receipt.PlanDigest {
				t.Fatal("causal replay preserved LaunchPlanDigest")
			}
		})
	}
}

func rebuildLaunchPlanRequest(
	t *testing.T,
	options Options,
	original RenderRequest,
	scope ports.AgentMicroVMNetworkScope,
) RenderRequest {
	t.Helper()
	bundle, err := ports.NewAgentMicroVMBundleDescriptorBinding(
		options.BundleDescriptorBindingAdapterRef,
		original.InputBundle.Artifact,
		scope,
	)
	if err != nil {
		t.Fatal(err)
	}
	networkPlan, err := ports.NewAgentMicroVMNetworkPlanContract(
		options.NetworkPlanAdapterRef,
		scope,
		digestText("network-policy:"+scope.ProjectRef.String()+":"+scope.SpecHash),
		digestText("network-plan:"+scope.ProjectRef.String()+":"+scope.SpecHash+
			":"+strconv.FormatUint(uint64(scope.PlanGeneration), 10)+
			":"+strconv.FormatUint(uint64(scope.AppSpecGeneration), 10)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return RenderRequest{
		NetworkPlan:    networkPlan,
		InputBundle:    bundle,
		IdempotencyKey: original.IdempotencyKey,
	}
}

func mustRenderError(options Options, request RenderRequest) error {
	_, err := Render(options, request)
	return err
}

func digestText(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
