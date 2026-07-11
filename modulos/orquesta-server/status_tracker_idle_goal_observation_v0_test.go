package orquestaserver

import (
	"testing"
	"time"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

func TestStatusTrackerIdleGoalObservationWithoutClosureBlocksIndependentAttestationV0(t *testing.T) {
	tracker := trackerWithIdleGoalSpecForObservationTestV0(t, idleGoalSpecForObservationTestV0(false))

	state := tracker.MarkIdleSelfImprovementGoalObservedV0(idleGoalCompleteResultForObservationTestV0(), nil, time.Now().UTC())

	assertIdleGoalClosureMissingAttestationV0(t, state)
}

func TestStatusTrackerIdleGoalObservationWithoutClosureBlocksRequiredAcceptanceCriteriaV0(t *testing.T) {
	spec := idleGoalSpecForObservationTestV0(true)
	spec.ClosurePolicy.RequireIndependentRequiredTestAttestation = false
	tracker := trackerWithIdleGoalSpecForObservationTestV0(t, spec)

	state := tracker.MarkIdleSelfImprovementGoalObservedV0(idleGoalCompleteResultForObservationTestV0(), nil, time.Now().UTC())

	assertIdleGoalClosureMissingAttestationV0(t, state)
}

func TestStatusTrackerIdleGoalObservationWithoutClosureKeepsLegacyFallbackV0(t *testing.T) {
	spec := idleGoalSpecForObservationTestV0(false)
	spec.ClosurePolicy = orquestagoal.GoalClosurePolicyV0{}
	tracker := trackerWithIdleGoalSpecForObservationTestV0(t, spec)

	state := tracker.MarkIdleSelfImprovementGoalObservedV0(idleGoalCompleteResultForObservationTestV0(), nil, time.Now().UTC())

	if state.IdleSelfImprovementGoalClosure == nil || !state.IdleSelfImprovementGoalClosure.Accepted ||
		state.IdleSelfImprovementGoalClosure.NeedsRework ||
		state.IdleSelfImprovementOperationalMessage == nil ||
		state.IdleSelfImprovementOperationalMessage.ReasonCode != idleSelfImprovementGoalClosureAcceptedReasonV0 {
		t.Fatalf("state=%+v", state)
	}
}

func trackerWithIdleGoalSpecForObservationTestV0(t *testing.T, spec orquestagoal.GoalWorkSpecV0) *StatusTrackerV0 {
	t.Helper()
	tracker := NewStatusTrackerV0(ConfigV0{}, time.Now().UTC())
	tracker.updateV0(func(state *StateV0) {
		goalSpec := copyGoalWorkSpecForServerStateV0(spec)
		state.IdleSelfImprovementGoalSpec = &goalSpec
	})
	return tracker
}

func idleGoalSpecForObservationTestV0(requiredAcceptanceCriteria bool) orquestagoal.GoalWorkSpecV0 {
	writeSet := []orquestagoal.GoalWriteScopeV0{{Path: "modulos/orquesta-server"}}
	spec := orquestagoal.GoalWorkSpecV0{
		GoalRef:        "goal-ref-idle-observation-001",
		Objective:      "Validate observed closure before accepting an idle goal.",
		DirectorKind:   orquestagoal.GoalDirectorKindRuntimeGoalV0,
		WriteSet:       writeSet,
		WriteSetSHA256: orquestagoal.GoalWriteSetSHA256V0(writeSet),
		RequiredTests: []orquestagoal.GoalRequiredTestV0{orquestagoal.FreezeGoalRequiredTestV0(orquestagoal.GoalRequiredTestV0{
			TestRef:    "test-ref-idle-observation-001",
			CommandRef: "command-ref-idle-observation-001",
			Command:    "go test ./modulos/orquesta-server",
		})},
		ClosurePolicy: orquestagoal.GoalClosurePolicyV0{
			RequireIndependentRequiredTestAttestation: true,
		},
	}
	if requiredAcceptanceCriteria {
		spec.RequiredTests[0].AcceptanceCriteriaRefs = []string{"criterion-ref-idle-observation-001"}
		spec.RequiredTests[0] = orquestagoal.FreezeGoalRequiredTestV0(spec.RequiredTests[0])
		spec.ClosurePolicy.RequiredAcceptanceCriteriaRefs = []string{"criterion-ref-idle-observation-001"}
	}
	return spec
}

func idleGoalCompleteResultForObservationTestV0() orquestagoal.GoalWorkResultV0 {
	return orquestagoal.GoalWorkResultV0{
		Status:  orquestagoal.GoalStatusCompleteV0,
		GoalRef: "goal-ref-idle-observation-001",
		Summary: "Goal completed according to its own result.",
	}
}

func assertIdleGoalClosureMissingAttestationV0(t *testing.T, state StateV0) {
	t.Helper()
	if state.IdleSelfImprovementGoalClosure == nil ||
		state.IdleSelfImprovementGoalClosure.Status != orquestagoal.GoalStatusBlockedV0 ||
		state.IdleSelfImprovementGoalClosure.Accepted ||
		!state.IdleSelfImprovementGoalClosure.NeedsRework ||
		!idleGoalClosureHasIssueCodeForObservationTestV0(*state.IdleSelfImprovementGoalClosure, orquestagoal.ErrGoalRequiredTestAttestationMissingV0) {
		t.Fatalf("state=%+v", state)
	}
}

func idleGoalClosureHasIssueCodeForObservationTestV0(closure orquestagoal.GoalClosureValidationV0, code string) bool {
	for _, issue := range closure.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
