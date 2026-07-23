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
	h.process(t, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionLaunchAgent, application.ActionObserveAgent, application.ActionObserveAgent)
	if current := h.get(t, ref); len(current.CouncilDecisions) != 0 || len(current.CouncilFacts) != 2 {
		t.Fatalf("auto decided before third Council observation: facts=%+v decisions=%+v", current.CouncilFacts, current.CouncilDecisions)
	}
	h.process(t, application.ActionObserveAgent)
	record = h.get(t, ref)
	if len(record.CouncilDecisions) != 1 || record.CouncilDecisions[0].Decision.Outcome != council.OutcomeAccepted || len(record.CouncilDecisions[0].Decision.Dissent) != 1 {
		t.Fatalf("auto decision=%+v", record.CouncilDecisions)
	}
	v19RequireCouncilDeliveries(t, record)
	change := record.ChangeSets[0]
	if _, err := h.base.runtime.Orchestrator().SkipCouncil(context.Background(), h.base.access, application.SkipCouncilRequest{RequestRef: "skip:v19-auto-post-round", GoalRef: ref, ChangeRef: change.Ref, ExpectedGoalRevision: record.Goal.Revision(), ExpectedItemRevision: record.Goal.WorkItems()[0].Revision(), Reason: "must not replace an auto Council decision"}); err == nil {
		t.Fatal("post-round auto Council skip accepted")
	}
	admitted, err := h.base.runtime.Orchestrator().IntegrateChange(context.Background(), h.base.access, application.IntegrateChangeRequest{RequestRef: "request:v19-auto-integrate", GoalRef: ref, ChangeRef: change.Ref, ExpectedTargetOID: before})
	if err != nil || !admitted.Created || admitted.Action.CouncilResolution == nil ||
		admitted.Action.CouncilResolution.SubjectDigest != record.CouncilRounds[0].SubjectDigest ||
		admitted.Action.CouncilResolution.DecisionRef != record.CouncilDecisions[0].Ref ||
		admitted.Action.CouncilResolution.DecisionDigest != record.CouncilDecisions[0].DecisionDigest ||
		admitted.Action.CouncilResolution.SkipRef != "" || admitted.Action.CouncilResolution.SkipDigest != "" {
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

// Generation/launch substitution and malformed receipts stay adversarial unit
// contracts: TestPlanGenerationRejectsActionExecutionGenerationMismatch and
// TestMalformedLaunchReceiptReconcilesWithSameEffectKey.
func v19RequireCouncilDeliveries(t *testing.T, record application.GoalRecord) {
	t.Helper()
	if len(record.CouncilFacts) != 3 {
		t.Fatalf("Council facts=%+v", record.CouncilFacts)
	}
	executions := map[string]application.ExecutionRecord{}
	for _, execution := range record.Executions {
		if execution.CouncilSubjectDigest != "" {
			if execution.ExternalRef == "" || execution.LaunchReceiptRef == "" || execution.State != application.ExecutionSucceeded ||
				executions[execution.Ref.String()].Ref.String() != "" {
				t.Fatalf("Council execution=%+v", execution)
			}
			executions[execution.Ref.String()] = execution
		}
	}
	if len(executions) != 3 {
		t.Fatalf("Council executions=%+v", executions)
	}
	external, receipts, artifacts := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, fact := range record.CouncilFacts {
		execution, found := executions[fact.ExecutionRef]
		if !found || fact.LaunchReceiptRef != execution.LaunchReceiptRef || fact.ExternalRef != execution.ExternalRef ||
			fact.ArtifactRef == "" || fact.ArtifactDigest == "" || external[fact.ExternalRef] || receipts[fact.LaunchReceiptRef] || artifacts[fact.ArtifactRef] {
			t.Fatalf("Council fact=%+v", fact)
		}
		external[fact.ExternalRef], receipts[fact.LaunchReceiptRef], artifacts[fact.ArtifactRef] = true, true, true
	}
	stored := map[string]bool{}
	for _, artifact := range record.Artifacts {
		if artifact.Kind == application.ArtifactKindCouncilContribution {
			if _, found := executions[artifact.ExecutionRef.String()]; !found || artifact.Stored.MediaType != council.ContributionMediaType ||
				artifact.Stored.Ref.String() == "" || !artifacts[artifact.Stored.Ref.String()] || stored[artifact.Stored.Ref.String()] {
				t.Fatalf("Council artifact=%+v", artifact)
			}
			stored[artifact.Stored.Ref.String()] = true
		}
	}
	if len(stored) != 3 {
		t.Fatalf("Council stored artifacts=%+v", stored)
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
