package main

import (
	"context"
	"strings"

	orquestaappcodexstack "orquesta/modulos/orquesta-app-codex-stack"
	orquestaserver "orquesta/modulos/orquesta-server"
	orquestaservershutdown "orquesta/modulos/orquesta-server-shutdown"
)

type serverShutdownSnapshotPortV0 struct {
	readers []orquestaservershutdown.ActiveShutdownWorkReaderPortV0
}

func serverShutdownSnapshotFromStackV0(
	stack orquestaappcodexstack.StackV0,
	backends serverCodexGoalBackendsV0,
) orquestaserver.ShutdownSnapshotPortV0 {
	readers := []orquestaservershutdown.ActiveShutdownWorkReaderPortV0{
		orquestaappcodexstack.NewShutdownActiveWorkReaderV0(orquestaappcodexstack.ConfigV0{
			Stores:                stack.Stores,
			AppGoalLauncher:       serverGoalWorkLauncherFromBackendV0(backends.AppGoal),
			AppGoalReworkLauncher: serverGoalWorkLauncherFromBackendV0(backends.AppGoal),
			AppGoalObserver:       serverGoalWorkObserverFromBackendV0(backends.AppGoal),
		}),
	}
	if idleReader := serverGoalActiveShutdownWorkReaderFromBackendV0(backends.IdleGoal); idleReader != nil {
		readers = append(readers, idleReader)
	}
	if autoprogrammingReader := serverGoalActiveShutdownWorkReaderFromBackendV0(backends.AutoprogrammingGoal); autoprogrammingReader != nil {
		readers = append(readers, autoprogrammingReader)
	}
	return serverShutdownSnapshotPortV0{readers: readers}
}

func (port serverShutdownSnapshotPortV0) SnapshotShutdownV0(
	ctx context.Context,
	request orquestaserver.ShutdownSnapshotRequestV0,
) (orquestaserver.ShutdownSnapshotResultV0, error) {
	out := orquestaserver.ShutdownSnapshotResultV0{}
	for _, reader := range port.readers {
		if reader == nil {
			continue
		}
		active, err := reader.ReadActiveShutdownWorkV0(ctx, orquestaservershutdown.ActiveShutdownWorkRequestV0{
			EvidenceRefs: append([]string(nil), request.EvidenceRefs...),
			MaxItems:     100,
		})
		if err != nil {
			return orquestaserver.ShutdownSnapshotResultV0{}, err
		}
		out = mergeServerShutdownSnapshotV0(out, active)
	}
	if out.ActiveWorkCount > 0 && strings.TrimSpace(out.Status) == "" {
		out.Status = serverShutdownSnapshotStatusV0(out.ActiveWorks)
	}
	return out, nil
}

func mergeServerShutdownSnapshotV0(
	left orquestaserver.ShutdownSnapshotResultV0,
	right orquestaservershutdown.ActiveShutdownWorkResultV0,
) orquestaserver.ShutdownSnapshotResultV0 {
	seen := map[string]struct{}{}
	for _, work := range left.ActiveWorks {
		seen[serverShutdownSnapshotWorkKeyV0(work)] = struct{}{}
	}
	for _, work := range right.ActiveWorks {
		next := orquestaserver.ShutdownSnapshotWorkV0{
			Kind:            strings.TrimSpace(work.Kind),
			RunRef:          strings.TrimSpace(work.RunRef),
			WorkRef:         strings.TrimSpace(work.WorkRef),
			ExternalWorkRef: strings.TrimSpace(work.ExternalWorkRef),
			Status:          strings.TrimSpace(work.Status),
		}
		if next.Kind == "" && next.RunRef == "" && next.WorkRef == "" && next.ExternalWorkRef == "" {
			continue
		}
		key := serverShutdownSnapshotWorkKeyV0(next)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		left.ActiveWorks = append(left.ActiveWorks, next)
	}
	left.ActiveWorkCount = len(left.ActiveWorks)
	return left
}

func serverShutdownSnapshotWorkKeyV0(work orquestaserver.ShutdownSnapshotWorkV0) string {
	return strings.Join([]string{
		strings.TrimSpace(work.Kind),
		strings.TrimSpace(work.RunRef),
		strings.TrimSpace(work.WorkRef),
		strings.TrimSpace(work.ExternalWorkRef),
		strings.TrimSpace(work.Status),
	}, "\x00")
}

func serverShutdownSnapshotStatusV0(works []orquestaserver.ShutdownSnapshotWorkV0) string {
	for _, work := range works {
		if strings.TrimSpace(work.Kind) == "goal_backend" ||
			strings.TrimSpace(work.Status) == orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0 {
			return orquestaservershutdown.ServerShutdownStatusBackendStillRunningV0
		}
	}
	return orquestaservershutdown.ServerShutdownStatusActiveGoalsPresentV0
}
