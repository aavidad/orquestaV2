package application

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"orquesta/internal/goal"
)

func TestV38LogicalCohortsDeriveAndScheduleWithoutStaticCeiling(t *testing.T) {
	for _, size := range []int{1, 16, 70, 500} {
		t.Run(fmt.Sprintf("cohort_%d", size), func(t *testing.T) {
			clock := &mutableClock{now: time.Date(2026, 7, 30, 20, 0, 0, 0, time.UTC)}
			repository := newMemoryRepository()
			orchestrator, _ := newTestOrchestrator(
				t, repository, clock, &scriptedAgent{now: clock.Now},
			)
			actor, project := testScope(t)
			result, err := orchestrator.Submit(
				context.Background(),
				accessForScope(t, actor, project),
				SubmitRequest{
					RequestRef: fmt.Sprintf("request:v38-logical-cohort-%d", size),
					Statement:  fmt.Sprintf("derive logical cohort of %d work items", size),
					Confirm:    true,
					Plan:       logicalCohortPlan(size),
				},
			)
			if err != nil {
				t.Fatalf("Submit(cohort=%d) error = %v", size, err)
			}

			items := result.Record.Goal.WorkItems()
			runnable := result.Record.Goal.RunnableWorkItems()
			ready := result.Record.Goal.ReadyWorkItems()
			if len(items) != size || len(runnable) != size || len(ready) != size ||
				len(result.Record.Executions) != size {
				t.Fatalf(
					"cohort=%d items/runnable/ready/executions = %d/%d/%d/%d",
					size, len(items), len(runnable), len(ready), len(result.Record.Executions),
				)
			}
			for index := range items {
				execution := result.Record.Executions[index]
				if runnable[index].Ref() != items[index].Ref() ||
					ready[index].Ref() != items[index].Ref() ||
					execution.WorkItemRef != items[index].Ref() ||
					execution.PlanGeneration != result.Record.Goal.PlanGeneration() ||
					execution.AttemptNo != 1 {
					t.Fatalf(
						"cohort=%d index=%d lost order or generation: item=%q runnable=%q ready=%q execution=%+v",
						size, index, items[index].Ref(), runnable[index].Ref(), ready[index].Ref(), execution,
					)
				}
			}

			existing := append([]ExecutionRecord(nil), result.Record.Executions...)
			executions, actions, events, err := orchestrator.scheduleReady(
				context.Background(),
				result.Record.Goal,
				existing,
				result.Record.WorkItemAuthorities,
				orchestrator.budgetPolicy.effectPolicy(),
				clock.Now(),
			)
			if err != nil {
				t.Fatalf("scheduleReady(existing cohort=%d) error = %v", size, err)
			}
			if len(executions) != 0 || len(actions) != 0 || len(events) != 0 ||
				!reflect.DeepEqual(existing, result.Record.Executions) {
				t.Fatalf(
					"existing cohort=%d was duplicated or mutated: executions/actions/events=%d/%d/%d",
					size, len(executions), len(actions), len(events),
				)
			}

			// Este es el límite de admisión durable que pertenece a scheduleReady.
			// V38-A04 demostrará por separado la espera con capacidad física parcial.
			admitted := size / 2
			executions, actions, events, err = orchestrator.scheduleReady(
				context.Background(),
				result.Record.Goal,
				existing[:admitted],
				result.Record.WorkItemAuthorities,
				orchestrator.budgetPolicy.effectPolicy(),
				clock.Now(),
			)
			if err != nil {
				t.Fatalf("scheduleReady(partial cohort=%d) error = %v", size, err)
			}
			if len(executions) != size-admitted ||
				len(actions) != size-admitted ||
				len(events) != size-admitted {
				t.Fatalf(
					"partial cohort=%d admitted=%d scheduled executions/actions/events=%d/%d/%d",
					size, admitted, len(executions), len(actions), len(events),
				)
			}
			for index := range executions {
				want := items[admitted+index]
				if executions[index].WorkItemRef != want.Ref() ||
					executions[index].AttemptNo != 1 ||
					actions[index].WorkItemRef != want.Ref() ||
					actions[index].Kind != ActionLaunchAgent ||
					events[index].WorkItemRef != want.Ref() {
					t.Fatalf(
						"partial cohort=%d index=%d did not preserve pending work item %q",
						size, index, want.Ref(),
					)
				}
			}
		})
	}
}

func logicalCohortPlan(size int) *PlanSpec {
	items := make([]WorkItemSpec, size)
	for index := range items {
		items[index] = WorkItemSpec{
			Key:            fmt.Sprintf("work-%03d", index),
			Objective:      fmt.Sprintf("execute logical work item %03d", index),
			Phase:          "phase:v38-logical-cohort",
			Role:           "role:worker",
			OutputContract: goal.OutputContractEvidenceBundle,
		}
	}
	return &PlanSpec{
		Phases: []PhaseSpec{{
			Ref:         "phase-instance:v38-logical-cohort",
			Key:         "phase:v38-logical-cohort",
			TemplateRef: "phase-template:v38-logical-cohort",
		}},
		WorkItems: items,
	}
}
