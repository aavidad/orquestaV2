# Handoff para Claude - mejora continua Orquesta 2026-07-03

<!--
checkpoint_started: task-ref-doc-cleanup-handoff-plan-20260704 rework-1.
scope: handoff/relevo/plan docs only.
intent: preserve the source task ref before required checks and keep Claude
pointed at the current closed/superseded status for BUG-149, MEJ-106 and the
remote-WIP triage.
evidence: current diff scan found reusable prior edits in the authorized
write-set.
-->

## Proposito

Este documento deja el estado para revision posterior de Claude tras la tanda de
mejora continua dirigida con Orquesta y reparaciones puntuales integradas por
Codex. No es un cierre total de la app ni de todos los conectores: es el corte
verificable de la ola actual y de las incidencias observadas.

## Estado corto

- Fecha de corte: 2026-07-03.
- Worktree: limpio tras commits y push de la tanda; si Claude ve cambios, deben
  tratarse como trabajo posterior.
- Procesos: tras la limpieza no quedaban procesos `orquesta-server run`,
  `codebase-memory-mcp` ni `codex app-server --listen`.
- Verificacion global ejecutada: verde con `go test -count=1 ./...`,
  `go build ./...` y `git diff --check`.
- Actualizacion posterior al commit `a9f3b455`: MEJ-104/T290 queda integrado y
  pusheado como reparacion acotada antes de relanzar automejora real. El smoke
  real de presupuesto queda solo como validacion condicional antes de reactivar
  pilotajes caros.
- Excepcion operativa: MEJ-206 se lanzo por Orquesta, pero el piloto quedo sin
  progreso util con consumo alto de tokens y checkpoint invalido. Se paro el
  runtime y Codex integro una reparacion acotada, dejando bugs documentados.

## Frentes integrados

### T285 - despertar por resultado materializado

Estado: integrado y probado.

Archivos principales:

- `modulos/orquesta-app-codex-stack/goal_materialized_result_watcher_v0.go`
- `modulos/orquesta-app-codex-stack/goal_materialized_result_watcher_v0_test.go`
- `modulos/orquesta-server/runtime_background_worker_v0.go`
- Wiring en `cmd/orquesta-server/stack.go` y paquetes `modulos/orquesta-server`.

Resultado funcional: el stack observa resultados materializados y despierta el
runtime/background worker cuando hay evidencia nueva.

### MEJ-202 - property tests y falsos verdes

Estado: integrado y probado.

Archivos principales:

- `modulos/orquesta-run-coordinator/external_work_reconciliation_v0.go`
- `modulos/orquesta-run-coordinator/external_work_reconciliation_properties_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/estado_backend_v0.go`
- `modulos/orquesta-runtime-codex-appserver/estado_backend_properties_v0_test.go`
- `modulos/orquesta-estado-vivo/proyeccion_properties_v0_test.go`
- `modulos/orquesta-estado-vivo/testdeps/rapid`
- `go.mod`

Nota para Claude: `pgregory.net/rapid` se resuelve mediante reemplazo local de
test en `modulos/orquesta-estado-vivo/testdeps/rapid`; revisar sin convertirlo
en dependencia productiva.

### MEJ-203 - piloto de mutation testing

Estado: integrado y probado.

Archivos principales:

- `scripts/orquesta_mutation_pilot.sh`
- `scripts/test_orquesta_mutation_pilot.sh`
- `docs/runbooks/mutation_testing_piloto_2026-07-04.md`
- Evidencias bajo `scripts/docs/`.

Resultado funcional: harness acotado de mutacion con runbook y smoke local.

### MEJ-205 - golden evals

Estado: integrado y probado.

Archivos principales:

- `scripts/orquesta_golden_evals.sh`
- `docs/evals/`
- `docs/runbooks/orquesta_golden_evals_2026-07-04.md`

Resultado funcional: harness de evaluaciones doradas para detectar regresiones
en casos compactos.

### MEJ-206 - habilidades curadas para autoprogramacion

Estado: integrado por reparacion acotada tras piloto Orquesta bloqueado.

Archivos principales:

- `modulos/orquesta-autoprogramming/curated_skills_v0.go`
- `modulos/orquesta-autoprogramming/curated_skills_v0_test.go`
- `cmd/orquesta-server/idle_self_improvement_skills_v0.go`
- `cmd/orquesta-server/idle_self_improvement_skills_v0_test.go`
- `cmd/orquesta-server/idle_self_improvement_stack_v0.go`
- `cmd/orquesta-server/stack.go`
- `modulos/orquesta-server/supervisor_idle_goal_first_v0.go`
- `modulos/orquesta-server/supervisor_loop_v0_test.go`

