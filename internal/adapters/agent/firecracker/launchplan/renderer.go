package launchplan

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"orquesta/internal/ports"
)

const (
	PlanSchema       = "orquesta.agent-firecracker-launch-plan.v1"
	ReceiptStatus    = ports.AgentMicroVMPlannedNotApplied
	RequirementState = ports.AgentMicroVMRequiredBeforeApply
)

var launchRequirements = []string{
	ports.AgentMicroVMRequirementVerifyBundleBytesFromCAS,
	"validate_bundle_traversal_links_devices_and_expansion",
	"enforce_one_physical_microvm_per_agent",
	"cleanup_fail_closed",
	"deliver_verified_bundle_via_allowlisted_vsock",
}

// Options contains only refs expected by composition. It deliberately embeds
// neither adapter configuration nor transport-specific request types.
type Options struct {
	AdapterRef                        string
	NetworkPlanAdapterRef             string
	BundleDescriptorBindingAdapterRef string
}

// RenderRequest composes only neutral ports contracts already produced and
// validated at their adapter boundaries.
type RenderRequest struct {
	NetworkPlan    ports.AgentMicroVMNetworkPlanContract
	InputBundle    ports.AgentMicroVMBundleDescriptor
	IdempotencyKey string
}

type PlanReceipt struct {
	Status                       string
	AdapterRef                   string
	ProjectRef                   string
	GoalRef                      string
	WorkItemRef                  string
	ExecutionRef                 string
	PlanGeneration               uint64
	AppSpecGeneration            uint64
	ExecutionAttempt             uint64
	SpecHash                     string
	AgentRef                     string
	NetworkPlanAdapterRef        string
	NetworkPolicyDigest          string
	NetworkPlanDigest            string
	NetworkPlanReceiptRef        string
	InputArtifactRef             string
	InputArtifactDigest          string
	InputArtifactSize            int64
	InputArtifactMediaType       string
	InputDescriptorBindingDigest string
	InputDescriptorBindingRef    string
	PlanDigest                   string
	IdempotencyKey               string
	ReceiptRef                   string
}

// AgentMicroVMLaunchPlan remains planned_not_applied. Its requirements are
// future gates, not claims of physical execution, byte verification, cleanup
// or delivery.
type AgentMicroVMLaunchPlan struct {
	Document []byte
	Receipt  PlanReceipt
}

type Error struct{ Code string }

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func ErrorCode(err error) string {
	var planErr *Error
	if errors.As(err, &planErr) {
		return planErr.Code
	}
	return ""
}

func Render(options Options, request RenderRequest) (AgentMicroVMLaunchPlan, error) {
	if !validRef(options.AdapterRef) || !validRef(options.NetworkPlanAdapterRef) {
		return AgentMicroVMLaunchPlan{}, planError("options_invalid")
	}
	scope := request.InputBundle.Scope
	if err := ports.ValidateAgentMicroVMBundleDescriptorBinding(
		options.BundleDescriptorBindingAdapterRef,
		scope,
		request.InputBundle,
	); err != nil {
		return AgentMicroVMLaunchPlan{}, planError("input_descriptor_binding_invalid")
	}
	if err := ports.ValidateAgentMicroVMNetworkPlanContract(
		options.NetworkPlanAdapterRef,
		scope,
		request.NetworkPlan,
	); err != nil {
		return AgentMicroVMLaunchPlan{}, planError("network_plan_contract_invalid")
	}
	if !validRef(request.IdempotencyKey) {
		return AgentMicroVMLaunchPlan{}, planError("idempotency_key_invalid")
	}
	document := planProjection(options.AdapterRef, request)
	payload, err := json.Marshal(document)
	if err != nil {
		return AgentMicroVMLaunchPlan{}, planError("render_failed")
	}
	planDigest := digestBytes(payload)
	receipt := planReceiptProjection(options.AdapterRef, request, planDigest)
	return AgentMicroVMLaunchPlan{Document: payload, Receipt: receipt}, nil
}

// ValidateReceipt anchors both producer refs in expected composition. It
// never derives Options from a candidate receipt.
func ValidateReceipt(
	options Options,
	request RenderRequest,
	candidate AgentMicroVMLaunchPlan,
) error {
	if digestBytes(candidate.Document) != candidate.Receipt.PlanDigest {
		return planError("document_digest_mismatch")
	}
	expected, err := Render(options, request)
	if err != nil {
		return err
	}
	if !bytes.Equal(candidate.Document, expected.Document) || candidate.Receipt != expected.Receipt {
		return planError("receipt_mismatch")
	}
	return nil
}

type planDocument struct {
	Schema         string            `json:"schema"`
	Status         string            `json:"status"`
	AdapterRef     string            `json:"adapter_ref"`
	Scope          planScope         `json:"scope"`
	Network        networkDescriptor `json:"network"`
	Input          inputDescriptor   `json:"input"`
	Requirements   []planRequirement `json:"requirements"`
	IdempotencyKey string            `json:"idempotency_key"`
}

type planScope struct {
	ProjectRef        string `json:"project_ref"`
	GoalRef           string `json:"goal_ref"`
	WorkItemRef       string `json:"work_item_ref"`
	ExecutionRef      string `json:"execution_ref"`
	PlanGeneration    uint64 `json:"plan_generation"`
	AppSpecGeneration uint64 `json:"app_spec_generation"`
	ExecutionAttempt  uint64 `json:"execution_attempt"`
	SpecHash          string `json:"spec_hash"`
	AgentRef          string `json:"agent_ref"`
}

