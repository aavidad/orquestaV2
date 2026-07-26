package application

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/wizard/stages"
)

var ErrWizardStagePlanInvalid = errors.New(
	"application.wizard_stage_plan_invalid",
)

// WizardStageTestBinding binds one template test kind to an already registered
// attestor tool. It contains no provider, credential or execution policy.
type WizardStageTestBinding struct {
	ToolRef          goal.ToolRef
	Arguments        []string
	WorkingDirectory string
}

// WizardStageTestBindings is explicit so compilation never guesses a runner
// from a test name, unit role, write-set or provider.
type WizardStageTestBindings struct {
	Unit          WizardStageTestBinding
	Contract      WizardStageTestBinding
	Integration   WizardStageTestBinding
	Security      WizardStageTestBinding
	Review        WizardStageTestBinding
	Postcondition WizardStageTestBinding
}

// WizardStageEffortBudgets binds declared template effort to concrete demand.
// The caller remains responsible for choosing demands that fit deployment
// policy; this pure compiler neither reads configuration nor creates envelopes.
type WizardStageEffortBudgets struct {
	Routine governance.ResourceVector
	Focused governance.ResourceVector
	Deep    governance.ResourceVector
}

// WizardStagePlanInput contains only application-owned values absent from a
// stages.Template. Refs are typed before entering this compiler.
type WizardStagePlanInput struct {
	Objective      string
	InputRefs      []goal.InputRef
	SkillRefs      []goal.SkillRef
	ToolRefs       []goal.ToolRef
	CapabilityRefs []goal.CapabilityRef
	Tests          WizardStageTestBindings
	Budgets        WizardStageEffortBudgets
}

// WizardStagePlanCompilation keeps PlanSpec honest about its representation
// boundary. Projection retains exact stage catalog semantics that PlanSpec
// cannot carry; it does not authorize or execute any projected effect.
type WizardStagePlanCompilation struct {
	Plan       PlanSpec
	Projection WizardStagePlanProjection
}

type WizardStagePlanProjection struct {
	TemplateRef           stages.TemplateRef
	RoadmapCapabilityRefs []stages.RoadmapCapabilityRef
	Stages                []WizardStageProjection
	Units                 []WizardStageUnitProjection
}

type WizardStageProjection struct {
	Ref       stages.StageRef
	Sequence  int
	Phase     stages.PhaseKind
	DependsOn []stages.StageRef
	PlanKey   string
}

type WizardStageUnitProjection struct {
	Ref                 stages.UnitRef
	StageRef            stages.StageRef
	DependsOn           []stages.UnitRef
	PlanDependencies    []string
	Role                stages.RoleRef
	WriteSet            []stages.WriteScope
	RequiredTests       []stages.RequiredTest
	AcceptanceCriteria  []stages.Criterion
	Effects             []stages.Effect
	Policy              stages.ExecutionPolicy
	BudgetDemand        governance.BudgetDemand
	CouncilPolicy       council.Policy
	SecurityCriticality governance.SecurityCriticality
	ReasoningEffort     governance.ReasoningEffort
}

// CompileWizardStagePlan performs a deterministic, side-effect-free
// translation. Goal remains the sole lifecycle owner; this function creates
// neither a Goal nor a plan generation.
func CompileWizardStagePlan(
	template stages.Template,
	input WizardStagePlanInput,
) (WizardStagePlanCompilation, error) {
	if template.Ref().String() == "" ||
		len(template.Stages()) == 0 ||
		len(template.Units()) == 0 {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("template", nil)
	}
	if strings.TrimSpace(input.Objective) == "" ||
		strings.TrimSpace(input.Objective) != input.Objective {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("objective", nil)
	}

	inputRefs, err := canonicalPlanRefs(input.InputRefs)
	if err != nil {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("input_refs", err)
	}
	skillRefs, err := canonicalPlanRefs(input.SkillRefs)
	if err != nil {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("skill_refs", err)
	}
	toolRefs, err := canonicalPlanRefs(input.ToolRefs)
	if err != nil {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("tool_refs", err)
	}
	capabilityRefs, err := canonicalPlanRefs(input.CapabilityRefs)
	if err != nil {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("capability_refs", err)
	}

	stageValues := template.Stages()
	unitValues := template.Units()
	unitsByStage := indexWizardUnitsByStage(unitValues)
	plan := PlanSpec{
		Phases:    make([]PhaseSpec, 0, len(stageValues)),
		WorkItems: make([]WorkItemSpec, 0, len(unitValues)),
	}
	projection := WizardStagePlanProjection{
		TemplateRef:           template.Ref(),
		RoadmapCapabilityRefs: template.RoadmapCapabilityRefs(),
		Stages:                make([]WizardStageProjection, 0, len(stageValues)),
		Units:                 make([]WizardStageUnitProjection, 0, len(unitValues)),
	}

	for _, stage := range stageValues {
		criterionRefs := wizardStageCriterionRefs(stage.Ref(), unitsByStage)
		stageRef := stage.Ref().String()
		plan.Phases = append(plan.Phases, PhaseSpec{
			Ref: "phase-instance:" + stageRef,
			Key: stageRef, TemplateRef: stageRef,
			InputRefs: inputRefs, CriterionRefs: criterionRefs,
		})
		projection.Stages = append(projection.Stages, WizardStageProjection{
			Ref: stage.Ref(), Sequence: stage.Sequence(), Phase: stage.Phase(),
			DependsOn: stage.DependsOn(), PlanKey: stageRef,
		})
	}

	stagesByRef := make(map[string]stages.Stage, len(stageValues))
	for _, stage := range stageValues {
		stagesByRef[stage.Ref().String()] = stage
	}
	for _, unit := range unitValues {
		item, projected, compileErr := compileWizardStageUnit(
			unit, stagesByRef, unitsByStage, input,
			skillRefs, toolRefs, capabilityRefs,
		)
		if compileErr != nil {
			return WizardStagePlanCompilation{}, compileErr
		}
		plan.WorkItems = append(plan.WorkItems, item)
		projection.Units = append(projection.Units, projected)
	}

	// This uses the current PhaseSpec/WorkItemSpec compilers and goal.Plan
	// validation with synthetic refs only. It performs no persistence or Goal
	// lifecycle transition.
	if err := validateIntakeDossierPlan(plan); err != nil {
		return WizardStagePlanCompilation{},
			invalidWizardStagePlan("plan", err)
	}
	return WizardStagePlanCompilation{
		Plan: plan, Projection: projection,
	}, nil
}

