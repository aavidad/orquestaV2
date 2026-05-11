package orquestaruncontrol

import "strings"

type RunControlEvaluationV0 struct {
	SchedulingAllowed  bool `json:"scheduling_allowed"`
	DispatchAllowed    bool `json:"dispatch_allowed"`
	CheckpointRequired bool `json:"checkpoint_required"`
	StopAgentsAllowed  bool `json:"stop_agents_allowed"`
	Terminal           bool `json:"terminal"`
}

func EvaluateRunControlV0(state RunControlStateV0) RunControlEvaluationV0 {
	switch NormalizeRunControlStatusV0(state.Status) {
	case RunControlStatusRunningV0:
		return RunControlEvaluationV0{
			SchedulingAllowed: true,
			DispatchAllowed:   true,
		}
	case RunControlStatusPausedV0:
		return RunControlEvaluationV0{}
	case RunControlStatusStopRequestedV0, RunControlStatusCancelRequestedV0:
		checkpointRequired := !state.Forced && !state.CheckpointRecorded
		return RunControlEvaluationV0{
			CheckpointRequired: checkpointRequired,
			StopAgentsAllowed:  !checkpointRequired,
		}
	case RunControlStatusStoppedV0, RunControlStatusCanceledV0:
		return RunControlEvaluationV0{Terminal: true}
	default:
		return RunControlEvaluationV0{}
	}
}

func NormalizeRunControlStatusV0(status RunControlStatusV0) RunControlStatusV0 {
	return RunControlStatusV0(strings.ToLower(strings.TrimSpace(string(status))))
}

func IsTerminalRunControlStatusV0(status RunControlStatusV0) bool {
	switch NormalizeRunControlStatusV0(status) {
	case RunControlStatusStoppedV0, RunControlStatusCanceledV0:
		return true
	default:
		return false
	}
}
