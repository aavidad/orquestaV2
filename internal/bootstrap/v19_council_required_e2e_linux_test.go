//go:build linux && v19_real_e2e

package bootstrap

import (
	"context"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
)

func TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E(t *testing.T) {
	if !v19RealGate(t, "TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E") {
		return
	}
	h := newV19CouncilHarness(t, council.PolicyRequired, map[council.Role]council.Ballot{
		council.RoleProposer: council.BallotAccept, council.RoleCritic: council.BallotAccept, council.RoleArbiter: council.BallotSecurityVeto,
	})
	defer h.shutdown(t)
	ref := h.submit(t, "request:v19-required")
	h.driveApprovedGate(t, ref)
	record := h.get(t, ref)
	if len(record.CouncilRounds) != 0 || v19CouncilExecutions(record) != 0 {
		t.Fatalf("required auto-opened=%+v", record.CouncilRounds)
	}
	if _, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "integrate:v19-required-pre-open", GoalRef: ref, ChangeRef: record.ChangeSets[0].Ref, ExpectedTargetOID: h.target(t)}); err == nil {
		t.Fatal("required integration admitted before Council resolution")
	}
	lease, err := h.base.runtime.Orchestrator().ClaimDirector(context.Background(), h.base.access, application.ClaimDirectorRequest{RequestRef: "claim:v19-required", GoalRef: ref})
	if err != nil {
		t.Fatal(err)
	}
	change, item := record.ChangeSets[0], record.Goal.WorkItems()[0]
	_, err = h.base.runtime.Orchestrator().OpenCouncilRound(context.Background(), h.base.access, application.OpenCouncilRoundRequest{RequestRef: "open:v19-required-stale", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence + 1})
	if err == nil {
		t.Fatal("stale required open accepted")
	}
	opened, err := h.base.runtime.Orchestrator().OpenCouncilRound(context.Background(), h.base.access, application.OpenCouncilRoundRequest{RequestRef: "open:v19-required", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), LeaseToken: lease.Lease.Token, LeaseFence: lease.Lease.Fence})
	if err != nil || !opened.Created {
		t.Fatalf("required open=%+v err=%v", opened, err)
	}
	h.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	if current := h.get(t, ref); len(current.CouncilDecisions) != 0 {
		t.Fatalf("veto decided early=%+v", current.CouncilDecisions)
	}
	h.process(t, application.ActionObserveAgent)
	if current := h.get(t, ref); len(current.CouncilDecisions) != 1 || current.CouncilDecisions[0].Decision.Outcome != council.OutcomeBlockedSecurity ||
		!v19CouncilFactHasBallot(current, council.RoleArbiter, council.BallotSecurityVeto) {
		t.Fatalf("veto final=%+v", current.CouncilDecisions)
	}
	h.restart(t)
	persisted := h.get(t, ref)
	if len(persisted.CouncilDecisions) != 1 || persisted.CouncilDecisions[0].Decision.Outcome != council.OutcomeBlockedSecurity {
		t.Fatalf("veto restart=%+v", persisted.CouncilDecisions)
	}
	if _, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "integrate:v19-veto", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: h.target(t)}); err == nil {
		t.Fatal("veto admitted integration")
	}
	v19ReplanBlockedCouncil(t, h, ref, lease.Lease, persisted)
	second := newV19CouncilHarness(t, council.PolicyAuto, map[council.Role]council.Ballot{council.RoleProposer: council.BallotAccept, council.RoleCritic: council.BallotReject, council.RoleArbiter: council.BallotAbstain})
	defer second.shutdown(t)
	secondRef := second.submit(t, "request:v19-no-consensus")
	second.driveApprovedGate(t, secondRef)
	second.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	if result := second.get(t, secondRef); len(result.CouncilDecisions) != 1 || result.CouncilDecisions[0].Decision.Outcome != council.OutcomeNoConsensus || len(result.CouncilDecisions[0].Decision.Dissent) != 3 {
		t.Fatalf("second Council no-consensus=%+v", result.CouncilDecisions)
	}
}

func v19CouncilFactHasBallot(record application.GoalRecord, role council.Role, ballot council.Ballot) bool {
	for _, fact := range record.CouncilFacts {
		if fact.Role == role && fact.Ballot == ballot {
			return true
		}
	}
	return false
}