func compileWizardStageUnit(
	unit stages.Unit,
	stagesByRef map[string]stages.Stage,
	unitsByStage map[string][]stages.Unit,
	input WizardStagePlanInput,
	skillRefs []string,
	toolRefs []string,
	capabilityRefs []string,
) (WorkItemSpec, WizardStageUnitProjection, error) {
	dependencies := wizardUnitPlanDependencies(
		unit, stagesByRef, unitsByStage,
	)
	requiredTests := make([]RequiredTestSpec, 0, len(unit.RequiredTests()))
	for _, test := range unit.RequiredTests() {
		binding, found := wizardTestBinding(input.Tests, test.Kind)
		if !found {
			return WorkItemSpec{}, WizardStageUnitProjection{},
				invalidWizardStagePlan(
					"tests."+string(test.Kind), nil,
				)
		}
		requiredTests = append(requiredTests, RequiredTestSpec{
			Ref: test.Ref.String(), ToolRef: binding.ToolRef.String(),
			Arguments:        append([]string(nil), binding.Arguments...),
			WorkingDirectory: binding.WorkingDirectory,
		})
	}

	policy := unit.Policy()
	resources, found := wizardEffortBudget(input.Budgets, policy.Effort)
	if !found {
		return WorkItemSpec{}, WizardStageUnitProjection{},
			invalidWizardStagePlan(
				"budgets."+string(policy.Effort), nil,
			)
	}
	if err := governance.ValidateResourceVector(resources); err != nil ||
		resources.ProcessSlots == 0 {
		return WorkItemSpec{}, WizardStageUnitProjection{},
			invalidWizardStagePlan(
				"budgets."+string(policy.Effort), err,
			)
	}
	demand := governance.BudgetDemand{
		Ref:       "budget-demand:" + unit.Ref().String(),
		Resources: resources,
	}
	councilPolicy := wizardCouncilPolicy(policy.Security.Approval)
	criticality := wizardSecurityCriticality(policy.Security)
	effort := wizardReasoningEffort(policy.Effort)

	item := WorkItemSpec{
		Key:       unit.Ref().String(),
		Objective: input.Objective + " [" + unit.Ref().String() + "]",
		Phase:     unit.StageRef().String(), Role: unit.Role().String(),
		Dependencies:  dependencies,
		WriteSet:      wizardWriteSet(unit.WriteSet()),
		CouncilPolicy: councilPolicy,
		RequiredTests: requiredTests,
		SkillRefs:     skillRefs, ToolRefs: toolRefs,
		CapabilityRefs: capabilityRefs,
		OutputContract: goal.OutputContractEvidenceBundle,
		BudgetDemand:   demand, SecurityCriticality: criticality,
		ReasoningEffort: effort,
	}
	projected := WizardStageUnitProjection{
		Ref: unit.Ref(), StageRef: unit.StageRef(),
		DependsOn:        unit.DependsOn(),
		PlanDependencies: append([]string(nil), dependencies...),
		Role:             unit.Role(), WriteSet: unit.WriteSet(),
		RequiredTests:      unit.RequiredTests(),
		AcceptanceCriteria: unit.AcceptanceCriteria(),
		Effects:            unit.Effects(), Policy: policy,
		BudgetDemand: demand, CouncilPolicy: councilPolicy,
		SecurityCriticality: criticality, ReasoningEffort: effort,
	}
	return item, projected, nil
}

