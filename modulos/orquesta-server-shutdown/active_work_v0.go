package orquestaservershutdown

import (
	"context"
	"strings"
)

const defaultActiveShutdownWorkMaxItemsV0 = 100

func blockingActiveShutdownWorkV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
) (ServerShutdownResultV0, bool, []string, error) {
	if deps.ActiveWorkReader == nil {
		return ServerShutdownResultV0{}, false, nil, nil
	}
	active, err := deps.ActiveWorkReader.ReadActiveShutdownWorkV0(ctx, ActiveShutdownWorkRequestV0{
		QueueRef:      command.QueueRef,
		AppRefs:       append([]string(nil), command.AppRefs...),
		CorrelationID: command.CorrelationID,
		EvidenceRefs:  append([]string(nil), command.EvidenceRefs...),
		MaxItems:      activeShutdownWorkMaxItemsV0(command.QueueLimit),
	})
	if err != nil {
		return ServerShutdownResultV0{}, false, nil, err
	}
	works := compactActiveShutdownWorksV0(active.ActiveWorks)
	cleanupEvidenceRefs := []string{}
	if command.CleanupGoalBackends && deps.ActiveWorkCleaner != nil {
		var cleaned ActiveShutdownWorkCleanupResultV0
		var cleanupRan bool
		cleaned, cleanupRan, err = cleanupActiveGoalBackendsV0(ctx, deps, command, works)
		if err != nil {
			return activeShutdownWorkCleanupErrorResultV0(command, active, works), true, nil, nil
		}
		if cleanupRan {
			cleanupEvidenceRefs = compactServerShutdownStringsV0(cleaned.EvidenceRefs)
			active, err = deps.ActiveWorkReader.ReadActiveShutdownWorkV0(ctx, ActiveShutdownWorkRequestV0{
				QueueRef:      command.QueueRef,
				AppRefs:       append([]string(nil), command.AppRefs...),
				CorrelationID: command.CorrelationID,
				EvidenceRefs: compactServerShutdownStringsV0(append(
					append([]string(nil), command.EvidenceRefs...),
					cleanupEvidenceRefs...,
				)),
				MaxItems: activeShutdownWorkMaxItemsV0(command.QueueLimit),
			})
			if err != nil {
				return ServerShutdownResultV0{}, false, nil, err
			}
			works = compactActiveShutdownWorksV0(active.ActiveWorks)
		}
	}
	if command.Forced {
		works = forcedBlockingActiveShutdownWorksV0(works)
	}
	if len(works) == 0 {
		return ServerShutdownResultV0{}, false, cleanupEvidenceRefs, nil
	}
	status := activeShutdownWorkBlockingStatusV0(works)
	result := ServerShutdownResultV0{
		SchemaVersion:   ServerShutdownSchemaVersionV0,
		Status:          status,
		ShutdownReady:   false,
		ActiveWorkCount: len(works),
		ActiveWorks:     works,
		EvidenceRefs: compactServerShutdownStringsV0(append(
			append([]string(nil), command.EvidenceRefs...),
			append(append(active.EvidenceRefs, cleanupEvidenceRefs...), activeShutdownWorkBlockingEvidenceV0(status))...,
		)),
	}
	return withServerShutdownRecommendedActionV0(result), true, nil, nil
}

func cleanupActiveGoalBackendsV0(
	ctx context.Context,
	deps ServerShutdownDepsV0,
	command ServerShutdownCommandV0,
	works []ActiveShutdownWorkV0,
) (ActiveShutdownWorkCleanupResultV0, bool, error) {
	backendWorks := activeShutdownBackendWorksV0(works)
	if len(backendWorks) == 0 || len(backendWorks) != len(works) {
		return ActiveShutdownWorkCleanupResultV0{}, false, nil
	}
	cleaned, err := deps.ActiveWorkCleaner.CleanupActiveShutdownWorkV0(ctx, ActiveShutdownWorkCleanupCommandV0{
		QueueRef:            command.QueueRef,
		AppRefs:             append([]string(nil), command.AppRefs...),
		CorrelationID:       command.CorrelationID,
		EvidenceRefs:        append([]string(nil), command.EvidenceRefs...),
		ActiveWorks:         backendWorks,
		CleanupGoalBackends: true,
	})
	if err != nil {
		return ActiveShutdownWorkCleanupResultV0{}, true, err
	}
	cleaned.EvidenceRefs = compactServerShutdownStringsV0(append(
		cleaned.EvidenceRefs,
		"evidence-ref-shutdown-goal-backend-cleanup-requested",
	))
	return cleaned, true, nil
}