func v19ReplanBlockedCouncil(t *testing.T, h *v19CouncilHarness, ref goal.GoalRef, lease application.DirectorLeaseRecord, record application.GoalRecord) {
	t.Helper()
	source := record.Goal.WorkItems()[0]
	authorRef, bound := source.Execution()
	author, found := v19Execution(record, authorRef)
	decision := record.CouncilDecisions[0]
	if !bound || !found || author.State != application.ExecutionAwaitingIntegration || decision.Decision.Outcome != council.OutcomeBlockedSecurity {
		t.Fatalf("blocked Council replan source=%+v author=%+v decision=%+v", source, author, decision)
	}
	request := application.ProposeDirectorPlanRequest{
		RequestRef: "replan:v19-blocked-security", GoalRef: ref,
		ExpectedGoalRevision: record.Goal.Revision(), ExpectedPlanGeneration: record.Goal.PlanGeneration(),
		LeaseToken: lease.Token, LeaseFence: lease.Fence, Cause: goal.ReplanCauseGovernanceDecision,
		SourceWorkItemRef: source.Ref(), ExpectedWorkItemRevision: source.Revision(),
		SourceExecutionRef: author.Ref, SourceExecutionAttempt: author.AttemptNo,
		CouncilSubjectDigest: decision.SubjectDigest, CouncilDecisionRef: decision.Ref, CouncilDecisionDigest: decision.DecisionDigest,
		Reason: "rework required after exact blocked security Council decision",
		Plan: application.PlanSpec{WorkItems: []application.WorkItemSpec{{
			Key: "rework-v19-blocked-security", Objective: "rework exact blocked security Council source",
			Phase: source.Phase().String(), Role: source.Role().String(), WriteSet: []string{"subject/rework"},
			CouncilPolicy:  council.PolicyRequired,
			RequiredTests:  []application.RequiredTestSpec{{Ref: "required-test:v19-rework", ToolRef: "tool:go", Arguments: []string{"test", "-buildvcs=false", "./...", "-count=1"}, WorkingDirectory: "subject"}},
			OutputContract: goal.OutputContractEvidenceBundle,
		}}},
	}
	result, err := h.base.runtime.Orchestrator().ProposeDirectorPlan(context.Background(), h.base.access, request)
	if err != nil || !result.Created || result.Decision.Cause != goal.ReplanCauseGovernanceDecision ||
		result.Decision.CouncilSubjectDigest != decision.SubjectDigest || result.Decision.CouncilDecisionRef != decision.Ref ||
		result.Decision.CouncilDecisionDigest != decision.DecisionDigest {
		t.Fatalf("blocked Council replan=%+v err=%v", result, err)
	}
	after := h.get(t, ref)
	updated, found := after.Goal.WorkItem(source.Ref())
	successor := after.Goal.WorkItems()[len(after.Goal.WorkItems())-1]
	reworkOf, linked := successor.ReworkOf()
	retired, retiredFound := v19Execution(after, author.Ref)
	if !found || updated.State() != goal.WorkItemStateSuperseded || after.Goal.PlanGeneration() != record.Goal.PlanGeneration()+1 ||
		!linked || reworkOf != source.Ref() || !retiredFound || retired.State != application.ExecutionCanceled ||
		retired.FailureCode != "application.execution_superseded" || len(after.IntegrationReceipts) != 0 {
		t.Fatalf("blocked Council replan durable state=%+v successor=%+v retired=%+v", after.Goal, successor, retired)
	}
	h.restart(t)
	replay, err := h.base.runtime.Orchestrator().ProposeDirectorPlan(context.Background(), h.base.access, request)
	if err != nil || replay.Created || replay.Decision != result.Decision {
		t.Fatalf("blocked Council replan restart replay=%+v err=%v", replay, err)
	}
}

func v19Execution(record application.GoalRecord, ref goal.ExecutionRef) (application.ExecutionRecord, bool) {
	for _, execution := range record.Executions {
		if execution.Ref == ref {
			return execution, true
		}
	}
	return application.ExecutionRecord{}, false
}
