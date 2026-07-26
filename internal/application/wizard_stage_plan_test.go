package application

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/governance"
	"orquesta/internal/wizard/stages"
)

func TestCompileWizardStagePlanCoversEveryBuiltInTemplate(t *testing.T) {
	t.Parallel()

	catalog := stages.BuiltIn()
	templates := catalog.Templates()
	if got, want := len(templates), 6; got != want {
		t.Fatalf("templates=%d want=%d", got, want)
	}
	for _, template := range templates {
		template := template
		t.Run(template.Ref().String(), func(t *testing.T) {
			t.Parallel()

			input := wizardStagePlanTestInput(t)
			compilation, err := CompileWizardStagePlan(template, input)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			if err := validateIntakeDossierPlan(compilation.Plan); err != nil {
				t.Fatalf("current plan compilers rejected result: %v", err)
			}
			assertWizardTemplateCompilation(t, template, input, compilation)
		})
	}
}

func TestCompileWizardStagePlanIsDeterministicAcrossRefInputOrder(t *testing.T) {
	t.Parallel()

	template := wizardBuiltInTemplate(t, "template:build_app")
	leftInput := wizardStagePlanTestInput(t)
	rightInput := wizardStagePlanTestInput(t)
	slices.Reverse(rightInput.InputRefs)
	slices.Reverse(rightInput.SkillRefs)
	slices.Reverse(rightInput.ToolRefs)
	slices.Reverse(rightInput.CapabilityRefs)

	left, err := CompileWizardStagePlan(template, leftInput)
	if err != nil {
		t.Fatalf("compile left: %v", err)
	}
	right, err := CompileWizardStagePlan(template, rightInput)
	if err != nil {
		t.Fatalf("compile right: %v", err)
	}
	if !reflect.DeepEqual(left, right) {
		t.Fatalf("compilation depends on ref input order\nleft=%#v\nright=%#v", left, right)
	}

	leftInput.Tests.Contract.Arguments[0] = "mutated-after-compile"
	if left.Plan.WorkItems[0].RequiredTests[0].Arguments[0] == "mutated-after-compile" {
		t.Fatal("compiled required test aliases caller arguments")
	}
}

func TestCompileWizardStagePlanRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	research := wizardBuiltInTemplate(t, "template:research")
	valid := func(t *testing.T) WizardStagePlanInput {
		return wizardStagePlanTestInput(t)
	}
	tests := []struct {
		name     string
		template stages.Template
		mutate   func(*WizardStagePlanInput)
	}{
		{
			name: "zero template",
		},
		{
			name: "blank objective", template: research,
			mutate: func(input *WizardStagePlanInput) {
				input.Objective = " "
			},
		},
		{
			name: "duplicate typed input ref", template: research,
			mutate: func(input *WizardStagePlanInput) {
				input.InputRefs[1] = input.InputRefs[0]
			},
		},
		{
			name: "missing contract test tool", template: research,
			mutate: func(input *WizardStagePlanInput) {
				input.Tests.Contract = WizardStageTestBinding{}
			},
		},
		{
			name: "unsafe test working directory", template: research,
			mutate: func(input *WizardStagePlanInput) {
				input.Tests.Contract.WorkingDirectory = "../escape"
			},
		},
		{
			name: "zero focused process slots", template: research,
			mutate: func(input *WizardStagePlanInput) {
				input.Budgets.Focused.ProcessSlots = 0
			},
		},
		{
			name: "negative deep budget", template: research,
			mutate: func(input *WizardStagePlanInput) {
				input.Budgets.Deep.Tokens = -1
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			input := valid(t)
			if test.mutate != nil {
				test.mutate(&input)
			}
			_, err := CompileWizardStagePlan(test.template, input)
			if !errors.Is(err, ErrWizardStagePlanInvalid) {
				t.Fatalf("error=%v want ErrWizardStagePlanInvalid", err)
			}
		})
	}
}

