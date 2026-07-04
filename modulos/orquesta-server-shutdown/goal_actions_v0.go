package orquestaservershutdown

import "strings"

const (
	serverShutdownGoalActionWaitCheckpointEvidenceV0   = "evidence-ref-shutdown-goal-action-wait-checkpoint"
	serverShutdownGoalActionStopRequestedEvidenceV0    = "evidence-ref-shutdown-goal-action-stop-requested"
	serverShutdownGoalActionForcedStopEvidenceV0       = "evidence-ref-shutdown-goal-action-forced-stop-requested"
	serverShutdownGoalActionObserveEvidenceV0          = "evidence-ref-shutdown-goal-action-observe-active"
	serverShutdownGoalActionCleanupRequiredEvidenceV0  = "evidence-ref-shutdown-goal-action-cleanup-required"
	serverShutdownGoalActionCleanupRequestedEvidenceV0 = "evidence-ref-shutdown-goal-action-cleanup-requested"
	serverShutdownGoalActionCleanupAttemptedEvidenceV0 = "evidence-ref-shutdown-goal-action-cleanup-attempted"
	serverShutdownGoalActionCleanupCompletedEvidenceV0 = "evidence-ref-shutdown-goal-action-cleanup-completed"
)

func annotateActiveShutdownWorksV0(
	works []ActiveShutdownWorkV0,
	result ServerShutdownResultV0,
	cleanupEvidenceRefs []string,
) ([]ActiveShutdownWorkV0, []ServerShutdownGoalActionV0) {
	out := make([]ActiveShutdownWorkV0, 0, len(works))
	actions := make([]ServerShutdownGoalActionV0, 0, len(works))
	for _, work := range works {
		action, evidence := activeShutdownWorkActionV0(work, result, cleanupEvidenceRefs)
		work.ActionTaken = action
		work.ActionEvidenceRefs = compactServerShutdownStringsV0(append(work.ActionEvidenceRefs, evidence...))
		work.EvidenceRefs = compactServerShutdownStringsV0(append(work.EvidenceRefs, work.ActionEvidenceRefs...))
		out = append(out, work)
		actions = append(actions, serverShutdownGoalActionFromWorkV0(work))
	}
	return out, compactServerShutdownGoalActionsV0(actions)
}

func activeShutdownWorkActionV0(
	work ActiveShutdownWorkV0,
	result ServerShutdownResultV0,
	cleanupEvidenceRefs []string,
) (string, []string) {
	if activeShutdownWorkIsBackendStillRunningV0(work) {
		if len(compactServerShutdownStringsV0(cleanupEvidenceRefs)) > 0 ||
			serverShutdownEvidenceContainsV0(result.EvidenceRefs, "evidence-ref-shutdown-goal-backend-cleanup-requested") {
			return ServerShutdownGoalActionCleanupAttemptedV0, []string{serverShutdownGoalActionCleanupAttemptedEvidenceV0}
		}
		return ServerShutdownGoalActionCleanupRequiredV0, []string{serverShutdownGoalActionCleanupRequiredEvidenceV0}
	}
	if run, ok := serverShutdownRunByRefV0(result.Runs, work.RunRef); ok {
		switch {
		case run.ForcedAfterCheckpointDeadline:
			return ServerShutdownGoalActionForcedStopRequestedV0, []string{serverShutdownGoalActionForcedStopEvidenceV0}
		case run.CheckpointRequired:
			return ServerShutdownGoalActionWaitCheckpointV0, []string{serverShutdownGoalActionWaitCheckpointEvidenceV0}
		case run.StopRequested:
			return ServerShutdownGoalActionStopRequestedWaitV0, []string{serverShutdownGoalActionStopRequestedEvidenceV0}
		}
	}
	switch strings.TrimSpace(result.Status) {
	case ServerShutdownStatusWaitingCheckpointV0:
		return ServerShutdownGoalActionWaitCheckpointV0, []string{serverShutdownGoalActionWaitCheckpointEvidenceV0}
	case ServerShutdownStatusWaitingDrainV0:
		return ServerShutdownGoalActionStopRequestedWaitV0, []string{serverShutdownGoalActionStopRequestedEvidenceV0}
	default:
		return ServerShutdownGoalActionObserveActiveV0, []string{serverShutdownGoalActionObserveEvidenceV0}
	}
}

