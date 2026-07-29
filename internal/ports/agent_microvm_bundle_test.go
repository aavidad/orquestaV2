package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"orquesta/internal/goal"
)

func validAgentMicroVMBundleDescriptorFixture(t *testing.T) (
	string,
	StoredArtifact,
	AgentMicroVMNetworkScope,
) {
	t.Helper()
	content := []byte("descriptor fixture bytes are not passed to the binding")
	digestBytes := sha256.Sum256(content)
	digest := hex.EncodeToString(digestBytes[:])
	artifactRef, err := goal.NewArtifactRef("artifact:sha256:" + digest)
	if err != nil {
		t.Fatal(err)
	}
	projectRef, _ := goal.NewProjectRef("project:bundle")
	goalRef, _ := goal.NewGoalRef("goal:bundle")
	workItemRef, _ := goal.NewWorkItemRef("work:bundle")
	executionRef, _ := goal.NewExecutionRef("execution:bundle")
	return "adapter:agent-bundle-descriptor",
		StoredArtifact{
			Ref: artifactRef, Digest: digest, Size: int64(len(content)),
			MediaType: AgentMicroVMGitBundleMediaType,
		},
		AgentMicroVMNetworkScope{
			ProjectRef: projectRef, GoalRef: goalRef, WorkItemRef: workItemRef,
			ExecutionRef: executionRef, PlanGeneration: 1, AppSpecGeneration: 1,
			ExecutionAttempt: 2, SpecHash: strings.Repeat("b", 64), AgentRef: "agent:codex",
		}
}

func TestAgentMicroVMBundleDescriptorBindsMetadataWithoutAccreditingContent(t *testing.T) {
	config, artifact, scope := validAgentMicroVMBundleDescriptorFixture(t)
	bundle, err := NewAgentMicroVMBundleDescriptorBinding(config, artifact, scope)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAgentMicroVMBundleDescriptorBinding(config, scope, bundle); err != nil {
		t.Fatal(err)
	}
	binding := bundle.Binding
	if binding.Schema != AgentMicroVMBundleDescriptorBindingSchema ||
		binding.BindingAdapterRef != config ||
		binding.ArtifactRef != artifact.Ref.String() ||
		binding.ArtifactDigest != artifact.Digest ||
		binding.ArtifactSize != artifact.Size ||
		binding.ArtifactMediaType != artifact.MediaType ||
		binding.GoalRef != scope.GoalRef.String() ||
		binding.WorkItemRef != scope.WorkItemRef.String() ||
		binding.ExecutionRef != scope.ExecutionRef.String() ||
		binding.PlanGeneration != uint64(scope.PlanGeneration) ||
		binding.AppSpecGeneration != uint64(scope.AppSpecGeneration) ||
		binding.ExecutionAttempt != scope.ExecutionAttempt ||
		binding.SpecHash != scope.SpecHash ||
		binding.AgentRef != scope.AgentRef ||
		binding.RequiredBeforeApply != AgentMicroVMRequirementVerifyBundleBytesFromCAS ||
		binding.BindingRef != "agent-microvm-bundle-descriptor-binding:"+binding.BindingDigest {
		t.Fatalf("incomplete descriptor binding: %+v", binding)
	}

	// Matching metadata remains only a descriptor: different bytes still fail
	// the independent ArtifactContent contract and therefore cannot be treated
	// as accredited content.
	content := ArtifactContent{
		Ref: artifact.Ref, Digest: artifact.Digest, Size: artifact.Size,
		MediaType: artifact.MediaType, Content: []byte("different bytes"),
	}
	if ValidateArtifactContent(content) == nil {
		t.Fatal("descriptor metadata was incorrectly accepted as content accreditation")
	}
}

