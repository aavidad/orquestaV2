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
	IntakeStore             IntakeStore
	WizardGapsOutcomes      WizardGapsOutcomeStore
	IntakeDossierStore      IntakeDossierStore
	Access                  AccessRepository
	Launcher                AgentLauncher
	Observer                AgentObserver
	Controller              AgentController
	Artifacts               ArtifactStore
	WorkspaceManager        WorkspaceManager
	VersionControl          VersionControl
	TestAttestor            TestAttestor
	TestAttestationPolicy   TestAttestationPolicy
	Clock                   Clock
	IDs                     IDGenerator
	MaxOutputBytes          int64
	MaxMailboxEnvelopeBytes int64
	MaxExecutionAttempts    uint64
	MaxChildrenPerParent    int
	ClaimLease              time.Duration
	AttestTestClaimLease    time.Duration
	DirectorLeaseDuration   time.Duration
	EffectApprovalTTL       time.Duration
	BudgetPolicy            BudgetPolicy
	ObservationDelay        time.Duration
	ExecutionTimeout        time.Duration
	AgentCapabilities       ports.AgentCapabilities
	ExecutionSessions       ports.ExecutionSessionBroker
	PostArtifactMailbox     PostArtifactMailboxAdmitter
}

type Orchestrator struct {
	state                   StateRepository
	intake                  *IntakeService
	wizardGaps              *WizardGapsService
	intakeDossier           *IntakeDossierService
	access                  AccessRepository
	launcher                AgentLauncher
	observer                AgentObserver
	controller              AgentController
	artifacts               ArtifactStore
	workspaceManager        WorkspaceManager
	versionControl          VersionControl
	testAttestor            TestAttestor
	testAttestationPolicy   TestAttestationPolicy
	clock                   Clock
	ids                     IDGenerator
	maxOutputBytes          int64
	maxMailboxEnvelopeBytes int64
	maxExecutionAttempts    uint64
	maxChildrenPerParent    int
	claimLease              time.Duration
	attestTestClaimLease    time.Duration
	directorLeaseDuration   time.Duration
	budgetPolicy            BudgetPolicy
	observationDelay        time.Duration
	executionTimeout        time.Duration
	agentCapabilities       ports.AgentCapabilities
	executionSessions       ports.ExecutionSessionBroker
	postArtifactMailbox     PostArtifactMailboxAdmitter
}

func New(dependencies Dependencies) (*Orchestrator, error) {
	switch {
	case (dependencies.IntakeStore == nil) !=
		(dependencies.WizardGapsOutcomes == nil):
		return nil, errors.New("application.wizard_gaps_store_composition_invalid")
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
	case dependencies.AttestTestClaimLease < 0:
		return nil, errors.New("application.attest_test_claim_lease_invalid")
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
	case dependencies.TestAttestor == nil && dependencies.TestAttestationPolicy != (TestAttestationPolicy{}):
		return nil, errors.New("application.test_attestor_required")
	case dependencies.TestAttestor != nil && ValidateTestAttestationPolicy(dependencies.TestAttestationPolicy) != nil:
		return nil, errors.New("application.test_attestation_policy_invalid")
	}
	controller := dependencies.Controller
	if controller == nil {
		controller = unsupportedAgentController{}
	}
	var intakeService *IntakeService
	var wizardGapsService *WizardGapsService
	if dependencies.IntakeStore != nil {
		var err error
		intakeService, err = NewIntakeService(dependencies.IntakeStore)
		if err != nil {
			return nil, err
		}
		wizardGapsService, err = NewWizardGapsService(
			intakeService, dependencies.WizardGapsOutcomes,
		)
		if err != nil {
			return nil, err
		}
	}
	var intakeDossierService *IntakeDossierService
	if dependencies.IntakeDossierStore != nil {
		var err error
		intakeDossierService, err = NewIntakeDossierService(
			dependencies.IntakeStore,
			dependencies.IntakeDossierStore,
		)
		if err != nil {
			return nil, err
		}
	}
	return &Orchestrator{
		state:                   dependencies.State,
		intake:                  intakeService,
		wizardGaps:              wizardGapsService,
		intakeDossier:           intakeDossierService,
		access:                  dependencies.Access,
		launcher:                dependencies.Launcher,
		observer:                dependencies.Observer,
		controller:              controller,
		artifacts:               dependencies.Artifacts,
		workspaceManager:        dependencies.WorkspaceManager,
		versionControl:          dependencies.VersionControl,
		testAttestor:            dependencies.TestAttestor,
		testAttestationPolicy:   dependencies.TestAttestationPolicy,
		clock:                   dependencies.Clock,
		ids:                     dependencies.IDs,
		maxOutputBytes:          dependencies.MaxOutputBytes,
		maxMailboxEnvelopeBytes: dependencies.MaxMailboxEnvelopeBytes,
		maxExecutionAttempts:    dependencies.MaxExecutionAttempts,
		maxChildrenPerParent:    dependencies.MaxChildrenPerParent,
		claimLease:              dependencies.ClaimLease,
		attestTestClaimLease:    dependencies.AttestTestClaimLease,
		directorLeaseDuration:   dependencies.DirectorLeaseDuration,
		budgetPolicy:            dependencies.BudgetPolicy,
		observationDelay:        dependencies.ObservationDelay,
		executionTimeout:        dependencies.ExecutionTimeout,
		agentCapabilities:       cloneAgentCapabilities(dependencies.AgentCapabilities),
		executionSessions:       dependencies.ExecutionSessions,
		postArtifactMailbox:     dependencies.PostArtifactMailbox,
	}, nil
}

func cloneAgentCapabilities(source ports.AgentCapabilities) ports.AgentCapabilities {
	source.RoleKeys = append([]string(nil), source.RoleKeys...)
	source.SkillRefs = append([]string(nil), source.SkillRefs...)
	source.ToolRefs = append([]string(nil), source.ToolRefs...)
	source.CapabilityRefs = append([]string(nil), source.CapabilityRefs...)
	return source
}
