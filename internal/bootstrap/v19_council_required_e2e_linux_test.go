//go:build linux && v19_real_e2e

package bootstrap

import (
	"context"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
)

func TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E(t *testing.T) {
	if !v19RealGate(t, "TestV19CouncilRequiredStaleOpenAndVetoWaitsThreeE2E") {
		return
	}
	h := newV19CouncilHarness(t, council.PolicyRequired, map[council.Role]council.Ballot{
		council.RoleProposer: council.BallotSecurityVeto, council.RoleCritic: council.BallotAccept, council.RoleArbiter: council.BallotAccept,
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
	if current := h.get(t, ref); len(current.CouncilDecisions) != 1 || current.CouncilDecisions[0].Decision.Outcome != council.OutcomeBlockedSecurity {
		t.Fatalf("veto final=%+v", current.CouncilDecisions)
	}
	h.restart(t)
	if _, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "integrate:v19-veto", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: h.target(t)}); err == nil {
		t.Fatal("veto admitted integration")
	}
	second := newV19CouncilHarness(t, council.PolicyAuto, map[council.Role]council.Ballot{council.RoleProposer: council.BallotAccept, council.RoleCritic: council.BallotReject, council.RoleArbiter: council.BallotAbstain})
	defer second.shutdown(t)
	secondRef := second.submit(t, "request:v19-no-consensus")
	second.driveApprovedGate(t, secondRef)
	second.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	if result := second.get(t, secondRef); len(result.CouncilDecisions) != 1 || result.CouncilDecisions[0].Decision.Outcome != council.OutcomeNoConsensus || len(result.CouncilDecisions[0].Decision.Dissent) != 3 {
		t.Fatalf("second Council no-consensus=%+v", result.CouncilDecisions)
	}
}