func activeShutdownBackendWorksV0(
	works []ActiveShutdownWorkV0,
) []ActiveShutdownWorkV0 {
	out := make([]ActiveShutdownWorkV0, 0, len(works))
	for _, work := range works {
		if activeShutdownWorkIsBackendStillRunningV0(work) {
			out = append(out, work)
		}
	}
	if out == nil {
		return []ActiveShutdownWorkV0{}
	}
	return out
}

func activeShutdownWorkBlockingStatusV0(
	works []ActiveShutdownWorkV0,
) string {
	for _, work := range works {
		if activeShutdownWorkIsBackendStillRunningV0(work) {
			return ServerShutdownStatusBackendStillRunningV0
		}
	}
	return ServerShutdownStatusActiveGoalsPresentV0
}

func forcedBlockingActiveShutdownWorksV0(
	works []ActiveShutdownWorkV0,
) []ActiveShutdownWorkV0 {
	out := make([]ActiveShutdownWorkV0, 0, len(works))
	for _, work := range works {
		if activeShutdownWorkIsBackendStillRunningV0(work) {
			out = append(out, work)
		}
	}
	if out == nil {
		return []ActiveShutdownWorkV0{}
	}
	return out
}

func activeShutdownWorkIsBackendStillRunningV0(
	work ActiveShutdownWorkV0,
) bool {
	return strings.TrimSpace(work.Kind) == "goal_backend" ||
		strings.TrimSpace(work.Status) == ServerShutdownStatusBackendStillRunningV0
}

func activeShutdownWorkCleanupErrorResultV0(
	command ServerShutdownCommandV0,
	active ActiveShutdownWorkResultV0,
	works []ActiveShutdownWorkV0,
) ServerShutdownResultV0 {
	status := activeShutdownWorkBlockingStatusV0(works)
	return withServerShutdownRecommendedActionV0(ServerShutdownResultV0{
		SchemaVersion:   ServerShutdownSchemaVersionV0,
		Status:          status,
		ShutdownReady:   false,
		ActiveWorkCount: len(works),
		ActiveWorks:     works,
		EvidenceRefs: compactServerShutdownStringsV0(append(
			append([]string(nil), command.EvidenceRefs...),
			append(active.EvidenceRefs,
				"evidence-ref-shutdown-goal-backend-cleanup-requested",
				"evidence-ref-shutdown-goal-backend-cleanup-error",
				activeShutdownWorkBlockingEvidenceV0(status),
			)...,
		)),
	})
}

func activeShutdownWorkBlockingEvidenceV0(status string) string {
	if strings.TrimSpace(status) == ServerShutdownStatusBackendStillRunningV0 {
		return "evidence-ref-shutdown-backend-still-running"
	}
	return "evidence-ref-shutdown-active-goals-present"
}

func activeShutdownWorkMaxItemsV0(limit int) int {
	if limit > 0 {
		return limit
	}
	return defaultActiveShutdownWorkMaxItemsV0
}

func compactActiveShutdownWorksV0(
	works []ActiveShutdownWorkV0,
) []ActiveShutdownWorkV0 {
	out := make([]ActiveShutdownWorkV0, 0, len(works))
	seen := map[string]struct{}{}
	for _, work := range works {
		work = normalizeActiveShutdownWorkV0(work)
		if work.Kind == "" && work.RunRef == "" && work.WorkRef == "" && work.ExternalWorkRef == "" {
			continue
		}
		key := strings.Join([]string{work.Kind, work.RunRef, work.WorkRef, work.ExternalWorkRef, work.Status}, "\x00")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, work)
	}
	if out == nil {
		return []ActiveShutdownWorkV0{}
	}
	return out
}

func normalizeActiveShutdownWorkV0(
	work ActiveShutdownWorkV0,
) ActiveShutdownWorkV0 {
	work.Kind = strings.TrimSpace(work.Kind)
	work.RunRef = strings.TrimSpace(work.RunRef)
	work.WorkRef = strings.TrimSpace(work.WorkRef)
	work.ExternalWorkRef = strings.TrimSpace(work.ExternalWorkRef)
	work.Status = strings.TrimSpace(work.Status)
	work.EvidenceRefs = compactServerShutdownStringsV0(work.EvidenceRefs)
	return work
}
