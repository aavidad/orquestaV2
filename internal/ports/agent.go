package ports

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"orquesta/internal/goal"
)

type AgentStatus string

const (
	AgentPending   AgentStatus = "pending"
	AgentRunning   AgentStatus = "running"
	AgentCompleted AgentStatus = "completed"
	AgentFailed    AgentStatus = "failed"
)

type AgentCapabilities struct {
	ProviderRef string
}

type AgentLaunchRequest struct {
	ExecutionRef       goal.ExecutionRef
	GoalRef            goal.GoalRef
	WorkItemRef        goal.WorkItemRef
	SpecHash           string
	ActorRef           goal.ActorRef
	ProjectRef         goal.ProjectRef
	Objective          string
	PhaseRef           string
	PhaseKey           string
	PhaseTemplateRef   string
	PhaseInputRefs     []string
	PhaseCriterionRefs []string
	RoleKey            string
	SkillRefs          []string
	ToolRefs           []string
	CapabilityRefs     []string
	WriteSet           []string
	OutputContract     string
	ArtifactMediaType  string
	IdempotencyKey     string
	MaxOutputBytes     int64
}

type AgentLaunchReceipt struct {
	ExecutionRef   goal.ExecutionRef
	SpecHash       string
	ProviderRef    string
	ExternalRef    string
	IdempotencyKey string
	AcceptedAt     time.Time
}

type AgentObservation struct {
	ExecutionRef goal.ExecutionRef
	SpecHash     string
	Status       AgentStatus
	MediaType    string
	Content      []byte
	ErrorCode    string
	ObservedAt   time.Time
}

type AgentContractError struct {
	Code string
}

func (err *AgentContractError) Error() string {
	if err == nil {
		return ""
	}
	return err.Code
}

func AgentContractErrorCode(err error) string {
	var contractErr *AgentContractError
	if errors.As(err, &contractErr) {
		return contractErr.Code
	}
	return ""
}

func ValidateAgentCapabilities(capabilities AgentCapabilities) error {
	if strings.TrimSpace(capabilities.ProviderRef) == "" {
		return &AgentContractError{Code: "agent.provider_ref_required"}
	}
	return nil
}

func ValidateAgentLaunchRequest(request AgentLaunchRequest) error {
	switch {
	case request.ExecutionRef.String() == "":
		return &AgentContractError{Code: "agent.execution_ref_required"}
	case request.GoalRef.String() == "":
		return &AgentContractError{Code: "agent.goal_ref_required"}
	case request.WorkItemRef.String() == "":
		return &AgentContractError{Code: "agent.work_item_ref_required"}
	case request.SpecHash == "":
		return &AgentContractError{Code: "agent.spec_hash_required"}
	case !goal.IsCanonicalAppSpecHash(request.SpecHash):
		return &AgentContractError{Code: "agent.spec_hash_invalid"}
	case request.ActorRef.String() == "":
		return &AgentContractError{Code: "agent.actor_ref_required"}
	case request.ProjectRef.String() == "":
		return &AgentContractError{Code: "agent.project_ref_required"}
	case strings.TrimSpace(request.Objective) == "":
		return &AgentContractError{Code: "agent.objective_required"}
	case !validPhaseRefValue(request.PhaseRef):
		return &AgentContractError{Code: "agent.phase_ref_invalid"}
	case !validPhaseKey(request.PhaseKey):
		return &AgentContractError{Code: "agent.phase_key_invalid"}
	case !validPhaseTemplateRefValue(request.PhaseTemplateRef):
		return &AgentContractError{Code: "agent.phase_template_ref_invalid"}
	case !validInputRefs(request.PhaseInputRefs):
		return &AgentContractError{Code: "agent.phase_input_refs_invalid"}
	case !validCriterionRefs(request.PhaseCriterionRefs):
		return &AgentContractError{Code: "agent.phase_criterion_refs_invalid"}
	case !validRoleKey(request.RoleKey):
		return &AgentContractError{Code: "agent.role_key_invalid"}
	case !validSkillRefs(request.SkillRefs):
		return &AgentContractError{Code: "agent.skill_refs_invalid"}
	case !validToolRefs(request.ToolRefs):
		return &AgentContractError{Code: "agent.tool_refs_invalid"}
	case !validCapabilityRefs(request.CapabilityRefs):
		return &AgentContractError{Code: "agent.capability_refs_invalid"}
	case !validOutputContract(request.OutputContract):
		return &AgentContractError{Code: "agent.output_contract_invalid"}
	case !validWriteSet(request.WriteSet):
		return &AgentContractError{Code: "agent.write_set_invalid"}
	case strings.TrimSpace(request.ArtifactMediaType) == "":
		return &AgentContractError{Code: "agent.artifact_media_type_required"}
	case strings.TrimSpace(request.IdempotencyKey) == "":
		return &AgentContractError{Code: "agent.idempotency_key_required"}
	case request.MaxOutputBytes <= 0:
		return &AgentContractError{Code: "agent.max_output_bytes_invalid"}
	default:
		return nil
	}
}