Resultado funcional:

- Carga metadata compacta desde `skills/*/SKILL.md` solo desde composicion/cmd.
- Filtra entradas inseguras o con rutas absolutas/secretos.
- Enriquecimiento de peticiones idle con `SkillRefs` y `context_refs` de tipo
  `skill_ref`.
- No genera ni comitea automaticamente `SKILL.md`; solo puede proponer
  destilacion revisable.

Nota para Claude: no clasificar MEJ-206 como cierre autonomo de Orquesta. El
runtime se uso, pero quedo bloqueado; la integracion final es una excepcion de
reparacion local documentada.

### MEJ-104 - gobernador de presupuesto de automejora idle

Estado: integrado localmente con pruebas focales y validacion global verdes; no
reactivar automejora productiva hasta ejecutar un smoke real controlado.

Motivo de la excepcion: BUG-ORQ-20260703-154 demostro que relanzar Orquesta
sin gobernador podia volver a consumir muchos tokens sin progreso. Por eso
Codex hizo una reparacion local acotada antes de otro goal real.

Hecho:

- `orquesta-autoprogramming` incorpora
  `DecideAutoprogrammingIdleSelfImprovementBudgetV0` y tipos de config/uso/
  decision para `budget_unconfigured`, `within_budget`, `budget_deferred` y
  `budget_degraded`.
- `orquesta-server` decide antes de lanzar automejora idle, reduce el lote si
  el presupuesto restante solo permite parte de las goals, o aplaza con
  `budget_deferred` si no cabe.
- El uso diario se estima desde estado durable: goals idle usadas hoy, budget
  de contexto observado en receipt/result, y tokens de prompt cache publicados
  como evidencias `evidence-ref-codex-goal-cached-input-tokens-*`.
- El estado del servidor y `/api/v0/server/status` publican
  `idle_self_improvement_budget`.
- `orquesta.autoprogramming.status.v0` publica el mismo presupuesto mediante
  puerto opcional inyectado, no por dependencia directa del runtime.
- `cmd/orquesta-server` registra
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_GOAL_BUDGET` y
  `ORQUESTA_SERVER_IDLE_SELF_IMPROVEMENT_DAILY_CONTEXT_BUDGET_BYTES`.

Archivos principales:

- `modulos/orquesta-autoprogramming/idle_self_improvement_v0.go`
- `modulos/orquesta-server/supervisor_idle_budget_v0.go`
- `modulos/orquesta-server/status_tracker_idle_budget_v0.go`
- `modulos/orquesta-server/status_public_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0.go`
- `cmd/orquesta-server/autoprogramming_idle_budget_source_v0.go`
- Wiring en `modulos/orquesta-app-codex-stack`, `modulos/orquesta-app-gateway`
  y `cmd/orquesta-server/stack.go`.

Pruebas verdes tras el cambio:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming ./modulos/orquesta-server ./modulos/orquesta-mcp ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server
git diff --check
go test -count=1 ./...
go build ./...
```

Estado actualizado tras T290/Codex:

- El presupuesto pre-launch de MEJ-104 ya queda en codigo y tests:
  `budget_deferred`/`budget_degraded` viajan por state, status publico y
  `orquesta.autoprogramming.status.v0`.
- La parte "durante launch" queda cubierta por T290 (`eab3be97`): un goal idle
  activo con consumo creciente sin progreso util entra en
  `goal_high_consumption_without_progress`, se persiste como `blocked`, conserva
  rework accionable y solicita stop cooperativo por run-control. Incluye el caso
  de checkpoint invalido repetido.
- Queda recomendado un smoke real acotado de presupuesto antes de reactivar
  automejora/pilotajes caros. No se lanza aqui por la congelacion operativa
  vigente.

## Bugs documentados

Inventario actualizado en `docs/inventario_bugs_orquesta_2026-06-30.md` con:

- `BUG-ORQ-20260703-154`: cerrado por MEJ-104/T290; queda solo smoke real
  acotado como validacion antes de reactivar automejora/pilotajes.
- `BUG-ORQ-20260703-155`: cerrado; MEJ-206 ya no hereda criterios base ajenos
  en secciones ejecutables.
- `BUG-ORQ-20260703-156`: cerrado; shutdown ejecuta hooks tambien en rutas de
  timeout.
