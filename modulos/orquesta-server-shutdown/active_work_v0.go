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
) (ServerShutdownResultV0, bool, error) {
	if command.Forced || deps.ActiveWorkReader == nil {
		return ServerShutdownResultV0{}, false, nil
	}
	active, err := deps.ActiveWorkReader.ReadActiveShutdownWorkV0(ctx, ActiveShutdownWorkRequestV0{
		QueueRef:      command.QueueRef,
		AppRefs:       append([]string(nil), command.AppRefs...),
		CorrelationID: command.CorrelationID,
		EvidenceRefs:  append([]string(nil), command.EvidenceRefs...),
		MaxItems:      activeShutdownWorkMaxItemsV0(command.QueueLimit),
	})
	if err != nil {
		return ServerShutdownResultV0{}, false, err
	}
	works := compactActiveShutdownWorksV0(active.ActiveWorks)
	if len(works) == 0 {
		return ServerShutdownResultV0{}, false, nil
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
			append(active.EvidenceRefs, activeShutdownWorkBlockingEvidenceV0(status))...,
		)),
	}
	return result, true, nil
}

func activeShutdownWorkBlockingStatusV0(
	works []ActiveShutdownWorkV0,
) string {
	for _, work := range works {
		if strings.TrimSpace(work.Kind) == "goal_backend" ||
			strings.TrimSpace(work.Status) == ServerShutdownStatusBackendStillRunningV0 {
			return ServerShutdownStatusBackendStillRunningV0
		}
	}
	return ServerShutdownStatusActiveGoalsPresentV0
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
