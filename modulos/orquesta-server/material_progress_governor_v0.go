package orquestaserver

import (
	"context"
	"strings"
	"time"

	orquestaautoprogramming "orquesta/modulos/orquesta-autoprogramming"
	orquestagoal "orquesta/modulos/orquesta-goal"
)

const (
	materialProgressGovernedEvidenceV0 = "evidence-ref-material-progress-governed-v0"
	materialProgressWarningCodeV0      = "material_progress_warning"
	materialProgressReplanCodeV0       = "material_progress_replan_required"
	materialProgressHardStopCodeV0     = "material_progress_hard_stop_required"
	materialProgressNoDiffStopCodeV0   = "material_progress_no_diff_stop_required"
)

func (runtime *RuntimeV0) reconcileMaterialProgressV0(
	ctx context.Context,
	result orquestagoal.GoalWorkObserveActiveResultV0,
) orquestagoal.GoalWorkObserveActiveResultV0 {
	if runtime == nil || runtime.goalStateStore == nil || runtime.materialProgressStore == nil || runtime.materialProgressEvidence == nil {
		return result
	}
	for index := range result.Observations {
		observation, handled := runtime.governMaterialProgressObservationV0(ctx, result.Observations[index])
		if !handled {
			continue
		}
		result.Observations[index] = observation
		result.EvidenceRefs = compactConfigStringsV0(append(result.EvidenceRefs, observation.EvidenceRefs...))
	}
	return result
}

func (runtime *RuntimeV0) governMaterialProgressObservationV0(
	ctx context.Context,
	observation orquestagoal.GoalWorkObserveResultV0,
) (orquestagoal.GoalWorkObserveResultV0, bool) {
	state := observation.State
	result := orquestagoal.NormalizeGoalWorkResultV0(observation.Result)
	usage := result.UsageObservation
	if strings.TrimSpace(result.Status) != orquestagoal.GoalStatusRunningV0 ||
		orquestagoal.GoalUsageObservationEmptyV0(usage) || usage.ObservedAt == "" {
		return observation, false
	}
	evidence, err := runtime.materialProgressEvidence.ClassifyMaterialProgressV0(ctx, orquestaautoprogramming.MaterialProgressEvidenceRequestV0{State: state, Result: result})
	if err != nil || !evidence.Verified {
		return observation, false
	}
	progress, expectedVersion, replay := runtime.nextMaterialProgressStateV0(ctx, state, result, evidence)
	validation := orquestaautoprogramming.ValidateMaterialProgressStateV0(progress)
	if !validation.Accepted {
		return observation, false
	}
	saved := validation.State
	if !replay {
		saved, err = runtime.materialProgressStore.CompareAndSwapMaterialProgressStateV0(ctx, expectedVersion, validation.State)
		if err != nil {
			return observation, true
		}
	}
	observation.EvidenceRefs = compactConfigStringsV0(append(observation.EvidenceRefs, materialProgressGovernedEvidenceV0, saved.LastCheckpointRef))
	switch saved.LastDecision.Action {
	case orquestaautoprogramming.MaterialProgressActionWarningV0:
		return materialProgressWarningObservationV0(observation, saved), true
	case orquestaautoprogramming.MaterialProgressActionReplanRequiredV0:
		if saved.LastCheckpoint.MaterialClass == orquestaautoprogramming.MaterialProgressClassNoneV0 {
			return runtime.stopForMaterialProgressV0(ctx, observation, saved, materialProgressNoDiffStopCodeV0, false), true
		}
		return runtime.stopForMaterialProgressV0(ctx, observation, saved, materialProgressReplanCodeV0, true), true
	case orquestaautoprogramming.MaterialProgressActionHardStopRequiredV0:
		return runtime.stopForMaterialProgressV0(ctx, observation, saved, materialProgressHardStopCodeV0, false), true
	default:
		return observation, true
	}
}