- `BUG-ORQ-20260703-157`: cerrado; el helper comun de readiness rechaza PID
  muerto antes de aceptar `addr`.
- `BUG-ORQ-20260703-159`: cerrado; los retries idle tras error/prepare_failed ya
  no quedan bloqueados por idempotencia stale.

Actualizacion posterior: `BUG-ORQ-20260703-149` queda cerrado como WIP remoto
triado/obsoleto: no integrar el patch completo ni extraer mas piezas salvo bug
nuevo con write-set propio y pruebas actuales. Las filas largas antiguas de
OPES/goal-first/shutdown/write-set/status siguen como deuda amplia y no son
regresion nueva de estos commits.

## Pruebas ejecutadas

Verificaciones focales:

```bash
go test -count=1 ./modulos/orquesta-autoprogramming
go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0IdleSelfImprovementGoalFirstPropagaSkillRefsDeRequestV0'
go test -count=1 ./cmd/orquesta-server -run 'TestServerStackSupervisorV0FiltroAnadeSkillCuradaCasadaV0|TestServerCuratedSkillsFromProjectV0'
go test -count=1 ./modulos/orquesta-autoprogramming ./cmd/orquesta-server ./modulos/orquesta-server
go test -count=1 ./
```

Verificaciones globales:

```bash
go build ./...
git diff --check
go test -count=1 ./...
```

Tras documentar bugs y este handoff se debe repetir al menos:

```bash
git diff --check
```

## Limpieza operativa

Se pararon los pilotos y backends temporales asociados a los workdirs bajo
`/tmp/claude-1000/.../scratchpad/pilot-*`.

Comprobacion final usada:

```bash
pgrep -af 'orquesta-server run|codebase-memory-mcp|codex app-server --listen'
```

Resultado observado: sin salida.

## Advertencias para revision de Claude

- No integrar truncados de `docs/autoprogramacion_orquesta_pendientes_2026-05-23.md`
  generados en worktrees piloto. En main no se integro ese cambio.
- No reabrir T285, MEJ-202, MEJ-203, MEJ-205 ni MEJ-206 salvo regresion
  demostrada por prueba o diff concreto.
- No usar `codebase-memory-mcp` para esta revision salvo consulta de grafo muy
  concreta; para este handoff bastan `rg`, `git diff` y tests.
- Revisar especialmente MEJ-206: aislamiento de composicion, ausencia de rutas
  absolutas/secretos en metadata compacta, y propagacion de `SkillRefs` a
  `GoalWorkSpec`.
- Verificar que las evidencias de pilotos no se confunden con codigo productivo.

## Pendiente aproximado

Olas cerradas en terminos de integracion verificable:

- T285.
- MEJ-201 / T291.
- MEJ-202.
- MEJ-203.
- MEJ-204 / T292.
- MEJ-104 / T290, con presupuesto pre-launch y corte durante ejecucion cubiertos
  por codigo/fakes; queda solo smoke real acotado antes de reactivar
  automejora/pilotajes caros.
- MEJ-205.
- MEJ-206, con excepcion documentada.

Cola congelada por orden del operador:

- No relanzar T286-EXP, MEJ-102, MEJ-105, MEJ-207 ni nuevos pilotajes de
  automejora desde este handoff.
- MEJ-103 residente queda supersedida por el director de escalada para esta
  tanda.

Condicionales, si siguen vigentes tras revision de backlog:

- MEJ-101: cierre OPES real temporal opt-in como validacion final de campo.

Cerrados/supersedidos tras este handoff:

- MEJ-106: deuda residual/ratchets cerrada por Codex local en modo deuda
  gobernada. La retirada de `legacy_director_loop` no se ejecuto aqui y queda
  como decision separada del operador, condicionada a checklist verificable.

## Actualizacion 2026-07-04 - limpieza documental con Orquesta

Codex lanzo Orquesta con tres goals paralelos, todos con write-set documental
disjunto:

- `task-ref-doc-cleanup-inventario-20260704` sobre
  `docs/inventario_bugs_orquesta_2026-06-30.md`.
- `task-ref-doc-cleanup-bitacora-20260704` sobre
  `docs/bitacora_correccion_pericial_2026-07-03.md`.
- `task-ref-doc-cleanup-handoff-plan-20260704` sobre este handoff,
  `docs/relevo_claude_orquestador_2026-07-03.md` y
  `docs/plan_mejora_continua_orquesta_2026-07-04.md`.

Resultado documental:

