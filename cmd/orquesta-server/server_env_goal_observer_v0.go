package main

import (
	"os"
	"strings"
)

const (
	envServerGoalObserverEnabledV0            = "ORQUESTA_SERVER_GOAL_OBSERVER_ENABLED"
	envServerGoalObserverIntervalMSV0         = "ORQUESTA_SERVER_GOAL_OBSERVER_INTERVAL_MS"
	envServerGoalObserverMaxItemsV0           = "ORQUESTA_SERVER_GOAL_OBSERVER_MAX_ITEMS"
	envServerGoalObserverFingerprintEnabledV0 = "ORQUESTA_SERVER_GOAL_OBSERVER_FINGERPRINT_ENABLED"
)

func init() {
	serverEffectiveEnvRegistryV0[envServerGoalObserverEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "goal_observer",
		Label:       "Observador goal-first",
		Description: "Activa el loop residente que observa goals activos y refleja cierres sin usar el Director residente legacy.",
	}
	serverEffectiveEnvRegistryV0[envServerGoalObserverIntervalMSV0] = serverEnvSettingMetadataV0{
		Scope:       "goal_observer",
		Label:       "Intervalo observador goal-first ms",
		Description: "Intervalo propio del observador residente de goals activos; si no se define, hereda el tick general del servidor.",
	}
	serverEffectiveEnvRegistryV0[envServerGoalObserverMaxItemsV0] = serverEnvSettingMetadataV0{
		Scope:       "goal_observer",
		Label:       "Max goals observados",
		Description: "Limite de goals activos que observa cada tick residente.",
	}
	serverEffectiveEnvRegistryV0[envServerGoalObserverFingerprintEnabledV0] = serverEnvSettingMetadataV0{
		Scope:       "goal_observer",
		Label:       "Fingerprint observador goal-first",
		Description: "Activa una guarda opt-in para saltar observaciones residentes cuando la senal externa del goal no ha cambiado.",
	}
}

func serverGoalObserverEnabledFromEnvV0() (bool, bool) {
	if strings.TrimSpace(os.Getenv(envServerGoalObserverEnabledV0)) == "" {
		return true, false
	}
	return boolEnvOrDefaultV0(envServerGoalObserverEnabledV0, false), true
}

func serverGoalObserverFingerprintEnabledFromEnvV0() bool {
	return boolEnvOrDefaultV0(envServerGoalObserverFingerprintEnabledV0, false)
}
