package orquestaserver

import (
	"context"
	"strings"
	"time"
)

type ShutdownSnapshotPortV0 interface {
	SnapshotShutdownV0(
		context.Context,
		ShutdownSnapshotRequestV0,
	) (ShutdownSnapshotResultV0, error)
}

type ShutdownSnapshotRequestV0 struct {
	Route        string   `json:"route,omitempty"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type ShutdownSnapshotResultV0 struct {
	Status          string                   `json:"status,omitempty"`
	ActiveWorkCount int                      `json:"active_work_count,omitempty"`
	ActiveWorks     []ShutdownSnapshotWorkV0 `json:"active_works,omitempty"`
	EvidenceRefs    []string                 `json:"evidence_refs,omitempty"`
}

type ShutdownSnapshotWorkV0 struct {
	Kind            string `json:"kind,omitempty"`
	RunRef          string `json:"run_ref,omitempty"`
	WorkRef         string `json:"work_ref,omitempty"`
	ExternalWorkRef string `json:"external_work_ref,omitempty"`
	Status          string `json:"status,omitempty"`
}

func (runtime *RuntimeV0) snapshotShutdownActiveWorkForHTTPV0(ctx context.Context) {
	if runtime == nil || runtime.shutdownSnapshot == nil || runtime.tracker == nil {
		return
	}
	snapshot, err := runtime.shutdownSnapshot.SnapshotShutdownV0(ctx, ShutdownSnapshotRequestV0{
		Route:        serverShutdownRoutePathV0,
		EvidenceRefs: []string{"evidence-ref-shutdown-http-pre-snapshot"},
	})
	if err != nil {
		runtime.auditEventV0(ctx, "server_shutdown_snapshot", "failed", "", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}
	if shutdownSnapshotEmptyV0(snapshot) {
		return
	}
	runtime.persistStateTransitionV0(
		ctx,
		runtime.tracker.MarkShutdownSnapshotV0(snapshot, runtime.clock.Now()),
		"shutdown_active_work_snapshot",
	)
}

func (tracker *StatusTrackerV0) MarkShutdownSnapshotV0(
	snapshot ShutdownSnapshotResultV0,
	now time.Time,
) StateV0 {
	status := strings.TrimSpace(snapshot.Status)
	if status == "" {
		status = "active_work_snapshot"
	}
	refs := shutdownSnapshotActiveWorkRefsV0(snapshot.ActiveWorks)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.LastShutdownAt = formatTimeV0(now)
		state.ShutdownInProgress = true
		state.ShutdownStatus = status
		state.ShutdownReady = false
		state.ShutdownActiveWorkCount = nonNegativeServerIntV0(firstNonZeroServerIntV0(
			snapshot.ActiveWorkCount,
			len(refs),
		))
		state.ShutdownActiveWorkRefs = compactServerStringsV0(refs)
		state.SupervisorFrozen = true
		state.SupervisorTickActive = false
	})
}

func shutdownSnapshotActiveWorkRefsV0(
	works []ShutdownSnapshotWorkV0,
) []string {
	projection := make([]serverShutdownHTTPWorkProjectionV0, 0, len(works))
	for _, work := range works {
		projection = append(projection, serverShutdownHTTPWorkProjectionV0{
			Kind:            work.Kind,
			RunRef:          work.RunRef,
			WorkRef:         work.WorkRef,
			ExternalWorkRef: work.ExternalWorkRef,
			Status:          work.Status,
		})
	}
	return shutdownProjectionActiveWorkRefsV0(projection)
}

func shutdownSnapshotEmptyV0(snapshot ShutdownSnapshotResultV0) bool {
	return strings.TrimSpace(snapshot.Status) == "" &&
		snapshot.ActiveWorkCount <= 0 &&
		len(snapshot.ActiveWorks) == 0 &&
		len(snapshot.EvidenceRefs) == 0
}

func firstNonZeroServerIntV0(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
