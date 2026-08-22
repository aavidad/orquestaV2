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
	WizardGapsStore         WizardGapsStore
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
	EgressPolicies          EgressPolicyResolver
	ProviderCatalogSources  []ProviderCatalogSource
	CapacitySources         []FuenteCapacidadColocacionAgente
	CapacityObservationWait time.Duration
	AgentLifecycle          *AgentEnvironmentLifecycleComposition
}

type AgentEnvironmentLifecycleComposition struct {
	Store             AgentEnvironmentLifecycleStore
	Physical          ports.AgentEnvironmentLifecycle
	Reconciler        ports.AgentEnvironmentLifecycleReconciler
	BuildPreservation AgentEnvironmentPreservationBuilder
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
	egressPolicies          EgressPolicyResolver
	providerCatalogSources  []normalizedProviderCatalogSource
	capacitySources         []FuenteCapacidadColocacionAgente
	capacityObservationWait time.Duration
	agentLifecycle          *AgentEnvironmentLifecycleService
}

func New(dependencies Dependencies) (*Orchestrator, error) {
	providerCatalogSources, providerCatalogErr := normalizeProviderCatalogSources(dependencies.ProviderCatalogSources)
	if providerCatalogErr != nil {
		return nil, providerCatalogErr
	}
	capacitySources, capacityErr := normalizarFuentesCapacidadColocacion(dependencies.CapacitySources)
	if capacityErr != nil {
		return nil, errors.New("application.agent_capacity_sources_invalid")
	}
	switch {
	case dependencies.IntakeStore != nil && dependencies.WizardGapsStore != nil:
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
	case len(capacitySources) > 0 && dependencies.CapacityObservationWait <= 0:
		return nil, errors.New("application.agent_capacity_observation_wait_invalid")
	case dependencies.TestAttestor == nil && dependencies.TestAttestationPolicy != (TestAttestationPolicy{}):
		return nil, errors.New("application.test_attestor_required")
	case dependencies.TestAttestor != nil && ValidateTestAttestationPolicy(dependencies.TestAttestationPolicy) != nil:
		return nil, errors.New("application.test_attestation_policy_invalid")
	case agentEnvironmentLifecycleDependenciesPartial(dependencies):
		return nil, errors.New("application.agent_environment_lifecycle_composition_invalid")
	}
	controller := dependencies.Controller
	if controller == nil {
		controller = unsupportedAgentController{}
	}
	var intakeService *IntakeService
	var wizardGapsService *WizardGapsService
	effectiveIntakeStore := dependencies.IntakeStore
	if dependencies.WizardGapsStore != nil {
		effectiveIntakeStore = dependencies.WizardGapsStore
	}
	if effectiveIntakeStore != nil {
		var err error
		intakeService, err = NewIntakeService(effectiveIntakeStore)
		if err != nil {
			return nil, err
		}
	}
	if dependencies.WizardGapsStore != nil {
		var err error
		wizardGapsService, err = NewWizardGapsService(
			dependencies.WizardGapsStore,
		)
		if err != nil {
			return nil, err
		}
	}
	var intakeDossierService *IntakeDossierService
	if dependencies.IntakeDossierStore != nil {
		var err error
		intakeDossierService, err = NewIntakeDossierService(
			effectiveIntakeStore,
			dependencies.IntakeDossierStore,
		)
		if err != nil {
			return nil, err
		}
	}
	orchestrator := &Orchestrator{
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
		egressPolicies:          dependencies.EgressPolicies,
		providerCatalogSources:  providerCatalogSources,
		capacitySources:         capacitySources,
		capacityObservationWait: dependencies.CapacityObservationWait,
	}
	if dependencies.AgentLifecycle != nil {
		service, err := NewAgentEnvironmentLifecycleService(AgentEnvironmentLifecycleServiceDependencies{
			Store:             dependencies.AgentLifecycle.Store,
			Physical:          dependencies.AgentLifecycle.Physical,
			Reconciler:        dependencies.AgentLifecycle.Reconciler,
			Clock:             dependencies.Clock,
			BuildNextAction:   orchestrator.BuildAgentEnvironmentLifecycleAction,
			BuildPreservation: dependencies.AgentLifecycle.BuildPreservation,
		})
		if err != nil {
			return nil, errors.New("application.agent_environment_lifecycle_composition_invalid")
		}
		orchestrator.agentLifecycle = service
	}
	return orchestrator, nil
}

func agentEnvironmentLifecycleDependenciesPartial(dependencies Dependencies) bool {
	if dependencies.AgentLifecycle == nil {
		return false
	}
	present := 0
	for _, dependency := range []bool{
		dependencies.AgentLifecycle.Store != nil,
		dependencies.AgentLifecycle.Physical != nil,
		dependencies.AgentLifecycle.Reconciler != nil,
		dependencies.AgentLifecycle.BuildPreservation != nil,
	} {
		if dependency {
			present++
		}
	}
	return present != 4
}

func cloneAgentCapabilities(source ports.AgentCapabilities) ports.AgentCapabilities {
	source.RoleKeys = append([]string(nil), source.RoleKeys...)
	source.SkillRefs = append([]string(nil), source.SkillRefs...)
	source.ToolRefs = append([]string(nil), source.ToolRefs...)
	source.CapabilityRefs = append([]string(nil), source.CapabilityRefs...)
	return source
}
