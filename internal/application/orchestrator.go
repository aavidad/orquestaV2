package application

import (
	"errors"
	"time"

	"orquesta/internal/ports"
)

const (
	agentArtifactMediaType  = "text/plain"
	outputAttestationPolicy = "agent_output_present"
)

type Dependencies struct {
	State                   StateRepository
	Access                  AccessRepository
	Launcher                AgentLauncher
	Observer                AgentObserver
	Controller              AgentController
	Artifacts               ArtifactStore
	Clock                   Clock
	IDs                     IDGenerator
	MaxOutputBytes          int64
	MaxMailboxEnvelopeBytes int64
	MaxExecutionAttempts    uint64
	MaxChildrenPerParent    int
	ClaimLease              time.Duration
	DirectorLeaseDuration   time.Duration
	EffectApprovalTTL       time.Duration
	BudgetPolicy            BudgetPolicy
	ObservationDelay        time.Duration
	ExecutionTimeout        time.Duration
	AgentCapabilities       ports.AgentCapabilities
}

type Orchestrator struct {
	state                   StateRepository
	access                  AccessRepository
	launcher                AgentLauncher
	observer                AgentObserver
	controller              AgentController
	artifacts               ArtifactStore
	clock                   Clock
	ids                     IDGenerator
	maxOutputBytes          int64
	maxMailboxEnvelopeBytes int64
	maxExecutionAttempts    uint64
	maxChildrenPerParent    int
	claimLease              time.Duration
	directorLeaseDuration   time.Duration
	budgetPolicy            BudgetPolicy
	observationDelay        time.Duration
	executionTimeout        time.Duration
	agentCapabilities       ports.AgentCapabilities
}

func New(dependencies Dependencies) (*Orchestrator, error) {
	switch {
	case dependencies.State == nil:
		return nil, errors.New("application.state_required")
	case dependencies.Access == nil:
		return nil, errors.New("application.access_required")
	case dependencies.Launcher == nil:
		return nil, errors.New("application.launcher_required")
	case dependencies.Observer == nil:
		return nil, errors.New("application.observer_required")
	case dependencies.Artifacts == nil:
		return nil, errors.New("application.artifacts_required")
	case dependencies.Clock == nil:
		return nil, errors.New("application.clock_required")
	case dependencies.IDs == nil:
		return nil, errors.New("application.ids_required")
	case dependencies.MaxOutputBytes <= 0:
		return nil, errors.New("application.max_output_bytes_invalid")
	case dependencies.MaxMailboxEnvelopeBytes <= 0:
		return nil, errors.New("application.max_mailbox_envelope_bytes_invalid")
	case dependencies.MaxExecutionAttempts == 0:
		return nil, errors.New("application.max_execution_attempts_invalid")
	case dependencies.MaxChildrenPerParent <= 0:
		return nil, errors.New("application.max_children_per_parent_invalid")
	case ports.ValidateAgentCapabilities(dependencies.AgentCapabilities) != nil:
		return nil, errors.New("application.agent_capabilities_invalid")
	case dependencies.ClaimLease <= 0:
		return nil, errors.New("application.claim_lease_invalid")
	case dependencies.DirectorLeaseDuration <= 0:
		return nil, errors.New("application.director_lease_duration_invalid")
	case dependencies.EffectApprovalTTL <= 0 ||
		dependencies.EffectApprovalTTL != dependencies.BudgetPolicy.EffectApprovalTTL:
		return nil, errors.New("application.effect_approval_ttl_invalid")
	case ValidateBudgetPolicy(dependencies.BudgetPolicy) != nil:
		return nil, errors.New("application.budget_policy_invalid")
	case dependencies.ObservationDelay <= 0:
		return nil, errors.New("application.observation_delay_invalid")
	case dependencies.ExecutionTimeout <= 0:
		return nil, errors.New("application.execution_timeout_invalid")
	}
	controller := dependencies.Controller
	if controller == nil {
		controller = unsupportedAgentController{}
	}
	return &Orchestrator{
		state:                   dependencies.State,
		access:                  dependencies.Access,
		launcher:                dependencies.Launcher,
		observer:                dependencies.Observer,
		controller:              controller,
		artifacts:               dependencies.Artifacts,
		clock:                   dependencies.Clock,
		ids:                     dependencies.IDs,
		maxOutputBytes:          dependencies.MaxOutputBytes,
		maxMailboxEnvelopeBytes: dependencies.MaxMailboxEnvelopeBytes,
		maxExecutionAttempts:    dependencies.MaxExecutionAttempts,
		maxChildrenPerParent:    dependencies.MaxChildrenPerParent,
		claimLease:              dependencies.ClaimLease,
		directorLeaseDuration:   dependencies.DirectorLeaseDuration,
		budgetPolicy:            dependencies.BudgetPolicy,
		observationDelay:        dependencies.ObservationDelay,
		executionTimeout:        dependencies.ExecutionTimeout,
		agentCapabilities:       cloneAgentCapabilities(dependencies.AgentCapabilities),
	}, nil
}

func cloneAgentCapabilities(source ports.AgentCapabilities) ports.AgentCapabilities {
	source.RoleKeys = append([]string(nil), source.RoleKeys...)
	source.SkillRefs = append([]string(nil), source.SkillRefs...)
	source.ToolRefs = append([]string(nil), source.ToolRefs...)
	source.CapabilityRefs = append([]string(nil), source.CapabilityRefs...)
	return source
}