func (runtime *RuntimeV0) nextMaterialProgressStateV0(
	ctx context.Context,
	goalState orquestagoal.GoalWorkStateV0,
	result orquestagoal.GoalWorkResultV0,
	evidence orquestaautoprogramming.MaterialProgressEvidenceV0,
) (orquestaautoprogramming.MaterialProgressStateV0, uint64, bool) {
	policy := materialProgressPolicyFromConfigV0(runtime.config, goalState.Spec)
	segment := orquestaautoprogramming.MaterialProgressSegmentV0{
		StartSequence:          1,
		StartTokensAccumulated: result.UsageObservation.TokensAccumulated,
		ContextRevisionRef:     strings.TrimSpace(evidence.ContextRevisionRef),
		ReplansUsed:            materialProgressReplansUsedV0(goalState.Spec),
	}
	expectedVersion := uint64(0)
	sequence := int64(1)
	var current orquestaautoprogramming.MaterialProgressStateV0
	if loaded, err := runtime.materialProgressStore.LoadMaterialProgressStateV0(ctx, goalState.RunRef, goalState.GoalRef); err == nil {
		current = loaded
		policy = current.Policy
		segment = current.Segment
		expectedVersion = current.StoreVersion
		sequence = current.LastCheckpoint.Sequence + 1
	}
	checkpoint := orquestaautoprogramming.MaterialProgressCheckpointV0{
		Sequence: sequence, TokensAccumulated: result.UsageObservation.TokensAccumulated,
		ContextRevisionRef: strings.TrimSpace(evidence.ContextRevisionRef),
		MaterialClass:      evidence.MaterialClass, EvidenceRefs: compactConfigStringsV0(evidence.EvidenceRefs),
	}
	if expectedVersion > 0 && materialProgressCheckpointObservationEqualV0(current.LastCheckpoint, checkpoint) {
		return current, expectedVersion, true
	}
	decision := orquestaautoprogramming.DecideMaterialProgressV0(orquestaautoprogramming.MaterialProgressInputV0{
		Policy: policy, Segment: segment, Checkpoint: checkpoint,
	})
	observedAt, _ := time.Parse(time.RFC3339Nano, result.UsageObservation.ObservedAt)
	state := orquestaautoprogramming.MaterialProgressStateV0{
		SchemaVersion: orquestaautoprogramming.MaterialProgressStateSchemaVersionV0,
		StoreVersion:  expectedVersion + 1,
		RunRef:        goalState.RunRef, GoalRef: goalState.GoalRef, Policy: policy,
		Segment: decision.Segment, LastCheckpoint: checkpoint, LastDecision: decision,
		BaselineRef: strings.TrimSpace(evidence.BaselineRef), WriteSetSHA256: strings.TrimSpace(evidence.WriteSetSHA256),
		ContextRevisionRef: strings.TrimSpace(evidence.ContextRevisionRef), ObservedAt: observedAt,
		EvidenceRefs: compactConfigStringsV0(append(evidence.EvidenceRefs, result.UsageObservation.EvidenceRefs...)),
	}
	state.LastCheckpointRef = orquestaautoprogramming.MaterialProgressCheckpointRefV0(state.RunRef, state.GoalRef, state.BaselineRef, state.WriteSetSHA256, checkpoint)
	state.LastActionIdempotencyKey = orquestaautoprogramming.MaterialProgressActionIdempotencyKeyV0(state.RunRef, state.GoalRef, state.LastCheckpointRef, decision.Action)
	return state, expectedVersion, false
}

func materialProgressCheckpointObservationEqualV0(
	left orquestaautoprogramming.MaterialProgressCheckpointV0,
	right orquestaautoprogramming.MaterialProgressCheckpointV0,
) bool {
	if left.TokensAccumulated != right.TokensAccumulated || left.ContextRevisionRef != right.ContextRevisionRef ||
		left.MaterialClass != right.MaterialClass {
		return false
	}
	leftRefs := compactConfigStringsV0(left.EvidenceRefs)
	rightRefs := compactConfigStringsV0(right.EvidenceRefs)
	if len(leftRefs) != len(rightRefs) {
		return false
	}
	for _, leftRef := range leftRefs {
		found := false
		for _, rightRef := range rightRefs {
			if leftRef == rightRef {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func materialProgressPolicyFromConfigV0(config ConfigV0, spec orquestagoal.GoalWorkSpecV0) orquestaautoprogramming.MaterialProgressPolicyV0 {
	hard := config.AutoprogrammingGoalProgressPolicy.CheckpointOnlyHighConsumptionTokens
	if hard < 4 {
		hard = 100000
	}
	maxReplans := spec.ReworkPolicy.MaxReworkGoals
	if maxReplans <= 0 {
		maxReplans = spec.Budget.MaxReworkGoals
	}
	return orquestaautoprogramming.MaterialProgressPolicyV0{
		WarningAfterTokens: hard / 4, ReplanRequiredAfterTokens: hard / 2,
		HardStopRequiredAfterTokens: hard, MaxReplans: maxReplans,
	}
}

func materialProgressReplansUsedV0(spec orquestagoal.GoalWorkSpecV0) int {
	used := 0
	for _, ref := range spec.ContextRefs {
		if strings.TrimSpace(ref.Kind) == "source_goal" {
			used++
		}
	}
	return used
}

func materialProgressWarningObservationV0(observation orquestagoal.GoalWorkObserveResultV0, state orquestaautoprogramming.MaterialProgressStateV0) orquestagoal.GoalWorkObserveResultV0 {
	observation.Result.Issues = append(observation.Result.Issues, orquestagoal.GoalWorkIssueV0{Code: materialProgressWarningCodeV0, Field: "goal_progress"})
	observation.Result.EvidenceRefs = compactConfigStringsV0(append(observation.Result.EvidenceRefs, state.LastCheckpointRef))
	return observation
}

func goalObservationHasEvidenceV0(observation orquestagoal.GoalWorkObserveResultV0, expected string) bool {
	for _, ref := range append(append([]string{}, observation.EvidenceRefs...), observation.Result.EvidenceRefs...) {
		if strings.TrimSpace(ref) == expected {
			return true
		}
	}
	return false
}
