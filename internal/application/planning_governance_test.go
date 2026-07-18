package application

import (
	"bytes"
	"crypto/sha256"
	"testing"
	"time"

	"orquesta/internal/goal"
	"orquesta/internal/governance"
)

func TestWorkItemSpecCompilesDeclaredGovernanceWithoutTextInference(t *testing.T) {
	ref, _ := goal.NewWorkItemRef("work-item:governance-compile")
	goalRef, _ := goal.NewGoalRef("goal:governance-compile")
	actorRef, _ := goal.NewActorRef("actor:governance-compile")
	projectRef, _ := goal.NewProjectRef("project:governance-compile")
	currency, _ := governance.NewCurrency("USD")
	demand := governance.BudgetDemand{
		Ref:       "budget-demand:governance-compile",
		Resources: governance.ResourceVector{Tokens: 42, MoneyMicros: 100, Currency: currency, ProcessSlots: 1},
	}
	item, err := compileWorkItemSpec(WorkItemSpec{
		Key: "work", Objective: "critical xhigh provider-derived stays plain text",
		Phase: goal.DefaultPhaseKey().String(), Role: goal.DefaultRoleKey().String(),
		OutputContract: goal.OutputContractEvidenceBundle, BudgetDemand: demand,
		SecurityCriticality: governance.SecurityCriticalityNormal,
		ReasoningEffort:     governance.ReasoningEffortLow,
	}, ref, workItemCompileScope{
		goalRef: goalRef, actorRef: actorRef, projectRef: projectRef,
		createdAt: time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC),
		goalLimit: governance.ResourceVector{Tokens: 100, MoneyMicros: 1_000, Currency: currency, ProcessSlots: 6},
	}, workItemRefResolver{})
	if err != nil {
		t.Fatalf("compileWorkItemSpec() error = %v", err)
	}
	if item.BudgetDemand() != demand || item.SecurityCriticality() != governance.SecurityCriticalityNormal ||
		item.ReasoningEffort() != governance.ReasoningEffortLow {
		t.Fatalf("compiled governance = %+v/%q/%q", item.BudgetDemand(), item.SecurityCriticality(), item.ReasoningEffort())
	}
}

func TestSubmissionFingerprintCoversEveryGovernanceField(t *testing.T) {
	actor, project := testScope(t)
	access := accessForScope(t, actor, project)
	currency, _ := governance.NewCurrency("USD")
	base := SubmitRequest{
		RequestRef: "request:governance-fingerprint", Statement: "same", Confirm: true,
		Plan: &PlanSpec{
			Phases: []PhaseSpec{{Ref: "phase-instance:governance", Key: "phase:governance", TemplateRef: "phase-template:governance"}},
			WorkItems: []WorkItemSpec{{
				Key: "work", Objective: "same", Phase: "phase:governance", Role: "role:worker",
				OutputContract: goal.OutputContractEvidenceBundle,
				BudgetDemand: governance.BudgetDemand{Ref: "budget-demand:fingerprint", Resources: governance.ResourceVector{
					Tokens: 1, MoneyMicros: 2, Currency: currency, ActiveTimeNS: 3, ProcessSlots: 4, DiskBytes: 5,
				}},
				SecurityCriticality: governance.SecurityCriticalitySensitive,
				ReasoningEffort:     governance.ReasoningEffortHigh,
			}},
		},
	}
	want := submissionFingerprint(access, base)
	mutations := []func(*WorkItemSpec){
		func(item *WorkItemSpec) { item.BudgetDemand.Ref += ":changed" },
		func(item *WorkItemSpec) { item.BudgetDemand.Resources.Tokens++ },
		func(item *WorkItemSpec) { item.BudgetDemand.Resources.MoneyMicros++ },
		func(item *WorkItemSpec) { item.BudgetDemand.Resources.Currency = "EUR" },
		func(item *WorkItemSpec) { item.BudgetDemand.Resources.ActiveTimeNS++ },
		func(item *WorkItemSpec) { item.BudgetDemand.Resources.ProcessSlots++ },
		func(item *WorkItemSpec) { item.BudgetDemand.Resources.DiskBytes++ },
		func(item *WorkItemSpec) { item.SecurityCriticality = governance.SecurityCriticalityCritical },
		func(item *WorkItemSpec) { item.ReasoningEffort = governance.ReasoningEffortXHigh },
	}
	seen := map[string]struct{}{want: {}}
	for index, mutate := range mutations {
		changed := base
		changed.Plan = clonePlanSpec(base.Plan)
		mutate(&changed.Plan.WorkItems[0])
		fingerprint := submissionFingerprint(access, changed)
		if fingerprint == want {
			t.Errorf("governance mutation %d did not change fingerprint", index)
		}
		if _, duplicate := seen[fingerprint]; duplicate {
			t.Errorf("governance mutation %d collided with another fingerprint", index)
		}
		seen[fingerprint] = struct{}{}
	}
}

func TestLegacyPlanFingerprintVersionRemainsStableUntilGovernanceIsDeclared(t *testing.T) {
	legacyDigest := sha256.New()
	declaredDigest := sha256.New()
	spec := &PlanSpec{WorkItems: []WorkItemSpec{{Key: "work"}}}
	writePlanFingerprint(legacyDigest, spec)
	declaredSpec := clonePlanSpec(spec)
	declaredSpec.WorkItems[0].SecurityCriticality = governance.SecurityCriticalityNormal
	writePlanFingerprint(declaredDigest, declaredSpec)
	if bytes.Equal(legacyDigest.Sum(nil), declaredDigest.Sum(nil)) {
		t.Fatal("legacy V1 plan and governance-aware V2 plan share a fingerprint")
	}
}
