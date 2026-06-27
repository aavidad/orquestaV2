package orquestaserver

import "strings"

type SupervisorWakeupV0 struct {
	Cause string
}

type ResidentDirectorWakeupV0 struct {
	Cause string
}

type GoalObservationWakeupV0 struct {
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

func (runtime *RuntimeV0) RequestResidentDirectorWakeupV0(cause string) bool {
	if runtime == nil ||
		runtime.residentDirector == nil ||
		!runtime.config.ResidentDirectorEnabled ||
		runtime.residentDirectorPausedV0() ||
		runtime.residentDirectorWakeups == nil {
		return false
	}
	if runtime.supervisorFrozenForShutdownV0() {
		return false
	}
	wakeup := ResidentDirectorWakeupV0{Cause: strings.TrimSpace(cause)}
	if wakeup.Cause == "" {
		wakeup.Cause = "resident_director_wakeup"
	}
	select {
	case runtime.residentDirectorWakeups <- wakeup:
		return true
	default:
		return false
	}
}

func (runtime *RuntimeV0) RequestGoalObservationWakeupV0(cause string) bool {
	if runtime == nil ||
		!runtime.goalObservationAvailableV0() ||
		runtime.goalObservationWakeups == nil {
		return false
	}
	if runtime.supervisorFrozenForShutdownV0() {
		return false
	}
	wakeup := GoalObservationWakeupV0{Cause: strings.TrimSpace(cause)}
	if wakeup.Cause == "" {
		wakeup.Cause = "goal_observation_wakeup"
	}
	select {
	case runtime.goalObservationWakeups <- wakeup:
		return true
	default:
		return false
	}
}
