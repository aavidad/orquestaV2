package commands

import (
	"context"
	"encoding/json"
	"errors"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/identity"
	"orquesta/internal/intake"
	"orquesta/internal/ports"
)

type applicationAPI interface {
	Submit(context.Context, application.Access, application.SubmitRequest) (application.SubmitResult, error)
	Amend(context.Context, application.Access, application.AmendRequest) (application.AmendResult, error)
	GetGoal(context.Context, application.Access, goal.GoalRef) (application.GoalRecord, error)
	ListGoals(context.Context, application.Access, int) ([]application.GoalSummary, error)
	GetArtifact(context.Context, application.Access, goal.GoalRef, goal.ArtifactRef) (ports.ArtifactContent, error)
	Status(context.Context, application.Access) (application.RepositoryStatus, error)
	GrantMembership(context.Context, application.Access, identity.MembershipGrantRequest, identity.Principal) (identity.Membership, identity.MembershipAuditReceipt, bool, error)
	RevokeMembership(context.Context, application.Access, identity.MembershipRevokeRequest) (identity.Membership, identity.MembershipAuditReceipt, bool, error)
	ClaimDirector(context.Context, application.Access, application.ClaimDirectorRequest) (application.DirectorLeaseResult, error)
	RenewDirector(context.Context, application.Access, application.RenewDirectorRequest) (application.DirectorLeaseResult, error)
	ProposeDirectorPlan(context.Context, application.Access, application.ProposeDirectorPlanRequest) (application.DirectorPlanResult, error)
	Control(context.Context, application.Access, application.ControlRequest) (application.ControlResult, error)
	DecideEffect(context.Context, application.Access, application.DecideEffectRequest) (application.DecideEffectResult, error)
	ListPendingChanges(context.Context, application.Access, application.ListPendingChangesRequest) (application.ListPendingChangesResult, error)
	IntegrateChange(context.Context, application.Access, application.IntegrateChangeRequest) (application.IntegrateChangeResult, error)
	AdmitMailbox(context.Context, application.Access, application.AdmitMailboxRequest) (application.MailboxAdmissionResult, error)
	ClaimMailbox(context.Context, application.Access, application.ClaimMailboxRequest) (application.MailboxClaimResult, error)
	MarkMailboxDelivered(context.Context, application.Access, application.MarkMailboxDeliveredRequest) (application.MailboxMutationResult, error)
	ConsumeMailbox(context.Context, application.Access, application.ConsumeMailboxRequest) (application.MailboxMutationResult, error)
	GetMailbox(context.Context, application.Access, application.GetMailboxRequest) (application.MailboxRecord, error)
	ListMailbox(context.Context, application.Access, application.ListMailboxRequest) (application.ListMailboxResult, error)
	AcknowledgeMailbox(context.Context, application.Access, application.AcknowledgeMailboxRequest) (application.MailboxResolutionResult, error)
	BlockMailbox(context.Context, application.Access, application.BlockMailboxRequest) (application.MailboxResolutionResult, error)
	OpenCouncilRound(context.Context, application.Access, application.OpenCouncilRoundRequest) (application.OpenCouncilRoundResult, error)
	SkipCouncil(context.Context, application.Access, application.SkipCouncilRequest) (application.SkipCouncilResult, error)
	CreateIntake(context.Context, application.Access, application.CreateIntakeRequest) (application.IntakeResult, error)
	GetIntake(context.Context, application.Access, application.GetIntakeRequest) (application.IntakeRecord, error)
	ApplyIntake(context.Context, application.Access, application.ApplyIntakeRequest) (application.IntakeResult, error)
	ApplyWizardGaps(context.Context, application.Access, application.ApplyWizardGapsRequest) (application.ApplyWizardGapsResult, error)
	AcceptIntakeRecommendations(context.Context, application.Access, application.AcceptIntakeRecommendationsRequest) (application.IntakeResult, error)
	GetIntakeContext(context.Context, application.Access, application.GetIntakeContextRequest) (intake.Context, error)
	PrepareIntakeDossier(context.Context, application.Access, application.PrepareIntakeDossierRequest) (application.IntakeDossierResult, error)
	PrepareWizardDossier(context.Context, application.Access, application.PrepareWizardDossierRequest) (application.WizardDossierResult, error)
	GetIntakeDossier(context.Context, application.Access, application.GetIntakeDossierRequest) (application.IntakeDossierRecord, error)
	ConfirmIntakeDossier(context.Context, application.Access, application.ConfirmIntakeDossierRequest) (application.ConfirmIntakeDossierResult, error)
}

