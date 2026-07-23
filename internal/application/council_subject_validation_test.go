package application

import (
	"context"
	"testing"

	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestValidatePersistedCouncilSubjectRequiresExactV18GateAndGeneration(t *testing.T) {
	system := newCouncilSystem(t, council.PolicyAuto)
	system.processCommit(t)
	system.process(t, ActionAttestTest, ActionLaunchAgent, ActionLaunchAgent, ActionObserveAgent, ActionObserveAgent)
	record := system.record(t)
	if len(record.CouncilRounds) != 1 {
		t.Fatalf("Council rounds=%+v", record.CouncilRounds)
	}
	subject := record.CouncilRounds[0].Subject
	if err := ValidatePersistedCouncilSubject(record, subject); err != nil {
		t.Fatalf("ValidatePersistedCouncilSubject() error = %v", err)
	}

	fakeGate := subject
	fakeGate.ReviewGateDigest = "sha256:" + string(makeDigest('f'))
	if err := ValidatePersistedCouncilSubject(record, fakeGate); err == nil {
		t.Fatal("fake V18 gate was accepted")
	}

	fakeGeneration := subject
	fakeGeneration.PlanGeneration++
	if err := ValidatePersistedCouncilSubject(record, fakeGeneration); err == nil {
		t.Fatal("fake plan generation was accepted")
	}
}

func TestValidatePersistedCouncilSubjectKeepsIntegratedHistoricalSubject(t *testing.T) {
	system := newCouncilSystem(t, council.PolicySkipByOperator)
	system.processCommit(t)
	system.process(t, ActionAttestTest)
	system.approveReviews(t)
	record := system.record(t)
	if len(record.CouncilSkips) != 1 {
		t.Fatalf("Council skips=%+v", record.CouncilSkips)
	}
	subject := record.CouncilSkips[0].Subject
	item := record.Goal.WorkItems()[0]
	_, err := system.orchestrator.IntegrateChange(context.Background(), system.access, IntegrateChangeRequest{
		RequestRef: "request:integrate:council-historical", GoalRef: record.Goal.Ref(), ChangeRef: record.ChangeSets[0].Ref,
		ExpectedTargetOID: record.WorkspaceBindings[0].BaseOID,
	})
	appTestNoError(t, err)
	system.process(t, ActionIntegrateChange)
	after := system.record(t)
	if err := ValidatePersistedCouncilSubject(after, subject); err != nil {
		t.Fatalf("integrated historical subject rejected: %v (item=%+v)", err, item)
	}
}

func TestValidatePersistedCouncilSubjectKeepsSupersededHistoricalSubject(t *testing.T) {
	record, source, author, decision, system := councilDecisionRecord(t, council.OutcomeRejected)
	if len(record.CouncilRounds) != 1 {
		t.Fatalf("Council rounds=%+v", record.CouncilRounds)
	}
	subject := record.CouncilRounds[0].Subject
	principal, project, err := system.access.values()
	appTestNoError(t, err)
	system.orchestrator.access.(*memoryAccessRepository).setRole(principal.Ref, project, identity.RoleProjectOwner)
	lease, err := system.orchestrator.ClaimDirector(context.Background(), system.access, ClaimDirectorRequest{
		RequestRef: "director-claim:council-historical", GoalRef: record.Goal.Ref(),
	})
	appTestNoError(t, err)
	request := councilReplanRequest(record, source, author, decision)
	request.RequestRef, request.GoalRef = "director-plan:council-historical", record.Goal.Ref()
	request.ExpectedGoalRevision, request.ExpectedPlanGeneration = record.Goal.Revision(), record.Goal.PlanGeneration()
	request.LeaseToken, request.LeaseFence = lease.Lease.Token, lease.Lease.Fence
	request.Reason = "replan preserves historical Council subject"
	request.Plan = PlanSpec{WorkItems: []WorkItemSpec{{
		Key: "successor-council-historical", Objective: "rework after Council rejection",
		Phase: source.Phase().String(), Role: source.Role().String(), WriteSet: []string{"internal/workspace"},
		CouncilPolicy: council.PolicyRequired, RequiredTests: requiredTestSpecs("required-test:council-historical"),
		OutputContract: goal.OutputContractEvidenceBundle,
	}}}
	_, err = system.orchestrator.ProposeDirectorPlan(context.Background(), system.access, request)
	appTestNoError(t, err)
	if err := ValidatePersistedCouncilSubject(system.record(t), subject); err != nil {
		t.Fatalf("superseded historical subject rejected: %v", err)
	}
}
