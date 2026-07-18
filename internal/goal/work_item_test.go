package goal_test

import (
	"testing"
	"time"

	domain "orquesta/internal/goal"
	"orquesta/internal/governance"
)

func TestWorkItemTransitionsProduceImmutableRevisions(t *testing.T) {
	item := newWorkItem(t, "failed stop garbage are data, not state")
	if item.State() != domain.WorkItemStatePending || item.Revision() != 1 {
		t.Fatalf("new item state/revision = %q/%d, want pending/1", item.State(), item.Revision())
	}

	execution := mustRef(t, "execution:001", domain.NewExecutionRef)
	_, err := item.Start(99, execution, baseTime().Add(3*time.Minute))
	requireCode(t, err, domain.ErrorRevisionConflict)
	if item.State() != domain.WorkItemStatePending {
		t.Fatalf("failed transition mutated original state to %q", item.State())
	}

	running, err := item.Start(item.Revision(), execution, baseTime().Add(3*time.Minute))
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if running.State() != domain.WorkItemStateRunning || running.Revision() != 2 {
		t.Fatalf("running state/revision = %q/%d, want running/2", running.State(), running.Revision())
	}
	if item.State() != domain.WorkItemStatePending || item.Revision() != 1 {
		t.Fatal("Start mutated the source snapshot")
	}
	if got, ok := running.Execution(); !ok || got != execution {
		t.Fatalf("Execution() = %q/%v, want %q/true", got, ok, execution)
	}

	artifact := mustRef(t, "artifact:001", domain.NewArtifactRef)
	attestation := mustRef(t, "attestation:001", domain.NewAttestationRef)
	_, err = running.Succeed(running.Revision(), nil, []domain.AttestationRef{attestation}, baseTime().Add(4*time.Minute))
	requireCode(t, err, domain.ErrorEvidenceRequired)
	_, err = running.Succeed(running.Revision(), []domain.ArtifactRef{artifact}, nil, baseTime().Add(4*time.Minute))
	requireCode(t, err, domain.ErrorEvidenceRequired)

	succeeded, err := running.Succeed(
		running.Revision(),
		[]domain.ArtifactRef{artifact},
		[]domain.AttestationRef{attestation},
		baseTime().Add(4*time.Minute),
	)
	if err != nil {
		t.Fatalf("Succeed() error = %v", err)
	}
	if succeeded.State() != domain.WorkItemStateSucceeded || succeeded.Revision() != 3 || !succeeded.IsTerminal() {
		t.Fatalf("succeeded state/revision/terminal = %q/%d/%v", succeeded.State(), succeeded.Revision(), succeeded.IsTerminal())
	}
	if running.State() != domain.WorkItemStateRunning || running.Revision() != 2 {
		t.Fatal("Succeed mutated the running snapshot")
	}

	artifacts := succeeded.Artifacts()
	artifacts[0] = mustRef(t, "artifact:changed", domain.NewArtifactRef)
	if got := succeeded.Artifacts()[0]; got != artifact {
		t.Fatalf("mutating returned artifacts changed snapshot: got %q", got)
	}
	attestations := succeeded.Attestations()
	attestations[0] = mustRef(t, "attestation:changed", domain.NewAttestationRef)
	if got := succeeded.Attestations()[0]; got != attestation {
		t.Fatalf("mutating returned attestations changed snapshot: got %q", got)
	}

	_, err = succeeded.Fail(succeeded.Revision(), baseTime().Add(5*time.Minute))
	requireCode(t, err, domain.ErrorInvalidTransition)
}

func TestWorkItemFailureRequiresExplicitTransition(t *testing.T) {
	item := newWorkItem(t, "failed capacity_limited garbage")
	if item.State() != domain.WorkItemStatePending {
		t.Fatalf("objective words changed state to %q", item.State())
	}

	running, err := item.Start(
		item.Revision(),
		mustRef(t, "execution:failure", domain.NewExecutionRef),
		baseTime().Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if running.State() != domain.WorkItemStateRunning {
		t.Fatalf("objective words changed running state to %q", running.State())
	}
	_, err = running.Fail(1, baseTime().Add(4*time.Minute))
	requireCode(t, err, domain.ErrorRevisionConflict)

	failed, err := running.Fail(running.Revision(), baseTime().Add(4*time.Minute))
	if err != nil {
		t.Fatalf("Fail() error = %v", err)
	}
	if failed.State() != domain.WorkItemStateFailed || failed.Revision() != 3 || !failed.IsTerminal() {
		t.Fatalf("failed state/revision/terminal = %q/%d/%v", failed.State(), failed.Revision(), failed.IsTerminal())
	}
}

func TestWorkItemRejectsInvalidInputAndTransitionTimes(t *testing.T) {
	valid := domain.NewWorkItemInput{
		Ref:       mustRef(t, "work-item:valid", domain.NewWorkItemRef),
		Goal:      mustRef(t, "goal:valid", domain.NewGoalRef),
		Actor:     mustRef(t, "actor:valid", domain.NewActorRef),
		Project:   mustRef(t, "project:valid", domain.NewProjectRef),
		Objective: "perform the unit of work",
		CreatedAt: baseTime().Add(2 * time.Minute),
	}
	tests := []struct {
		name  string
		input domain.NewWorkItemInput
		code  domain.ErrorCode
	}{
		{name: "work item ref", input: withWorkItemRef(valid, domain.WorkItemRef{}), code: domain.ErrorInvalidRef},
		{name: "goal ref", input: withWorkItemGoal(valid, domain.GoalRef{}), code: domain.ErrorInvalidRef},
		{name: "actor ref", input: withWorkItemActor(valid, domain.ActorRef{}), code: domain.ErrorInvalidRef},
		{name: "project ref", input: withWorkItemProject(valid, domain.ProjectRef{}), code: domain.ErrorInvalidRef},
		{name: "blank objective", input: withObjective(valid, "\t"), code: domain.ErrorInvalidArgument},
		{name: "zero created at", input: withWorkItemCreatedAt(valid, time.Time{}), code: domain.ErrorInvalidArgument},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := domain.NewWorkItem(test.input)
			requireCode(t, err, test.code)
		})
	}

	item, err := domain.NewWorkItem(valid)
	if err != nil {
		t.Fatalf("NewWorkItem(valid) error = %v", err)
	}
	_, err = item.Start(item.Revision(), domain.ExecutionRef{}, baseTime().Add(3*time.Minute))
	requireCode(t, err, domain.ErrorInvalidRef)
	_, err = item.Start(
		item.Revision(),
		mustRef(t, "execution:early", domain.NewExecutionRef),
		baseTime().Add(time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidArgument)

	running, err := item.Start(
		item.Revision(),
		mustRef(t, "execution:valid", domain.NewExecutionRef),
		baseTime().Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("Start(valid) error = %v", err)
	}
	_, err = running.Fail(running.Revision(), baseTime().Add(2*time.Minute))
	requireCode(t, err, domain.ErrorInvalidArgument)
	_, err = running.Succeed(
		running.Revision(),
		[]domain.ArtifactRef{{}},
		[]domain.AttestationRef{mustRef(t, "attestation:valid", domain.NewAttestationRef)},
		baseTime().Add(4*time.Minute),
	)
	requireCode(t, err, domain.ErrorInvalidRef)
}