func TestCompileWizardStagePlanMaterializesStageDAGAsDependencyKeys(t *testing.T) {
	t.Parallel()

	template := wizardStagePlanDAGFixture(t)
	compilation, err := CompileWizardStagePlan(
		template,
		wizardStagePlanTestInput(t),
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if got, want := compilation.Plan.WorkItems[1].Dependencies,
		[]string{"unit:compiler_test.first"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("stage dependencies=%v want=%v", got, want)
	}
	if got := compilation.Projection.Units[1].DependsOn; len(got) != 0 {
		t.Fatalf("source unit dependencies changed in projection: %v", got)
	}
	if got := compilation.Projection.Units[1].PlanDependencies; !reflect.DeepEqual(
		got,
		[]string{"unit:compiler_test.first"},
	) {
		t.Fatalf("projected plan dependencies=%v", got)
	}
	first := compilation.Plan.WorkItems[0]
	if first.SecurityCriticality != governance.SecurityCriticalityNormal ||
		first.ReasoningEffort != governance.ReasoningEffortLow ||
		first.BudgetDemand.Resources !=
			wizardStagePlanTestInput(t).Budgets.Routine {
		t.Fatalf(
			"routine mapping security/effort/budget=%q/%q/%+v",
			first.SecurityCriticality,
			first.ReasoningEffort,
			first.BudgetDemand.Resources,
		)
	}
}

func TestCompileWizardStagePlanProjectsUnrepresentableSemanticsExactly(t *testing.T) {
	t.Parallel()

	template := wizardBuiltInTemplate(t, "template:deploy")
	compilation, err := CompileWizardStagePlan(
		template,
		wizardStagePlanTestInput(t),
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	index := wizardPlanItemIndex(
		t, compilation.Plan, "unit:deploy.apply_release",
	)
	item := compilation.Plan.WorkItems[index]
	projected := compilation.Projection.Units[index]
	if item.CouncilPolicy != council.PolicyRequired ||
		item.SecurityCriticality != governance.SecurityCriticalityCritical ||
		item.ReasoningEffort != governance.ReasoningEffortHigh {
		t.Fatalf(
			"conservative governance=%q/%q/%q",
			item.CouncilPolicy,
			item.SecurityCriticality,
			item.ReasoningEffort,
		)
	}
	if projected.Policy.Security.Approval != stages.ApprovalBeforeMutation ||
		!projected.Policy.Security.LeastPrivilege ||
		!projected.Policy.Security.SensitiveInputs {
		t.Fatalf("exact security policy lost: %+v", projected.Policy.Security)
	}
	if got, want := projected.Effects, []stages.Effect{{
		Ref:  wizardEffectRef(t, "effect:deploy.apply_release"),
		Kind: stages.EffectMutateExternal, ApprovalRequired: true,
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("effects=%#v want=%#v", got, want)
	}
	if len(projected.RequiredTests) != 1 ||
		!reflect.DeepEqual(
			projected.RequiredTests[0].CriterionRefs,
			[]stages.CriterionRef{
				wizardCriterionRef(
					t,
					"criterion:deploy.apply_release.attempt_receipt_recorded",
				),
				wizardCriterionRef(
					t,
					"criterion:deploy.apply_release.mutation_matches_approval",
				),
			},
		) {
		t.Fatalf(
			"test-to-criteria projection=%#v",
			projected.RequiredTests,
		)
	}
	phase := compilation.Plan.Phases[2]
	if !slices.Contains(
		phase.CriterionRefs,
		"criterion:deploy.apply_release.mutation_matches_approval",
	) {
		t.Fatalf("phase criteria=%v", phase.CriterionRefs)
	}
	if item.OutputContract != goal.OutputContractEvidenceBundle {
		t.Fatalf("effect was treated as authorization: output=%q", item.OutputContract)
	}
}

func TestCompileWizardStagePlanMapsSecurityAndEffortAxesIndependently(t *testing.T) {
	t.Parallel()

	research, err := CompileWizardStagePlan(
		wizardBuiltInTemplate(t, "template:research"),
		wizardStagePlanTestInput(t),
	)
	if err != nil {
		t.Fatalf("compile research: %v", err)
	}
	collect := research.Plan.WorkItems[wizardPlanItemIndex(t, research.Plan, "unit:research.collect_evidence")]
	if collect.SecurityCriticality != governance.SecurityCriticalitySensitive ||
		collect.ReasoningEffort != governance.ReasoningEffortMedium ||
		collect.CouncilPolicy != council.PolicyAuto {
		t.Fatalf(
			"sensitive focused mapping=%q/%q/%q",
			collect.SecurityCriticality,
			collect.ReasoningEffort,
			collect.CouncilPolicy,
		)
	}

	selfChange, err := CompileWizardStagePlan(
		wizardBuiltInTemplate(t, "template:self_change"),
		wizardStagePlanTestInput(t),
	)
	if err != nil {
		t.Fatalf("compile self_change: %v", err)
	}
	mutation := selfChange.Plan.WorkItems[wizardPlanItemIndex(t, selfChange.Plan, "unit:self_change.implement_change")]
	if mutation.SecurityCriticality != governance.SecurityCriticalityCritical ||
		mutation.ReasoningEffort != governance.ReasoningEffortHigh ||
		mutation.CouncilPolicy != council.PolicyRequired {
		t.Fatalf(
			"critical deep mapping=%q/%q/%q",
			mutation.SecurityCriticality,
			mutation.ReasoningEffort,
			mutation.CouncilPolicy,
		)
	}
}

func assertWizardTemplateCompilation(
	t *testing.T,
	template stages.Template,
	input WizardStagePlanInput,
	compilation WizardStagePlanCompilation,
) {
	t.Helper()

	stageValues := template.Stages()
	unitValues := template.Units()
	if len(compilation.Plan.Phases) != len(stageValues) ||
		len(compilation.Plan.WorkItems) != len(unitValues) {
		t.Fatalf(
			"shape phases/items=%d/%d want=%d/%d",
			len(compilation.Plan.Phases),
			len(compilation.Plan.WorkItems),
			len(stageValues),
			len(unitValues),
		)
	}
	if compilation.Projection.TemplateRef != template.Ref() ||
		!reflect.DeepEqual(
			compilation.Projection.RoadmapCapabilityRefs,
			template.RoadmapCapabilityRefs(),
		) {
		t.Fatal("template traceability projection changed")
	}

	for index, stage := range stageValues {
		phase := compilation.Plan.Phases[index]
		if phase.Ref != "phase-instance:"+stage.Ref().String() ||
			phase.Key != stage.Ref().String() ||
			phase.TemplateRef != stage.Ref().String() ||
			!reflect.DeepEqual(phase.InputRefs, []string{"input:a", "input:z"}) {
			t.Fatalf("phase[%d]=%+v stage=%s", index, phase, stage.Ref().String())
		}
		projected := compilation.Projection.Stages[index]
		if projected.Ref != stage.Ref() ||
			projected.Sequence != stage.Sequence() ||
			projected.Phase != stage.Phase() ||
			!reflect.DeepEqual(projected.DependsOn, stage.DependsOn()) {
			t.Fatalf("stage projection[%d]=%+v", index, projected)
		}
	}

	for index, unit := range unitValues {
		item := compilation.Plan.WorkItems[index]
		projected := compilation.Projection.Units[index]
		if item.Key != unit.Ref().String() ||
			item.Objective != input.Objective+" ["+unit.Ref().String()+"]" ||
			item.Phase != unit.StageRef().String() ||
			item.Role != unit.Role().String() ||
			!reflect.DeepEqual(item.WriteSet, wizardWriteSet(unit.WriteSet())) ||
			!reflect.DeepEqual(item.SkillRefs, []string{"skill:a", "skill:z"}) ||
			!reflect.DeepEqual(item.ToolRefs, []string{"tool:a", "tool:z"}) ||
			!reflect.DeepEqual(
				item.CapabilityRefs,
				[]string{"capability:a", "capability:z"},
			) {
			t.Fatalf("work item[%d]=%+v unit=%s", index, item, unit.Ref().String())
		}
		if got, want := item.Dependencies,
			wizardUnitRefStrings(unit.DependsOn()); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s dependencies=%v want=%v", item.Key, got, want)
		}
		if len(item.RequiredTests) != len(unit.RequiredTests()) {
			t.Fatalf("%s required tests=%v", item.Key, item.RequiredTests)
		}
		for testIndex, required := range unit.RequiredTests() {
			compiled := item.RequiredTests[testIndex]
			binding, _ := wizardTestBinding(input.Tests, required.Kind)
			if compiled.Ref != required.Ref.String() ||
				compiled.ToolRef != binding.ToolRef.String() ||
				!reflect.DeepEqual(compiled.Arguments, binding.Arguments) ||
				compiled.WorkingDirectory != binding.WorkingDirectory {
				t.Fatalf("%s test[%d]=%+v", item.Key, testIndex, compiled)
			}
		}
		expectedBudget, _ := wizardEffortBudget(input.Budgets, unit.Policy().Effort)
		if item.BudgetDemand.Ref != "budget-demand:"+unit.Ref().String() ||
			item.BudgetDemand.Resources != expectedBudget {
			t.Fatalf("%s budget=%+v", item.Key, item.BudgetDemand)
		}
		if !reflect.DeepEqual(projected.RequiredTests, unit.RequiredTests()) ||
			!reflect.DeepEqual(projected.AcceptanceCriteria, unit.AcceptanceCriteria()) ||
			!reflect.DeepEqual(projected.Effects, unit.Effects()) ||
			projected.Policy != unit.Policy() {
			t.Fatalf("%s exact projection lost fields: %+v", item.Key, projected)
		}
	}
}

func wizardStagePlanTestInput(t *testing.T) WizardStagePlanInput {
	t.Helper()

	testBinding := func(kind string) WizardStageTestBinding {
		return WizardStageTestBinding{
			ToolRef:   wizardToolRef(t, "tool:test-"+kind),
			Arguments: []string{"run", kind}, WorkingDirectory: ".",
		}
	}
	return WizardStagePlanInput{
		Objective: "deliver the confirmed objective",
		InputRefs: []goal.InputRef{
			wizardInputRef(t, "input:z"), wizardInputRef(t, "input:a"),
		},
		SkillRefs: []goal.SkillRef{
			wizardSkillRef(t, "skill:z"), wizardSkillRef(t, "skill:a"),
		},
		ToolRefs: []goal.ToolRef{
			wizardToolRef(t, "tool:z"), wizardToolRef(t, "tool:a"),
		},
		CapabilityRefs: []goal.CapabilityRef{
			wizardCapabilityRef(t, "capability:z"),
			wizardCapabilityRef(t, "capability:a"),
		},
		Tests: WizardStageTestBindings{
			Unit: testBinding("unit"), Contract: testBinding("contract"),
			Integration: testBinding("integration"),
			Security:    testBinding("security"), Review: testBinding("review"),
			Postcondition: testBinding("postcondition"),
		},
		Budgets: WizardStageEffortBudgets{
			Routine: governance.ResourceVector{
				Tokens: 100, ActiveTimeNS: 100, ProcessSlots: 1, DiskBytes: 100,
			},
			Focused: governance.ResourceVector{
				Tokens: 200, ActiveTimeNS: 200, ProcessSlots: 1, DiskBytes: 200,
			},
			Deep: governance.ResourceVector{
				Tokens: 300, ActiveTimeNS: 300, ProcessSlots: 1, DiskBytes: 300,
			},
		},
	}
}

func wizardStagePlanDAGFixture(t *testing.T) stages.Template {
	t.Helper()

	firstStageRef := wizardStageRef(t, "stage:compiler_test.first")
	secondStageRef := wizardStageRef(t, "stage:compiler_test.second")
	firstStage, err := stages.NewStage(stages.StageInput{
		Ref: firstStageRef, Sequence: 10, Phase: stages.PhaseDiscover,
	})
	if err != nil {
		t.Fatalf("first stage: %v", err)
	}
	secondStage, err := stages.NewStage(stages.StageInput{
		Ref: secondStageRef, Sequence: 20, Phase: stages.PhaseProduce,
		DependsOn: []stages.StageRef{firstStageRef},
	})
	if err != nil {
		t.Fatalf("second stage: %v", err)
	}
	first := wizardFixtureUnit(
		t, "first", firstStageRef, "artifacts/compiler_test/first",
	)
	second := wizardFixtureUnit(
		t, "second", secondStageRef, "artifacts/compiler_test/second",
	)
	ref, err := stages.NewTemplateRef("template:compiler_test")
	if err != nil {
		t.Fatalf("template ref: %v", err)
	}
	capability, err := stages.NewRoadmapCapabilityRef("WIZ-24")
	if err != nil {
		t.Fatalf("roadmap ref: %v", err)
	}
	template, err := stages.NewTemplate(stages.TemplateInput{
		Ref: ref, RoadmapCapabilityRefs: []stages.RoadmapCapabilityRef{capability},
		Stages: []stages.Stage{secondStage, firstStage},
		Units:  []stages.Unit{second, first},
	})
	if err != nil {
		t.Fatalf("template: %v", err)
	}
	return template
}

func wizardFixtureUnit(
	t *testing.T,
	name string,
	stageRef stages.StageRef,
	writeScope string,
) stages.Unit {
	t.Helper()

	unitRef, err := stages.NewUnitRef("unit:compiler_test." + name)
	if err != nil {
		t.Fatalf("unit ref: %v", err)
	}
	roleRef, err := stages.NewRoleRef("role:worker")
	if err != nil {
		t.Fatalf("role ref: %v", err)
	}
	testRef, err := stages.NewTestRef("test:compiler_test." + name)
	if err != nil {
		t.Fatalf("test ref: %v", err)
	}
	criterionRef := wizardCriterionRef(
		t, "criterion:compiler_test."+name+".accepted",
	)
	effectRef := wizardEffectRef(
		t, "effect:compiler_test."+name,
	)
	unit, err := stages.NewUnit(stages.UnitInput{
		Ref: unitRef, StageRef: stageRef, Role: roleRef,
		WriteSet: []stages.WriteScope{{Path: writeScope}},
		RequiredTests: []stages.RequiredTest{{
			Ref: testRef, Kind: stages.TestUnit,
			CriterionRefs: []stages.CriterionRef{criterionRef},
		}},
		AcceptanceCriteria: []stages.Criterion{{Ref: criterionRef}},
		Effects: []stages.Effect{{
			Ref: effectRef, Kind: stages.EffectWriteArtifact,
		}},
		Policy: stages.ExecutionPolicy{
			Effort: stages.EffortRoutine,
			Security: stages.SecurityPolicy{
				Risk: stages.RiskLow, Approval: stages.ApprovalNone,
				LeastPrivilege: true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unit: %v", err)
	}
	return unit
}

func wizardBuiltInTemplate(t *testing.T, value string) stages.Template {
	t.Helper()

	ref, err := stages.NewTemplateRef(value)
	if err != nil {
		t.Fatalf("template ref: %v", err)
	}
	template, found := stages.BuiltIn().Template(ref)
	if !found {
		t.Fatalf("template %s not found", value)
	}
	return template
}

func wizardPlanItemIndex(t *testing.T, plan PlanSpec, key string) int {
	t.Helper()

	for index, item := range plan.WorkItems {
		if item.Key == key {
			return index
		}
	}
	t.Fatalf("work item %s not found", key)
	return -1
}

func wizardUnitRefStrings(refs []stages.UnitRef) []string {
	values := make([]string, len(refs))
	for index, ref := range refs {
		values[index] = ref.String()
	}
	return values
}

func wizardInputRef(t *testing.T, value string) goal.InputRef {
	t.Helper()
	ref, err := goal.NewInputRef(value)
	if err != nil {
		t.Fatalf("input ref %s: %v", value, err)
	}
	return ref
}

func wizardSkillRef(t *testing.T, value string) goal.SkillRef {
	t.Helper()
	ref, err := goal.NewSkillRef(value)
	if err != nil {
		t.Fatalf("skill ref %s: %v", value, err)
	}
	return ref
}

func wizardToolRef(t *testing.T, value string) goal.ToolRef {
	t.Helper()
	ref, err := goal.NewToolRef(value)
	if err != nil {
		t.Fatalf("tool ref %s: %v", value, err)
	}
	return ref
}

func wizardCapabilityRef(t *testing.T, value string) goal.CapabilityRef {
	t.Helper()
	ref, err := goal.NewCapabilityRef(value)
	if err != nil {
		t.Fatalf("capability ref %s: %v", value, err)
	}
	return ref
}

func wizardStageRef(t *testing.T, value string) stages.StageRef {
	t.Helper()
	ref, err := stages.NewStageRef(value)
	if err != nil {
		t.Fatalf("stage ref %s: %v", value, err)
	}
	return ref
}

func wizardCriterionRef(t *testing.T, value string) stages.CriterionRef {
	t.Helper()
	ref, err := stages.NewCriterionRef(value)
	if err != nil {
		t.Fatalf("criterion ref %s: %v", value, err)
	}
	return ref
}

func wizardEffectRef(t *testing.T, value string) stages.EffectRef {
	t.Helper()
	ref, err := stages.NewEffectRef(value)
	if err != nil {
		t.Fatalf("effect ref %s: %v", value, err)
	}
	return ref
}