var _ applicationAPI = (*application.Orchestrator)(nil)

var expectedHandlerPermissions = map[string]string{
	"Submit": "goals.create", "Amend": "goals.amend", "GetGoal": "goals.get", "ListGoals": "goals.list",
	"GetArtifact": "artifacts.read", "Status": "project.status", "GrantMembership": "project.membership.manage",
	"RevokeMembership": "project.membership.manage", "ClaimDirector": "goals.direct", "RenewDirector": "goals.direct",
	"ProposeDirectorPlan": "goals.direct", "Control": "goals.direct", "DecideEffect": "effects.approve",
	"ListPendingChanges": "goals.list", "IntegrateChange": "changes.integrate", "AdmitMailbox": "goals.direct",
	"ClaimMailbox": "goals.get", "MarkMailboxDelivered": "goals.get", "ConsumeMailbox": "goals.get",
	"GetMailbox": "goals.get", "ListMailbox": "goals.get", "AcknowledgeMailbox": "goals.get", "BlockMailbox": "goals.get",
	"OpenCouncilRound": "goals.direct", "SkipCouncil": "council.skip",
	"CreateIntake": "goals.create", "GetIntake": "goals.get", "ApplyIntake": "goals.create",
	"ApplyWizardGaps":             "goals.create",
	"AcceptIntakeRecommendations": "goals.create", "GetIntakeContext": "goals.get",
	"PrepareIntakeDossier": "goals.create", "PrepareWizardDossier": "goals.create",
	"GetIntakeDossier": "goals.get", "ConfirmIntakeDossier": "goals.create",
}

func (dispatcher *Dispatcher) applicationHandlers() map[string]handler {
	wrap := func(call func(context.Context, applicationAPI, handlerContext, json.RawMessage) (json.RawMessage, error)) handler {
		return func(ctx context.Context, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
			return call(ctx, dispatcher.application, bound, payload)
		}
	}
	return map[string]handler{
		"Submit": wrap(handleSubmit), "Amend": wrap(handleAmend), "GetGoal": wrap(handleGetGoal),
		"ListGoals": wrap(handleListGoals), "GetArtifact": wrap(handleGetArtifact), "Status": wrap(handleStatus),
		"GrantMembership": wrap(handleGrantMembership), "RevokeMembership": wrap(handleRevokeMembership),
		"ClaimDirector": wrap(handleClaimDirector), "RenewDirector": wrap(handleRenewDirector),
		"ProposeDirectorPlan": wrap(handleProposeDirectorPlan), "Control": wrap(handleControl),
		"DecideEffect": wrap(handleDecideEffect), "ListPendingChanges": wrap(handleListPendingChanges),
		"IntegrateChange": wrap(handleIntegrateChange), "AdmitMailbox": wrap(handleAdmitMailbox),
		"ClaimMailbox": wrap(handleClaimMailbox), "MarkMailboxDelivered": wrap(handleMarkMailboxDelivered),
		"ConsumeMailbox": wrap(handleConsumeMailbox), "GetMailbox": wrap(handleGetMailbox),
		"ListMailbox": wrap(handleListMailbox), "AcknowledgeMailbox": wrap(handleAcknowledgeMailbox),
		"BlockMailbox": wrap(handleBlockMailbox), "OpenCouncilRound": wrap(handleOpenCouncilRound),
		"SkipCouncil": wrap(handleSkipCouncil),
		"CreateIntake": func(ctx context.Context, bound handlerContext, payload json.RawMessage) (json.RawMessage, error) {
			return handleCreateIntake(ctx, dispatcher.application, bound, payload, dispatcher.intakePolicy)
		},
		"GetIntake": wrap(handleGetIntake), "ApplyIntake": wrap(handleApplyIntake),
		"ApplyWizardGaps":             wrap(handleApplyWizardGaps),
		"AcceptIntakeRecommendations": wrap(handleAcceptIntakeRecommendations),
		"GetIntakeContext":            wrap(handleGetIntakeContext),
		"PrepareIntakeDossier":        wrap(handlePrepareIntakeDossier),
		"PrepareWizardDossier":        wrap(handlePrepareWizardDossier),
		"GetIntakeDossier":            wrap(handleGetIntakeDossier),
		"ConfirmIntakeDossier":        wrap(handleConfirmIntakeDossier),
	}
}

