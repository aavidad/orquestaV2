package ports

import "errors"

const (
	AgentMicroVMBundleDescriptorBindingSchema = "orquesta.agent-microvm-bundle-descriptor-binding.v1"
	AgentMicroVMGitBundleMediaType            = "application/vnd.git.bundle"
	AgentMicroVMWorkspaceSnapshotMediaType    = "application/vnd.orquesta.workspace-snapshot.v1+tar"

	// AgentMicroVMRequirementVerifyBundleBytesFromCAS remains mandatory until
	// an adapter reads ArtifactContent from CAS and validates its digest and
	// size. A StoredArtifact descriptor alone never satisfies this requirement.
	AgentMicroVMRequirementVerifyBundleBytesFromCAS = "verify_bundle_bytes_from_cas"
)

// AgentMicroVMBundleDescriptorBinding binds StoredArtifact metadata to one
// causal launch scope. It is deliberately neither a receipt nor a seal of
// artifact bytes: content accreditation requires ArtifactContent validation.
type AgentMicroVMBundleDescriptorBinding struct {
	Schema              string
	BindingAdapterRef   string
	ArtifactRef         string
	ArtifactDigest      string
	ArtifactSize        int64
	ArtifactMediaType   string
	ProjectRef          string
	GoalRef             string
	WorkItemRef         string
	ExecutionRef        string
	PlanGeneration      uint64
	AppSpecGeneration   uint64
	ExecutionAttempt    uint64
	SpecHash            string
	AgentRef            string
	BindingDigest       string
	BindingRef          string
	RequiredBeforeApply string
}

// AgentMicroVMBundleDescriptor contains no path, mount, inline content or
// assertion that CAS bytes exist. The launch plan must preserve
// verify_bundle_bytes_from_cas as required_before_apply.
type AgentMicroVMBundleDescriptor struct {
	Artifact StoredArtifact
	Scope    AgentMicroVMNetworkScope
	Binding  AgentMicroVMBundleDescriptorBinding
}

type AgentMicroVMBundleContractError struct{ Code string }

