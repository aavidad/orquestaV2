package orquestaserver

import "strings"

type SupervisorWakeupV0 struct {
	Cause string
}

func (runtime *RuntimeV0) RequestSupervisorWakeupV0(cause string) bool {
	if runtime == nil || runtime.supervisor == nil || runtime.supervisorWakeups == nil {
		return false
	}
	if runtime.supervisorFrozenForShutdownV0() {
		return false
	}
	wakeup := SupervisorWakeupV0{Cause: strings.TrimSpace(cause)}
	if wakeup.Cause == "" {
		wakeup.Cause = "supervisor_wakeup"
	}
	select {
	case runtime.supervisorWakeups <- wakeup:
		return true
	default:
		return false
	}
}
