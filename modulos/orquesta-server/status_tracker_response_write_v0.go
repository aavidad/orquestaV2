package orquestaserver

import (
	"strings"
	"time"
)

func (tracker *StatusTrackerV0) MarkResponseWriteFailedV0(
	stage string,
	now time.Time,
) StateV0 {
	stage = compactResponseWriteStageV0(stage)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResponseWriteFailures++
		state.ResponseWriteLastFailedAt = formatTimeV0(now)
		state.ResponseWriteLastCode = serverResponseWriteFailedCodeV0
		state.ResponseWriteLastStage = stage
		appendRecentServerErrorV0(
			state,
			now,
			serverResponseWriteFailedCodeV0,
			"http_response",
			serverResponseWriteFailedCodeV0,
			nil,
		)
	})
}

func (tracker *StatusTrackerV0) MarkResponseEncodeFailedV0(
	stage string,
	now time.Time,
) StateV0 {
	stage = compactResponseWriteStageV0(stage)
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResponseWriteFailures++
		state.ResponseWriteLastFailedAt = formatTimeV0(now)
		state.ResponseWriteLastCode = serverResponseEncodeFailedCodeV0
		state.ResponseWriteLastStage = stage
		appendRecentServerErrorV0(
			state,
			now,
			serverResponseEncodeFailedCodeV0,
			"http_response",
			serverResponseEncodeFailedCodeV0,
			nil,
		)
	})
}

func (tracker *StatusTrackerV0) MarkResponseWriteSucceededV0(now time.Time) StateV0 {
	return tracker.updateV0(func(state *StateV0) {
		state.LastHeartbeatAt = formatTimeV0(now)
		state.ResponseWriteLastCode = ""
		state.ResponseWriteLastStage = ""
	})
}

func (tracker *StatusTrackerV0) ResponseWriteRecoveryPendingV0() bool {
	if tracker == nil {
		return false
	}
	state := tracker.SnapshotV0()
	return strings.TrimSpace(state.ResponseWriteLastCode) != ""
}

func compactResponseWriteStageV0(stage string) string {
	switch strings.TrimSpace(stage) {
	case "encode_before_header", "body_before_header", "body_after_header", "fallback_body_after_header":
		return strings.TrimSpace(stage)
	default:
		return "unknown"
	}
}