func TestWorkItemGovernanceDefaultsAndValidationAreTyped(t *testing.T) {
	item := newWorkItem(t, "critical xhigh provider-derived are only objective data")
	if item.BudgetDemand() != (governance.BudgetDemand{Ref: "budget-demand:" + item.Ref().String()}) ||
		item.SecurityCriticality() != governance.SecurityCriticalityNormal ||
		item.ReasoningEffort() != governance.ReasoningEffortMedium {
		t.Fatalf("governance defaults = %+v/%q/%q", item.BudgetDemand(), item.SecurityCriticality(), item.ReasoningEffort())
	}

	valid := domain.NewWorkItemInput{
		Ref:       mustRef(t, "work-item:governance-validation", domain.NewWorkItemRef),
		Goal:      mustRef(t, "goal:governance-validation", domain.NewGoalRef),
		Actor:     mustRef(t, "actor:governance-validation", domain.NewActorRef),
		Project:   mustRef(t, "project:governance-validation", domain.NewProjectRef),
		Objective: "validate declared governance", CreatedAt: baseTime().Add(2 * time.Minute),
		BudgetDemand:        governance.BudgetDemand{Ref: "budget-demand:governance-validation"},
		SecurityCriticality: governance.SecurityCriticalitySensitive,
		ReasoningEffort:     governance.ReasoningEffortHigh,
	}
	for _, test := range []struct {
		name   string
		mutate func(*domain.NewWorkItemInput)
	}{
		{"demand ref", func(input *domain.NewWorkItemInput) { input.BudgetDemand.Ref = " bad" }},
		{"negative demand", func(input *domain.NewWorkItemInput) { input.BudgetDemand.Resources.Tokens = -1 }},
		{"criticality", func(input *domain.NewWorkItemInput) { input.SecurityCriticality = "provider-derived" }},
		{"effort", func(input *domain.NewWorkItemInput) { input.ReasoningEffort = "auto" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			test.mutate(&candidate)
			_, err := domain.NewWorkItem(candidate)
			requireCode(t, err, domain.ErrorInvalidPlan)
		})
	}
}

func newWorkItem(t *testing.T, objective string) domain.WorkItem {
	t.Helper()
	item, err := domain.NewWorkItem(domain.NewWorkItemInput{
		Ref:       mustRef(t, "work-item:001", domain.NewWorkItemRef),
		Goal:      mustRef(t, "goal:001", domain.NewGoalRef),
		Actor:     mustRef(t, "actor:local", domain.NewActorRef),
		Project:   mustRef(t, "project:orquesta", domain.NewProjectRef),
		Objective: objective,
		CreatedAt: baseTime().Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("NewWorkItem() error = %v", err)
	}
	return item
}

func withWorkItemRef(input domain.NewWorkItemInput, ref domain.WorkItemRef) domain.NewWorkItemInput {
	input.Ref = ref
	return input
}

func withWorkItemGoal(input domain.NewWorkItemInput, ref domain.GoalRef) domain.NewWorkItemInput {
	input.Goal = ref
	return input
}

func withWorkItemActor(input domain.NewWorkItemInput, ref domain.ActorRef) domain.NewWorkItemInput {
	input.Actor = ref
	return input
}

func withWorkItemProject(input domain.NewWorkItemInput, ref domain.ProjectRef) domain.NewWorkItemInput {
	input.Project = ref
	return input
}

func withObjective(input domain.NewWorkItemInput, objective string) domain.NewWorkItemInput {
	input.Objective = objective
	return input
}

func withWorkItemCreatedAt(input domain.NewWorkItemInput, createdAt time.Time) domain.NewWorkItemInput {
	input.CreatedAt = createdAt
	return input
}
