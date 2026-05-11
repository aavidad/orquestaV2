package orquestaruncontrol

import "strings"

type RunControlStateNotFoundErrorV0 struct {
	RunRef string
}

func (err RunControlStateNotFoundErrorV0) Error() string {
	if strings.TrimSpace(err.RunRef) == "" {
		return "run_control_state_not_found"
	}
	return "run_control_state_not_found: " + strings.TrimSpace(err.RunRef)
}

func DefaultRunControlStateV0(runRef string) RunControlStateV0 {
	return RunControlStateV0{
		RunRef: strings.TrimSpace(runRef),
		Status: RunControlStatusRunningV0,
	}
}