- `BUG-085` queda historico/supersedido por `BUG-164`; no contarlo como vivo
  salvo smoke OPES/productivo especifico o fallo externo de sandbox/proveedor.
- `BUG-088` queda cerrado funcionalmente; las menciones antiguas a "sigue
  abierto" son historicas y no deben alimentar el conteo vivo.
- `BUG-120` queda cerrado; sus notas intermedias de "no cierra el bug padre"
  estan marcadas como historicas cuando correspondia a ese bug.
- `BUG-149` queda cerrado como WIP remoto triado/obsoleto; no integrar el patch
  completo de 71 ficheros.
- `MEJ-106` queda cerrado como deuda gobernada; retirada de
  `legacy_director_loop` sigue siendo decision separada.

Incidencia nueva para Claude:

- `BUG-ORQ-20260704-165` queda abierto. Orquesta ejecuto y escribio los docs,
  pero `autoprogramming/status` y `observe_goal` devolvieron timeouts; antes de
  reiniciar con backend, `runs/control stop forced=true` sobre T260 fallo con
  `control_not_propagated_to_goal_backend` pese a no observarse app-server local
  vivo. Al cierre, `orquesta-server stop --force --reason ...` quedo sin
  respuesta mas de 60s con active works stale; se aborto el cliente y se paro el
  servidor local por SIGINT. No quedaron procesos `orquesta-server run`,
  `codex app-server`, tmux `orquesta-goal-*` ni `codebase-memory-mcp`; el state
  residual puede aparecer `stopped/degraded`. Revisar observabilidad/control
  goal-first, no reabrir la limpieza documental por este fallo.

## Actualizacion Codex 2026-07-04 noche 28

Commits ya empujados antes de este bloque:

- `63f68213 fix: reconciliar goal stale sin proceso vivo`.
- `89e390ac fix: materializar checkpoint antes del turn start`.

Avance de este bloque:

- `BUG-ORQ-20260701-073` pasa a cerrado funcionalmente: el checkpoint runtime
  inicial ya no se interpreta como artefacto parcial ni como falta de
  checkpoint.
- `autoprogramming/status` distingue `active_no_checkpoint_yet` de
  `active_checkpoint_only_yet`. Con checkpoint inicial pide
  `observe_goal_backend_require_next_artifact`; tambien lo hace
  `active_timeout_checkpoint_recent`.
- `BUG-ORQ-20260701-079` sigue abierto solo por limite duro pre-tool/stdout del
  app-server/proveedor y smoke real largo. No reabrirlo por checkpoint inicial:
  ese borde queda reducido por codigo y pruebas.
- Conteo vigente del inventario tras este bloque: 208 filas, 173 IDs unicos, 6
  bugs abiertos reales: `BUG-165`, `BUG-058`, `BUG-065`, `BUG-066`, `BUG-075`,
  `BUG-079`.

Pruebas focales ejecutadas:

- `go test -count=1 ./modulos/orquesta-mcp -run 'TestMCPAutoprogrammingStatusExecutorV0(GoalBloqueadoConBackendActivoPublicaSnapshotAccionable|CheckpointOnlyBajoConsumoPideSiguienteArtefacto|CheckpointOnlyHighConsumptionEsBloqueante|CheckpointOnlyConsumoMedioAvisaAntesDeUmbralAlto|GoalActiveTimeoutConCheckpointRecienteNoReplanificaAun|GoalActiveTimeoutConCheckpointEstancadoPromueveReplanV0)|TestMCPDomainWorkStatusHTTPHandlerV0NormalizaSenalesCheckpointComoEstadoOperacional'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0DetectaCheckpointEnWriteSet|TestStackGoalMaterializedRefsSourceV0DetectaArtefactosTxtBUG088V0'`
- `git diff --check`
- `go test -count=1 ./...`

## Checklist de Claude

1. Revisar `git status --short` y separar cambios de cada frente.
2. Revisar diff de MEJ-206 con foco en contratos y frontera core/composicion.
3. Confirmar que `docs/inventario_bugs_orquesta_2026-06-30.md` contiene los
   bugs 154-157, 163-165 y la lectura vigente 2026-07-04.
4. Confirmar MEJ-104 en `autoprogramming/status` con presupuesto bajo si se
   decide ejecutar smoke real acotado; no hacerlo automaticamente.
5. Confirmar que no quedan procesos vivos con el `pgrep` indicado arriba.
6. Si todo sigue verde, no preparar ola nueva salvo orden explicita: la cola
   esta congelada.
