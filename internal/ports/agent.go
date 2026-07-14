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
	ExecutionRef      goal.ExecutionRef
	GoalRef           goal.GoalRef
	WorkItemRef       goal.WorkItemRef
	ActorRef          goal.ActorRef
	ProjectRef        goal.ProjectRef
	Objective         string
	ArtifactMediaType string
	IdempotencyKey    string
	MaxOutputBytes    int64
}

type AgentLaunchReceipt struct {
	ExecutionRef   goal.ExecutionRef
	ProviderRef    string
	ExternalRef    string
	IdempotencyKey string
	AcceptedAt     time.Time
}

type AgentObservation struct {
	ExecutionRef goal.ExecutionRef
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
	case request.ActorRef.String() == "":
		return &AgentContractError{Code: "agent.actor_ref_required"}
	case request.ProjectRef.String() == "":
		return &AgentContractError{Code: "agent.project_ref_required"}
	case strings.TrimSpace(request.Objective) == "":
		return &AgentContractError{Code: "agent.objective_required"}
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

func ValidateAgentLaunchReceipt(request AgentLaunchRequest, receipt AgentLaunchReceipt) error {
	if receipt.ExecutionRef != request.ExecutionRef {
		return &AgentContractError{Code: "agent.receipt_execution_mismatch"}
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
