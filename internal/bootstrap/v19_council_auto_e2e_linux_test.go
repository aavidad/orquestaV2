//go:build linux && v19_real_e2e

package bootstrap

import (
	"context"
	"reflect"
	"testing"

	"orquesta/internal/application"
	"orquesta/internal/council"
	"orquesta/internal/goal"
	"orquesta/internal/ports"
)

func TestV19CouncilAutoSQLiteFilesystemRestartE2E(t *testing.T) {
	if !v19RealGate(t, "TestV19CouncilAutoSQLiteFilesystemRestartE2E") {
		return
	}
	h := newV19CouncilHarness(t, council.PolicyAuto, map[council.Role]council.Ballot{
		council.RoleProposer: council.BallotAccept, council.RoleCritic: council.BallotAccept, council.RoleArbiter: council.BallotReject,
	})
	defer h.shutdown(t)
	ref, before := h.submit(t, "request:v19-auto"), h.target(t)
	h.driveApprovedGate(t, ref)
	record := h.get(t, ref)
	if len(record.CouncilRounds) != 1 || v19CouncilExecutions(record) != 3 || len(record.CouncilFacts) != 0 {
		t.Fatalf("auto opening=%+v", record)
	}
	h.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	record = h.get(t, ref)
	if len(record.CouncilDecisions) != 1 || record.CouncilDecisions[0].Decision.Outcome != council.OutcomeAccepted || len(record.CouncilDecisions[0].Decision.Dissent) != 1 {
		t.Fatalf("auto decision=%+v", record.CouncilDecisions)
	}
	change := record.ChangeSets[0]
	if _, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), h.base.access, application.SkipCouncilRequest{RequestRef: "skip:v19-auto-post-round", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: record.Goal.WorkItems()[0].Revision(), Reason: "must not replace an auto Council decision"}); err == nil {
		t.Fatal("post-round auto Council skip accepted")
	}
	admitted, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "request:v19-auto-integrate", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: before})
	if err != nil || !admitted.Created || admitted.Action.CouncilResolution == nil {
		t.Fatalf("auto integration=%+v err=%v", admitted, err)
	}
	h.restart(t)
	replay, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "request:v19-auto-integrate", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: before})
	if err != nil || replay.Created || !reflect.DeepEqual(replay.Action, admitted.Action) {
		t.Fatalf("auto replay=%+v err=%v", replay, err)
	}
	h.process(t, application.ActionIntegrateChange)
	if closed := h.get(t, ref); closed.Goal.State() != goal.GoalStateSucceeded || h.target(t) == before {
		t.Fatalf("auto close=%+v", closed.Goal)
	}
	h.restart(t)
	if replayed := h.get(t, ref); replayed.Goal.State() != goal.GoalStateSucceeded {
		t.Fatalf("auto restart terminal=%s", replayed.Goal.State())
	}
	if idle, err := h.base.runtime.Orchestrator().ProcessNext(context.Background(), "worker:v19-auto-idle"); err != nil || idle.Processed {
		t.Fatalf("auto restart duplicated work=%+v err=%v", idle, err)
	}
	cross, _ := ports.NewChangeSetRef("change-set:cross-substitution")
	if _, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "request:v19-auto-cross", GoalRef: ref, ChangeRef: cross, ExpectedTargetOID: before}); err == nil {
		t.Fatal("cross change substitution accepted")
	}
}

func v19CouncilExecutions(record application.GoalRecord) int {
	count := 0
	for _, execution := range record.Executions {
		if execution.CouncilSubjectDigest != "" {
			count++
		}
	}
	return count
}
