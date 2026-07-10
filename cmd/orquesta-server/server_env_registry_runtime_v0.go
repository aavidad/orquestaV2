package main

func init() {
	for key, metadata := range map[string]serverEnvSettingMetadataV0{
		envCodexCommandV0: {
			Scope: "codex_runtime", Label: "Comando Codex",
			Description: "Ruta o comando del ejecutable Codex usado por la composicion local.",
		},
		envCodexProjectWorkDirV0: {
			Scope: "codex_runtime", Label: "Proyecto Codex",
			Description: "Directorio del proyecto que la composicion usa para resolver configuracion y ejecutar trabajo.",
		},
		envCodexPathV0: {
			Scope: "codex_runtime", Label: "PATH Codex",
			Description: "PATH proyectado al proceso Codex cuando la composicion necesita resolver comandos auxiliares.",
		},
		envCodexApprovalPolicyV0: {
			Scope: "codex_runtime", Label: "Aprobacion Codex",
			Description: "Politica no interactiva de aprobacion usada por ejecuciones Codex gobernadas.",
		},
		envCodexDirectorApprovalPolicyV0: {
			Scope: "codex_director", Label: "Aprobacion Director Codex",
			Description: "Politica de aprobacion especifica para tareas materializadas por el Director Operativo.",
		},
		envCodexAllowInteractiveApprovalV0: {
			Scope: "codex_runtime", Label: "Aprobacion interactiva Codex",
			Description: "Opt-in explicito para aprobacion interactiva; por defecto la composicion no la permite.",
		},
		envCodexExtraArgsV0: {
			Scope: "codex_runtime", Label: "Argumentos extra Codex",
			Description: "Argumentos adicionales auditables para el proceso Codex; no sustituye al routing tipado.",
		},
		envCodexSkillInstructionsJSONV0: {
			Scope: "codex_runtime", Label: "Instrucciones de skills Codex",
			Description: "Instrucciones compactas de skills proyectadas al runtime Codex por la composicion.",
		},
		envCodexWaitIntervalMSV0: {
			Scope: "codex_runtime", Label: "Intervalo espera Codex",
			Description: "Intervalo en milisegundos para observacion cooperativa de procesos Codex.",
		},
		envCodexStalledTicksV0: {
			Scope: "codex_runtime", Label: "Ticks sin progreso Codex",
			Description: "Numero de ticks sin progreso antes de publicar diagnostico de atasco Codex.",
		},
		envCodexLoopTicksV0: {
			Scope: "codex_runtime", Label: "Ticks repetidos Codex",
			Description: "Limite de acciones repetidas que el observador Codex admite antes de escalar el caso.",
		},
		envCodexNoActivitySecondsV0: {
			Scope: "codex_runtime", Label: "Sin actividad Codex",
			Description: "Segundos sin actividad observable antes de que el presupuesto de progreso requiera rework.",
		},
		envCodexDirectorProjectRefV0: {
			Scope: "codex_director", Label: "Proyecto Director Codex",
			Description: "Ref opaca del proyecto que el Director Operativo asocia a la ola materializada.",
		},
		envCodexDirectorDomainRefsV0: {
			Scope: "codex_director", Label: "Dominios Director Codex",
			Description: "Refs opacas de dominio inyectadas al Director Operativo para una ola concreta.",
		},
		envCodexDirectorDomainContextFilesV0: {
			Scope: "codex_director", Label: "Contexto Director Codex",
			Description: "Rutas locales de contexto de dominio que el adaptador transforma en recibos auditables.",
		},
		envDirectorMaxBurstsV0: {
			Scope: "legacy_director", Label: "Rafagas Director",
			Description: "Limite de rafagas del loop historico, mantenido solo para compatibilidad correctiva.",
		},
		envDirectorMaxStepsV0: {
			Scope: "legacy_director", Label: "Pasos Director",
			Description: "Limite de pasos del loop historico, mantenido solo para compatibilidad correctiva.",
		},
		envDirectorMaxDispatchesV0: {
			Scope: "legacy_director", Label: "Despachos Director",
			Description: "Limite de despachos del loop historico, mantenido solo para compatibilidad correctiva.",
		},
		envDirectorMaxCommandsV0: {
			Scope: "legacy_director", Label: "Comandos Director",
			Description: "Limite de comandos del loop historico, mantenido solo para compatibilidad correctiva.",
		},
		envDirectorMaxOutboxV0: {
			Scope: "legacy_director", Label: "Outbox Director",
			Description: "Limite de outbox del loop historico, mantenido solo para compatibilidad correctiva.",
		},
		envDirectorMaxExternalWaitsV0: {
			Scope: "legacy_director", Label: "Esperas externas Director",
			Description: "Limite de esperas externas del loop historico, mantenido solo para compatibilidad correctiva.",
		},
		envServerAllowRepeatedRunsV0: {
			Scope: "server_supervisor", Label: "Runs repetidos",
			Description: "Opt-in que permite reintentar runs repetidos bajo control del supervisor residente.",
		},
		envServerSupervisorMaxTicksV0: {
			Scope: "server_supervisor", Label: "Ticks supervisor",
			Description: "Limite de ticks de una invocacion acotada del supervisor residente.",
		},
		envServerDrainMaxBurstsV0: {
			Scope: "server_supervisor", Label: "Rafagas drain",
			Description: "Limite de rafagas que un drain del supervisor puede procesar.",
		},
		envServerDrainMaxStepsV0: {
			Scope: "server_supervisor", Label: "Pasos drain",
			Description: "Limite de pasos que un drain del supervisor puede procesar.",
		},
		envServerDrainMaxDecisionsV0: {
			Scope: "server_supervisor", Label: "Decisiones drain",
			Description: "Limite de decisiones causales que un drain del supervisor puede materializar.",
		},
		envServerDefaultPriorityV0: {
			Scope: "server_supervisor", Label: "Prioridad servidor",
			Description: "Prioridad por defecto de requests aceptados por el servidor.",
		},
		envReviewGateStrictGoLineBudgetV0: {
			Scope: "review_gate", Label: "Presupuesto Go revision",
			Description: "Opt-in de presupuesto estricto de lineas Go para la puerta de revision.",
		},
		envStartupCleanupModeV0: {
			Scope: "startup_cleanup", Label: "Modo limpieza arranque",
			Description: "Modo gobernado de reconciliacion de estado al arrancar; no borra evidencia sin recibo.",
		},
		envStartupCleanupScopeRefsV0: {
			Scope: "startup_cleanup", Label: "Scope limpieza arranque",
			Description: "Refs opacas que acotan la reconciliacion de estado al arrancar.",
		},
		envStartupQueueLimitV0: {
			Scope: "startup_cleanup", Label: "Cola arranque",
			Description: "Limite de elementos que la reconciliacion de arranque examina en una pasada.",
		},
	} {
		serverEffectiveEnvRegistryV0[key] = metadata
	}
}
