# Contrato del veredicto causal en superficies (v0)

Fecha: 2026-07-10. Autor: Claude Fable (revisor/director).
Implantado en la etapa A de `docs/guia_continuacion_agentes_2026-07-10.md`
(commits `5cf602a21`, `f24926d87`, `074310fb6`, `941b8897a`, `97648addf`).
Nucleo: `DerivarVeredictoCausalV0` en
`modulos/orquesta-estado-vivo/veredicto_causal_v0.go` (contratos del modulo
en `modulos/orquesta-estado-vivo/docs/contratos.md`).

## Regla comun a todas las superficies

- El veredicto viaja como campos nuevos OPCIONALES: `causal_verdict`
  (clase) y `causal_reason_code` (razon), sin romper campos existentes.
- Una superficie NO publica `running` si el veredicto no es
  `running_confirmed`. Si el veredicto contradice un state `running`
  (clases `terminal_by_artifact` o `process_dead_state_stale`), la
  superficie no publica running limpio y recomienda `reconcile_goal_state`.
- Sin fuente de evidencias inyectada, el comportamiento previo queda
  intacto (los campos no aparecen).

## Superficies y donde vive cada contrato

### observe_goal (MCP `orquesta.apps.observe_director_goal.v0`)

- Codigo: `modulos/orquesta-mcp/observe_app_director_goal_estado_vivo_v0.go`
  (`withCausalVerdictV0`, `applyMCPObserveAppDirectorGoalCausalVerdictV0`);
  wiring de la fuente real en
  `modulos/orquesta-app-codex-stack/goal_first_observe_mcp_executor_v0.go`.
- Disparo de `reconcile_goal_state`: `goal_status=running` + clase
  `terminal_by_artifact` o `process_dead_state_stale`.
- Tests: `modulos/orquesta-mcp/observe_app_director_goal_causal_verdict_v0_test.go`
  (`TestMCPObserveAppDirectorGoalExecuteProcesoMuertoNoPublicaRunningV0`,
  `...TerminalDurableNoPublicaRunningV0`,
  `...RunningConfirmadoConservaRunningV0`, `...SinFuenteNoPublicaVeredictoV0`)
  y `modulos/orquesta-app-codex-stack/goal_first_observe_mcp_executor_v0_test.go`
  (`TestCodexStackObserveAppDirectorGoalExecutorV0ExecuteCableaVeredictoCausalV0`).

### autoprogramming/status (MCP `orquesta.autoprogramming.status.v0`)

- Codigo: `modulos/orquesta-mcp/autoprogramming_status_estado_vivo_v0.go`
  (`applyCausalVerdictToStatusResultMCPAutoprogrammingV0`,
  `mcpAutoprogrammingEstadoVivoRunningContradichoV0`).
- Publica `causal_verdict`/`causal_reason_code` a nivel de resultado (para
  el run consultado) y en cada actionable de `stale_running`. Si el
  veredicto contradice un state running y la fase proyectada aun no publica
  el terminal, emite actionable bloqueante
  `estado_vivo_reconcile_goal_state` con `recommended_action=reconcile_goal_state`.
- Tests: `modulos/orquesta-mcp/autoprogramming_status_causal_verdict_v0_test.go`
  (`TestMCPAutoprogrammingStatusExecutorV0ProcesoMuertoPublicaVeredictoYReconcileV0`,
  `...RunningConfirmadoNoPideReconcileV0`).

### director/stats (MCP `orquesta.director.stats.v0`)

- Codigo: `modulos/orquesta-mcp/director_stats_estado_vivo_v0.go`
  (`applyEstadoVivoProjectionV0` devuelve el veredicto) y
  `director_stats_tool_v0.go` (campos en el resultado).
- La proyeccion de estado vivo ya bloqueaba running contradicho (marca
  closure blocked); ahora ademas expone el veredicto.
- Tests: `modulos/orquesta-mcp/director_stats_causal_verdict_v0_test.go`
  (`TestMCPDirectorStatsToolExecutorV0PublicaVeredictoCausalProcesoMuertoV0`,
  `...SinFuenteNoPublicaVeredictoCausalV0`).

### supervisor residente (no publica; decide)

- Codigo: `modulos/orquesta-app-codex-stack/goal_first_resident_rework_v0.go`
  (`maybeReconcileGoalFirstResidentDeadProcessV0`): antes de decidir
  rework/espera deriva el veredicto; con `process_dead_state_stale` marca el
  goal blocked+needs_rework (terminal reconciliable) y prepara rework
  acotado, en vez de esperar infinito. Proceso vivo confirmado queda intacto.
- Tests: `modulos/orquesta-app-codex-stack/goal_first_resident_rework_v0_test.go`
  (`TestRunSupervisorGoalFirstResidentReconciliaProcesoMuertoConVeredictoCausalV0`,
  `TestRunSupervisorGoalFirstResidentProcesoVivoNoReconciliaV0`).

## Clases y razones (referencia rapida)

| Clase | Publica running | Efecto en superficie |
| --- | --- | --- |
| `running_confirmed` | si | estado limpio |
| `terminal_by_artifact` | no | publica terminal; si state decia running: reconcile |
| `process_dead_state_stale` | no | reconcile_goal_state / rework residente |
| `divergent_needs_repair` | no | bloqueo con `reconcile_estado_vivo` (ya existia) |
| `indeterminate` | no | observar de nuevo; nunca verde |

Verificacion de este documento: los tests citados se ejecutaron en verde el
2026-07-10 (`go test -count=1` real por paquete; ver
`docs/pruebas_revisor_208h_2026-07-10.md` para la disciplina de conteo).
