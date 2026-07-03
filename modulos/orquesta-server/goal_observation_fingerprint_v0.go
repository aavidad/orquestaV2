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
		if runtime.goalObservationFingerprintUnchangedV0(runRef, fingerprint) {
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
			runtime.deleteGoalObservationFingerprintV0(runRef)
			continue
		}
		if fingerprint, ok := pending[runRef]; ok {
			runtime.storeGoalObservationFingerprintV0(runRef, fingerprint)
			continue
		}
		if runtime.goalFingerprint != nil {
			if fingerprint, ok, err := runtime.goalFingerprint.FingerprintGoalObservationV0(ctx, observation.State); err == nil && ok {
				fingerprint = orquestagoal.NormalizeGoalObservationFingerprintV0(fingerprint)
				if fingerprint.RunRef == "" {
					fingerprint.RunRef = runRef
				}
				runtime.storeGoalObservationFingerprintV0(runRef, fingerprint)
				continue
			}
		}
	}
}

func (runtime *RuntimeV0) ForgetGoalObservationFingerprintV0(runRef string) bool {
	runRef = strings.TrimSpace(runRef)
	if runtime == nil || runRef == "" {
		return false
	}
	runtime.goalObservationFingerprintsMu.Lock()
	defer runtime.goalObservationFingerprintsMu.Unlock()
	if runtime.goalObservationFingerprints == nil {
		return false
	}
	if _, ok := runtime.goalObservationFingerprints[runRef]; !ok {
		return false
	}
	delete(runtime.goalObservationFingerprints, runRef)
	return true
}

func (runtime *RuntimeV0) goalObservationFingerprintUnchangedV0(
	runRef string,
	fingerprint orquestagoal.GoalObservationFingerprintV0,
) bool {
	runRef = strings.TrimSpace(runRef)
	if runtime == nil || runRef == "" {
		return false
	}
	runtime.goalObservationFingerprintsMu.Lock()
	defer runtime.goalObservationFingerprintsMu.Unlock()
	if runtime.goalObservationFingerprints == nil {
		return false
	}
	previous, ok := runtime.goalObservationFingerprints[runRef]
	return ok && orquestagoal.GoalObservationUnchangedV0(previous, fingerprint)
}

func (runtime *RuntimeV0) storeGoalObservationFingerprintV0(
	runRef string,
	fingerprint orquestagoal.GoalObservationFingerprintV0,
) {
	runRef = strings.TrimSpace(runRef)
	if runtime == nil || runRef == "" {
		return
	}
	runtime.goalObservationFingerprintsMu.Lock()
	defer runtime.goalObservationFingerprintsMu.Unlock()
	if runtime.goalObservationFingerprints == nil {
		runtime.goalObservationFingerprints = map[string]orquestagoal.GoalObservationFingerprintV0{}
	}
	runtime.goalObservationFingerprints[runRef] = fingerprint
}

func (runtime *RuntimeV0) deleteGoalObservationFingerprintV0(runRef string) {
	runRef = strings.TrimSpace(runRef)
	if runtime == nil || runRef == "" {
		return
	}
	runtime.goalObservationFingerprintsMu.Lock()
	defer runtime.goalObservationFingerprintsMu.Unlock()
	delete(runtime.goalObservationFingerprints, runRef)
}