func indexWizardUnitsByStage(
	units []stages.Unit,
) map[string][]stages.Unit {
	index := make(map[string][]stages.Unit)
	for _, unit := range units {
		key := unit.StageRef().String()
		index[key] = append(index[key], unit)
	}
	return index
}

// wizardUnitPlanDependencies retains explicit unit edges and turns direct
// stage edges into conservative completion gates over every unit in the
// predecessor stage. PlanSpec has no separate phase DAG.
func wizardUnitPlanDependencies(
	unit stages.Unit,
	stagesByRef map[string]stages.Stage,
	unitsByStage map[string][]stages.Unit,
) []string {
	seen := make(map[string]struct{})
	for _, dependency := range unit.DependsOn() {
		seen[dependency.String()] = struct{}{}
	}
	stage := stagesByRef[unit.StageRef().String()]
	for _, stageDependency := range stage.DependsOn() {
		for _, dependency := range unitsByStage[stageDependency.String()] {
			seen[dependency.Ref().String()] = struct{}{}
		}
	}
	values := make([]string, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func wizardStageCriterionRefs(
	stageRef stages.StageRef,
	unitsByStage map[string][]stages.Unit,
) []string {
	var values []string
	for _, unit := range unitsByStage[stageRef.String()] {
		for _, criterion := range unit.AcceptanceCriteria() {
			values = append(values, criterion.Ref.String())
		}
	}
	sort.Strings(values)
	return values
}

func wizardWriteSet(scopes []stages.WriteScope) []string {
	values := make([]string, len(scopes))
	for index, scope := range scopes {
		values[index] = scope.Path
	}
	return values
}

func wizardTestBinding(
	bindings WizardStageTestBindings,
	kind stages.TestKind,
) (WizardStageTestBinding, bool) {
	switch kind {
	case stages.TestUnit:
		return bindings.Unit, true
	case stages.TestContract:
		return bindings.Contract, true
	case stages.TestIntegration:
		return bindings.Integration, true
	case stages.TestSecurity:
		return bindings.Security, true
	case stages.TestReview:
		return bindings.Review, true
	case stages.TestPostcondition:
		return bindings.Postcondition, true
	default:
		return WizardStageTestBinding{}, false
	}
}

func wizardEffortBudget(
	budgets WizardStageEffortBudgets,
	effort stages.EffortLevel,
) (governance.ResourceVector, bool) {
	switch effort {
	case stages.EffortRoutine:
		return budgets.Routine, true
	case stages.EffortFocused:
		return budgets.Focused, true
	case stages.EffortDeep:
		return budgets.Deep, true
	default:
		return governance.ResourceVector{}, false
	}
}

func wizardCouncilPolicy(approval stages.ApprovalPolicy) council.Policy {
	if approval == stages.ApprovalNone {
		return council.PolicyAuto
	}
	// Required council is conservative review metadata. Before-mutation and
	// before-publish approval remain projection-only and are not claimed to be
	// satisfied by council.
	return council.PolicyRequired
}

func wizardSecurityCriticality(
	policy stages.SecurityPolicy,
) governance.SecurityCriticality {
	switch policy.Risk {
	case stages.RiskCritical, stages.RiskHigh:
		return governance.SecurityCriticalityCritical
	case stages.RiskModerate:
		return governance.SecurityCriticalitySensitive
	default:
		if policy.SensitiveInputs {
			return governance.SecurityCriticalitySensitive
		}
		return governance.SecurityCriticalityNormal
	}
}

func wizardReasoningEffort(
	effort stages.EffortLevel,
) governance.ReasoningEffort {
	switch effort {
	case stages.EffortRoutine:
		return governance.ReasoningEffortLow
	case stages.EffortDeep:
		return governance.ReasoningEffortHigh
	default:
		return governance.ReasoningEffortMedium
	}
}

type planStringRef interface {
	String() string
}

func canonicalPlanRefs[T planStringRef](refs []T) ([]string, error) {
	values := make([]string, len(refs))
	seen := make(map[string]struct{}, len(refs))
	for index, ref := range refs {
		value := ref.String()
		if value == "" {
			return nil, errors.New("application.wizard_stage_plan_ref_invalid")
		}
		if _, duplicate := seen[value]; duplicate {
			return nil, errors.New("application.wizard_stage_plan_ref_duplicate")
		}
		seen[value] = struct{}{}
		values[index] = value
	}
	sort.Strings(values)
	return values, nil
}

func invalidWizardStagePlan(field string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%w: %s", ErrWizardStagePlanInvalid, field)
	}
	return fmt.Errorf("%w: %s: %v", ErrWizardStagePlanInvalid, field, cause)
}
