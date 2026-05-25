package main

import (
	"strconv"
	"time"

	orquestaserver "orquesta/modulos/orquesta-server"
)

const defaultCodexDirectorMaxSubagentsPerAgentV0 = 6

func serverEffectiveConfigFromEnvV0(config orquestaserver.ConfigV0) orquestaserver.ServerEffectiveConfigV0 {
	config = orquestaserver.NormalizeConfigV0(config)
	codexRuntime := codexRuntimeEnvConfigFromEnvV0()
	stackCapacity := codexStackCapacityEnvConfigFromEnvV0()
	directorWaveLimits := codexDirectorWaveLimitsEnvConfigFromEnvV0()
	return orquestaserver.NormalizeServerEffectiveConfigV0(orquestaserver.ServerEffectiveConfigV0{
		SchemaVersion: orquestaserver.ServerEffectiveConfigSchemaVersionV0,
		Settings: []orquestaserver.ServerConfigSettingV0{
			serverConfigSettingV0("ORQUESTA_SERVER_MAX_RUNS_PER_TICK", strconv.Itoa(config.SupervisorCommand.MaxRunsPerTick), "server_supervisor", "Runs por tick", "Runs candidatos por pulso residente."),
			serverConfigSettingV0("ORQUESTA_SERVER_MAX_EXECUTIONS_PER_TICK", strconv.Itoa(config.SupervisorCommand.MaxExecutions), "server_supervisor", "Ejecuciones por tick", "Ejecuciones lanzadas por pulso residente."),
			serverConfigSettingV0("ORQUESTA_SERVER_DRAIN_MAX_EXTERNAL_WAITS", strconv.Itoa(config.SupervisorCommand.DrainLimits.MaxExternalWaits), "server_supervisor", "Esperas externas", "Esperas externas maximas por drain del supervisor residente."),
			serverConfigSettingV0("ORQUESTA_SERVER_TICK_INTERVAL_MS", strconv.Itoa(int(config.TickInterval/time.Millisecond)), "server_supervisor", "Intervalo tick ms", "Intervalo entre pulsos automaticos del servidor."),
			serverConfigSettingV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_TARGET_QUEUE", strconv.Itoa(config.IdleSelfImprovementTargetQueue), "autoprogramming", "Cola objetivo", "Tamano objetivo de cola de automejora."),
			serverConfigSettingV0("ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_MAX_REQUESTS", strconv.Itoa(config.IdleSelfImprovementMaxRequests), "autoprogramming", "Nuevas tareas por tanda", "Maximo de tareas nuevas por tanda de automejora."),
			serverConfigSettingV0("ORQUESTA_CODEX_MAX_BATCH_READY", strconv.Itoa(codexRuntime.Limits.MaxBatchReady), "codex_runtime", "Batch Codex ready", "Agentes ready a despachar por tanda Codex."),
			serverConfigSettingV0("ORQUESTA_CODEX_MAX_CONCURRENCY", strconv.Itoa(codexRuntime.Limits.MaxLiveProcesses), "codex_runtime", "Procesos Codex vivos", "Limite global de procesos Codex vivos; si esta lleno, la tarea queda pendiente sin consumirse."),
			serverConfigSettingV0("ORQUESTA_CODEX_REASONING_EFFORT", codexRuntime.ReasoningEffort, "codex_runtime", "Reasoning Codex", "Esfuerzo de razonamiento para agentes Codex."),
			serverConfigSettingV0("ORQUESTA_CAPACITY_REASONING_EFFORT", string(stackCapacity.ReasoningEffort), "capacity", "Reasoning capacidad", "Recomendacion de capacidad que recibe el stack."),
			serverConfigSettingV0("ORQUESTA_CODEX_DIRECTOR_WAVE_AGENTS", strconv.Itoa(directorWaveLimits.Agents), "director_wave", "Agentes por ola", "Agentes padre que puede lanzar el Director en una ola."),
			serverConfigSettingV0("ORQUESTA_CODEX_DIRECTOR_MAX_SUBAGENTS_PER_AGENT", strconv.Itoa(directorWaveLimits.MaxSubagentsPerAgent), "director_wave", "Subagentes por agente", "Fanout maximo de subagentes por agente padre en delegacion recursiva."),
			serverConfigSettingV0("ORQUESTA_CODEX_DIRECTOR_RECURSIVE_AGENT_BUDGET", strconv.Itoa(directorWaveLimits.RecursiveAgentBudget), "director_wave", "Presupuesto recursivo", "Presupuesto global de agentes en arbol recursivo."),
		},
	})
}

func serverConfigSettingV0(
	key string,
	value string,
	scope string,
	label string,
	description string,
) orquestaserver.ServerConfigSettingV0 {
	return orquestaserver.ServerConfigSettingV0{
		Key:             key,
		Value:           value,
		Scope:           scope,
		Label:           label,
		Description:     description,
		RestartBehavior: "restart_required",
		Editable:        true,
		Canonical:       true,
	}
}