func TestAgentMicroVMBundleDescriptorRejectsTamperingAndAdapterSubstitution(t *testing.T) {
	config, artifact, scope := validAgentMicroVMBundleDescriptorFixture(t)
	bundle, err := NewAgentMicroVMBundleDescriptorBinding(config, artifact, scope)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*AgentMicroVMBundleDescriptor){
		"artifact ref": func(value *AgentMicroVMBundleDescriptor) {
			value.Artifact.Ref, _ = goal.NewArtifactRef("artifact:sha256:" + strings.Repeat("c", 64))
		},
		"digest": func(value *AgentMicroVMBundleDescriptor) {
			value.Artifact.Digest = strings.Repeat("c", 64)
		},
		"size": func(value *AgentMicroVMBundleDescriptor) {
			value.Artifact.Size++
		},
		"media type": func(value *AgentMicroVMBundleDescriptor) {
			value.Artifact.MediaType = AgentMicroVMWorkspaceSnapshotMediaType
		},
		"binding digest": func(value *AgentMicroVMBundleDescriptor) {
			value.Binding.BindingDigest = strings.Repeat("d", 64)
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			candidate := bundle
			mutate(&candidate)
			if ValidateAgentMicroVMBundleDescriptorBinding(config, scope, candidate) == nil {
				t.Fatal("tampered descriptor binding accepted")
			}
		})
	}
	if code := AgentMicroVMBundleContractErrorCode(
		ValidateAgentMicroVMBundleDescriptorBinding(
			"adapter:substitute",
			scope,
			bundle,
		),
	); code != "agent_microvm_bundle.binding_mismatch" {
		t.Fatalf("adapter substitution code = %q", code)
	}
}

func TestAgentMicroVMBundleDescriptorRejectsCausalReplay(t *testing.T) {
	config, artifact, scope := validAgentMicroVMBundleDescriptorFixture(t)
	bundle, err := NewAgentMicroVMBundleDescriptorBinding(config, artifact, scope)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func(*AgentMicroVMNetworkScope){
		"project": func(value *AgentMicroVMNetworkScope) {
			value.ProjectRef, _ = goal.NewProjectRef("project:other")
		},
		"plan generation": func(value *AgentMicroVMNetworkScope) { value.PlanGeneration++ },
		"app spec generation": func(value *AgentMicroVMNetworkScope) {
			value.AppSpecGeneration++
		},
		"spec":    func(value *AgentMicroVMNetworkScope) { value.SpecHash = strings.Repeat("c", 64) },
		"attempt": func(value *AgentMicroVMNetworkScope) { value.ExecutionAttempt++ },
		"agent":   func(value *AgentMicroVMNetworkScope) { value.AgentRef = "agent:other" },
		"goal": func(value *AgentMicroVMNetworkScope) {
			value.GoalRef, _ = goal.NewGoalRef("goal:other")
		},
		"work item": func(value *AgentMicroVMNetworkScope) {
			value.WorkItemRef, _ = goal.NewWorkItemRef("work:other")
		},
		"execution": func(value *AgentMicroVMNetworkScope) {
			value.ExecutionRef, _ = goal.NewExecutionRef("execution:other")
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			replayScope := scope
			mutate(&replayScope)
			if code := AgentMicroVMBundleContractErrorCode(
				ValidateAgentMicroVMBundleDescriptorBinding(config, replayScope, bundle),
			); code != "agent_microvm_bundle.scope_mismatch" {
				t.Fatalf("causal replay code = %q", code)
			}
		})
	}
}

func TestAgentMicroVMBundleDescriptorRejectsUnsupportedMediaType(t *testing.T) {
	config, artifact, scope := validAgentMicroVMBundleDescriptorFixture(t)
	artifact.MediaType = "application/octet-stream"
	if _, err := NewAgentMicroVMBundleDescriptorBinding(
		config,
		artifact,
		scope,
	); AgentMicroVMBundleContractErrorCode(err) != "agent_microvm_bundle.artifact_descriptor_invalid" {
		t.Fatalf("unsupported media type error = %v", err)
	}
}