func (err *AgentMicroVMBundleContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentMicroVMBundleContractErrorCode(err error) string {
	var contractErr *AgentMicroVMBundleContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

// NewAgentMicroVMBundleDescriptorBinding binds an immutable descriptor. It
// intentionally accepts no byte content and therefore cannot accredit content.
func NewAgentMicroVMBundleDescriptorBinding(
	bindingAdapterRef string,
	artifact StoredArtifact,
	scope AgentMicroVMNetworkScope,
) (AgentMicroVMBundleDescriptor, error) {
	if !validWorkspaceLogicalRef(bindingAdapterRef) {
		return AgentMicroVMBundleDescriptor{}, agentMicroVMBundleError("binding_adapter_invalid")
	}
	if !validAgentMicroVMNetworkScope(scope) {
		return AgentMicroVMBundleDescriptor{}, agentMicroVMBundleError("scope_invalid")
	}
	if !validAgentMicroVMBundleArtifactDescriptor(artifact) {
		return AgentMicroVMBundleDescriptor{}, agentMicroVMBundleError("artifact_descriptor_invalid")
	}
	binding := agentMicroVMBundleDescriptorBindingProjection(bindingAdapterRef, artifact, scope)
	binding.BindingDigest = agentMicroVMNetworkDocumentDigest(
		agentMicroVMBundleDescriptorBindingDocumentProjection(binding),
	)
	binding.BindingRef = "agent-microvm-bundle-descriptor-binding:" + binding.BindingDigest
	return AgentMicroVMBundleDescriptor{Artifact: artifact, Scope: scope, Binding: binding}, nil
}

func ValidateAgentMicroVMBundleDescriptorBinding(
	expectedBindingAdapterRef string,
	expectedScope AgentMicroVMNetworkScope,
	bundle AgentMicroVMBundleDescriptor,
) error {
	if !validWorkspaceLogicalRef(expectedBindingAdapterRef) {
		return agentMicroVMBundleError("expected_binding_adapter_invalid")
	}
	if !validAgentMicroVMNetworkScope(expectedScope) {
		return agentMicroVMBundleError("scope_invalid")
	}
	if bundle.Scope != expectedScope {
		return agentMicroVMBundleError("scope_mismatch")
	}
	expected, err := NewAgentMicroVMBundleDescriptorBinding(
		expectedBindingAdapterRef,
		bundle.Artifact,
		bundle.Scope,
	)
	if err != nil {
		return err
	}
	if bundle.Binding != expected.Binding {
		return agentMicroVMBundleError("binding_mismatch")
	}
	return nil
}

func validAgentMicroVMBundleArtifactDescriptor(artifact StoredArtifact) bool {
	if artifact.Ref.String() != artifactSHA256RefPrefix+artifact.Digest ||
		!validWorkspaceDigest(artifact.Digest) ||
		artifact.Size <= 0 {
		return false
	}
	switch artifact.MediaType {
	case AgentMicroVMGitBundleMediaType, AgentMicroVMWorkspaceSnapshotMediaType:
		return true
	default:
		return false
	}
}

func agentMicroVMBundleDescriptorBindingProjection(
	adapterRef string,
	artifact StoredArtifact,
	scope AgentMicroVMNetworkScope,
) AgentMicroVMBundleDescriptorBinding {
	return AgentMicroVMBundleDescriptorBinding{
		Schema: AgentMicroVMBundleDescriptorBindingSchema, BindingAdapterRef: adapterRef,
		ArtifactRef: artifact.Ref.String(), ArtifactDigest: artifact.Digest,
		ArtifactSize: artifact.Size, ArtifactMediaType: artifact.MediaType,
		ProjectRef: scope.ProjectRef.String(), GoalRef: scope.GoalRef.String(),
		WorkItemRef: scope.WorkItemRef.String(), ExecutionRef: scope.ExecutionRef.String(),
		PlanGeneration:    uint64(scope.PlanGeneration),
		AppSpecGeneration: uint64(scope.AppSpecGeneration),
		ExecutionAttempt:  scope.ExecutionAttempt, SpecHash: scope.SpecHash,
		AgentRef:            scope.AgentRef,
		RequiredBeforeApply: AgentMicroVMRequirementVerifyBundleBytesFromCAS,
	}
}

func agentMicroVMBundleDescriptorBindingDocumentProjection(
	binding AgentMicroVMBundleDescriptorBinding,
) agentMicroVMBundleDescriptorBindingDocument {
	return agentMicroVMBundleDescriptorBindingDocument{
		Schema: binding.Schema, BindingAdapterRef: binding.BindingAdapterRef,
		ArtifactRef: binding.ArtifactRef, ArtifactDigest: binding.ArtifactDigest,
		ArtifactSize: binding.ArtifactSize, ArtifactMediaType: binding.ArtifactMediaType,
		ProjectRef: binding.ProjectRef, GoalRef: binding.GoalRef,
		WorkItemRef: binding.WorkItemRef, ExecutionRef: binding.ExecutionRef,
		PlanGeneration: binding.PlanGeneration, AppSpecGeneration: binding.AppSpecGeneration,
		ExecutionAttempt: binding.ExecutionAttempt, SpecHash: binding.SpecHash,
		AgentRef: binding.AgentRef, RequiredBeforeApply: binding.RequiredBeforeApply,
	}
}

type agentMicroVMBundleDescriptorBindingDocument struct {
	Schema              string `json:"schema"`
	BindingAdapterRef   string `json:"binding_adapter_ref"`
	ArtifactRef         string `json:"artifact_ref"`
	ArtifactDigest      string `json:"artifact_digest"`
	ArtifactSize        int64  `json:"artifact_size"`
	ArtifactMediaType   string `json:"artifact_media_type"`
	ProjectRef          string `json:"project_ref"`
	GoalRef             string `json:"goal_ref"`
	WorkItemRef         string `json:"work_item_ref"`
	ExecutionRef        string `json:"execution_ref"`
	PlanGeneration      uint64 `json:"plan_generation"`
	AppSpecGeneration   uint64 `json:"app_spec_generation"`
	ExecutionAttempt    uint64 `json:"execution_attempt"`
	SpecHash            string `json:"spec_hash"`
	AgentRef            string `json:"agent_ref"`
	RequiredBeforeApply string `json:"required_before_apply"`
}

func agentMicroVMBundleError(suffix string) error {
	return &AgentMicroVMBundleContractError{Code: "agent_microvm_bundle." + suffix}
}
