package application

import (
	"context"
	"encoding/json"
	"sort"
	"testing"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
	"orquesta/internal/ports"
	"orquesta/internal/review"
)

func TestDirectorReviewChangesRequestedUsesPersistedHistoricalGate(t *testing.T) {
	system := newTestAttestationSystem(t, ports.TestAttestationPassed)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent)
	record := system.record(t)
	reviewers := make([]ExecutionRecord, 0, 2)
	for _, execution := range record.Executions {
		if isReviewerExecution(execution) {
			reviewers = append(reviewers, execution)
		}
	}
	sort.Slice(reviewers, func(left, right int) bool {
		return reviewers[left].Ref.String() < reviewers[right].Ref.String()
	})
	observations := make([]ports.AgentObservation, 0, 2)
	for _, execution := range reviewers {
		role, found := reviewerRole(execution)
		if !found {
			t.Fatalf("reviewer role missing: %+v", execution)
		}
		payload, err := json.Marshal(review.Artifact{
			SchemaVersion: 1, SubjectDigest: execution.ReviewSubjectDigest,
			Role: role, Verdict: review.VerdictChangesRequested,
			Summary: "material correction required",
			Findings: []review.Finding{{
				Code: "review.finding", Severity: review.SeverityHigh,
				EvidenceRef: "change:exact",
			}},
		})
		appTestNoError(t, err)
		observations = append(observations, ports.AgentObservation{
			Status: ports.AgentCompleted, MediaType: review.AssessmentMediaType, Content: payload,
		})
	}
	agent := system.orchestrator.launcher.(*scriptedAgent)
	agent.mu.Lock()
	agent.observations = observations
	agent.mu.Unlock()
	system.process(t, ActionObserveAgent, ActionObserveAgent)

	record = system.record(t)
	source := record.Goal.WorkItems()[0]
	author, found := authorBound(record, source)
	if !found || author.State != ExecutionFailed ||
		author.FailureCode != string(goal.ReplanCauseReviewChangesRequested) {
		t.Fatalf("review changes author=%+v source=%+v", author, source)
	}
	principal, project, _ := system.access.values()
	system.orchestrator.access.(*memoryAccessRepository).setRole(
		principal.Ref, project, identity.RoleProjectOwner,
	)
	lease, err := system.orchestrator.ClaimDirector(context.Background(), system.access, ClaimDirectorRequest{
		RequestRef: "director-claim:review-changes", GoalRef: record.Goal.Ref(),
	})
	appTestNoError(t, err)
	result, err := system.orchestrator.ProposeDirectorPlan(context.Background(), system.access, ProposeDirectorPlanRequest{
		RequestRef: "director-plan:review-changes", GoalRef: record.Goal.Ref(),
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence,
		Cause: goal.ReplanCauseReviewChangesRequested, SourceWorkItemRef: source.Ref(),
		ExpectedWorkItemRevision: source.Revision(), SourceExecutionRef: author.Ref,
		SourceExecutionAttempt: author.AttemptNo, Reason: "repair exact review findings",
		Plan: PlanSpec{WorkItems: []WorkItemSpec{{
			Key: "review-successor", Objective: "repair exact review findings",
			Phase: source.Phase().String(), Role: source.Role().String(),
			WriteSet: []string{"internal/workspace"}, CouncilPolicy: council.PolicyAuto,
			RequiredTests:  requiredTestSpecs("required-test:review-replan"),
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	})
	if err != nil || !result.Created {
		t.Fatalf("review replan result=%+v err=%v", result, err)
	}
	after := system.record(t)
	updated, _ := after.Goal.WorkItem(source.Ref())
	successor := after.Goal.WorkItems()[len(after.Goal.WorkItems())-1]
	reworkOf, linked := successor.ReworkOf()
	if updated.State() != goal.WorkItemStateSuperseded || !linked || reworkOf != source.Ref() {
		t.Fatalf("review replan source=%+v successor=%+v", updated, successor)
	}
}