func serverShutdownRunByRefV0(
	runs []ServerShutdownRunResultV0,
	runRef string,
) (ServerShutdownRunResultV0, bool) {
	runRef = strings.TrimSpace(runRef)
	if runRef == "" {
		return ServerShutdownRunResultV0{}, false
	}
	for _, run := range runs {
		if strings.TrimSpace(run.RunRef) == runRef {
			return run, true
		}
	}
	return ServerShutdownRunResultV0{}, false
}

func serverShutdownGoalActionsForWorksV0(
	works []ActiveShutdownWorkV0,
	action string,
	actionEvidenceRefs []string,
) []ServerShutdownGoalActionV0 {
	actions := make([]ServerShutdownGoalActionV0, 0, len(works))
	action = strings.TrimSpace(action)
	actionEvidenceRefs = compactServerShutdownStringsV0(actionEvidenceRefs)
	for _, work := range works {
		work = normalizeActiveShutdownWorkV0(work)
		work.ActionTaken = action
		work.ActionEvidenceRefs = compactServerShutdownStringsV0(append(work.ActionEvidenceRefs, actionEvidenceRefs...))
		work.EvidenceRefs = compactServerShutdownStringsV0(append(work.EvidenceRefs, work.ActionEvidenceRefs...))
		actions = append(actions, serverShutdownGoalActionFromWorkV0(work))
	}
	return compactServerShutdownGoalActionsV0(actions)
}

func mergeServerShutdownGoalActionsV0(
	left []ServerShutdownGoalActionV0,
	right []ServerShutdownGoalActionV0,
) []ServerShutdownGoalActionV0 {
	return compactServerShutdownGoalActionsV0(append(append([]ServerShutdownGoalActionV0(nil), left...), right...))
}

func serverShutdownGoalActionFromWorkV0(
	work ActiveShutdownWorkV0,
) ServerShutdownGoalActionV0 {
	actionEvidence := compactServerShutdownStringsV0(work.ActionEvidenceRefs)
	return ServerShutdownGoalActionV0{
		Kind:               strings.TrimSpace(work.Kind),
		RunRef:             strings.TrimSpace(work.RunRef),
		WorkRef:            strings.TrimSpace(work.WorkRef),
		ExternalWorkRef:    strings.TrimSpace(work.ExternalWorkRef),
		Status:             strings.TrimSpace(work.Status),
		ActionTaken:        strings.TrimSpace(work.ActionTaken),
		ActionEvidenceRefs: actionEvidence,
		EvidenceRefs:       compactServerShutdownStringsV0(append(work.EvidenceRefs, actionEvidence...)),
	}
}

func compactServerShutdownGoalActionsV0(
	actions []ServerShutdownGoalActionV0,
) []ServerShutdownGoalActionV0 {
	out := make([]ServerShutdownGoalActionV0, 0, len(actions))
	seen := map[string]struct{}{}
	for _, action := range actions {
		action = normalizeServerShutdownGoalActionV0(action)
		if action.ActionTaken == "" ||
			(action.Kind == "" && action.RunRef == "" && action.WorkRef == "" && action.ExternalWorkRef == "") {
			continue
		}
		key := strings.Join([]string{action.Kind, action.RunRef, action.WorkRef, action.ExternalWorkRef, action.Status, action.ActionTaken}, "\x00")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, action)
	}
	if out == nil {
		return []ServerShutdownGoalActionV0{}
	}
	return out
}

func normalizeServerShutdownGoalActionV0(
	action ServerShutdownGoalActionV0,
) ServerShutdownGoalActionV0 {
	action.Kind = strings.TrimSpace(action.Kind)
	action.RunRef = strings.TrimSpace(action.RunRef)
	action.WorkRef = strings.TrimSpace(action.WorkRef)
	action.ExternalWorkRef = strings.TrimSpace(action.ExternalWorkRef)
	action.Status = strings.TrimSpace(action.Status)
	action.ActionTaken = strings.TrimSpace(action.ActionTaken)
	action.ActionEvidenceRefs = compactServerShutdownStringsV0(action.ActionEvidenceRefs)
	action.EvidenceRefs = compactServerShutdownStringsV0(action.EvidenceRefs)
	return action
}