type networkDescriptor struct {
	AdapterRef   string `json:"adapter_ref"`
	PolicyDigest string `json:"policy_digest"`
	PlanDigest   string `json:"plan_digest"`
	ReceiptRef   string `json:"receipt_ref"`
}

type inputDescriptor struct {
	ArtifactRef       string `json:"artifact_ref"`
	Digest            string `json:"digest"`
	Size              int64  `json:"size"`
	MediaType         string `json:"media_type"`
	BindingAdapterRef string `json:"descriptor_binding_adapter_ref"`
	BindingDigest     string `json:"descriptor_binding_digest"`
	BindingRef        string `json:"descriptor_binding_ref"`
}

type planRequirement struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

func planProjection(adapterRef string, request RenderRequest) planDocument {
	scope := request.InputBundle.Scope
	requirements := make([]planRequirement, 0, len(launchRequirements))
	for _, name := range launchRequirements {
		requirements = append(requirements, planRequirement{Name: name, State: RequirementState})
	}
	return planDocument{
		Schema: PlanSchema, Status: ReceiptStatus, AdapterRef: adapterRef,
		Scope: planScope{
			ProjectRef: scope.ProjectRef.String(), GoalRef: scope.GoalRef.String(),
			WorkItemRef: scope.WorkItemRef.String(), ExecutionRef: scope.ExecutionRef.String(),
			PlanGeneration:    uint64(scope.PlanGeneration),
			AppSpecGeneration: uint64(scope.AppSpecGeneration),
			ExecutionAttempt:  scope.ExecutionAttempt, SpecHash: scope.SpecHash,
			AgentRef: scope.AgentRef,
		},
		Network: networkDescriptor{
			AdapterRef:   request.NetworkPlan.AdapterRef,
			PolicyDigest: request.NetworkPlan.PolicyDigest,
			PlanDigest:   request.NetworkPlan.PlanDigest,
			ReceiptRef:   request.NetworkPlan.ReceiptRef,
		},
		Input: inputDescriptor{
			ArtifactRef:       request.InputBundle.Artifact.Ref.String(),
			Digest:            request.InputBundle.Artifact.Digest,
			Size:              request.InputBundle.Artifact.Size,
			MediaType:         request.InputBundle.Artifact.MediaType,
			BindingAdapterRef: request.InputBundle.Binding.BindingAdapterRef,
			BindingDigest:     request.InputBundle.Binding.BindingDigest,
			BindingRef:        request.InputBundle.Binding.BindingRef,
		},
		Requirements: requirements, IdempotencyKey: request.IdempotencyKey,
	}
}

func planReceiptProjection(
	adapterRef string,
	request RenderRequest,
	planDigest string,
) PlanReceipt {
	scope := request.InputBundle.Scope
	bundle := request.InputBundle
	receipt := PlanReceipt{
		Status: ReceiptStatus, AdapterRef: adapterRef,
		ProjectRef: scope.ProjectRef.String(), GoalRef: scope.GoalRef.String(),
		WorkItemRef: scope.WorkItemRef.String(), ExecutionRef: scope.ExecutionRef.String(),
		PlanGeneration:    uint64(scope.PlanGeneration),
		AppSpecGeneration: uint64(scope.AppSpecGeneration),
		ExecutionAttempt:  scope.ExecutionAttempt, SpecHash: scope.SpecHash,
		AgentRef:                     scope.AgentRef,
		NetworkPlanAdapterRef:        request.NetworkPlan.AdapterRef,
		NetworkPolicyDigest:          request.NetworkPlan.PolicyDigest,
		NetworkPlanDigest:            request.NetworkPlan.PlanDigest,
		NetworkPlanReceiptRef:        request.NetworkPlan.ReceiptRef,
		InputArtifactRef:             bundle.Artifact.Ref.String(),
		InputArtifactDigest:          bundle.Artifact.Digest,
		InputArtifactSize:            bundle.Artifact.Size,
		InputArtifactMediaType:       bundle.Artifact.MediaType,
		InputDescriptorBindingDigest: bundle.Binding.BindingDigest,
		InputDescriptorBindingRef:    bundle.Binding.BindingRef,
		PlanDigest:                   planDigest, IdempotencyKey: request.IdempotencyKey,
	}
	receipt.ReceiptRef = launchPlanReceiptRef(receipt.AdapterRef, receipt.PlanDigest)
	return receipt
}

func launchPlanReceiptRef(adapterRef string, planDigest string) string {
	document := struct {
		Schema     string `json:"schema"`
		AdapterRef string `json:"adapter_ref"`
		PlanDigest string `json:"plan_digest"`
	}{
		Schema:     "orquesta.agent-microvm-launch-plan-receipt-ref.v1",
		AdapterRef: adapterRef,
		PlanDigest: planDigest,
	}
	payload, err := json.Marshal(document)
	if err != nil {
		panic("agent Firecracker launch plan receipt ref cannot fail: " + err.Error())
	}
	return "agent-microvm-launch-plan-receipt:" + digestBytes(payload)
}

func digestBytes(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

func validRef(value string) bool {
	return value != "" && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "\x00\r\n")
}

func planError(suffix string) error {
	return &Error{Code: "agent_firecracker_launch_plan." + suffix}
}
