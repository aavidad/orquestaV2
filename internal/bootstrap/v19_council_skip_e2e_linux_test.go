//go:build linux && v19_real_e2e

package bootstrap

import (
	"context"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/identity"
)

func TestV19CouncilSkipHumanReplayAndIntegrationE2E(t *testing.T) {
	if !v19RealGate(t, "TestV19CouncilSkipHumanReplayAndIntegrationE2E") {
		return
	}
	h := newV19CouncilHarness(t, council.PolicySkipByOperator, nil)
	defer h.shutdown(t)
	ref, before := h.submit(t, "request:v19-skip"), h.target(t)
	h.driveApprovedGate(t, ref)
	record := h.get(t, ref)
	change, item := record.ChangeSets[0], record.Goal.WorkItems()[0]
	if len(record.Reviews) != 2 || len(record.Attestations) < 1 {
		t.Fatalf("skip lost approved V18 gate: reviews=%d attestations=%d", len(record.Reviews), len(record.Attestations))
	}
	if _, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "integrate:v19-skip-before-resolution", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: before}); err == nil {
		t.Fatal("skip-policy integration admitted without human Council resolution")
	}
	serviceRef, _ := identity.NewPrincipalRef("principal:v19-skip-service")
	serviceActor, _ := goal.NewActorRef("actor:v19-skip-service")
	service, _ := identity.NewPrincipal(serviceRef, serviceActor, identity.PrincipalKindService, "v19")
	serviceAccess, _ := application.NewAccess(service, record.Goal.Project())
	if _, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), serviceAccess, application.SkipCouncilRequest{RequestRef: "skip:v19-service", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), Reason: "service cannot skip"}); err == nil {
		t.Fatal("nonhuman skip accepted")
	}
	request := application.SkipCouncilRequest{RequestRef: "skip:v19", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), Reason: "human approved documented exception"}
	skipped, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), h.base.access, request)
	if err != nil || !skipped.Created {
		t.Fatalf("skip=%+v err=%v", skipped, err)
	}
	replay, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), h.base.access, request)
	if err != nil || replay.Created || replay.Skip != skipped.Skip {
		t.Fatalf("skip replay=%+v err=%v", replay, err)
	}
	request.Reason = "different reason"
	if _, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), h.base.access, request); err == nil {
		t.Fatal("skip replay mismatch accepted")
	}
	foreignProject, _ := goal.NewProjectRef("project:v19-skip-foreign")
	foreignRef, _ := identity.NewPrincipalRef("principal:v19-skip-foreign")
	foreignActor, _ := goal.NewActorRef("actor:v19-skip-foreign")
	foreignPrincipal, _ := identity.NewPrincipal(foreignRef, foreignActor, identity.PrincipalKindHuman, "v19")
	foreignAccess, _ := application.NewAccess(foreignPrincipal, foreignProject)
	if _, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), foreignAccess, application.SkipCouncilRequest{RequestRef: "skip:v19-foreign", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: item.Revision(), Reason: "foreign project"}); err == nil {
		t.Fatal("cross-project skip accepted")
	}
	if _, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), h.base.access, application.SkipCouncilRequest{RequestRef: "skip:v19-stale", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision() + 1, ExpectedItemRevision: item.Revision(), Reason: "stale generation"}); err == nil {
		t.Fatal("stale skip generation accepted")
	}
	record = h.get(t, ref)
	if v19CouncilExecutions(record) != 0 || len(record.CouncilRounds) != 0 || len(record.CouncilSkips) != 1 {
		t.Fatalf("skip launches/rounds=%+v", record)
	}
	admitted, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "integrate:v19-skip", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: before})
	if err != nil || admitted.Action.CouncilResolution == nil || admitted.Action.CouncilResolution.SkipRef == "" {
		t.Fatalf("skip integration=%+v err=%v", admitted, err)
	}
	h.process(t, application.ActionIntegrateChange)
	if closed := h.get(t, ref); closed.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("skip close=%s", closed.Goal.State())
	}
}