type planInput struct {
	Phases    []phaseInput    `json:"phases"`
	WorkItems []workItemInput `json:"work_items"`
}

type phaseInput struct {
	Ref           string   `json:"ref"`
	Key           string   `json:"key"`
	TemplateRef   string   `json:"template_ref"`
	InputRefs     []string `json:"input_refs,omitempty"`
	CriterionRefs []string `json:"criterion_refs,omitempty"`
}

type workItemInput struct {
	Key                 string              `json:"key"`
	Objective           string              `json:"objective"`
	Phase               string              `json:"phase"`
	Role                string              `json:"role"`
	Parent              string              `json:"parent,omitempty"`
	HandoffRequired     bool                `json:"handoff_required,omitempty"`
	Dependencies        []string            `json:"dependencies"`
	WriteSet            []string            `json:"write_set"`
	RequiredTests       []requiredTestInput `json:"required_tests,omitempty"`
	SkillRefs           []string            `json:"skill_refs,omitempty"`
	ToolRefs            []string            `json:"tool_refs,omitempty"`
	CapabilityRefs      []string            `json:"capability_refs,omitempty"`
	EgressPolicyRef     string              `json:"egress_policy_ref,omitempty"`
	OutputContract      string              `json:"output_contract"`
	SecurityCriticality string              `json:"security_criticality,omitempty"`
	ReasoningEffort     string              `json:"reasoning_effort,omitempty"`
	CouncilPolicy       string              `json:"council_policy,omitempty"`
}

type requiredTestInput struct {
	Ref              string   `json:"ref"`
	ToolRef          string   `json:"tool_ref"`
	Arguments        []string `json:"arguments"`
	WorkingDirectory string   `json:"working_directory"`
}

func applicationPlan(input *planInput) *application.PlanSpec {
	if input == nil {
		return nil
	}
	result := &application.PlanSpec{Phases: make([]application.PhaseSpec, 0, len(input.Phases)), WorkItems: make([]application.WorkItemSpec, 0, len(input.WorkItems))}
	for _, phase := range input.Phases {
		result.Phases = append(result.Phases, application.PhaseSpec{Ref: phase.Ref, Key: phase.Key, TemplateRef: phase.TemplateRef, InputRefs: append([]string(nil), phase.InputRefs...), CriterionRefs: append([]string(nil), phase.CriterionRefs...)})
	}
	for _, item := range input.WorkItems {
		tests := make([]application.RequiredTestSpec, 0, len(item.RequiredTests))
		for _, test := range item.RequiredTests {
			tests = append(tests, application.RequiredTestSpec{Ref: test.Ref, ToolRef: test.ToolRef, Arguments: append([]string(nil), test.Arguments...), WorkingDirectory: test.WorkingDirectory})
		}
		result.WorkItems = append(result.WorkItems, application.WorkItemSpec{
			Key: item.Key, Objective: item.Objective, Phase: item.Phase, Role: item.Role,
			Parent: item.Parent, HandoffRequired: item.HandoffRequired,
			Dependencies: append([]string(nil), item.Dependencies...), WriteSet: append([]string(nil), item.WriteSet...),
			RequiredTests: tests, SkillRefs: append([]string(nil), item.SkillRefs...), ToolRefs: append([]string(nil), item.ToolRefs...),
			CapabilityRefs: append([]string(nil), item.CapabilityRefs...), EgressPolicyRef: item.EgressPolicyRef,
			OutputContract:      goal.OutputContractKind(item.OutputContract),
			SecurityCriticality: governance.SecurityCriticality(item.SecurityCriticality), ReasoningEffort: governance.ReasoningEffort(item.ReasoningEffort),
			CouncilPolicy: council.Policy(item.CouncilPolicy),
		})
	}
	return result
}

func marshalApplication(value any, err error) (json.RawMessage, error) {
	if err != nil {
		return nil, normalizeApplicationError(err)
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("commands.output_invalid")
	}
	return encoded, nil
}