func validPhaseKey(value string) bool {
	key, err := goal.NewPhaseKey(value)
	return err == nil && key.String() == value
}

func validPhaseRefValue(value string) bool {
	ref, err := goal.NewPhaseRef(value)
	return err == nil && ref.String() == value
}

func validPhaseTemplateRefValue(value string) bool {
	ref, err := goal.NewPhaseTemplateRef(value)
	return err == nil && ref.String() == value
}

func validInputRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewInputRef(value)
		return err == nil && ref.String() == value
	})
}

func validCriterionRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewCriterionRef(value)
		return err == nil && ref.String() == value
	})
}

func validRoleKey(value string) bool {
	key, err := goal.NewRoleKey(value)
	return err == nil && key.String() == value
}

func validSkillRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewSkillRef(value)
		return err == nil && ref.String() == value
	})
}

func validToolRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewToolRef(value)
		return err == nil && ref.String() == value
	})
}

func validCapabilityRefs(values []string) bool {
	return validUniqueAgentRefs(values, func(value string) bool {
		ref, err := goal.NewCapabilityRef(value)
		return err == nil && ref.String() == value
	})
}

func validUniqueAgentRefs(values []string, valid func(string) bool) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !valid(value) {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validOutputContract(value string) bool {
	contract, err := goal.NewOutputContract(goal.OutputContractKind(value))
	return err == nil && string(contract.Kind()) == value
}

func validWriteSet(values []string) bool {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		scope, err := goal.NewWriteScope(value)
		if err != nil || scope.String() != value {
			return false
		}
		if _, duplicate := seen[value]; duplicate {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func ValidateAgentLaunchReceipt(request AgentLaunchRequest, receipt AgentLaunchReceipt) error {
	if receipt.ExecutionRef != request.ExecutionRef {
		return &AgentContractError{Code: "agent.receipt_execution_mismatch"}
	}
	if receipt.SpecHash == "" {
		return &AgentContractError{Code: "agent.receipt_spec_hash_required"}
	}
	if !goal.IsCanonicalAppSpecHash(receipt.SpecHash) {
		return &AgentContractError{Code: "agent.receipt_spec_hash_invalid"}
	}
	if receipt.SpecHash != request.SpecHash {
		return &AgentContractError{Code: "agent.receipt_spec_hash_mismatch"}
	}
	if receipt.IdempotencyKey != request.IdempotencyKey {
		return &AgentContractError{Code: "agent.receipt_idempotency_mismatch"}
	}
	if strings.TrimSpace(receipt.ProviderRef) == "" {
		return &AgentContractError{Code: "agent.receipt_provider_ref_required"}
	}
	if strings.TrimSpace(receipt.ExternalRef) == "" {
		return &AgentContractError{Code: "agent.receipt_external_ref_required"}
	}
	if receipt.AcceptedAt.IsZero() {
		return &AgentContractError{Code: "agent.receipt_accepted_at_required"}
	}
	return nil
}

func ValidateAgentObservation(observation AgentObservation, maxOutputBytes int64) error {
	if observation.ExecutionRef.String() == "" {
		return &AgentContractError{Code: "agent.observation_execution_ref_required"}
	}
	if observation.SpecHash == "" {
		return &AgentContractError{Code: "agent.observation_spec_hash_required"}
	}
	if !goal.IsCanonicalAppSpecHash(observation.SpecHash) {
		return &AgentContractError{Code: "agent.observation_spec_hash_invalid"}
	}
	if observation.ObservedAt.IsZero() {
		return &AgentContractError{Code: "agent.observation_observed_at_required"}
	}
	if int64(len(observation.Content)) > maxOutputBytes {
		return &AgentContractError{Code: "agent.observation_output_too_large"}
	}
	switch observation.Status {
	case AgentPending, AgentRunning:
		if len(observation.Content) > 0 || observation.ErrorCode != "" {
			return &AgentContractError{Code: "agent.observation_nonterminal_payload"}
		}
	case AgentCompleted:
		if len(observation.Content) == 0 {
			return &AgentContractError{Code: "agent.observation_content_required"}
		}
		if strings.TrimSpace(observation.MediaType) == "" {
			return &AgentContractError{Code: "agent.observation_media_type_required"}
		}
		if observation.ErrorCode != "" {
			return &AgentContractError{Code: "agent.observation_terminal_conflict"}
		}
	case AgentFailed:
		if strings.TrimSpace(observation.ErrorCode) == "" {
			return &AgentContractError{Code: "agent.observation_error_code_required"}
		}
		if len(observation.Content) > 0 {
			return &AgentContractError{Code: "agent.observation_terminal_conflict"}
		}
	default:
		return &AgentContractError{Code: fmt.Sprintf("agent.observation_status_invalid:%s", observation.Status)}
	}
	return nil
}
