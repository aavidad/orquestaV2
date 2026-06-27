package orquestaserver

import (
	"context"
	"strings"

	orquestagoal "orquesta/modulos/orquesta-goal"
)

type goalObservationFingerprintPlanV0 struct {
	Request      orquestagoal.GoalWorkObserveActiveRequestV0
	Pending      map[string]orquestagoal.GoalObservationFingerprintV0
	Skipped      int
	FilterActive bool
}

func (runtime *RuntimeV0) goalObservationFingerprintPlanV0(
	ctx context.Context,
	request orquestagoal.GoalWorkObserveActiveRequestV0,
) goalObservationFingerprintPlanV0 {
	plan := goalObservationFingerprintPlanV0{Request: request}
	if runtime == nil || runtime.goalFingerprint == nil || runtime.goalStateStore == nil {
		return plan
	}
	lister, ok := runtime.goalStateStore.(orquestagoal.GoalWorkStateListPortV0)
	if !ok {
		return plan
	}
	states, err := lister.ListGoalWorkStatesV0(ctx, request.List)
	if err != nil || len(states) == 0 {
		return plan
	}
	pending := map[string]orquestagoal.GoalObservationFingerprintV0{}
	var runRefs []string
	for _, state := range states {
		runRef := strings.TrimSpace(state.RunRef)
		if runRef == "" {
			continue
		}
		fingerprint, fingerprintOK, err := runtime.goalFingerprint.FingerprintGoalObservationV0(ctx, state)
		if err != nil || !fingerprintOK {
			runRefs = append(runRefs, runRef)
			continue
		}
		fingerprint = orquestagoal.NormalizeGoalObservationFingerprintV0(fingerprint)
		if fingerprint.RunRef == "" {
			fingerprint.RunRef = runRef
		}
		if previous, ok := runtime.goalObservationFingerprints[runRef]; ok &&
			orquestagoal.GoalObservationUnchangedV0(previous, fingerprint) {
			plan.Skipped++
			continue
		}
		runRefs = append(runRefs, runRef)
		pending[runRef] = fingerprint
	}
	if len(pending) > 0 {
		plan.Pending = pending
	}
	if plan.Skipped == 0 {
		return plan
	}
	plan.FilterActive = true
	plan.Request.List.RunRefs = compactConfigStringsV0(runRefs)
	return plan
}

func (runtime *RuntimeV0) rememberGoalObservationFingerprintsV0(
	ctx context.Context,
	result orquestagoal.GoalWorkObserveActiveResultV0,
	pending map[string]orquestagoal.GoalObservationFingerprintV0,
) {
	if runtime == nil || len(pending) == 0 {
		return
	}
	for _, observation := range result.Observations {
		runRef := strings.TrimSpace(observation.State.RunRef)
		if runRef == "" {
			continue
		}
		if observation.Terminal || orquestagoal.GoalWorkResultTerminalV0(observation.State.Status) {
			delete(runtime.goalObservationFingerprints, runRef)
			continue
		}
		if fingerprint, ok := pending[runRef]; ok {
			runtime.goalObservationFingerprints[runRef] = fingerprint
			continue
		}
		if runtime.goalFingerprint != nil {
			if fingerprint, ok, err := runtime.goalFingerprint.FingerprintGoalObservationV0(ctx, observation.State); err == nil && ok {
				fingerprint = orquestagoal.NormalizeGoalObservationFingerprintV0(fingerprint)
				if fingerprint.RunRef == "" {
					fingerprint.RunRef = runRef
				}
				runtime.goalObservationFingerprints[runRef] = fingerprint
				continue
			}
		}
	}
}
