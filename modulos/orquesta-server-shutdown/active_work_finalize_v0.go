package orquestaservershutdown

import (
	"context"
	"strings"
)

func finalizeShutdownActiveWorkV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	result ServerShutdownResultV0,
	initialWorks []ActiveShutdownWorkV0,
) (ServerShutdownResultV0, error) {
	if deps.ActiveWorkReader == nil {
		result.GoalActions = mergeServerShutdownGoalActionsV0(
			result.GoalActions,
			initialShutdownGoalActionsV0(command, result, initialWorks),
		)
		return withServerShutdownRecommendedActionV0(result), nil
	}
	active, err := deps.ActiveWorkReader.ReadActiveShutdownWorkV0(ctx, ActiveShutdownWorkRequestV0{
		QueueRef:      command.QueueRef,
		AppRefs:       append([]string(nil), command.AppRefs...),
		CorrelationID: command.CorrelationID,
		EvidenceRefs:  compactServerShutdownStringsV0(append(command.EvidenceRefs, result.EvidenceRefs...)),
		MaxItems:      activeShutdownWorkMaxItemsV0(command.QueueLimit),
	})
	if err != nil {
		return ServerShutdownResultV0{}, err
	}
	works := compactActiveShutdownWorksV0(active.ActiveWorks)
	if command.Forced {
		works = forcedBlockingActiveShutdownWorksV0(works)
	}
	if len(works) == 0 {
		result.EvidenceRefs = compactServerShutdownStringsV0(append(result.EvidenceRefs, active.EvidenceRefs...))
		result.GoalActions = mergeServerShutdownGoalActionsV0(
			result.GoalActions,
			initialShutdownGoalActionsV0(command, result, initialWorks),
		)
		return withServerShutdownRecommendedActionV0(result), nil
	}
	works, actions := annotateActiveShutdownWorksV0(works, result, nil)
	status := activeShutdownWorkBlockingStatusV0(works)
	result.ActiveWorks = works
	result.ActiveWorkCount = len(works)
	result.GoalActions = mergeServerShutdownGoalActionsV0(result.GoalActions, actions)
	result.EvidenceRefs = compactServerShutdownStringsV0(append(
		append(result.EvidenceRefs, active.EvidenceRefs...),
		activeShutdownWorkBlockingEvidenceV0(status),
	))
	if result.ShutdownReady || strings.TrimSpace(result.Status) == ServerShutdownStatusReadyV0 {
		result.Status = status
		result.ShutdownReady = false
	}
	return withServerShutdownRecommendedActionV0(result), nil
}

func initialShutdownGoalActionsV0(
	command ServerShutdownCommandV0,
	result ServerShutdownResultV0,
	initialWorks []ActiveShutdownWorkV0,
) []ServerShutdownGoalActionV0 {
	initialWorks = compactActiveShutdownWorksV0(initialWorks)
	if len(initialWorks) == 0 {
		return []ServerShutdownGoalActionV0{}
	}
	actions := make([]ServerShutdownGoalActionV0, 0, len(initialWorks))
	for _, work := range initialWorks {
		if command.Forced {
			actions = append(actions, serverShutdownGoalActionsForWorksV0(
				[]ActiveShutdownWorkV0{work},
				ServerShutdownGoalActionForcedStopRequestedV0,
				[]string{serverShutdownGoalActionForcedStopEvidenceV0},
			)...)
			continue
		}
		action, evidence := activeShutdownWorkActionV0(work, result, nil)
		actions = append(actions, serverShutdownGoalActionsForWorksV0([]ActiveShutdownWorkV0{work}, action, evidence)...)
	}
	return compactServerShutdownGoalActionsV0(actions)
}
