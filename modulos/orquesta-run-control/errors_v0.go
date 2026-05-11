package orquestaruncontrol

import "strings"

type RunControlStateNotFoundErrorV0 struct {
	RunRef string
}

type RunControlCompletionTargetErrorV0 struct {
	TargetStatus RunControlStatusV0
}

func (err RunControlStateNotFoundErrorV0) Error() string {
	if strings.TrimSpace(err.RunRef) == "" {
		return "run_control_state_not_found"
	}
	return "run_control_state_not_found: " + strings.TrimSpace(err.RunRef)
}

func (err RunControlCompletionTargetErrorV0) Error() string {
	target := strings.TrimSpace(string(err.TargetStatus))
	if target == "" {
		return "run_control_completion_target_invalid"
	}
	return "run_control_completion_target_invalid: " + target
}

func DefaultRunControlStateV0(runRef string) RunControlStateV0 {
	return RunControlStateV0{
		RunRef: strings.TrimSpace(runRef),
		Status: RunControlStatusRunningV0,
	}
}
