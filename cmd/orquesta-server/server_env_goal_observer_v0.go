package main

import (
	"os"
	"strings"
)

const (
	envServerGoalObserverEnabledV0  = "ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED"
	envServerGoalObserverMaxItemsV0 = "ORQUESTA_SERVER_GOAL_OBSERVER_MAX_ITEMS"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerGoalObserverEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "goal_observer",
		Label:       "Observador goal-first",
		Description: "Activa el loop residente que observa goals activos y refleja cierres sin usar el Director residente legacy.",
	}
	serverEffectiveEnvRegistryV0[envServerGoalObserverMaxItemsV0] = serverEnvSettingMetadataV0{
		Scope:       "goal_observer",
		Label:       "Max goals observados",
		Description: "Limite de goals activos que observa cada tick residente.",
	}
}

func serverGoalObserverEnabledFromEnvV0() (bool, bool) {
	if strings.TrimSpace(os.Getenv(envServerGoalObserverEnabledV0)) == "" {
		return true, false
	}
	return boolEnvOrDefaultV0(envServerGoalObserverEnabledV0, false), true
}
