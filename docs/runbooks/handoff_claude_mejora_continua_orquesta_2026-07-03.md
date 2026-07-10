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

## Actualizacion Codex 2026-07-04 tarde 7

Claude anadio y commiteo las ampliaciones del wizard en la rama
`trabajo/plataforma-agentes` hasta `5eba766a`:

- `a0bb1627`: capa tecnica T1-T8 y motor de exclusion.
- `33b8532d`: ayuda en lenguaje llano por opcion.
- `f3b06ccb`: boton "explicamelo todo" y preguntas libres.
- `bbfbaf9e` y `5eba766a`: bot guia/RAG y LLM opt-in con presupuesto.

Codex intento ejecutar en paralelo T1-T7 por Orquesta. Resultado: no contar esa
tanda como avance funcional. El lote conjunto fallo por lifecycle/ownership de
`app_server_tmux`; los relanzamientos aislados terminaron en checkpoint
invalid, `blocked` por alto consumo o `backend_still_running` en shutdown. Se
documento como reproduccion viva de `BUG-ORQ-20260704-165` en el inventario y
en la bitacora.

Subagentes read-only usados:

- Euler: confirma que G2 del wizard esta, pero faltan T1-T8, exclusiones por
  hechos, ayuda total, glosario, "explicamelo todo" y bot RAG/MCP.
- Darwin: confirma que los pilotajes no son verdes; hay que reparar lifecycle
  goal-first/Codex tmux y shutdown cooperativo antes de volver a declarar
  autonomia amplia.

Siguiente trabajo correcto:

1. Relanzar por Orquesta un goal T7A estrecho: `orquesta-web` solo, contratos
   del wizard, motor de hechos/exclusion, ayuda `HelpKey`/`ExampleKey`,
   contraste con justificacion y consulta de comprension sin avanzar turno.
2. No meter bot RAG/MCP en T7A. Dejar T7B para corpus/indice determinista y
   T7C para tool `orquesta.nueva_app.wizard.bot.v0` + web chat + LLM opt-in.
3. En todos los goals: arquitectura hexagonal, i18n, conectores por
   puertos/adaptadores, y cero opciones "a pelo" en nucleo generico.
4. Usar `ORQUESTA_AUTOPROGRAMMING_CHECKPOINT_ONLY_HIGH_CONSUMPTION_TOKENS` si
   se sube el umbral; el nombre anterior usado por Codex era incorrecto y dejo
   el default de 100000.

## Actualizacion Codex 2026-07-04 tarde 8

Se revisaron los dos documentos indicados por el operador:

- `docs/auditoria_envs_pisadas_2026-07-04.md`
- `docs/instrucciones_director_codex_2026-07-04.md`, TAREA-8 completa.

Primer corte aplicado de la ola 1 de TAREA-8:

- `ORQUESTA_CODEX_CODE_HOME` es la canónica para auth/config Codex.
  `CODEX_HOME` queda como alias legacy con `deprecated_env_used` si es la unica
  fuente, y `env_alias_conflict` si difiere de la canónica. No se publican rutas
  crudas en `effective_config`.
- `ORQUESTA_OPES_BASE_URL` es la canónica para OPES. `OPES_BASE_URL` queda como
  alias legacy con el mismo diagnostico.
- `ORQUESTA_CODEX_HOME` se mantiene, pero solo como HOME del proceso Codex; no
  sustituye a `ORQUESTA_CODEX_CODE_HOME`.
- `BUG-ORQ-20260704-173` documenta el cierre parcial.

Verificado:

- `go test -count=1 ./cmd/orquesta-server`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=511`

Pendiente para Claude si sigue TAREA-8:

- `ORQUESTA_SERVER_URL`/`ORQUESTA_BASE_URL`: diagnostico de alias/conflicto.
- Timeouts de smoke con unidades mezcladas: corregir doc stale y mover a
  prefijo/perfil claro.
- `ORQUESTA_GUARDIAN_*`: registry de child-process env o excepcion explicita.
- TAREA-8.1/8.2: fichero canónico y guard `config_projection_mismatch`.

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

## Actualizacion Codex 2026-07-04 noche 29

Commits ya empujados antes de este bloque:

- `a3cdcff8 fix: distinguir checkpoint inicial en status goal-first`.

Avance de este bloque:

- `BUG-ORQ-20260704-165` y `BUG-ORQ-20260701-065` quedan reducidos en el borde
  CLI/HTTP de shutdown: si el `POST /api/v0/server/shutdown` falla por
  transporte, timeout o conexion cortada sin cuerpo, `orquesta-server stop`
  consulta `/status` y devuelve `shutdown_not_ready ...` con
  `backend_still_running`, `active_work` y refs compactas cuando el state
  publico conserva trabajo activo. Ese error bloquea tambien la escalada por
  `--force`; ya no queda un `shutdown_timeout` opaco cuando `/status` contiene
  causa accionable.
- Auditoria OPES read-only por subagente, sin ediciones ni
  `codebase-memory-mcp`: `BUG-075` es el siguiente frente mas cerrable por
  codigo/test; `BUG-058` requiere fake/temporal de ciclo de paquete para no
  sobrecerrar; `BUG-066` necesita smoke temporal real de external-work/OPES y
  reconciliacion de cleanup, no solo unit tests.

Conteo operativo tras esta tanda local:

- Inventario: 208 filas de bugs, 173 IDs unicos.
- Filas abiertas mantenidas: `BUG-165`, `BUG-058`, `BUG-065`, `BUG-066`,
  `BUG-075`, `BUG-079`.
- Deduplicando por ultimo registro, solo `BUG-058` y `BUG-066` quedan como IDs
  claramente abiertos, pero no borrar las filas abiertas antiguas sin limpieza
  documental explicita.

Pruebas focales ejecutadas antes del commit de esta tanda:

- `go test -count=1 ./cmd/orquesta-server -run 'TestRequestServerShutdownV0(ErrorTransporteConsultaStatusAccionable|NoPropagaBodyCrudoEnError|ReadyNoSaltaActiveWorkPersistido|ReadyNoSaltaActiveWorksEstructurados|ReadyNoSaltaGoalActionsSinActiveWork|CortaEsperaSiStatusBackendTieneRunsPendientes)|TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado|TestShutdownClientReadyV0'`
- `git diff --check`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./...`

Pendiente para Claude si reanuda desde aqui:

- Validar `git status --short --branch`.
- Confirmar el commit/push de esta tanda si aparece como pendiente.
- No cerrar `BUG-075/058/066` sin las evidencias indicadas arriba.

## Actualizacion Codex 2026-07-04 noche 30

Avance de este bloque:

- `BUG-ORQ-20260701-075` queda reducido por un contrato ejecutable nuevo en
  `orquesta-opes-director`: `OPESArtifactQualityContractV0`.
- El contrato valida campos/refs estructurados minimos para HTML, visuales,
  audio, tutor/RAG, fuentes, revisiones, supuestos, juegos, ayuda y
  reutilizacion visual. No usa rails por texto libre.
- Si una entrega terminal trae evidencia nominal `opes-final-evidence:*` pero
  no trae manifest/refs/QA estructurada del artefacto, el registro queda
  `needs_rework`, settlement `artifact_quality_contract_failed` y followup
  `review_artifact_quality`.
- Si falta toda la evidencia minima, se conserva el diagnostico anterior
  `required_evidence_missing`; no se mezcla con artifact quality.
- `BUG-075` sigue abierto para smoke temporal OPES/external-work con artefactos
  reales y para demostrar que no hay reescritura tardia tras suficiente entrega.

Incidencia operativa cerrada:

- `BUG-ORQ-20260704-172`: `go test` fallo inicialmente por `/home` al 100% y
  cache Go de 17G en `/home/alberto/.cache/go-build`. Se limpio con
  `go clean -cache` y se verifico usando `GOCACHE=/tmp/orquesta-codex-gocache`
  y `GOTMPDIR=/tmp/orquesta-codex-gotmp`.

Pruebas ejecutadas en esta tanda:

- `go test -count=1 ./modulos/orquesta-opes-director -run 'TestValidateOPESArtifactQualityContractV0CubreWorkKindsMinimos|TestProduceOPESCausalJobsV0(VisualConEvidenceRefPeroContratoFallidoCreaRework|HTMLConContratoArtifactQualityPassNoCreaRework|QuestionBankConEvidenceRefPeroContratoFallidoCreaRework|QuestionBankConContratoPassNoCreaRework)'`
- `go test -count=1 ./modulos/orquesta-opes-director`
- `go test -count=1 ./modulos/orquesta-opes-director ./modulos/orquesta-opes-bridge`

## Actualizacion Codex 2026-07-04 noche 31

Avance de este bloque:

- `BUG-ORQ-20260701-058` queda reducido en `orquesta-opes-director`.
- El registro OPES ya no libera un `completed_syllabus_package` aunque el
  `manifest_cierre` traiga HTML/RAG/audio/tests/tutor/visual/QA, QA estricta y
  reports, si faltan refs durables de resultados `OPESTopicQualityContractV0`
  por tema (`topic_quality_contract_result_refs` o
  `topic_quality_contract_results`).
- En el caso incompleto mantiene `registry_action=update`,
  `proposed_status=pendiente_validacion_paquete_final`,
  `pending_refs=final-package-manifest-closure-evidence-required` y rework
  causal `finalize_temario_package`.
- El caso positivo de liberacion exige ahora `topic-quality-contract-result-ref`
  en el manifest y lo conserva como `settled_refs`.
- No cerrar `BUG-058` todavia: esto prueba el borde local de registro y lo
  alinea con `orquesta-app-codex-stack`, pero falta smoke temporal OPES de todo
  el arbol tema asentado -> derivados -> paquete final.

Archivos tocados en esta tanda:

- `modulos/orquesta-opes-director/topic_registry_v0.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `modulos/orquesta-opes-director/README.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Pruebas ejecutadas en esta tanda:

- `go test -count=1 ./modulos/orquesta-opes-director -run 'TestProduceOPESCausalJobsV0PaqueteFinal'`
- `go test -count=1 ./modulos/orquesta-opes-director`

## Actualizacion Codex 2026-07-04 noche 32

Avance de este bloque:

- `BUG-ORQ-20260701-058/066` queda reducido con un smoke offline transversal
  en `orquesta-app-codex-stack`: goal-first OPES escribe artefacto, el stack
  produce receipt de dominio, el productor OPES genera `update_topic_registry`
  y el registro pasa de texto asentado con derivados pendientes a paquete final
  liberado solo cuando hay refs de resultados `OPESTopicQualityContractV0`.
- Se corrigio una frontera real de `orquesta-opes-director`: los refs agregados
  de calidad por tema que viajan en un `final_domain_package` ya no activan por
  si solos la QA textual de tema. Si un paquete final trae campos textuales o
  estado QA explicito, la validacion textual sigue pudiendo declararse; el
  manifest agregado se valida por el contrato de paquete final.
- `BUG-ORQ-20260701-079` queda mejor documentado, no cerrado: el contrato
  Codex Goal declara `direction_contract`, checkpoint temprano,
  `max_text_bytes=16384` y `thread_read_max_bytes=256 KiB`, pero el limite duro
  previo a stdout de herramientas internas sigue dependiendo del proveedor o de
  mediacion real del app-server.
- Subagentes read-only usados en esta tanda: 4 terminados, 0 corriendo. Dos
  confirmaron que `BUG-058/066` solo se podia reducir por smoke local sin
  cerrar el E2E real; otro confirmo que `BUG-079` no tenia patch runtime seguro
  sin soporte de proveedor.

Archivos tocados en esta tanda:

- `modulos/orquesta-app-codex-stack/external_work_goal_first_opes_sequence_v0_test.go`
- `modulos/orquesta-opes-director/topic_registry_quality_v0.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `modulos/orquesta-runtime-codex-goal/docs/contratos.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Pruebas focales ejecutadas antes de la verificacion amplia:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0OPESGoalFirstLifecycleAsientaDerivadosYCierraRegistroFinalConTopicQualityV0'`
- `go test -count=1 ./modulos/orquesta-opes-director -run 'TestProduceOPESCausalJobsV0PaqueteFinalCompleteConManifestCompatibleYQATernaLiberaRegistro'`

Pendiente para no sobrecerrar:

- `BUG-058/066`: smoke temporal OPES real via `/api/v0/external-work/observe`,
  sin OPES productivo, demostrando no reescritura tardia y registro final sin
  active/stale goal residual.
- `BUG-079`: soporte runtime/proveedor para limite duro pre-tool stdout o
  mediacion del app-server, mas smoke largo con proveedor real.

## Actualizacion Codex 2026-07-04 noche 33

Avance de este bloque:

- `BUG-ORQ-20260701-075` queda reducido con un smoke offline transversal en
  `orquesta-app-codex-stack`.
- El nuevo test
  `TestCodexStackV0OPESGoalFirstArtifactQualityHTMLDerivadoReworkYPassV0`
  cubre `/api/v0/external-work/run` fake, goal-first fake,
  `ObserveAppDirectorGoalV0`, submit DomainWork y productor OPES.
- Caso negativo: HTML terminal con evidencia nominal
  `opes-final-evidence:html_site_publicable` pero sin
  `html_topic_pages_manifest` ni `html_validation_report` genera
  `artifact_quality_status=needs_rework`,
  `settlement_reason=artifact_quality_contract_failed` y followup
  `review_director_consolidation` con
  `recommended_action=review_artifact_quality`.
- Caso positivo: HTML con manifest y reporte estructurado queda
  `artifact_quality_status=complete`, `artifact_quality_issue_count=0` y no
  crea followup de rework.

Archivos tocados en esta tanda:

- `modulos/orquesta-app-codex-stack/external_work_goal_first_opes_sequence_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Prueba focal ejecutada:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0OPESGoalFirstArtifactQualityHTMLDerivadoReworkYPassV0'`

No sobrecerrar:

- No prueba OPES temporal real, proveedor real, servidor real, ausencia de
  reescritura tardia ni calidad semantica/links HTML reales.
- Solo prueba pipeline offline stack -> DomainWork -> OPES director para
  `artifact_quality_failed/pass`.

## Actualizacion Codex 2026-07-04 noche 34

Avance de este bloque:

- `BUG-ORQ-20260704-165` y `BUG-ORQ-20260701-065` quedan reducidos en el
  cliente CLI shutdown.
- `postServerShutdownRequestV0` usa ahora un timeout por intento, y
  `waitServerShutdownReadyV0` acota cada re-POST al deadline restante.
- Si el POST inicial o un re-POST de `/api/v0/server/shutdown` se cuelga, el
  cliente consulta `/status` y devuelve `shutdown_not_ready` con
  `active_work_refs` compactas. No permite señal forzada si el status conserva
  active work.

Archivos tocados en esta tanda:

- `cmd/orquesta-server/shutdown_client.go`
- `cmd/orquesta-server/shutdown_client_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Pruebas focales ejecutadas:

- `go test -count=1 ./cmd/orquesta-server -run 'TestRequestServerShutdownV0(PostColgadoConsultaStatusAccionable|ErrorTransporteConsultaStatusAccionable|CortaEsperaSiStatusBackendTieneRunsPendientes)|TestWaitServerShutdownReadyV0(RepostColgadoRespetaDeadlineYDevuelveStatusAccionable|CortaTrasRepostNoRecuperable)|TestShutdownRequestErrorAllowsSignalV0SoloConTimeoutYEstadoDrenado'`

No sobrecerrar:

- Esto cierra solo el subcaso CLI: no quedarse sin cuerpo accionable si el POST
  shutdown cuelga y `/status` conserva active work.
- `BUG-165/065` siguen abiertos hasta smoke real amplio con proveedor/status
  lento y coordinacion automatica completa
  `backend/checkpoint/stop/cancel/wait`.

## Actualizacion Codex 2026-07-04 noche 35

Avance de este bloque:

- `BUG-ORQ-20260701-058/066` quedan reducidos por dos cambios locales
  complementarios.
- `external_job_stats` de `orquesta-app-codex-stack` ahora recibe el ledger de
  DomainWork y revalida un cierre goal-first aceptado antes de publicar
  `completed`. Si el receipt aceptado no cubre el contrato durable del ledger,
  degrada el external job a `blocked` y conserva el issue causal.
- `orquesta-opes-director` deja de publicar `review_director_consolidation` y
  `assemble_topic` como `next_required_work_kinds` despues de `settled_text`;
  el texto asentado deriva a visuales, tests, revisiones, audio/tutor/html y
  paquete final, no a reescritura textual sin rework.

Archivos tocados en esta tanda:

- `modulos/orquesta-app-codex-stack/external_job_stats_source_v0.go`
- `modulos/orquesta-app-codex-stack/external_job_stats_source_v0_test.go`
- `modulos/orquesta-app-codex-stack/external_work_goal_first_opes_sequence_v0_test.go`
- `modulos/orquesta-app-codex-stack/sources_v0.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `modulos/orquesta-opes-director/topic_registry_settlement_v0.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Pruebas focales ejecutadas:

- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackExternalJobStatsSourceV0GoalFirst(AceptadoCompletaJob|AcceptedConIssuesNoCompletaJob|AcceptedSinReceiptNoCompletaJob|AcceptedConLedgerIncompletoNoCompletaJob)'`
- `go test -count=1 ./modulos/orquesta-opes-director -run 'TestProduceOPESCausalJobsV0NoBloqueaRegistroConQATemaCompletaV0'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestCodexStackV0OPESGoalFirstLifecycleAsientaDerivadosYCierraRegistroFinalConTopicQualityV0'`

Verificacion amplia:

- `git diff --check`
- `GOCACHE=/tmp/orquesta-codex-gocache GOTMPDIR=/tmp/orquesta-codex-gotmp go test -count=1 ./...`

No sobrecerrar:

- No prueba OPES temporal real ni proveedor real.
- No demuestra cierre agregado de todo el arbol OPES.
- No demuestra por si solo ausencia de reescritura tardia en un goal ya vivo;
  solo elimina dos work-kinds textuales de la lista de siguientes trabajos tras
  `settled_text` y evita falso `completed` si el ledger contradice el receipt.

## Actualizacion Codex 2026-07-04 noche 36

Avance de este bloque:

- `BUG-ORQ-20260701-079` queda reducido en el borde `app_server_command`.
- El protocolo command ya no acepta lineas JSON-RPC de hasta 1 MiB: el default
  queda en 256 KiB, alineado con `thread_read_max_bytes`.
- `thread/read` mantiene `codex_app_server_thread_read_response_too_large`.
  Los demas metodos, incluido `turn/start`, devuelven
  `codex_app_server_command_response_too_large` si la respuesta stdout supera
  el presupuesto.

Archivos tocados en esta tanda:

- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_command_protocol_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `modulos/orquesta-runtime-codex-goal/docs/contratos.md`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Pruebas focales ejecutadas:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestCodexAppServerCommandProtocol(TurnStartResponseBudget|ThreadReadResponseBudget)V0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-goal -run 'TestBuildCodexGoalStartPacketV0IncluyeContratoDeDireccion'`

No sobrecerrar:

- No es enforcement duro pre-tool dentro del proveedor. Reduce stdout gigante en
  el transporte command de Orquesta, pero si el proveedor ejecuta una herramienta
  interna y quema tokens antes de exponer el resultado, sigue haciendo falta
  soporte del runtime/proveedor o mediacion real de herramientas por app-server.
- Falta smoke largo real con proveedor.

## Corte Codex para auditoria 2026-07-04 tarde

El operador paro la implementacion para preparar auditoria. No se habia aplicado
ningun parche despues de `f4c9984e`.

Estado verificado:

- Rama `trabajo/plataforma-agentes`.
- Git limpio y sincronizado con origin (`0 0` en `@{u}...HEAD`).
- HEAD `f4c9984e fix: acotar respuestas command del app-server`.
- Inventario: 209 filas, 167 IDs unicos, 6 bugs abiertos reales.
- No quedaban procesos `orquesta-server run`, `codex app-server`,
  `codebase-memory-mcp`, `orquesta-goal-*` ni `go test` vivos al cierre de este
  corte.

Lo que se estaba revisando:

- `BUG-ORQ-20260704-165` y `BUG-ORQ-20260701-065`, por ser los residuales mas
  transversales del nucleo: observabilidad/control goal-first lento o stale,
  shutdown con backend Goal propio y reconciliacion tras cortes externos.

Hallazgos confirmados:

- `MarkServerProcessStaleStateV0` ya limpia actividad viva cuando el proceso
  registrado no existe.
- `NormalizeStoppedServerSnapshotV0` ya limpia `shutdown_active_work`,
  `shutdown_goal_actions`, async work y timeout stale en snapshots `stopped`.
- `MarkStoppedV0` y `MarkRuntimeStoppedV0` limpian proyecciones heredadas de
  startup/shutdown.
- `orquesta-server status` reconcilia snapshots `stopped` sucios y persiste la
  limpieza.
- `shutdown_freeze` conserva `active_work_refs` y `goal_actions` cuando una
  respuesta HTTP sin cuerpo/conflictiva podria borrar evidencia necesaria.
- `goal_actions` bloqueantes se publican en status y evitan falso `ready`;
  `cleanup_completed` no cuenta como bloqueante.

Mensaje para Claude:

No resolveria `BUG-165/065` con otro refactor amplio. La siguiente accion debe
ser un test focal. Construir un snapshot `Status="stopped"` heredado de timeout,
sin active work vivo, pero con narrativa contradictoria de shutdown
(`shutdown_status`/`shutdown_ready`). Si el test demuestra incoherencia real,
ajustar `NormalizeStoppedServerSnapshotV0` para publicar una parada coherente,
no solo campos activos limpios. Si el test no falla, no inventar codigo: pasar
al smoke real amplio de observabilidad/control lento.

Write-set recomendado si se continua:

- `modulos/orquesta-server/status_process_stale_v0.go`
- `modulos/orquesta-server/status_process_stale_v0_test.go`
- opcionalmente `cmd/orquesta-server/command_public_output_v0_test.go` si hace
  falta cubrir la proyeccion CLI.

Comandos recomendados:

```bash
git status --short --branch
GOCACHE=/tmp/orquesta-codex-gocache GOTMPDIR=/tmp/orquesta-codex-gotmp go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server
```

No actualizar el inventario como cierre salvo que haya test rojo, parche y
verificacion. No tocar OPES/MCP en este microfrente.

## Actualizacion Codex 2026-07-04 tarde 2

Avance de este bloque:

- La hipotesis del corte de auditoria se confirmo con test rojo.
- `NormalizeStoppedServerSnapshotV0` ahora trata como snapshot sucio un
  `Status="stopped"` que conserve narrativa vieja de shutdown aunque no tenga
  active work: por ejemplo `shutdown_status=stop_timeout` y
  `shutdown_ready=false`.
- La reconciliacion fija `shutdown_status=stopped` y `shutdown_ready=true` al
  limpiar la proyeccion de shutdown de un servidor ya parado.
- Tambien limpia contadores residuales de shutdown (`shutdown_runs_*`, agentes
  en vuelo y checkpoints pendientes), para que un `stopped` no conserve
  narrativa de trabajo activo.
- `orquesta-server status` queda cubierto para persistir esa correccion en el
  statefile, evitando que el operador vea un servidor `stopped` con narrativa de
  timeout activo ya sin proceso ni active work.
- Revision subagente `Hume`: veredicto coherente, riesgo bajo; no encontro
  contrato interno que dependa de `stopped + shutdown_status=stop_timeout`.
  Sugirio blindar `shutdown_runs_*`, cubierto en el test de contadores.

Archivos tocados:

- `modulos/orquesta-server/status_process_stale_v0.go`
- `modulos/orquesta-server/status_process_stale_v0_test.go`
- `cmd/orquesta-server/command_public_output_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`

Pruebas focales ejecutadas:

- `go test -count=1 ./modulos/orquesta-server -run 'TestNormalizeStoppedServerSnapshotV0|TestStatusTracker(Stopped|RuntimeStopped)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestStatusServerCommandV0ReconciliaStopped'`
- `go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server`

No sobrecerrar:

- Esto reduce `BUG-165/065` en el subcaso statefile stopped con narrativa stale.
- No cierra el smoke real amplio de observabilidad/control lento.
- No cierra la coordinacion automatica completa
  backend/checkpoint/stop/cancel/wait.

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

## Actualizacion Codex 2026-07-04 tarde 3

Avance aplicado y verificado para relevo de Claude:

- App-server (`BUG-079`): se redujeron bordes locales de salida gigante. El
  lector JSON-RPC legacy queda en 256 KiB, `stderr` command usa buffer tail
  acotado y la deteccion de errores por log lee solo la cola del fichero.
- MCP/autoprogramming (`BUG-079` / `BUG-165` residual): un goal `running` con
  backend activo, alto consumo y cero checkpoint/artefactos/receipts ya entra
  en `goal_active_no_checkpoint_high_consumption` con
  `replan_narrow_context`. La guarda excluye backend stale sin proceso, que
  sigue por `run_control_reconcile_external_cleanup`.
- Shutdown (`BUG-065/165`): test combinado para dos goals activos valida
  rePOSTs con `cleanup_goal_backends=true` hasta `cleanup_completed` no
  bloqueante.
- OPES connector (`BUG-058/066`): receipts con `CompleteJob=true` aceptan
  estados terminales nativos `completed`, `done`, `settled`; `pending` sigue
  invalido.
- TAREA-6/MEJ-106: no se cambio el ratchet. La metrica real actual es
  `env_vars_orquesta=513`; el objetivo `511` requiere retirar/consolidar dos
  nombres `ORQUESTA_*` antes de bajar `env_vars_budget_test.go`.

Archivos usados/modificados:

- `cmd/orquesta-server/shutdown_client_v0_test.go`
- `modulos/orquesta-mcp/autoprogramming_status_stale_running_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0.go`
- `modulos/orquesta-mcp/autoprogramming_status_tool_v0_test.go`
- `modulos/orquesta-opes-connector/result_v0.go`
- `modulos/orquesta-opes-connector/rest_client_receipt_validation_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_command_protocol_v0.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_rpc_v0.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Verificacion:

- `git diff --check`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-opes-connector ./modulos/orquesta-runtime-codex-appserver ./cmd/orquesta-server`
- `go test -count=1 ./...`

Pendiente real:

- `BUG-079`: enforcement duro pre-tool del runtime/proveedor y smoke largo real.
- `BUG-065/165`: smoke real amplio con proveedor/status lento y coordinacion
  automatica completa.
- `BUG-058/066`: lifecycle OPES end-to-end con instancia temporal y
  external-work real.
- `MEJ-106`: bajar de 513 a 511 tras consolidar dos env vars reales.

Nota de cierre operativo:

- Git quedo limpio y empujado hasta `535ab198`.
- Sigue vivo un pilot ajeno previo bajo
  `/tmp/claude-1000/-home-alberto-Trabajo-orquesta/8be66426-b9e9-428a-a040-c4ee62477495/scratchpad/pilot-t297`.
  No se paro porque `/api/v0/server/shutdown` informo goal activo
  `run_ref=request-ref-t297-wizard-g1-20260704-001`,
  `goal_ref=goal-ref-task-autoprogramming-5b1bf1c819cc-g01` y backend
  `orquesta-goal-2b3a252e09893ce4` vivo. `autoprogramming/status` lo clasifica
  como `partial_artifacts_written` con accion
  `review_partial_artifacts:run:request-ref-t297-wizard-g1-20260704-001`.
  Si Claude retoma ese pilot, debe revisar/cerrar el goal o pararlo por control
  gobernado antes de limpiar procesos.

## Coordinacion wizard TAREA-7 (Claude, 2026-07-04)

Hubo doble implementacion simultanea de G1: la del director Codex en el repo
principal (canonica: ya cableada al endpoint guided, suite orquesta-web verde)
y la del pilot t297 de Orquesta (goal cerrado blocked solo por socket de
httptest en sandbox; suite verde fuera del sandbox). Decision: se conserva la
version del repo. La rama `pilot-t297` queda viva SOLO como cantera: contiene
12 tests focales (R1-R8 individuales, contraste, defaults, agenda, i18n owner)
en `modulos/orquesta-web/nueva_app_wizard_gaps_v0_test.go`. Codex: adoptar de
ahi los tests focales que su version no cubra y despues borrar la rama.
Pilot t297 limpiado (server+tmux); goal habia alcanzado result terminal.
Falta del diseno: G2 (tool MCP + render web completo) y G3 (catalogo i18n en
completo), segun docs/diseno_wizard_programacion_2026-07-04.md.

## Actualizacion Codex 2026-07-04 tarde 4

Continuacion de TAREA-7/7b tras revisar el pilot t297 y la implementacion
principal.

Integrado:

- Nucleo web del wizard en `modulos/orquesta-web/nueva_app_wizard_*`:
  preguntas ricas, opciones con recomendacion, contraste usuario vs
  recomendacion, defaults de ingenieria y aceptacion automatica de
  recomendaciones.
- Endpoint guided: `WebNuevaAppIntakeGuidedRequestV0` acepta
  `wizard_answers []WizardAnswerV0` y la respuesta devuelve `wizard` con
  decisiones/contrastes del turno aplicado.
- i18n: catalogos es/en y `nueva_app_i18n_keys_v0.go` incluyen todas las claves
  nuevas generadas por agenda, pagos, mapas, storage, mobile, deploy y usuarios
  compartidos. Se anadio test exacto por locale para evitar fallback silencioso.
- Riesgo corregido: `wizard-r5-integracion-gobierno` queda como `question_ref`
  estable; el indice de integracion permanece en `field`, evitando claves
  dinamicas `...-0`, `...-1`, etc.
- `docs/diseno_wizard_programacion_2026-07-04.md` actualizado: ya no declara
  "NO implementado"; queda como implementacion parcial verificada.

Verificado hasta ahora:

- `go test -count=1 ./modulos/orquesta-web -run 'TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'`
- `go test -count=1 ./modulos/orquesta-web`
- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-http-gateway`
- `git diff --check`
- `go test -count=1 ./...`

Pendiente real para Claude si retoma:

- Tool MCP equivalente `orquesta.nueva_app.wizard.v0`.
- Render web usable del turno: radios/opciones, recomendacion visible,
  contraste y panel de defaults.
- Commit y push de este corte si la sesion se cierra desde Codex.

## Actualizacion Codex 2026-07-04 tarde 5

Se cierra el pendiente inmediato de TAREA-7/7b G2 en el repo principal:

- Tool MCP `orquesta.nueva_app.wizard.v0` implementada y registrada en el
  transporte MCP. Acepta el mismo contrato operativo del endpoint guiado:
  `need`, `action_id`, `answer`, `wizard_answers`, `session`, `locale`,
  `nombre` e `idea`; devuelve `turn`, `session` y `wizard`.
- `orquesta-app-codex-stack` cablea el executor real del wizard MCP contra el
  handler guiado existente.
- `/nueva-app` renderiza turno rico: preguntas, opciones, recomendacion,
  racionales y defaults. Los clicks envian `wizard_answers` y el cliente
  repinta preguntas/contrastes/defaults con la respuesta.
- `docs/diseno_wizard_programacion_2026-07-04.md` queda actualizado: MCP y
  render web pasan a cerrados; la nueva taxonomia U1-U12 y packs de dominio se
  conserva como pendiente vinculante, no implementada en este corte.

Archivos principales usados/modificados:

- `modulos/orquesta-mcp/nueva_app_wizard_tool_v0.go`
- `modulos/orquesta-mcp/nueva_app_wizard_transport_v0.go`
- `modulos/orquesta-app-codex-stack/nueva_app_wizard_mcp_executor_v0.go`
- `modulos/orquesta-web/nueva_app_html_render_v0.go`
- `modulos/orquesta-web/nueva_app_html_handler_v0_test.go`
- `docs/diseno_wizard_programacion_2026-07-04.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Verificado:

- `go test -count=1 ./modulos/orquesta-web -run 'TestNuevaAppHTMLHandlerV0GETMuestraFormularioUsableSinDelegar|TestWizard|TestNuevaAppIntakeGuidedResponseV0IncluyeWizardRicoV0|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0|TestNuevaAppI18nCatalogV0CatalogosCubrenClavesRequeridas'`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-web`

Pendiente real para Claude:

- Implementar la taxonomia U1-U12 y packs de dominio de
  `docs/diseno_wizard_programacion_2026-07-04.md`.
- Anadir los tests de taxonomia: dimensiones universales, packs por keyword,
  packs combinados sin duplicados y pregunta abierta si no hay dominio.
- Mantener abiertos `BUG-079`, `BUG-165/065`, `BUG-058/066`, `BUG-075` y
  `MEJ-106` hasta sus smokes/reducciones especificas.

## Aviso a Codex: diseño del wizard ampliado (Claude, 2026-07-04)

El operador amplió el alcance del wizard DESPUÉS del G1 ya implementado.
Leer y ejecutar las secciones NUEVAS de
`docs/diseno_wizard_programacion_2026-07-04.md`: 9 (taxonomía U1-U12 +
packs de dominio), 10 (capa técnica T1-T8 + motor de exclusión; ejemplo
canónico kernel-C-sin-web), 11 y 11.5 (ayuda en llano por opción, botón
explícamelo-todo, pregunta libre), 12 (bot guía con RAG del catálogo;
grounding estricto, funciona sin LLM). Grupos pendientes: G2 (MCP+render),
G3 (i18n completo), G4-G5 (bot). Tests de aceptación listados en cada
sección; ninguno es opcional.

## Actualizacion Codex 2026-07-04 tarde 9

TAREA-8, ola 1 de variables pisadas: nuevo corte cerrado por Codex con dos
subagentes.

Hecho:

- `ORQUESTA_SERVER_URL` canónica; `ORQUESTA_BASE_URL` alias legacy con
  `deprecated_env_used`, `deprecated_env_duplicate` o `env_alias_conflict`.
  `scripts/lib/smoke_common.sh`, `scripts/smoke_opes_plan_temario_operadores.sh`
  y `scripts/smoke_opes_derivatives_rest.sh` respetan la precedencia canónica.
- Timeouts de smoke normalizados a `_MS`: Codex usa solo
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`; Claude/Gemini directos usan
  (`SMOKE_CLAUDE_GOAL_PROCESS_TIMEOUT_MS`,
  `SMOKE_GEMINI_GOAL_PROCESS_TIMEOUT_MS`) y Claude server request timeout
  (`SMOKE_CLAUDE_GOAL_PROCESS_SERVER_REQUEST_TIMEOUT_MS`).
- `ORQUESTA_GUARDIAN_*` emitidas por `cmd/orquesta-server` quedan declaradas
  como `child_process` y cubiertas por tests de registry/no herencia.
- Prueba Orquesta temporal: `orquesta-server run` con state/runtime en `/tmp`
  validó diagnósticos efectivos para `ORQUESTA_BASE_URL`, `CODEX_HOME` y
  `OPES_BASE_URL` sin exponer valores crudos.

Verificado:

- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack`
- `bash -n` de scripts de smoke tocados
- `git diff --check`

Atencion:

- `scripts/orquesta_metricas_deuda.sh --json` queda vigente en
  `env_vars_orquesta=513`: la ventana de compatibilidad Codex `_SECONDS` esta
  retirada y las dos envs Telegram operativas explican el techo temporal.
  No reintroducir `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS`.
- Siguen pendientes TAREA-8.1 (`orquesta.config.*`) y TAREA-8.3 completa. El
  guard `config_projection_mismatch` queda cerrado en primer corte para
  `prepare-run` y `apps/director`.

## Actualizacion Codex 2026-07-04 tarde 10

Codex cerro TAREA-8.2 para las entradas que lanzan trabajo:

- `orquesta.autoprogramming.prepare_run.v0` y
  `orquesta.apps.arrancar_director.v0` aceptan `required_settings`.
- El servidor proyecta `effective_config.settings` hacia el stack como DTO MCP
  (`key`, `value`, `sensitive`).
- `orquesta-app-codex-stack` valida antes de lanzar goal, persistir run legacy
  o encolar. Si no coincide, responde `config_projection_mismatch` con HTTP
  400 y sin valores sensibles en el mensaje.
- No se ha tocado core puro ni `orquesta-goal`; la validacion vive en
  adaptador/composicion.

Archivos usados/modificados:

- `modulos/orquesta-mcp/config_projection_v0.go`
- `modulos/orquesta-mcp/autoprogramming_prepare_run_tool_v0.go`
- `modulos/orquesta-mcp/arrancar_director_app_tool_v0.go`
- `modulos/orquesta-app-codex-stack/config_projection_guard_v0.go`
- `modulos/orquesta-app-codex-stack/config_v0.go`
- `modulos/orquesta-app-codex-stack/stack_v0.go`
- `modulos/orquesta-app-codex-stack/autoprogramming_prepare_run_mcp_executor_v0.go`
- `modulos/orquesta-app-codex-stack/queued_arrancar_director_v0.go`
- `cmd/orquesta-server/stack.go`
- `cmd/orquesta-server/stack_wiring_test.go`
- `modulos/orquesta-i18n-docs/public_error_catalog_v0.go`

Verificado:

- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-i18n-docs`
- `go test -count=1 ./modulos/orquesta-app-codex-stack`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./...`
- `git diff --check`
- Smoke Orquesta temporal: `POST /api/v0/apps/director` y
  `POST /api/v0/autoprogramming/prepare-run` devuelven
  `400/config_projection_mismatch` con `required_settings` divergente.

Pendiente para Claude:

- TAREA-8.1: diseñar/implementar fichero canónico `orquesta.config.*`.
- TAREA-8.3: ratchet bidireccional de envs registradas/leídas.
- Reducir `env_vars_orquesta=512` retirando aliases legacy cuando cierre la
  ventana de compatibilidad.

## Actualizacion Codex 2026-07-04 tarde 11

Codex añadió el primer corte ejecutable de TAREA-8.1 y un diagnóstico inicial de
TAREA-8.3.

Cambios principales:

- `cmd/orquesta-server/config_file_v0.go`: carga `orquesta.config.json` desde
  el project workdir con `schema_version:"orquesta_config.v0"`.
- `modulos/orquesta-server/config_v0.go`: `ConfigV0` conserva
  `AutoprogrammingGoalProgressPolicy` neutral.
- `cmd/orquesta-server/config.go`, `effective_config_v0.go`, `stack.go`: la
  política de autoprogramación sale de `env explícita > config file > default`
  y se publica en `effective_config`.
- `modulos/orquesta-app-gateway/config_v0.go` y `handler_v0.go`: el status REST
  recibe la misma política que el transporte MCP.
- `cmd/orquesta-server/server_env_registry_ast_v0_test.go`: diagnóstico AST de
  lecturas `ORQUESTA_*`; hoy informa 226 lecturas sin registry/allowlist y no
  falla todavía.

Bug cerrado durante validación:

- El smoke temporal mostró que `/api/v0/autoprogramming/status` seguía
  publicando defaults `100000/900/600` aunque el fichero traía `450000/1500/900`.
  La causa era que `orquesta-app-gateway` no recibía
  `AutoprogrammingGoalProgressPolicy`. Arreglado y cubierto por test.

Verificación:

- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server -run 'AutoprogrammingStatusAPIRouteV0PublicaGoalProgressPolicy|FicheroCanonico|EnvExplicito|SchemaInvalido|BuildStackFromEnvV0'`
- `go test -count=1 ./modulos/orquesta-server`
- `go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-app-codex-stack ./modulos/orquesta-i18n-docs`
- `go test -count=1 ./cmd/orquesta-server`
- `git diff --check`
- `scripts/orquesta_metricas_deuda.sh --json` -> `env_vars_orquesta=512`
- Smoke Orquesta temporal con `orquesta.config.json`: status publicó
  `450000/1500/900`; `required_settings=450000` no produjo mismatch;
  `required_settings=999999` produjo `400/config_projection_mismatch`.

Pendiente para Claude:

- No dar TAREA-8.1 por completa: faltan familias `server`, `codex_runtime`,
  `goal_backend`, `daemon_logs`, OPES/bridge y posible `--config`/snapshot de
  daemon.
- TAREA-8.3 sigue no estricta: convertir las 226 lecturas AST pendientes en
  ratchet por fases.
- Mantener `env_vars_orquesta=512` como techo temporal y bajarlo al retirar
  aliases legacy.

## Actualizacion Codex 2026-07-04 tarde 12

Codex amplió el fichero canónico y dejó una baseline más estricta para el AST.

Archivos clave nuevos/modificados en este tramo:

- `cmd/orquesta-server/config_file_v0.go`
- `cmd/orquesta-server/config.go`
- `cmd/orquesta-server/effective_config_v0.go`
- `cmd/orquesta-server/server_env_registry_v0.go`
- `cmd/orquesta-server/daemon_log_env_v0.go`
- `cmd/orquesta-server/server_env_registry_ast_v0_test.go`
- `cmd/orquesta-server/config_test.go`

Estado vigente:

- `orquesta.config.json` soporta `autoprogramming`, `server`, `daemon_logs` y
  `codex_runtime`.
- Precedencia por campo: env explícita > fichero > default.
- `effective_config` marca `source=config_file`; `state_dir` y
  `runtime_work_dir` salen como refs sensibles, no como paths.
- Ratchet AST de envs: baseline vigente `220` lecturas pendientes. El test falla
  si sube por encima de 220.

Verificación:

- `go test -count=1 ./cmd/orquesta-server -run 'FicheroCanonico|EnvExplicitoGana|AuditFileInvalido|EnvRegistryAST|EnvVars|Ratchet'`
- `go test -count=1 ./cmd/orquesta-server`
- `go test -count=1 ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack ./modulos/orquesta-mcp ./modulos/orquesta-i18n-docs ./modulos/orquesta-server`
- Smoke temporal ampliado de Orquesta verde con `addr/state/runtime` desde
  `orquesta.config.json`.

Pendiente para Claude:

- No reabrir lo cerrado salvo regresión: TAREA-8.2 y el primer bloque de TAREA-8.1
  ya tienen tests y smoke.
- Completar familias restantes del fichero canónico: `goal_backend`, límites
  Codex/agentes, OPES/bridges, codebase broker, rails/seguridad y domain work.
- Decidir/implementar `--config` o snapshot daemon.
- Bajar la baseline AST de 220 por categorías, sin meter child-process/smoke en
  `effective_config` si no corresponde.

## Actualizacion Codex 2026-07-04 tarde 13

Nuevo corte de TAREA-8.1 integrado en el workspace:

- `orquesta.config.json` añade `server_http` y `server_lifecycle`.
- Campos activos:
  - `server_http.read_header_timeout_ms`
  - `server_http.read_timeout_ms`
  - `server_http.write_timeout_ms`
  - `server_http.idle_timeout_ms`
  - `server_http.max_header_bytes`
  - `server_http.control_body_max_bytes`
  - `server_lifecycle.shutdown_grace_ms`
- `cmd/orquesta-server/config.go` consume esos campos con precedencia
  `env explícita > fichero > default`.
- `cmd/orquesta-server/effective_config_v0.go` publica `source=config_file`.
- Tests nuevos/extendidos en `cmd/orquesta-server/config_test.go`.

Verificado:

- `go test -count=1 ./cmd/orquesta-server -run 'LeeHTTPYLifecycle|LeeServerDaemonRuntime|EnvExplicitoGanaCampo|ShutdownGrace|EnvRegistryAST|EnvVarsOrquestaRatchet'`
- Smoke Orquesta temporal aislado: `orquesta_config_http_smoke=passed`.

Aclaración para no reabrir:

- `ORQUESTA_CAPACITY_MODEL_REF` y `ORQUESTA_CAPACITY_QUOTA_REF` no faltan en
  `effective_config`; fueron retiradas intencionalmente por MEJ-106. Reponerlas
  sube `env_vars_orquesta` a 514 y rompe el ratchet.

Pendiente real:

- Completar familias restantes: `goal_backend`, límites Codex/agentes,
  OPES/bridges, codebase broker, rails/seguridad, domain work y usage
  accounting.
- Resolver `--config`/snapshot daemon.
- Reducir AST 220 por categorías.

## Actualizacion Codex 2026-07-04 tarde 14

Nuevo corte de config canónica:

- `orquesta.config.json` añade `worktree_snapshot.max_files`,
  `worktree_snapshot.max_file_bytes` y `worktree_snapshot.max_total_bytes`.
- `cmd/orquesta-server/worktree_snapshot_budget_env_v0.go` lee esos valores con
  precedencia `env explícita > fichero > default`.
- `cmd/orquesta-server/stack.go` usa el presupuesto efectivo para
  `SnapshotReadBudget`.
- `effective_config` publica `source=config_file` para las tres claves.
- `server_env_registry_ast_v0_test.go` baja la baseline de 220 a 217.

Verificado:

- `go test -count=1 ./cmd/orquesta-server -run 'WorktreeSnapshot|EnvRegistryAST|EnvVarsOrquestaRatchet|LeeHTTPYLifecycle' -v`
- Smoke Orquesta temporal: `orquesta_config_snapshot_smoke=passed`.
- Métrica deuda: `env_vars_orquesta=512`.

Pendiente real:

- `goal_backend`, límites Codex/agentes, OPES/bridges, codebase broker,
  rails/seguridad, domain work, usage accounting, `--config`/snapshot daemon.
- AST pendiente: 217.

## Actualizacion Codex 2026-07-04 tarde 15

Nuevo corte de config canónica:

- `orquesta.config.json` añade `codex_usage_accounting.mode` y
  `codex_usage_accounting.log_max_bytes`.
- El stack solo activa métricas si el modo es `redacted_report` o
  `runtime_usage_report`.
- `effective_config` publica `ORQUESTA_CODEX_USAGE_ACCOUNTING` y
  `ORQUESTA_CODEX_USAGE_LOG_MAX_BYTES` con `source=config_file`.
- `server_env_registry_ast_v0_test.go` baja la baseline de 217 a 215.
- `BUG-ORQ-20260704-176` queda cerrado: no añadir nuevas familias directamente
  a `server_env_registry_v0.go`; usar registries por familia para no romper T90.

Verificado:

- `go test -count=1 ./cmd/orquesta-server -run 'CodexUsage|EnvRegistryAST|EnvVarsOrquestaRatchet' -v`
- `go test -count=1 ./cmd/orquesta-server -run 'ResidualGoFileBudget|CodexUsage|WorktreeSnapshot|EnvRegistryAST|EnvVarsOrquestaRatchet' -v`
- Smoke Orquesta temporal: `orquesta_config_usage_smoke=passed`.
- Métrica deuda: `env_vars_orquesta=512`.

Pendiente real:

- `goal_backend`, límites Codex/agentes, OPES/bridges, codebase broker,
  rails/seguridad, domain work y `--config`/snapshot daemon.
- AST pendiente: 215.

## Actualizacion Codex 2026-07-04 tarde 16

Nuevo corte de config canónica:

- `orquesta.config.json` añade la sección `codebase_broker`.
- Campos activos:
  - `provider_kind`
  - `external_indexer_enabled`
  - `max_concurrent`
  - `timeout_ms`
  - `state_dir`
  - `watchdog_enabled`
  - `watchdog_stop_orphans`
  - `watchdog_orphan_min_age_seconds`
  - `command`
  - `project_name`
- El broker central, el watchdog de code context y `effective_config` consumen
  esos campos con precedencia `env explícita > fichero > default`.
- `state_dir` y `command` se tratan como sensibles; en estado publico aparecen
  redactados. `provider_kind` tambien se redacted por politica publica general
  de claves con `PROVIDER`, pero conserva `source=config_file`.
- Cerrados:
  - `BUG-ORQ-20260704-177`: `/api/v0/codebase/query` estaba anunciado pero no
    montado en el handler directo. Arreglado en `cmd/orquesta-server/stack.go`.
  - `BUG-ORQ-20260704-178`: `scope:["."]` no buscaba en la raiz del proyecto.
    Arreglado en `serverRGCodeContextScopesV0`.

Verificado:

- Focales `CodeContextBroker`, `CodebaseQueryPublico`, `ScopePunto`,
  `EnvRegistryAST`, `EnvVarsOrquestaRatchet` y `ResidualGoFileBudget`.
- Smoke Orquesta temporal:
  `effective_config_codebase_broker=passed`, `codebase_status=passed`,
  `codebase_query=passed`.
- `go test -count=1 ./cmd/orquesta-server` estaba verde antes de los fixes de
  ruta/scope; repetirlo antes de cerrar commit final.

Pendiente real:

- Siguiente bloque seguro recomendado por subagente: `domain_work`.
- Resto TAREA-8: `goal_backend`, límites Codex/agentes, OPES/bridges,
  rails/seguridad y `--config`/snapshot daemon.
- AST pendiente: 215.

## Actualizacion Codex 2026-07-04 tarde 17

Nuevo corte de config canónica:

- `orquesta.config.json` añade la sección `domain_work`.
- Campos activos:
  - `http_base_url`
  - `http_domain_ref`
  - `file_dir`
  - `file_enabled`
  - `http_create_path`
  - `http_submit_path`
  - `http_timeout_seconds`
  - `http_egress_mode`
  - `http_allowed_hosts`
  - `delivery_ledger_path`
- El executor domain_work, el backend file, el adaptador HTTP neutral, el flag
  de delivery y el ledger consumen config file con precedencia
  `env explícita > fichero > default`.
- `effective_config` publica la familia con `source=config_file`; URL y rutas
  locales quedan sensibles/redactadas.
- `server_env_registry_ast_v0_test.go` baja baseline de 215 a 202.

Verificado:

- Focal `DomainWork` + ratchets `EnvRegistryAST`,
  `EnvVarsOrquestaRatchet`, `ResidualGoFileBudget`.
- Smoke Orquesta temporal:
  `effective_config_domain_work=passed`, `domain_work_create=passed`,
  `domain_work_snapshot=passed`.

Pendiente real:

- Repetir `go test -count=1 ./cmd/orquesta-server` tras documentacion.
- Resto TAREA-8: `goal_backend`, límites Codex/agentes, OPES/bridges,
  rails/seguridad y `--config`/snapshot daemon.
- AST pendiente: 202.

## Actualizacion Codex 2026-07-04 tarde 18

Nuevo corte de config canónica:

- `orquesta.config.json` añade limites de supervisor residente:
  - `server_supervisor.max_runs_per_tick`
  - `server_supervisor.max_executions_per_tick`
  - `server_supervisor.queue_limit`
  - `server_supervisor.drain_max_dispatches`
  - `server_supervisor.drain_max_commands`
  - `server_supervisor.drain_max_outbox`
  - `server_supervisor.drain_max_external_waits`
- Añade limites residentes auxiliares:
  - `server_idle_self_improvement.max_requests`
  - `server_resident_director.max_actions`
- `codex_runtime` añade `execution_mode`, `reasoning_effort`,
  `max_expected_seconds`, `max_batch_ready` y `max_concurrency`.
- `codex_director` añade `wave_agents`, `max_subagents_per_agent` y
  `recursive_agent_budget`.
- La precedencia sigue siendo `env explícita > fichero > default`; el modo
  `serial` conserva el clamp a `1`.
- `effective_config` publica los limites con `source=config_file`.
- `server_env_registry_ast_v0_test.go` baja baseline de 202 a 201.
- Cerrado `BUG-ORQ-20260704-179`: la redacción pública trataba
  `ORQUESTA_SERVER_DRAIN_MAX_COMMANDS` como sensible por el fragmento
  `COMMAND`; ahora `*_MAX_COMMANDS` no se redacta si no fue marcado sensible.

Verificado:

- Focal de limites/config + ratchets.
- Focal de status publico en `modulos/orquesta-server`.
- Smoke Orquesta temporal:
  `orquesta_config_limits_smoke=passed`.

Pendiente real:

- Repetir `go test -count=1 ./cmd/orquesta-server` y set transversal.
- Resto TAREA-8: `goal_backend`, OPES/bridges, rails/seguridad y
  `--config`/snapshot daemon.
- AST pendiente: 201.

## Actualizacion Codex 2026-07-04 tarde 19

Nuevo corte de config canonica:

- `orquesta-server run/start/status/stop --config <path>` cargan el fichero
  canonico explicito.
- `--config` usa el directorio del fichero como fallback de proyecto si no hay
  `ORQUESTA_CODEX_PROJECT_WORKDIR`.
- `start --config` escribe snapshot validado en
  `state/config-snapshots/orquesta.config.json` y lanza el daemon como
  `run --config <snapshot>`, para que las mutaciones posteriores del fichero
  original no cambien el daemon vivo.
- `effective_config` conserva `source=config_file` desde el snapshot mediante
  `ProjectConfigFilePath`.
- Cerrado `BUG-ORQ-20260704-180`: `start` devolvia `readiness_timeout` si
  readiness estricta estaba degradada por conectores externos aunque el daemon
  ya estuviera `startup_ready`.

Uso de Orquesta:

- `prepare-run` real contra servidor temporal con backend
  `claude_file_control`; accepted con
  `goal_ref=goal-ref-task-autoprogramming-00f701576d89-g01`.

Verificado:

- Focal `ReadinessOK|ConfigPath|DaemonRunArgs|DaemonStart|Status|CommandPublic|EnvRegistryAST|EnvVarsOrquestaRatchet`.
- Smoke real temporal:
  `orquesta_start_config_snapshot_smoke=passed`.

Pendiente real:

- Repetir `go test -count=1 ./cmd/orquesta-server`, set transversal y full
  `go test -count=1 ./...`.
- Resto TAREA-8: `goal_backend`, OPES/bridges y rails/seguridad.

## Actualizacion Codex 2026-07-04 tarde 20

Nuevo corte de config canonica:

- `orquesta.config.json` añade `goal_backend.kind`,
  `goal_backend.timeout_ms`, `goal_backend.preflight_timeout_ms` y
  `goal_backend.allow_app_server_proxy_diagnostic`.
- Cubre `ORQUESTA_CODEX_GOAL_BACKEND`,
  `ORQUESTA_CODEX_GOAL_TIMEOUT_MS`,
  `ORQUESTA_CODEX_GOAL_PREFLIGHT_TIMEOUT_MS` y
  `ORQUESTA_ALLOW_APP_SERVER_PROXY_DIAGNOSTIC`.
- Precedencia: `env explicita > fichero > default`.
- El backend efectivo alimenta launch/observe de Codex app-server,
  Claude/Gemini file-control/process, derivacion idle goal-first, diagnosticos
  de external_work, cleanup tmux y self-programming-only.
- `effective_config` marca esos valores con `source=config_file`.

Uso de Orquesta:

- Smoke real `start --config` con `goal_backend.kind=claude_file_control`.
- `prepare-run` aceptado y observado por file-control:
  `goal-ref-task-autoprogramming-88e68c22e8c4-g01`.
- Nota: el primer assert del arnes falló por esperar `.status=="accepted"`; el
  contrato real devuelve `accepted:true`. No se clasifica como bug de Orquesta.

Verificado:

- Focal `GoalBackend|GoalFirst|ExternalWorkLegacy|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget`.
- Smoke real temporal:
  `orquesta_goal_backend_config_smoke=passed`.

Pendiente real:

- Repetir paquete completo y full suite tras documentacion.
- Resto TAREA-8: OPES/bridges, OPES registry y rails/egress.

## Actualizacion Codex 2026-07-04 tarde 21

Nuevo corte de config canonica:

- `orquesta.config.json` añade `rails_security.security_mode`,
  `rails_security.rails_mode`, `rails_security.detail_prohibited_rails` y
  `rails_security.detail_prohibited_rails_scope`.
- Cubre `ORQUESTA_SECURITY_MODE`, `ORQUESTA_RAILS_MODE`,
  `ORQUESTA_DETAIL_PROHIBITED_RAILS` y
  `ORQUESTA_DETAIL_PROHIBITED_RAILS_SCOPE`.
- Los valores se normalizan a rails blandos: `rails_mode=enforced` en fichero
  sigue publicando/ejecutando `offline`, y `detail_prohibited_rails=on` sigue
  publicando/ejecutando `off`.
- `orquesta.config.json` añade `egress_sanitizer.enabled`,
  `egress_sanitizer.sanitizer_ref`, `egress_sanitizer.local_model.*` y
  `egress_sanitizer.sidecar.*`.
- Cubre la familia `ORQUESTA_EGRESS_SANITIZER_*`.
- `effective_config` marca la familia como `source=config_file` y redacta
  rutas, comandos, endpoints y refs sensibles en salida publica.
- `buildStackFromEnvV0` consume el sanitizer desde fichero/snapshot.
- Cerrado `BUG-ORQ-20260704-181`: en `start --config`, la proyeccion de rails
  al entorno del daemon hacia que `/status` mintiera `source=explicit`. Ahora
  solo el snapshot daemon reclasifica esa proyeccion como `config_file`; un
  `run --config` manual con env explicita conserva `explicit`.

Uso de Orquesta:

- Smoke real `start --config` con rails/egress canonicos.
- `/status` verificado con `source=config_file` para rails/egress y sin fuga de
  comando, ruta de modelo ni endpoint local.

Verificado:

- Focal `Rails|EgressSanitizer|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget|DaemonStartEnvironment`.
- Smoke real temporal:
  `orquesta_rails_egress_config_smoke=passed`.

Pendiente real:

- Repetir paquete completo y full suite tras cerrar OPES.
- Resto TAREA-8: OPES bridge/loop, OPES registry/finalpkg/topic y, si hay
  tiempo, ejemplos/scripts.

## Actualizacion Codex 2026-07-04 tarde 22

Nuevo corte de config canonica OPES:

- Nuevo `cmd/orquesta-server/opes_config_file_v0.go` para separar structs OPES
  del parser central. `cmd/orquesta-server/config_file_v0.go` queda en 846
  lineas y el ratchet T90 no rompe.
- `orquesta.config.json` soporta `opes.base_url`.
- `orquesta.config.json` soporta el subconjunto seguro de `opes_bridge`:
  `dry_run`, `job_type`, `job_type_sequence`, `job_ref`, `program_id`,
  `topic_id`, `correlation_id`, `limit`, `timeout_seconds`, `priority`,
  `interval_seconds`, `initial_delay_seconds`, `max_ticks`,
  `supervise_submitted`, `wait_resident_seconds`,
  `wait_resident_interval_ms`, `require_runtime_compatibility` y refs runtime.
- `orquesta.config.json` soporta `opes_registry_finalpkg.*` y
  `opes_topic_registry.*`.
- `effective_config` publica `source=config_file` para esas familias y redacta
  base URL/rutas/tool path.
- Se conserva el parser historico de `job_type_sequence` con comas,
  punto y coma, espacios y saltos de linea.
- Ratchet AST baja de 201 a 191.

Frontera importante:

- No se migraron a fichero: `ORQUESTA_OPES_BRIDGE_ENABLED`,
  `ORQUESTA_OPES_BRIDGE_CONFIRM`, `ORQUESTA_OPES_TEMPORAL_CONFIRM`,
  `ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM`,
  `ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED`,
  `ORQUESTA_OPES_BRIDGE_DESTINATION_EVIDENCE_REF`,
  `ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_DISABLED`,
  `ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH` ni comandos/preflight/readiness live
  de speech/remote QA.
- Motivo: no son simples settings; controlan efectos externos, bypass de
  filtros, idempotencia del ledger o ejecucion shell local. Migrarlos requiere
  una tarea gobernada con nuevas pruebas de seguridad/idempotencia.

Uso de Orquesta:

- Smoke real `start --config` con OPES bridge seguro + finalpkg + topic
  registry. Solo se inspecciono `/status`; no se ejecuto drain ni se tocaron
  jobs OPES.

Verificado:

- Focal `OPESBridgeConfigLeeFicheroCanonico|OPESDrainConfig|OPESBridgeLoopConfig|OPESSpeechSynthesis|ServerConfigFromEnvV0PublicaConfiguracionEfectivaCanonica|ServerConfigFromEnvV0PublicaOPESSpeechSynthesisPreflightRedactado|EnvRegistryAST|EnvVarsOrquestaRatchet|ResidualGoFileBudget`.
- Smoke real temporal:
  `orquesta_opes_config_smoke=passed`.
- `go test -count=1 ./cmd/orquesta-server` -> verde.
- `git diff --check` -> verde.
- `bash scripts/orquesta_metricas_deuda.sh --json` ->
  `{"env_vars_orquesta":512,"endpoints_status":16,"interfaces_estado":65,"modulos_director":17}`.
- `go test -count=1 ./...` -> verde.
- Sin daemons temporales de smoke ni `codebase-memory-mcp` vivos tras cierre.

Pendiente real:

- Si TAREA-8 quiere eliminar tambien confirmaciones/ledger/comandos de env,
  abrir sub-tarea gobernada; no trasladarlo a pelo.

## Actualizacion Codex 2026-07-04 tarde 23

TAREA-9 avanza con herramienta y primera limpieza segura.

Hecho:

- `scripts/orquesta_auditoria_codigo.sh` reproduce la auditoría de código:
  deadcode, módulos huérfanos, helpers copiados y ficheros grandes.
- El script genera JSON y SQLite local como caché/reporte derivado. No es motor
  de verdad operativa ni cambia la persistencia de Orquesta.
- `scripts/orquesta_smoke_nightly.sh` ejecuta la auditoría por defecto y aplica
  ratchet contra el último nightly verde con métricas: falla si suben
  `deadcode_candidates` o `helper_duplicate_definitions`.
- Tests nuevos/actualizados: `scripts/test_orquesta_auditoria_codigo.sh` y
  `scripts/test_orquesta_smoke_nightly.sh`.
- `BUG-ORQ-20260704-182` cerrado: el primer script caminaba `**/*.go` desde la
  raíz y podía tardar demasiado; ahora solo recorre `cmd/` y `modulos/`.
- Primera ola `orquesta-deploy`: borrados siete helpers exportados muertos
  `Has*IssueV0` sin consumidores internos. No se borra el módulo.

Números reproducibles actuales:

- Con `deadcode` real instalado en GOPATH:
  `deadcode_candidates=1230`, `helper_duplicate_definitions=288`,
  `orphan_modules=1`, `large_files_over_800=17`.
- `modulos/orquesta-deploy=87`; el snapshot histórico de Claude marcaba 94 para
  ese módulo.
- El global histórico `1188` ya no debe usarse como línea base vigente: era un
  snapshot antes de cambios posteriores. Usar el script como fuente de verdad
  de auditoría.

Verificado:

- `scripts/test_orquesta_auditoria_codigo.sh`.
- `scripts/test_orquesta_smoke_nightly.sh`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-deploy ./modulos/orquesta-app-planner`.

Siguiente recomendado:

- TAREA-2 primer parche: añadir al analizador operaciones fallback
  `callers`, `imports`, `module_exports`, `relevant_snippets` y actualizar el
  prompt Goal para exigir `orquesta.codebase.query.v0` antes de lecturas
  completas cuando el write-set sea código.

## Actualizacion Codex 2026-07-04 tarde 24

TAREA-2 primer parche queda implementado.

Hecho:

- `CodeContextQueryPortV0` acepta `callers`, `imports`, `module_exports` y
  `relevant_snippets`.
- MCP `orquesta.codebase.query.v0` publica los nuevos `query_kind`.
- `cmd/orquesta-server/code_context_structured_fallback_v0.go` implementa
  fallback determinista con `go/parser`; no arranca `codebase-memory-mcp`.
- El proveedor `codebase-memory-mcp` opt-in degrada esos modos a búsqueda
  central en vez de devolver `query_kind_unsupported`.
- El prompt Goal exige consultar `orquesta.codebase.query.v0` antes de leer
  ficheros completos cuando el write-set es de codigo; no lo inyecta en
  write-sets documentales.
- Docs locales `modulos/orquesta-context/docs/{contratos,pruebas,tareas}.md`
  actualizados.

Verificado:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-context ./modulos/orquesta-mcp ./modulos/orquesta-runtime-codex-goal`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'Test(ServerRGCodeContextProviderV0|MCPCodebaseQuery|BuildServerAppHandlerV0CodebaseQuery)'`.
- Smoke REST real con servidor temporal:
  `POST /api/v0/codebase/query` para `callers`, `imports`, `module_exports`
  y `relevant_snippets` -> `codebase_query_structured_smoke=passed`.
- `GOFLAGS=-buildvcs=false go test -count=1 ./...` -> verde.
- `scripts/test_orquesta_auditoria_codigo.sh`,
  `scripts/test_orquesta_smoke_nightly.sh`,
  `scripts/orquesta_metricas_deuda.sh --json` -> verde.
- `git diff --check` -> verde.
- Sin `orquesta-server run` ni `codebase-memory-mcp` vivos tras el smoke.

Pendiente de TAREA-2:

- Smoke sandbox real con un goal que use el analizador.
- Medición de tokens antes/después frente a baseline 165k.
- Conectar la caché derivada al motor único del despliegue cuando exista
  Postgres; no introducir un segundo motor operativo.

## Actualizacion Codex 2026-07-04 tarde 25

TAREA-2: preparación automática de analizador para goals de código.

Hecho:

- `orquesta-app-codex-stack` envuelve `GoalLauncher` y `GoalReworkLauncher` si
  existe `CodeContext`.
- Para write-sets de código, antes de lanzar el goal ejecuta `repo_map` por
  `CodeContextQueryPortV0` con scope del write-set.
- Añade `ContextRef` `code_context_prepared:repo_map:<query_hash>` y evidencias
  del broker al spec lanzado.
- Si el broker falla, añade `code_context_prepare_failed` pero no bloquea el
  goal.
- El receipt propaga `ContextBudget.CodeContextCacheStatus`.
- Goals documentales no disparan la precarga.
- `BUG-ORQ-20260704-183` cerrado: un helper `compact*` nuevo subia
  `helper_duplicate_definitions` de 288 a 289; se renombro a
  `uniqueCodeContextGoalStringsV0` y la auditoria volvio a 288.

Verificado:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestBuildDirectorPortsV0.*CodeContext|TestBuildDirectorPortsV0CableaAppGoalLauncher'`.
- Auditoria fresca: `deadcode_candidates=1230`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`.

Pendiente de TAREA-2:

- Smoke sandbox real con un goal que use el analizador.
- Medición de tokens antes/después frente a baseline 165k.
- Conectar caché derivada al motor único del despliegue cuando exista Postgres.

## Actualizacion Codex 2026-07-04 tarde 26

TAREA-9 segunda ola segura:

- `modulos/orquesta-capacity/capacity_policy_v0.go`: inlinados helpers privados
  de un solo uso.
- `modulos/orquesta-capacity/model_escalation_policy_helpers_v0.go`: inlinados
  `joinModelEscalationPathV0` e `isForbiddenModelEscalationKeyV0`.
- No se tocaron APIs exportadas ni helpers con consumidores internos.

Verificado:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-capacity ./modulos/orquesta-app-codex-stack ./modulos/orquesta-director`.
- Auditoria fresca: `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`, `modulos/orquesta-capacity=89`.

## Actualizacion Codex 2026-07-04 tarde 27

TAREA-9 ratchet/auditoria y conector Gemini:

- Cerrado `BUG-ORQ-20260704-184`: el auditor de codigo busca `deadcode` en
  `PATH`, `GOBIN` y `GOPATH/bin`; ya no cae al snapshot historico solo porque
  `/home/alberto/go/bin` no este en `PATH`.
- Cerrado `BUG-ORQ-20260704-185`: el nightly valida schema, metricas
  obligatorias y fuente de auditoria; rechaza `snapshot_file`,
  `unavailable` y `deadcode_tool_failed`.
- Cerrado `BUG-ORQ-20260704-186`: Gemini runtime no-goal usa
  `OutputFormat=text` por defecto.
- Verificado con scripts de auditoria/nightly, focal Gemini/GoalBackend y
  auditoria viva `deadcode_tool`:
  `deadcode_candidates=1228`, `helper_duplicate_definitions=288`,
  `orphan_modules=1`, `large_files_over_800=17`.
- Suite completa `GOFLAGS=-buildvcs=false go test -count=1 ./...` verde.
- Smoke Orquesta real por API publica: servidor temporal con
  `ORQUESTA_SERVER_ADDR=127.0.0.1:0`, `POST /api/v0/codebase/query`
  (`schema_version=code_context_query.v0`, `query_kind=relevant_snippets`,
  `query=geminiRuntimeConfigV0`) -> `provider_kind=fallback_rg`, `results=1`;
  shutdown limpio. Primer intento manual con schema incorrecto
  `orquesta_code_context_query.v0` devolvio 400 esperado, no bug.

Pendiente recomendado para siguiente tramo:

- Ejecutar smoke real goal-first que demuestre que un goal de codigo usa
  `orquesta.codebase.query.v0`, y medir tokens frente al baseline 165k.
- Decidir `app_server_proxy`: implementarlo de verdad o retirarlo del set de
  backends aceptados, hoy solo queda como diagnostico historico no operativo.
- Mover el default local OPES `/home/alberto/Trabajo/OPES` a config canonica
  obligatoria por composicion/conector.
- Completar wizard universal de `docs/diseno_wizard_programacion_2026-07-04.md`
  con taxonomia U1-U12, capa tecnica T1-T8, motor de exclusion, ayudas i18n y
  bot RAG.

## Actualizacion Codex 2026-07-04 tarde 28

Conector OPES/project workdir:

- Cerrado `BUG-ORQ-20260704-187`: quitado el default local
  `/home/alberto/Trabajo/OPES` del guard `external_work` OPES y del
  descubrimiento del topic registry.
- Fuente vigente: `opes.project_workdir` en `orquesta.config.json` o
  `ORQUESTA_OPES_PROJECT_WORKDIR`.
- Sin OPES configurado, no se instala ruta local implicita; con OPES
  configurado, se conserva la guarda de `project_work_dir` y el descubrimiento
  de `registro_trabajo_temas.py`.
- Verificado con focales de `cmd/orquesta-server` y
  `modulos/orquesta-app-codex-stack` para guard/config OPES.

## Actualizacion Codex 2026-07-04 tarde 29

i18n goal-first Claude/Gemini:

- Cerrado parcial `BUG-ORQ-20260704-188`: los backends goal-first Claude y
  Gemini aceptan `PromptLocale`; defaults compatibles `es-ES`.
- Nuevos builders `BuildClaudeGoalPromptWithLocaleV0` y
  `BuildGeminiGoalPromptWithLocaleV0` con soporte `en-*` para cabeceras y
  protocolo durable del resultado.
- Configuracion canonica por fichero:
  `goal_backend.prompt_locale` en `orquesta.config.json`, cableada a
  file-control y process. No se anadieron nuevas variables `ORQUESTA_*`.
- No se toca `orquesta-goal`: el locale queda en adaptador/composicion.
- Residual: prompts legacy de agente `Build*AgentPrompt*` siguen en español si
  se exige i18n transversal fuera de goal-first.
- Verificado con focales de runtimes Claude/Gemini y wiring de servidor, mas
  ratchet de envs y auditoria de deuda sin aumento.

## Actualizacion Codex 2026-07-04 tarde 30

Smoke REST/readiness y cierre de verificacion:

- Cerrado `BUG-ORQ-20260704-189`: el smoke REST local ya no exige HTTP 200
  estricto si `/api/v0/server/readiness` devuelve 503 con
  `startup_ready=true`, `status=running` y `availability_status=running`.
- `scripts/lib/smoke_common.sh` centraliza `smoke_orquesta_readiness_ok` y
  `scripts/smoke_orquesta_server_rest_director.sh` consume ese helper.
- Evidencia: focal `TestSmokeCommon(Readiness|Shutdown)` verde y smoke REST
  verde con `POST /api/v0/apps/director -> HTTP 200`, `agents_started=1`,
  progress/usage/process refs verificados y shutdown limpio.
- Suite amplia posterior: `GOFLAGS=-buildvcs=false go test -count=1 ./...`
  verde.
- Auditoria viva final: `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`; `env_vars_orquesta=512`.

## Actualizacion Codex 2026-07-04 tarde 31

Wizard dominio y OPES finalpkg live-config:

- Avance parcial wizard: se ampliaron packs de dominio y preguntas R3
  multi-pack en `modulos/orquesta-web`, con i18n es/en y pruebas focales.
  No declarar completo: faltan U1-U12, T1-T8, exclusiones runtime,
  HelpKey/ExampleKey completos, glosario y bot RAG.
- Cerrado `BUG-ORQ-20260704-190`: `cmd/orquesta-server` ya no aplica
  defaults OPES de `course_id`, `template_run_ref` ni `template_topic_id` en
  `dry_run=false`; live exige config/env explicitos y bloquea efectos si
  faltan.
- Validado con Orquesta real temporal: API wizard `guided-turn` verde,
  autoprogramming status `ok`, readiness `running`, y el loop
  `opes-registry-finalpkg` bloquea la config live incompleta con las tres
  claves faltantes. No se toco OPES productivo.

Verificado:

- Focales `./modulos/orquesta-web` para wizard/i18n.
- Focales `./cmd/orquesta-server` para OPES finalpkg config/dry-run/submit.
- Smoke API publica de Orquesta con servidor temporal aislado.
- Suite completa `GOFLAGS=-buildvcs=false go test -count=1 ./...` verde.
- Auditoria viva: `deadcode_candidates=1228`,
  `helper_duplicate_definitions=288`, `orphan_modules=1`,
  `large_files_over_800=17`; `env_vars_orquesta=512`.

Mantener abiertos para siguiente agente:

- `BUG-165`: residual global stop/observe/shutdown goal-first.
- `BUG-058`, `BUG-066`, `BUG-075`: requieren smoke OPES temporal real
  end-to-end con external-work/observe, proveedor, derivados y paquete final.
- `BUG-065`, `BUG-079`: no cerrar sin evidencia real indicada en inventario.
- Wizard universal: completar secciones 9-12 del diseno antes de venderlo como
  generador universal.

## Corte Codex local para relevo remoto 2026-07-04 noche

Claude anadio `TAREA-10` en
`docs/instrucciones_director_codex_2026-07-04.md` y la nota de bitacora sobre
codigo construido-y-nunca-cableado. Codex local reviso esa entrada con
subagentes antes de apagar la sesion.

Hallazgo principal: `TAREA-10.3` esta parcialmente resuelta en el HEAD actual,
aunque el texto de Claude dice que "nunca se llama". En codigo:

- `cmd/orquesta-server/stack.go` llama a
  `codeContextBrokerWiringFromEnvV0`.
- `BuildStackV0` recibe `CodeContext` y `CodeContextToolLeases`.
- Se exponen `orquesta.codebase.query.v0` y `/api/v0/codebase/query`.
- El launcher goal-first se envuelve con
  `goalLauncherWithCodeContextPrepareFromConfigV0` y precarga `repo_map` en
  goals con write-set de codigo.

No reescribir el broker. Lo que falta para cerrar TAREA-10.3 es test focal
integrado:

- construir `buildStackFromEnvWithGoalBackendV0` con `ProjectWorkDir` temporal;
- consultar el binding MCP/HTTP real de `CodebaseQuery`;
- lanzar un goal fake de codigo y comprobar `code_context_prepared:*`;
- decidir despues si hace falta proyeccion MCP local en `CODEX_HOME` aislado
  del agente. Si se hace, debe seguir pasando por el broker central.

Artefactos de pilotajes goal-first:

- Los `checkpoint_started.txt` y
  `orquesta_goal_result_goal-ref-task-autoprogramming-*.json` que quedan bajo
  `cmd/orquesta-server/docs` y `modulos/*/docs` se commitean para que el
  servidor remoto vea la misma evidencia de arranque.
- Esos JSON son `status=invalid` o equivalentes provisionales: no son cierre
  de tarea, solo handoff durable de trabajo interrumpido.

Aviso operativo: `scripts/bootstrap_agent_tooling.sh --status` marco
`attention_required` por un `codebase-memory-mcp` vivo. No se paro desde Codex
local porque parecia pertenecer a una sesion Claude activa. En el servidor
remoto, revisar procesos vivos antes de lanzar smokes largos.

## Actualizacion Codex 2026-07-04 noche 32

TAREA-10.3 ya tiene evidencia integrada:

- Nuevo test:
  `TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0` en
  `cmd/orquesta-server/stack_wiring_test.go`.
- Cubre stack desde env, broker real `fallback_rg`, binding MCP
  `CodebaseQuery`, handler HTTP `/api/v0/codebase/query` y precarga goal-first
  `code_context_prepared:repo_map:*`.
- Smoke Orquesta real temporal: `POST /api/v0/codebase/query` contra servidor
  aislado -> HTTP 200, `estado=ok`, `provider_kind=fallback_rg`,
  `results=1`, `cache_status=stored`.

Comandos verdes:

- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0'`
- `GOFLAGS=-buildvcs=false go test -count=1 ./cmd/orquesta-server -run 'TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0|TestBuildServerAppHandlerV0CodebaseQueryPublicoUsaBindingDirecto|TestServerCodeContext|TestCodeContextBroker'`
- `git diff --check`
- `GOFLAGS=-buildvcs=false go test -count=1 ./...`

Para el siguiente agente: no reescribir `code_context_broker_v0.go`. TAREA-10.3
queda cerrada para cableado servidor/MCP/HTTP/goal. Solo queda como posible
frente separado la proyeccion MCP local dentro del `CODEX_HOME` aislado del
agente, si se decide que la herramienta debe ser invocable sin HTTP; esa pieza
debe seguir usando el broker central.

TAREA-10.1 tambien tiene nueva evidencia parcial:

- Nuevo test:
  `TestWizardNuevoListoCierraFactoryRealPorPuertoV0` en
  `modulos/orquesta-web/nueva_app_wizard_rest_flow_v0_test.go`.
- Cubre sesion wizard nueva, `LaunchReady`, preview factory valida, rutas
  punteadas de conector `calendar` preservadas y cierre real por puerto
  `RESTSolicitarNuevaAppClientV0` contra `orquesta-factory-http`.
- No se mete `SolicitarNuevaAppV0` directo en `orquesta-web`: se conserva la
  frontera hexagonal web/cliente/adaptador factory.

Comando verde:

- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web -run 'TestWizardNuevoListoCierraFactoryRealPorPuertoV0|TestWizardAgendaDesdeSoloObjetivoV0|TestNuevaAppRESTFlowV0ValidaWebRESTFactory|TestNuevaAppIntakeGuidedResponseV0AplicaWizardAnswersV0'`
- `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-app-director-intake ./modulos/orquesta-factory-http -run 'Test(AdvanceAppDirectorIntakeWizardV0|ApplyWebNuevaAppIntake|WebNuevaAppIntakeSessionV0|NuevaAppIntakeGuided|Wizard|NuevaAppRESTFlow|AppSpecHTTPV0PostValido)'`

No borrar todavia `modulos/orquesta-app-director-intake` ni sus `wizard_*.go`:
el modulo sigue importado por `orquesta-app-director-service`/`orquesta-mcp` y
la retirada requiere deprecacion compatible o refactor con tests propios.

## Actualizacion Codex 2026-07-04 noche 33

TAREA-10.2 integrada:

- `AutonomousDirectorPolicyPortV0` entra en `StartAppDirectorPortsV0`.
- `startAppDirectorGoalFirstV0` llama a la politica con
  `BuildDirectorRunStatsV0(prepared.Run)` antes de lanzar el goal.
- La decision ajusta `GoalWorkSpecV0.Budget.MaxSubgoals` y anade evidencia y
  contexto `autonomous_director_policy:v0`.
- `BuildStackV0` cablea `ConfigV0.AutonomousDirectorPolicy`.
- `cmd/orquesta-server` inyecta `HeuristicAutonomousDirectorPolicyV0` en el
  stack real.

Evidencia:

- `TestStartAppDirectorV0GoalFirstAplicaPoliticaAutonomaV0`.
- `TestStartAppDirectorGoalSpecWithAutonomousPolicyV0StatsDistintosDecisionDistinta`.
- `TestBuildDirectorPortsV0CableaPoliticaAutonomaV0`.
- `TestBuildStackFromEnvV0CableaBrokerContextoEnMCPHTTPYGoalV0` ampliado con
  assert del puerto.
- Suite afectada verde:
  `GOFLAGS=-buildvcs=false go test -count=1 ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`.
- Smoke Orquesta seguro: servidor temporal sin backend Goal, POST
  `/api/v0/apps/director` en `goal_first` -> HTTP 500
  `goal_backend_unavailable`, sin fallback legacy.

Pendiente separado para Claude/servidor remoto:

- TAREA-3 no queda cerrada por esto. `task_cost_class` ya existe en
  `orquesta-runtime-codex-goal` y se deriva del write-set, pero aun falta
  usarlo para seleccionar backend/effort barato en tareas documentales desde la
  composicion, sin meter proveedor/modelo en core ni goal neutral.

## Actualizacion Codex 2026-07-04 noche 34

TAREA-3 cerrada para Codex app-server:

- `task_cost_class` se deriva en `orquesta-runtime-codex-goal`; ficheros
  `.md/.markdown` y carpetas `docs/...` clasifican como `doc`.
- `serverCodexGoalCostRoutingStarterV0` ya envolvia el app-server; ahora queda
  probado por `TestServerCodexGoalCostRoutingStarterV0BajaSoloDocumentacionALowV0`.
- Regla verificada: write-set solo Markdown -> `reasoning_effort=low`; write-set
  de codigo o mixto -> conserva el esfuerzo configurado.
- `TestServerCodexGoalTaskCostClassForPacketV0IgnoraDeclaradoIncoherenteV0`
  fija que la composicion no se deja enganar por un `TaskCostClass` declarado
  que contradiga el write-set.
- No se anaden variables nuevas ni se toca core/goal neutral.

Residual opcional, no bloqueo: si producto quiere mover docs a Gemini/Claude en
vez de Codex low, hacerlo como politica de composicion configurable y con
evidencia propia.

## Codex remoto 2026-07-05: Telegram, supervisor y accepted invisible

Ver handoff detallado: `docs/runbooks/handoff_codex_orquesta_remoto_telegram_supervisor_2026-07-05.md`.

Resumen: Codex verifico que Orquesta genera codigo y pasa focales recientes, pero el servidor vivo no esta ejecutando el build nuevo y el supervisor sigue con `director_tick_input_build_invalido: field=scheduler_input.payload`. Se documento `BUG-ORQ-20260705-TELEGRAM-NOLLM-ACCEPTED-INVISIBLE`: `prepare-run` acepta `request-ref-remoto-telegram-nollm-runtime-20260705-001` pero no aparece en `autoprogramming/status`. Telegram salida funciona; control por Hermes LLM falla por 429; queda pendiente runtime Telegram no-LLM propio de Orquesta.

Actualizacion Codex 2026-07-06: el runtime Telegram no-LLM tiene avance local
en `cmd/orquesta-server`: endpoint `POST /api/v0/operator/telegram/update`,
validacion de chat autorizado, comandos contra canal operador-Director y sender
Bot API directo (`telegramBotAPISenderV0`) inyectado por config
`telegram_operator.token`. Falta desplegar ese commit en `srv1651826`, reiniciar
solo Orquesta y validar update real desde Telegram movil.

## Actualizacion Codex 2026-07-07: BUG-066 OPES settlement local reducido

Avance integrado para `BUG-ORQ-20260701-066`, sin tocar OPES productivo:

- `update_topic_registry` ya no queda congelado por una actualizacion antigua
  del mismo artefacto: la idempotencia causal incluye `receipt_ref` ademas de
  source job, artifact, followup y work kind.
- `settlement_status=settled_text|settled_final` y aliases terminales se
  consumen en `orquesta-opes-director` solo si no hay blockers calculados de
  QA, evidencia requerida o lifecycle goal-first.
- Cuando el terminal es valido, `pending_refs/rework_refs` stale dejan de abrir
  `assemble_topic`, `review_director_consolidation` o `finalize_temario_package`.
- `settled_text` queda como `texto_asentado_pendiente_derivados` +
  `operational_status=waiting`; `settled_final` exige `CompleteJob=true` y
  manifest de cierre completo antes de `release`.

Archivos tocados:

- `modulos/orquesta-opes-director/job_requests_v0.go`
- `modulos/orquesta-opes-director/topic_registry_settlement_v0.go`
- `modulos/orquesta-opes-director/topic_registry_v0.go`
- `modulos/orquesta-opes-director/producer_v0_test.go`
- `docs/inventario_bugs_orquesta_2026-06-30.md`
- `docs/bitacora_correccion_pericial_2026-07-03.md`
- `docs/runbooks/handoff_claude_mejora_continua_orquesta_2026-07-03.md`

Prueba ejecutada:

- `go test -count=1 ./modulos/orquesta-opes-director`

No sobrecerrar:

- `BUG-066` sigue abierto para el tramo real/residente: smoke temporal
  OPES/external-work, observacion por `/api/v0/external-work/observe`,
  reconciliacion automatica tras corte externo/manual y verificacion de que no
  queda backend goal-first activo/stale residual.

## Actualizacion Codex 2026-07-07: BUG-194 OPES required_settings y scope en goal

Se reduce `BUG-ORQ-20260705-194` en codigo local. No se toco OPES productivo.

Hecho:

- `cmd/orquesta-server/opes_bridge_config.go`: `effective_config` publica
  `ORQUESTA_OPES_TEMPORAL_CONFIRM`, `ORQUESTA_OPES_BRIDGE_ENABLED`,
  `ORQUESTA_OPES_BRIDGE_CONFIRM`, `ORQUESTA_OPES_BRIDGE_DRY_RUN`, ademas de
  URL, scope y limite ya existentes. La URL sigue redactada como
  `opes-base-url-configured`.
- `modulos/orquesta-external-work-run/goal_spec_v0.go`: prioridad maxima para
  `required_settings`, base URLs configuradas, confirmaciones OPES, `limit=1`,
  `job_ref`, `program_id`, `topic_id` y `correlation_id`.
- `modulos/orquesta-external-work-run/goal_spec_opes_v0.go`: helper OPES
  separado para no romper el ratchet de tamano del archivo principal.
- El criterio de aceptacion OPES real queda durable en el goal: si falta
  required setting o scope, cerrar/bloquear con
  `reason_code=missing_required_settings` y no tocar colas ni OPES productivo.

Pruebas:

- `go test -count=1 ./modulos/orquesta-external-work-run ./cmd/orquesta-server`

Revision remoto:

- Consulta SSH solo lectura a `srv1651826:/srv/orquesta-self/worktrees/pilot-remoto-1`:
  HEAD `0188739c`, 24 commits por detras de
  `origin/trabajo/plataforma-agentes`; `origin=/tmp/orquesta-self.bundle`.
  No hay version remota mas avanzada de este bug. Queda pendiente sincronizar o
  redesplegar Orquesta remoto antes de repetir el field test.

Residual:

- `BUG-194` queda cerrado en codigo local para transporte/contrato, pero el
  field test real sigue pendiente de instancia OPES temporal/preproduccion,
  `ORQUESTA_BASE_URL`, confirmacion de bridge, `limit=1` y scope duro aportados
  por operador.

## Actualizacion Codex 2026-07-07: auth Codex invalidada clasificada

Los subagentes de apoyo fallaron por proveedor con
`access token could not be refreshed`, equivalente operativo a
`BUG-ORQ-20260705-CODEX-HOME-TOKEN-INVALIDADO`. No se leyeron tokens ni
`auth.json`.

Cierre de codigo local:

- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_command_protocol_v0.go`
  clasifica `token_invalidated`, `refresh_token_invalidated`,
  `refresh_token_reused`, `token_expired` y `access token could not be refreshed`
  como `codex_app_server_provider_unauthorized`.
- `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`
  fija la regresion con los mensajes reales.

Prueba:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`

Residual:

- El bug sigue abierto operativo en remoto hasta reautenticar el `CODEX_HOME`
  aislado y verificar una tarea nueva con `agent_ack.json`.

## Actualizacion Codex 2026-07-08: golden evals con metricas A/B

Hecho:

- `scripts/orquesta_golden_evals.sh` ya agrega `tasks[].metrics` y
  `summary.metrics`.
- `scripts/orquesta_golden_metrics_launcher.sh` queda como wrapper opt-in para
  launchers reales: mide `elapsed_ms`, exit code, ficheros declarados y diff Git
  si se le pasa `ORQUESTA_GOLDEN_TASK_WORKTREE`.
- `scripts/test_orquesta_golden_metrics_launcher.sh` cubre caso correcto y
  fallo de launcher sin `result.json`.
- `docs/runbooks/orquesta_golden_evals_2026-07-04.md` documenta como usarlo
  para comparar baseline vs `orquesta-programacion-minima`.
- `docs/auditoria_programacion_minima_tokens_2026-07-08.md` queda actualizada:
  no meter reglas anti-overengineering globales sin A/B empirico.

Residual:

- Falta launcher real de proveedor/agente que escriba tokens reales por brazo
  A/B. El wrapper no inventa consumo: solo normaliza lo que publique el
  proveedor y completa metricas deterministas locales.

## Actualizacion Codex 2026-07-09: alias seconds Codex retirado

Hecho:

- `orquesta-runtime-codex-delivery` y `orquesta-app-codex-stack` dejan de leer
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_SECONDS`; solo aceptan
  `ORQUESTA_CODEX_SMOKE_TIMEOUT_MS`.
- Scripts y ejemplos de smokes Codex reales quedan convertidos a milisegundos.
- `scripts/orquesta_metricas_deuda.sh --json` baja de 514 a
  `env_vars_orquesta=513`; el inventario documenta
  `env_vars_orquesta_allow_increase_to=513`.

Pruebas:

- `bash -n scripts/smoke_codex_real_required_test_runner.sh scripts/smoke_codex_real_operational_wave.sh scripts/smoke_codex_real_recursive_tree.sh`
- `go test -count=1 ./modulos/orquesta-runtime-codex-delivery ./modulos/orquesta-app-codex-stack -run Smoke`
- `go test -count=1 ./cmd/orquesta-server -run 'EnvVarsOrquestaRatchet|ServerEnvRegistry'`
- `go test -count=1 ./...`
- `git diff --check`

Residual:

- Este residual queda cerrado por el corte siguiente de Telegram config
  canonica. Mantener pendiente solo el despliegue/prueba remota del bot.

## Actualizacion Codex 2026-07-09: Telegram operator sin envs duplicadas

Hecho:

- `cmd/orquesta-server` deja de leer `ORQUESTA_TELEGRAM_OPERATOR_ENABLED` y
  `ORQUESTA_TELEGRAM_OPERATOR_TOKEN`.
- `telegram_operator.enabled` y `telegram_operator.token` viven solo en
  `orquesta.config.json` y se publican como settings canonicos; el token queda
  redactado.
- `telegram_operator.notification_target_ref` derivado desde
  `authorized_chat_refs[0]` se publica con `source=config_file`, no
  `defaulted`.
- `scripts/orquesta_metricas_deuda.sh --json` vuelve a
  `env_vars_orquesta=511`.

Pruebas:

- `go test -count=1 ./cmd/orquesta-server -run 'TestTelegramOperator|TestTelegramBotAPI|TestServerEnvRegistry|TestEnvVarsOrquestaRatchetMEJ106V0'`
- `bash scripts/orquesta_metricas_deuda.sh --json`

Residual:

- Telegram real no queda probado por este corte: falta `orquesta.config.json`
  local en remoto con token, reinicio de Orquesta y validacion desde
  webhook/poller o Telegram movil.

## Actualizacion Codex 2026-07-09: forced-stop smoke cubre status

Hecho:

- `scripts/smoke_goal_first_app_server_real.sh` incorpora
  `post_autoprogramming_status_snapshot`.
- El modo `SMOKE_GOAL_FIRST_FORCED_STOP_MODE=1` consulta
  `/api/v0/autoprogramming/status` antes de `runs/control` y despues del
  `observe` terminal.
- El snapshot exige visibilidad de `run_ref`/`goal_ref`/`external_goal_ref`; el
  snapshot posterior al forced-stop falla si el goal sigue como `running`.

Pruebas:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_forced_stop_backend_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|ForcedStop)'`
- Smoke real:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_forced_stop_backend_real.sh`
  -> `smoke_goal_first_forced_stop_backend_real=ok`,
  `autoprogramming_status_before_forced_stop_visible=true`,
  `run_control_status=stopped`, `run_control_goal_status_after=blocked`,
  `observe_after_forced_stop_goal_status=blocked`,
  `autoprogramming_status_after_forced_stop_not_running=true`,
  `app_server_tmux_processes_alive=0`.

Residual:

- Esto cierra el subcaso forced-stop/status/observe/shutdown con proveedor real.
  No cierra `BUG-165/065/079` global: quedan observabilidad lenta/stale fuera
  de este smoke y despliegue remoto.
- Evidencia saneada:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.5Y0PiP` con
  `status_before_control_response.json`, `run_control_response.json`,
  `observe_after_forced_stop_response.json`, `status_after_control_response.json`
  y state final; se elimino `codex-home` y el binario temporal.

## Actualizacion Codex 2026-07-09: revalidacion OPES lifecycle y golden local

Hecho:

- Reejecutado `scripts/smoke_opes_lifecycle_real.sh` en el repo actual tras los
  ultimos commits: `status=passed`, 24/24 work kinds hasta
  `finalize_temario_package`, `finalpkg_dry_run=false`,
  `settlement_status=settled_final` y `no_residual_processes=true`.
- Evidencia retenida:
  `/tmp/orquesta-opes-lifecycle-real-20260709T111212Z/out/opes_lifecycle_result.json`.
- Reejecutados harness locales de evaluacion:
  `test_orquesta_golden_evals`, `test_orquesta_golden_ab_launcher`,
  `test_orquesta_golden_agent_launcher` y `test_orquesta_golden_metrics_launcher`,
  todos en verde.

Lectura para Claude:

- No reabrir `BUG-ORQ-20260709-196` ni `BUG-ORQ-20260709-197` salvo regresion:
  ambos estan verdes en local actual.
- Residual real: proveedor/cuota para A/B de `orquesta-programacion-minima`,
  despliegue remoto/Telegram, y smoke amplio de observabilidad lenta/stale para
  `BUG-165/065/079`.

## Actualizacion Codex 2026-07-09: control_plane en config canonica

Hecho:

- `orquesta.config.json` acepta `control_plane.remote_access_opt_in`,
  `control_plane.token`, `control_plane.principal`,
  `control_plane.permission_ref` y `control_plane.public_reason`.
- Las envs `ORQUESTA_SERVER_REMOTE_CONTROL_PLANE_CONFIRM` y
  `ORQUESTA_SERVER_CONTROL_*` siguen funcionando como override deprecated.
- `effective_config` publica `source=config_file` cuando procede y no filtra el
  token crudo: solo `present/absent`.
- La metrica de deuda sigue en `env_vars_orquesta=511`.

Pruebas:

- `go test -count=1 ./cmd/orquesta-server -run 'TestServerConfigFromEnvV0(.*Control|LeeControlPlane|ControlPlaneEnv)|TestServerEnvRegistry|TestEnvVarsOrquestaRatchetMEJ106V0|TestServerEnvRegistryASTV0'`
- `go test -count=1 ./cmd/orquesta-server ./modulos/orquesta-server`
- `go test -count=1 ./...`
- `bash scripts/orquesta_metricas_deuda.sh --json`
- `git diff --check`

Pendiente:

- No mover todavia `scripts/orquesta_server_ctl.sh` ni el perfil remoto en el
  mismo corte. Eso debe hacerse con prueba de servidor remoto y sin exponer el
  token real en logs/status.

## Actualizacion Codex 2026-07-09: shutdown coordination real app_server_tmux

Hecho:

- Nuevo wrapper `scripts/smoke_goal_first_shutdown_coordination_real.sh`.
- Extendido `scripts/smoke_goal_first_app_server_real.sh` con modo
  `SMOKE_GOAL_FIRST_SHUTDOWN_COORDINATION_MODE=1`: consulta
  `/api/v0/autoprogramming/status` antes del shutdown y luego coordina por
  `/api/v0/server/shutdown` con `cleanup_goal_backends=true`; el wrapper no usa
  `/api/v0/runs/control`.
- Fix de producto: `app_server_tmux` ya no publica active work residual si no
  observa owner, tmux session, socket, pane vivo ni proceso `codex app-server`.
- Fix del harness: `assert_app_server_tmux_shutdown_ready` ya no revienta con
  `session_name` sin inicializar si el owner desaparecio antes del diagnostico.

Pruebas:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestCodexAppServerTmuxBackendV0(EnsureShutdownCleanupMigrado|ReadActiveShutdownWorkIgnoraEstadoDegradadoSinResiduoVivo)'`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirstShutdownCoordinationReal'`
- Smoke real con proveedor:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=3 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_shutdown_coordination_real.sh`
  -> `smoke_goal_first_shutdown_coordination_real=ok`,
  `autoprogramming_status_before_shutdown_visible=true`, `status=ready`,
  `shutdown_ready=true`, `active_work_count=0`, `cleanup_completed`,
  `app_server_tmux_processes_alive=0`.
- `go test -count=1 ./...`
- `git diff --check`

Evidencia:

- Verde saneado:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.Ya4ayF`.
- Reproduccion fallida saneada antes del fix:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.3QX7lP`.

Pendiente:

- No cerrar `BUG-165/065` global todavia. Este smoke cierra el falso
  `backend_still_running` sin residuo vivo (`BUG-ORQ-20260709-198`) y valida
  cleanup real del backend, pero el shutdown final tiene `runs_requested=0`;
  queda reconciliar/probar runs goal-first fuera de cola durante shutdown amplio
  y los escenarios lentos/stale/remotos.

## Actualizacion Codex 2026-07-09b: goal-first fuera de cola coordinado offline

Hecho despues del corte anterior:

- Se corrigio el stack Codex, no el core puro. `serverShutdownExecutorV0` ya no
  pasa la cola directa al shutdown: usa un wrapper del adaptador que anade
  candidatos goal-first terminales cuando el run no aparece en la cola
  ejecutable pero `RunControl` sigue en `stop_requested` o `cancel_requested`.
- `stackShutdownActiveWorkReaderV0` publica esos controles pendientes como
  `goal_first` activo mientras no haya backend Goal vivo; asi no queda un
  `ready` silencioso si la cola no lo ve.
- `stackShutdownRunControlWriterV0` completa el control forzado a
  `stopped/canceled` si el `GoalWorkState` ya esta terminal.

Evidencia:

- Test nuevo:
  `TestStackShutdownV0ForzadoCoordinaGoalFirstTerminalFueraDeColaV0`.
- Cubre: cola vacia, `GoalWorkState=complete` con closure aceptada,
  `RunControl=stop_requested`, shutdown forzado, resultado
  `runs_requested=1`, `runs_stopped=1`, `shutdown_ready=true`,
  `RunControl=stopped` y evidencia
  `evidence-ref-server-shutdown-goal-terminal-run-control-reconciled`.
- Focales ejecutados:
  `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdown(V0ForzadoCoordinaGoalFirstTerminalFueraDeCola|ActiveWorkReaderV0|RunControlWriterV0)'`,
  `go test -count=1 ./modulos/orquesta-app-codex-stack` y
  `go test -count=1 ./modulos/orquesta-server-shutdown`.

Pendiente para Claude:

- No cierres `BUG-165/065` global aun. Falta smoke real amplio con proveedor
  lento/stale/remoto que demuestre el mismo contrato fuera del test offline.

## Actualizacion Codex 2026-07-09c: HTTP semi-real y guard del smoke real

Hecho:

- Se anadio `TestServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeColaV0`
  en `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go`.
- Cubre por HTTP real sin cuota:
  `/api/v0/apps/director` -> `/api/v0/apps/director/goal/observe` ->
  `/api/v0/runs/control forced=false` con `stop` y `cancel` ->
  `/api/v0/server/shutdown forced=true cleanup_goal_backends=true`.
- Resultado esperado verificado: `runs_requested=1`, `runs_stopped=1`,
  `shutdown_ready=true`, `active_work_count=0`, `control_status=stopped` para
  `stop_requested` y `control_status=canceled` para `cancel_requested`.
- `stackShutdownRunControlWriterV0` preserva `cancel_requested -> canceled` en
  shutdown amplio.
- `scripts/smoke_goal_first_app_server_real.sh` queda endurecido: en modo
  `shutdown_coordination` ya no acepta `ready` si `runs_requested<1`,
  `runs_stopped<runs_requested` o algun `runs[].control_status!=stopped`.

Verificado:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_shutdown_coordination_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestServerAppHTTPGoalFirstShutdownCoordinaControlPendienteFueraDeCola|TestSmokeGoalFirstShutdownCoordinationReal'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackShutdownV0ForzadoCoordinaGoalFirstTerminalFueraDeCola|TestStackShutdownRunControlWriterV0ForcedStopMarcaGoalTerminalReplanificable'`

Siguiente paso:

- Ejecutar el smoke real amplio. Ahora debe fallar si vuelve el falso verde
  `shutdown_ready=true` con `runs_requested=0`.

## Actualizacion Codex 2026-07-09d: smoke real amplio ejecutado

Hecho:

- Se ejecuto `scripts/smoke_goal_first_shutdown_coordination_real.sh` con
  backend real `app_server_tmux`, doble confirmacion y `KEEP_SMOKE_DIR=1`.
- Comando usado:
  `ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_REAL_CONFIRM=1 ORQUESTA_CODEX_GOAL_FIRST_APP_SERVER_CODEX_EXECUTION_CONFIRMED=1 ORQUESTA_GOAL_FIRST_SMOKE_POLLS=50 ORQUESTA_GOAL_FIRST_SMOKE_SLEEP_SECONDS=3 ORQUESTA_KEEP_SMOKE_DIR=1 ORQUESTA_SMOKE_PARENT=/tmp/orquesta-smokes-codex ./scripts/smoke_goal_first_shutdown_coordination_real.sh`.
- Resultado real:
  `smoke_goal_first_shutdown_coordination_real=ok`,
  `autoprogramming_status_before_shutdown_visible=true`,
  `shutdown_ready=true`, `runs_requested=1`, `runs_stopped=1`,
  `run_control_statuses=stopped`,
  `shutdown_coordination_all_runs_stopped=true`,
  `goal_actions[0].action_taken=cleanup_completed` y
  `app_server_tmux_processes_alive=0`.
- Refs:
  `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-bd43a0d5c0ed93df6c8f195055979a74`,
  `external_goal_ref=019f46cc-ae82-7352-bd4f-563f6ff34200`.
- Evidencia saneada:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.kALS5q`
  (~376 KiB, sin `codex-home`, `auth.json`, `config.toml` ni binario temporal).

Lectura para Claude:

- El falso verde/falso vacio `runs_requested=0` queda cerrado en local con
  proveedor real `app_server_tmux`.
- El script no llamo `/api/v0/runs/control` como camino principal: el estado
  previo de alto consumo genero `RunControl=stop_requested` y
  `/api/v0/server/shutdown` lo reconcilio a `stopped`.
- Mantener como residual externo solo la repeticion en servidor remoto/stale si
  se quiere evidencia fuera de la maquina local. No hay procesos locales vivos
  tras el smoke.
- Siguiente bug local recomendado por revision paralela si no se trabaja remoto:
  `BUG-079`, pero no cerrarlo sin prueba real de que `toolOutputPolicy` se
  aplica antes de herramientas. El primer corte util es hacer visible
  `tool-output-policy-sent/accepted/fallback`.

## Actualizacion Codex 2026-07-09e: BUG-079 observable

Hecho:

- `orquesta-runtime-codex-appserver` anade evidencias en el receipt de
  `turn/start`:
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-sent`,
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-accepted` y
  `evidence-ref-codex-app-server-turn-start-tool-output-policy-fallback`.
- Si el app-server acepta `toolOutputPolicy`, el receipt conserva `sent` +
  `accepted`.
- Si el app-server rechaza el campo por schema legacy y Orquesta reintenta sin
  JSON estructurado, el receipt conserva `sent` + `fallback`, pero no
  `accepted`.

Verificado:

- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `git diff --check`

Pendiente:

- No cerrar `BUG-079`: estas refs prueban transporte/aceptacion de la politica,
  no enforcement pre-tool. Falta smoke/proveedor real que demuestre que stdout
  gigante se corta antes de quemar contexto.

## Actualizacion Codex 2026-07-09f: BUG-066 cerrado local/fake-residente

Para Claude:

- No reabras `BUG-ORQ-20260701-066` como bug local de codigo si el alcance es
  local/fake-residente.
- Evidencia local vigente:
  `scripts/smoke_opes_lifecycle_real.sh` paso con 24/24 work kinds,
  `finalpkg_dry_run=false`, `settlement_status=settled_final` y
  `no_residual_processes=true`.
- Evidencia retenida:
  `/tmp/orquesta-opes-lifecycle-real-20260709T111212Z/out/opes_lifecycle_result.json`.
- El residual local de backend/shutdown queda cubierto por el smoke real
  goal-first `app_server_tmux`: `runs_requested=1`, `runs_stopped=1`,
  `run_control_statuses=stopped`, cleanup completo y sin procesos residuales.
- Evidencia retenida:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.kALS5q`.

Lo que queda fuera del cierre es evidencia externa/productiva: repetir en
servidor remoto/stale, OPES temporal/preproduccion o proveedor real autorizado.
Clasificalo como residual de despliegue/evidencia, no como codigo local abierto.

## Actualizacion Orquesta/Codex 2026-07-09g: BUG-079 por ola real y BUG-199

Se uso Orquesta para programar:

- Comando: `codex-launch-director-wave`.
- `wave_ref=codex-bug079-smoke-policy-real-20260709`.
- Agentes: 1.
- Resultado del agente: completado con `codex_last_message.txt`; tests verdes
  reportados por el agente.

Cambios integrados de la ola:

- `scripts/smoke_goal_first_app_server_real.sh` ahora exige evidencia de
  transporte `toolOutputPolicy` antes de declarar verde:
  `tool_output_policy_transport=accepted|fallback|sent_without_accept_or_fallback|missing`.
- El smoke acepta `accepted` o `fallback`, pero falla con
  `sent_without_accept_or_fallback` o `missing`.
- `cmd/orquesta-server/goal_first_app_http_flow_v0_test.go` fija que el fake
  HTTP exponga `sent` + `accepted`.
- `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go` guarda que el
  script conserve esa comprobacion.

Bug operativo descubierto por usar Orquesta:

- `BUG-ORQ-20260709-199`: la ola real termino, pero no escribio
  `codex_process_done_v0`; el marcador dependia de un goroutine `cmd.Wait()`
  del CLI lanzador, que puede morir al salir el CLI.
- Cierre: el wrapper `orquesta_codex_exec_v0.sh` escribe el marcador durable
  `codex_process_done_v0=completed|failed` antes de salir y en trap de senal.
  El nombre canonico vive ahora como `CodexProcessDoneFileNameV0`.
- Reproduccion post-fix:
  `wave=codex-process-done-repro-20260709T135205Z`,
  runtime `/home/alberto/Trabajo/runtime/codex-process-done-repro-20260709T135205Z`,
  `agent-01/codex_process_done_v0=completed`.

Verificado:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_forced_stop_backend_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst(AppServerReal|ForcedStop|ShutdownCoordinationReal)|TestServerAppHTTPGoalFirstLanzaObservaYCierraV0|TestCodexLaunchDirectorWaveCommandV0RecursiveFakeRuntimeEjecutableConLinaje'`
- `go test -count=1 ./modulos/orquesta-runtime-codex`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`

Pendiente:

- No cierres `BUG-079` completo: falta prueba real de enforcement pre-tool en
  proveedor/app-server. Este corte solo evita falso verde del smoke sin
  evidencia de transporte.

## Actualizacion Codex 2026-07-09h: BUG-079 app-server real acepta toolOutputPolicy

Para Claude:

- Ya no trates `BUG-079` como duda de transporte local de `toolOutputPolicy`.
  El app-server real actual acepta la politica estructurada.

Evidencia:

- Preflight app-server:
  `smoke_goal_first_app_server_preflight=ok`, `goal_backend=app_server_tmux`.
- Smoke normal corto:
  `run_ref=run-spec-smoke-goal-first-req-smoke-goal-first-4bde3b2d0c8fc9f42c555b714a3521f0`.
  No cerro accepted en 50 polls y no es verde, pero publico evidencias
  `tool-output-policy-sent` y `tool-output-policy-accepted`; sin procesos
  residuales. Evidencia:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.ojE5Us`.
- Smoke alto consumo/checkpoint:
  `run_ref=run-spec-smoke-goal-first-bug088-req-smoke-goal-first-bug088-a93fa78e27925c5262c0b25f2445dc0f`,
  `tool_output_policy_transport=accepted`,
  `smoke_goal_first_high_consumption_real=ok`,
  `bug088_path=second_artifact_or_partial_artifacts`,
  `tokens_used=8392`,
  `app_server_tmux_processes_alive=0`.
  Evidencia:
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.CZC8Ek`.

Pendiente real de `BUG-079`:

- Ejecutar o construir smoke adversarial/largo que pruebe enforcement pre-tool
  ante stdout gigante. Si esa prueba muestra consumo grande antes del corte,
  el bug pasa a frontera runtime/proveedor; si pasa, se puede cerrar `BUG-079`
  con evidencia fuerte.

## Actualizacion Orquesta/Codex 2026-07-09i: harness adversarial BUG-079 no concluyente

Hecho con Orquesta:

- Ola `codex-launch-director-wave`,
  `wave_ref=codex-bug079-adversarial-smoke-20260709`, 1 agente.
- El agente termino con `codex_process_done_v0=completed`.
- Nuevo wrapper:
  `scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`.
- Nuevo modo:
  `ORQUESTA_GOAL_FIRST_SMOKE_TOOL_OUTPUT_POLICY_ADVERSARIAL_MODE=1` en
  `scripts/smoke_goal_first_app_server_real.sh`.
- Test de guarda:
  `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go`.

Contrato del nuevo smoke:

- Exige `tool_output_policy_transport=accepted`.
- Falla si ve
  `evidence-ref-codex-app-server-thread-output-sanitized` o
  `codex_app_server_thread_read_response_too_large`.
- Pide al agente checkpoint temprano, crear `probe_stdout.py`, ejecutar una vez
  un stdout grande controlado y escribir `probe_result.txt` antes de
  `ORQUESTA_GOAL_RESULT_V0`.

Ejecucion real:

- Intento 1: 30 polls, evidencia
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.YAf0mK`.
- Intento 2: 60 polls, evidencia
  `/tmp/orquesta-smokes-codex/orquesta-goal-first-app-server.6wB72G`.
- Ambos quedaron `goal_status=running`, `recommended_action=observe_later`,
  solo checkpoint temprano, `toolOutputPolicy accepted`, sin salida saneada y
  sin procesos residuales tras cleanup.

Lectura para continuar:

- No cierres `BUG-079`.
- Abre/usa `BUG-ORQ-20260709-200`: el harness adversarial existe, pero no fuerza
  de forma fiable que el agente ejecute el probe stdout gigante.
- Siguiente accion tecnica recomendada: bajar el adversarial a nivel
  app-server/protocolo directo o hacer el probe determinista sin depender de
  una decision libre del agente.

## Actualizacion Orquesta/Codex 2026-07-09j: BUG-200 reducido con olas paralelas

Estado de sincronizacion al escribir esta nota:

- Trabajo hecho primero en local, rama `trabajo/plataforma-agentes`.
- El usuario ha pedido sincronizar tambien el servidor remoto de Orquesta; no
  asumir despliegue remoto hasta ver commit/push y verificacion de servidor.

Se uso Orquesta en paralelo:

- Ola `codex-core-bug200-harness-20260709T141956Z`.
  - Write-set: `scripts/smoke_goal_first_app_server_real.sh`,
    `scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`,
    `cmd/orquesta-server/smoke_goal_first_scripts_v0_test.go`.
  - Resultado: `agent-01/codex_process_done_v0=completed`.
  - Cambio integrado: el modo adversarial exige
    `generated-apps/bug079-tool-output-policy/probe_result.txt`; si solo hay
    checkpoint falla con `reason=bug200_probe_not_executed`, y si falta
    resultado valido falla con `reason=no_probe_result`. Ya no puede declarar OK
    solo por checkpoint.
  - El agente pudo ejecutar `bash -n`, pero su `go test` quedo bloqueado por el
    CODEX_HOME/Go cache aislado: intento descargar `golang.org/x/text v0.38.0`
    y la sandbox nego DNS/socket. No fue fallo del cambio.

- Ola `codex-core-bug079-protocol-20260709T141956Z`.
  - Write-set:
    `modulos/orquesta-runtime-codex-appserver/codex_goal_app_server_migrated_v0_test.go`.
  - Resultado: `agent-01/codex_process_done_v0=completed`.
  - Cambio integrado: nuevo test
    `TestServerCodexAppServerGoalBackendV0TurnStartToolOutputPolicyFallbackSoloSchemaV0`.
    Si `turn/start` falla con error no-schema aunque mencione
    `toolOutputPolicy`, Orquesta no relaja la policy ni hace fallback legacy.

Verificacion local posterior por Codex padre:

- `bash -n scripts/smoke_goal_first_app_server_real.sh scripts/smoke_goal_first_tool_output_policy_adversarial_real.sh`
- `go test -count=1 ./cmd/orquesta-server -run 'TestSmokeGoalFirst'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0TurnStart|TestServerCodexAppServerTurnStartParamsV0'`
- `git diff --check`

Lectura para continuar:

- `BUG-ORQ-20260709-200` queda reducido, no cerrado como prueba de proveedor:
  el arnes ya falla de forma determinista si el probe no se ejecuta.
- `BUG-ORQ-20260701-079` sigue abierto por frontera runtime/proveedor:
  falta evidencia fuerte de enforcement pre-tool real ante stdout gigante.
- Para sincronizar remoto: commit/push de esta rama; despues pull, build y
  restart gobernado solo de Orquesta en el servidor remoto. No asumir que el
  remoto tiene estos cambios hasta verificar hash y readiness.

## Actualizacion Codex 2026-07-09k: E3/E5 cerrados antes de push remoto

Para Claude:

- Codex padre cerro E3 local y un subagente cerro E5 local.
- Nuevos cierres documentados:
  - `BUG-ORQ-20260709-213`: contratos E3 multisuperficie.
  - `BUG-ORQ-20260709-214`: guards wizard/i18n.
  - `BUG-ORQ-20260709-215`: flaky del fake WebSocket app-server.
  - `BUG-ORQ-20260709-216`: deploy atomico reinicia servidor vivo.
  - `BUG-ORQ-20260709-217`: deploy lee `runtime_identity.binary_sha256`.

Cambios clave:

- Telegram operador se declara en el manifest HTTP como
  `operator_telegram.update.v0`; el servidor usa la constante canonica
  `RouteOperatorTelegramUpdateV0`.
- `/api/v0/operator/telegram/update` aparece en discovery como ruta conocida
  pero `Mounted=false`, porque sigue siendo opt-in por config.
- Nuevo test cruzado en `cmd/orquesta-server` comprueba que los contratos
  internos tienen correspondencia entre manifest HTTP, discovery y DTOs MCP.
- El inventario MCP de tools internas exige por prefijo nuevas tools de
  `nueva_app`, `director.human_work` y `operator.director`.
- Los guards i18n/wizard detectan nuevas claves fuera de inventario y duplicados
  raw de campos destino no declarados.
- Durante `go test ./...` aparecio un flaky previo del test WebSocket de
  `thread/read` gigante; se corrigio para no fallar por `broken pipe` benigno
  cuando el cliente cierra tras detectar frame demasiado grande.
- Antes de desplegar en remoto se corrigio `scripts/orquesta_server_deploy.sh`:
  ahora ejecuta `orquesta_server_ctl.sh stop` despues del build y antes del
  swap. Si falla la parada, aborta con `deploy_stop_failed` y no cambia el
  binario. Esto evita que un `ctl start` sobre servidor ya vivo sea no-op.
- La primera ejecucion real del deploy fallo solo por verificacion de identidad:
  `/api/status` exponia `runtime_identity.binary_sha256` anidado y el script
  buscaba solo campos top-level. Queda corregido y cubierto por test.
- La segunda ejecucion real del deploy detecto que el primer deploy habia
  dejado el worktree en `detached HEAD`: HEAD estaba en el commit nuevo, pero la
  rama local `trabajo/plataforma-agentes` seguia stale. Queda corregido como
  `BUG-ORQ-20260709-218`: cuando `ORQUESTA_DEPLOY_REF` es una rama, el deploy
  usa `checkout -B <ref> <sha>` y el test
  `test_success_preserves_branch_worktree` lo cubre.

Verificacion local ya pasada antes del commit:

- `go test -count=1 ./modulos/orquesta-http-gateway ./modulos/orquesta-mcp ./cmd/orquesta-server -run 'TestPublicRouteManifestV0DeclaraContratosE3InternosV0|TestServerResourcesRouteManifestIncluyeDiscoveryOPESV0|TestServerE3ContractSurfaceCatalogV0|TestMCPInternal'`
- `go test -count=1 ./modulos/orquesta-web -run 'TestNuevaAppI18nCatalogV0|TestWizard'`
- `go test -count=1 ./modulos/orquesta-web`
- `go test -count=10 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0ToolOutputPolicyYThreadReadGigantePorWebSocketDeterministaV0'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver`
- `bash -n scripts/orquesta_server_deploy.sh scripts/test_orquesta_server_deploy.sh`
- `bash scripts/test_orquesta_server_deploy.sh`
- `git diff --check`

Pendiente al retomar en remoto:

- Pull/fetch de la rama `trabajo/plataforma-agentes` desde GitHub.
- En `/srv/orquesta-self/worktrees/orquesta`, reparar la rama con
  `git switch -C trabajo/plataforma-agentes <sha-subido>` porque ese worktree
  quedo detached por el deploy anterior.
- Ejecutar de nuevo el deploy atomico. Si pasa, el recibo debe quedar `ok` y
  `/api/status` debe exponer el mismo `runtime_identity.binary_sha256`.
- Despues de F0, activar F2 Telegram remoto con `orquesta.config.json`
  canonico y credenciales reales; OPES/F1 sigue gateado por orden del operador.

## Actualizacion Codex 2026-07-10: revision remota en modo revisor

Nuevo documento de revision para Claude:

- `docs/runbooks/revision_codex_orquesta_remoto_2026-07-10.md`

Lectura corta:

- Codex queda como revisor externo; Orquesta remoto sigue siendo quien ejecuta.
- Se acepta como avance parcial el goal
  `goal-ref-task-autoprogramming-f0ad7bb152dd-g01` y su rework
  `goal-ref-task-autoprogramming-f0ad7bb152dd-g01-rework-1`, ambos con
  required tests declarados en verde.
- No se acepta cierre global de autonomia: el binario remoto vivo sigue siendo
  anterior al commit pusheado, hay `stale_running` y quedan runs antiguos
  `invalid/blocked/running`.
- Siguiente corte recomendado: F3 drain gobernado + harness aislado, despues F5
  deploy del binario nuevo con guard de identidad. No lanzar mas goals amplios
  antes de ese corte.

## Actualizacion Codex 2026-07-10: recibos fuera del write-set

Commits locales sincronizados: `37575b1e4`, `098fa1741`, `6bb71bdbe` y
`27c06d251` sobre `trabajo/plataforma-agentes`.

Cambios realizados:

- Registro efectivo de variables de runtime, Director, supervisor y arranque:
  el ratchet AST baja de 105 a 59 lecturas sin clasificar. La limpieza amplia
  queda pospuesta hasta terminar el nucleo.
- S13/F4: Codex app-server materializa checkpoint y resultado por `goal_ref`
  bajo `.orquesta-runtime/goal-receipts/`, ignorado por Git. El observador
  prioriza el recibo runtime con refs correladas y conserva el arbol del
  proyecto solo como fallback para evidencia legacy.
- Claude y Gemini usan su `RuntimeWorkDir` externo para el resultado final y
  lo leen antes de `docs/`/write-set. Sus prompts ya no ordenan escribir
  recibos de ejecucion como artefactos de producto.

Pruebas locales ejecutadas:

```bash
go test -count=1 ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-runtime-codex-appserver
go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-runtime-gemini
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'Test.*(GoalFirst|Materialized|Observe).*V0'
bash scripts/test_orquesta_check_versioned_execution_artifacts.sh
git diff --check
```

No se ejecuto smoke real de proveedor ni una app de prueba artificial: el
operador pidio reservar esa cuota para una app real. S13 sigue abierto por la
migracion gobernada de 58 recibos movibles, proyeccion comun de procedencia y
smoke real aislado. Ver
`docs/incidencias/incidencia_orquesta_s13_destino_recibos_runtime_2026-07-10.md`
y el indice vivo, que prevalecen sobre este resumen historico.

## Actualizacion Codex 2026-07-10: revalidacion local de nucleo antes de limpieza

El operador fijo el orden: cerrar nucleo primero; despues, limpieza gobernada
de variables y codigo. PDF/tools, conectores y temarios quedan fuera de este
corte.

Revalidacion offline ejecutada sin arrancar servidor, agentes, apps ni tocar
remoto/OPES/uso-app:

```bash
ORQUESTA_DRAIN_TEST_CACHE_ROOT=/tmp/orquesta-f3-r2-drain \
  bash scripts/test_orquesta_server_drain.sh
ORQUESTA_BATCH_TEST_CACHE_ROOT=/tmp/orquesta-f3-r2-batches \
  bash scripts/test_orquesta_test_batches.sh
GOCACHE=/tmp/orquesta-core-attestation-gocache \
GOTMPDIR=/tmp/orquesta-core-attestation-tmp \
  go test -count=1 \
    ./modulos/orquesta-goal \
    ./modulos/orquesta-runtime-required-test \
    ./modulos/orquesta-state-file \
    ./modulos/orquesta-app-director-service \
    ./modulos/orquesta-app-codex-stack \
    ./modulos/orquesta-autoprogramming
```

Resultado: verde. F3 valida identidad, backup, proteccion incondicional de
`uso-app`, lock y lotes aislados; los seis paquetes validan atestacion
independiente, persistencia, director y stack goal-first. No es evidencia de
drain/deploy/API reales: los residuales vivos siguen siendo los del indice
`docs/inventario_bugs_estado_vivo.md` y requieren ventana operativa
autorizada.

Revalidacion focal adicional de F1/F5, tambien verde:

```bash
GOCACHE=/tmp/orquesta-core-identity-gocache \
GOTMPDIR=/tmp/orquesta-core-identity-tmp \
  go test -count=1 ./modulos/orquesta-estado-vivo ./modulos/orquesta-mcp \
  -run 'Test(DerivarVeredictoCausalV0|ConstruirProyeccionCicloVidaV0|MCPObserveAppDirectorGoalEstadoVivo)'
GOCACHE=/tmp/orquesta-core-identity-gocache \
GOTMPDIR=/tmp/orquesta-core-identity-tmp \
  go test -count=1 ./cmd/orquesta-server \
  -run 'Test(DegradedIdentityHTTPHandlerV0|ValidateServerWorktreeIdentityV0|ServerIdentityPreflightV0|ServerWorktreeDirFromEnvV0|ServerYControlScriptCompartenRemoteCanonicoYPoliticaSymlinkV0|RunMainV0BloqueaLaunchAmplioAntesDeCrearRuntimeV0|ServerCommandRequiresExactIdentityV0|VerifyLiveDaemonIdentityV0|ServerReadiness)'
```

La cobertura local confirma que el veredicto causal no publica `running` sin
liveness y que el servidor rechaza identidad/worktree degradados antes de crear
runtime. No sustituye la comprobacion por API contra el binario desplegado.

Comprobacion de integracion adicional, sin arrancar el binario:

```bash
GOCACHE=/tmp/orquesta-server-build-gocache \
GOTMPDIR=/tmp/orquesta-server-build-gotmp \
GOFLAGS=-buildvcs=false \
  go build -o /tmp/orquesta-server-build/orquesta-server ./cmd/orquesta-server
```

Resultado: binario local compilado (SHA-256
`8b49eb580231468e14844f5c3fd79ee56c5792d8af8f5968cfbc08ccb09f70e0`).
Ese digest solo acredita este build local aislado; el digest exigible para F5
es el que produzca el deploy remoto y publique despues `/api/status`.

El contrato offline del deploy tambien fue reejecutado sin red ni cambio de
binario vivo:

```bash
TMPDIR=/tmp/orquesta-deploy-test-audit \
ORQUESTA_TEST_CACHE_ROOT=/tmp/orquesta-deploy-test-cache \
GOTMPDIR=/tmp/orquesta-deploy-test-gotmp \
  bash scripts/test_orquesta_server_deploy.sh
```

Resultado: `orquesta_server_deploy_tests=ok`. Permanece pendiente el receipt
de un deploy real, que no se simula desde el equipo local.

Revalidacion no destructiva de S13/F4:

```bash
bash scripts/test_orquesta_check_versioned_execution_artifacts.sh
scripts/orquesta_check_versioned_execution_artifacts.sh --root "$PWD" --json
```

Resultado: verde, 68 candidatos Git clasificados y sin rutas stale o sin
clasificar: 58 `movable_runtime_artifact`, 5 evidencia historica, 4 fixtures
y una referencia pendiente. No cierra S13: siguen requeridos la migracion
gobernada, procedencia comun y smoke real aislado.

## Actualizacion Codex 2026-07-10: segunda ola de configuracion local

El operador indico continuar la limpieza local sin crear apps de prueba ni
ejecutar proveedores. Se consolido la familia Gemini en
`orquesta.config.json > gemini_runtime`:

- la seccion tipada admite `enabled`, comando, workdirs, `HOME`, `PATH`,
  modelo, aprobacion, formato y argumentos;
- runtime y backend goal-first reutilizan una sola resolucion de perfil;
- se conserva `env > fichero > default`; un
  `ORQUESTA_GEMINI_ENABLED=false` explicito prevalece sobre el fichero;
- las diez variables Gemini quedan registradas y el ratchet AST baja de 59 a
  49 lecturas no clasificadas.

Pruebas locales, sin agente ni proveedor real:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run 'Test(GeminiRuntimeConfigV0|ServerGoalBackendFromEnvV0Gemini(Process|FileControl)|ServerEnvRegistryASTV0LecturasORQUESTARegistradas|Config)'
go test -count=1 ./modulos/orquesta-runtime-gemini
```

Resultado: verde. La siguiente familia no debe mezclar OPES: priorizar runner
de tests requeridos o promotion/guardian con el mismo patron tipado y ratchet.

## Actualizacion Codex 2026-07-10: runner de tests requeridos tipado

Se consolido `orquesta.config.json > required_test_runner` sin ejecutar
tests de proyecto ni proveedores. La seccion contiene opt-in, comando Go,
allowlist adicional, salida aislada, entorno proyectado y limites de salida y
retencion. Los mapas de allowlist y entorno se ordenan antes de construir el
executor para conservar recibos deterministas.

Las variables historicas siguen como override de compatibilidad. El opt-in
explicito del entorno prevalece sobre el fichero y cualquier valor distinto de
`1` mantiene el runner desactivado, igual que antes.

Pruebas locales:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run 'TestRequiredTestRunnerFromEnvV0|TestServerEnvRegistryASTV0LecturasORQUESTARegistradas|TestConfig'
```

Resultado: verde; ratchet AST 49 -> 42. Siguiente familia candidata:
promotion/guardian, no OPES.

## Actualizacion Codex 2026-07-10: primera retirada de codigo privado

Se retiro el par privado sin llamadas
`codexGoalTimeoutMSFromEnvV0`/
`codexGoalPreflightTimeoutMSFromEnvV0`. No se elimina la configuracion de
timeouts: el backend vivo ya usa los lectores tipados de `goal_backend`.

Prueba focal y auditoria:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run 'Test(ServerGoalBackend|Config|ServerEnvRegistryASTV0LecturasORQUESTARegistradas)'
scripts/orquesta_auditoria_codigo.sh --root "$PWD" --no-sqlite
```

Resultado: verde; el indice ya no contiene ninguna de las dos funciones. No se
usan los 1.249 candidatos restantes como autorizacion de borrado masivo.

## Actualizacion Codex 2026-07-10: retirada de wrappers legacy Goal

Se retiraron doce wrappers privados sin llamadas de deteccion `*FromEnv` y
derivacion legacy de backend Codex/Claude/Gemini. No se elimina la capacidad de
seleccionar backend: la composicion viva consume
`codexGoalBackendFromProjectConfigFileV0` y sus adaptadores por valor.

La auditoria reproducible queda en 1.237 candidatas y no contiene los wrappers
retirados. Verificacion focal:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run 'Test(ServerGoalBackend|GeminiRuntimeConfigV0|Config|ServerEnvRegistryASTV0LecturasORQUESTARegistradas)'
scripts/orquesta_auditoria_codigo.sh --root "$PWD" --no-sqlite
```

Resultado: verde. No se arranco servidor, app, agente ni proveedor.

## Actualizacion Codex 2026-07-10: promocion de autoprogramacion tipada

Se consolido la parte no ejecutora de promotion en
`autoprogramming.promotion`: opt-in, archivo, refs opacas y mensaje de
commit. El entorno continua teniendo precedencia por compatibilidad. El guard
`self_programming_only` reutiliza la ruta resuelta en lugar de leer una
segunda configuracion distinta.

El guardian de reparacion se dejo inicialmente fuera porque ejecuta comandos y
tiene mas de veinte inputs; su consolidacion se cerro en el corte siguiente,
con pruebas especificas de aislamiento.

Pruebas locales:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run 'TestAutoprogrammingPromotionConfigFromEnvV0|TestSelfProgrammingOnlyConfigV0|TestServerEnvRegistryASTV0LecturasORQUESTARegistradas|TestConfig'
```

Resultado: verde; ratchet AST 42 -> 36. No se ejecuto promotion ni guardian.

Checkpoint de limpieza no destructiva sincronizado: `4820516ec`
(`chore: indexa funciones para limpieza segura`). El auditor genera un indice
lexico de 25.199 funciones y una SQLite derivada; sirve para priorizar
revision, nunca para borrar automaticamente. Detalle:
`docs/auditorias/indice_funciones_orquesta_2026-07-10.md`.

## Actualizacion Codex 2026-07-10: guardian de promocion tipado local

La cuarta familia de configuracion local es
`autoprogramming.promotion.guardian`. Sus 29 inputs pasan a
`orquesta.config.json`, con compatibilidad `env > fichero > default`,
precedencia negativa de `enabled=false`, y registro efectivo con alcance
propio. El proceso hijo conserva la allowlist y no recibe secretos del proceso
padre por defecto.

El corte fue solo local y focal: no se arranco servidor, promocion, guardian,
agente ni proveedor. La prueba de configuracion canonica, la proyeccion
auditable de Gemini/runner/promotion y las pruebas de allowlist/recibos pasan.
El ratchet AST baja de 36 a 5: quedan solo cinco lecturas de OPES, fuera del
nucleo. La confirmacion del harness MCP opt-in queda clasificada aparte y no
forma parte de configuracion de runtime.

## Actualizacion Codex 2026-07-10: cierre de regresion 208U

La verificacion amplia local descubrio dos regresiones del corte de
configuracion antes de cualquier push: `config_file_v0.go` y
`effective_config_v0.go` superaban T90, y los aliases históricos de timeout
del guardian aparecian como nombres sin unidad. Se extrajeron familias
cohesivas a ficheros propios; los timeout canonicos pasan a `*_TIMEOUT_MS` y
los aliases de duracion Go quedan marcados como deprecados, con precedencia
canonico > legacy > fichero > default.

Se reejecutaron T90, registry/AST, guardian, configuracion efectiva y MCP:
verde. No se arranco servidor, guardian, agente ni proveedor. El indice
canonico conserva el cierre como `BUG-ORQ-20260710-208U`.

Una tentativa posterior de `go test -count=1 ./cmd/orquesta-server` completo
activo un fake OPES interno y quedo esperando; se termino cooperativamente por
PID junto con su helper, sin cambios de fuente ni procesos residuales. No se
usa esa tentativa como evidencia de cierre: para este corte valen los focales
aislados anteriores. El paquete completo necesita ejecutarse solo en un corte
que admita sus fixtures OPES y tenga timeout/cleanup gobernados.

## Actualizacion Codex 2026-07-10: limpieza minima de wrappers privados

El indice actualizado se uso solo como triage. Se revisaron y retiraron dos
wrappers sin consumidores de produccion: el lector de politica de progreso
que servia exclusivamente a una prueba y el lector de limites Codex sin
llamadas. La prueba consulta ahora la configuracion viva equivalente. El
indice baja de 25.223 a 25.221 funciones y de 1.238 a 1.236 candidatas
estaticas; no hay borrado masivo ni cambio de contratos.

Una segunda pasada retiro dos wrappers privados mas sin referencias (lector de
politica desde directorio y adaptador runtime Codex desde directorio). El
indice reproducible queda en 25.219 funciones y 1.234 candidatas. La regla se
mantiene: candidatos estaticos no se borran sin verificar consumidores,
contratos y prueba focal.

La auditoria tambien marcaba siete modulos con `importer_count: 0`. Revision
manual: todos son contratos o adaptadores opt-in deliberadamente desacoplados
(`data-ingestion`, extraccion JSON/CSV/capability, SQL de referencia,
presentaciones y tool-capability-file). Ninguno se borra; la clasificacion y
las rutas de evidencia quedan en
`docs/auditorias/indice_funciones_orquesta_2026-07-10.md`.

El indice de funciones se refino sin tocar runtime: para privadas marcadas por
deadcode distingue referencias de produccion, solo de tests y ninguna, sin
contar la declaracion. Foto actual: 731 / 41 / 53 respectivamente; 409
exportadas mantienen la categoria contractual previa. Los campos JSON/SQLite
son aditivos y no autorizan ningun borrado automatico.

Primera aplicacion del grupo sin referencias: retirados dos wrappers privados
(allowlist legacy del runner guardian y acceso directo al broker de contexto).
El wiring vigente permanece; focales verdes. Indice actual: 25.217 funciones y
1.232 candidatas brutas.

## Actualizacion Codex 2026-07-11: tools de ingesta y presentaciones

Se integraron en paralelo los adaptadores hexagonales
`orquesta-data-ingestion-tool-capability` y
`orquesta-presentation-extraction-tool-capability`. Ambos reutilizan el SDK
`ToolBundle/AttachPlan`: descriptor neutral, autoridad de registro, intent por
refs, snapshot opaco, efecto idempotente y delegacion de lease/CAS/recibos al
SDK. Se verificaron sus cuatro modulos, el SDK y la frontera neutral raiz.

No hay parser real, OCR, filesystem, driver SQL, DSN, red, LibreOffice ni
composicion productiva en estos modulos. Esos adaptadores quedan como siguiente
fase opt-in, no como dependencia del nucleo.

## Actualizacion Codex 2026-07-11: MCP stdio local

El endpoint HTTP `/mcp` ya existia; se completo el hueco de transporte por
stdio con `orquesta-server mcp-stdio`. Lee una request JSON-RPC por linea,
reutiliza el dispatcher existente, reserva stdout para respuestas y stderr para
diagnostico, y no inicia listener HTTP. Los focales cubren handshake, listas,
lectura de recurso, notificacion y JSON invalido/trailing. El falso verde del
primer fixture de recurso queda registrado y cerrado como 208V.

## Actualizacion Codex 2026-07-11: e2e fake Claude y Gemini goal-first

La prueba raíz `TestGoalFirstProcessBackendsE2EV0ReworkThenClose` cubre ambos
backends de proceso reales contra ejecutables fake aislados. Deriva el spec,
artefactos y tests del preview, arranca `StartAppDirectorV0` goal-first,
fuerza rework por tests requeridos fallidos y cierra con evidencia/validación
accepted en una segunda entrega. No crea app externa ni consume proveedor.

Esto cierra el hueco de cobertura local completa de Claude/Gemini. No se debe
declarar proveedor real cerrado: Claude necesita smoke real opt-in y Gemini
sigue bloqueado por tier/autenticación hasta que haya acceso operativo.

## Actualizacion Codex 2026-07-11: adaptador CSV/JSON de ingesta

Se incorporo `orquesta-data-ingestion-file`, primer adaptador real fuera del
nucleo para fuente y perfilado CSV/JSON con stdlib. Usa catálogo de refs opacas,
root explícito, hash/snapshot/provenance y límites; rechaza traversal, symlinks,
mutación de snapshot y trailing JSON. Mapping/validación de dominio y otros
formatos o BBDD siguen fuera de este adaptador. La omisión inicial del trailing
JSON se detectó en revisión, se probó y quedó cerrada como 208W antes de commit.

## Actualizacion Codex 2026-07-11: limpieza antes de conectores y tools

Por decision del operador, la limpieza estructural pasa a preceder cualquier
nuevo conector o tool: primero configuracion canonica, duplicados, wrappers
muertos y separaciones cohesivas de coordinadores grandes que afecten al
nucleo/conectores; despues se retoma funcionalidad sobre esa base.

Se cerro la familia OPES residual de configuracion local. El snapshot tipado
`serverOPESConfigSnapshotV0` concentra workdir, timeout e intentos con
precedencia `env > orquesta.config.json > default`; guard, `domain_work` y
contexto OPES lo consumen. La secuencia de job types del bridge usa la utilidad
generica de listas del fichero canonico. El ratchet AST de lecturas
`ORQUESTA_*` no registradas pasa de 5 a 0 y queda estricto en 0.

Tambien se retiraron cinco wrappers privados sin consumidores y se extrajo el
helper de listas desde `domain_work` a configuracion comun. Focales verdes:
`TestServerEnvRegistryASTV0LecturasORQUESTARegistradas`, precedencia OPES,
configuracion del bridge, bucle seguro de secuencia, runtime Codex appserver y
state-file. No se arranco servidor, OPES, bridge, agente ni proveedor. Quedan
en revision separada los coordinadores grandes y candidatos de conectores;
ningun borrado masivo esta autorizado por el indice.

## Actualizacion Codex 2026-07-11: wrappers privados de proveedores

Se retiro una tanda independiente de cinco wrappers privados sin callers en
los adaptadores Codex/Claude/Gemini. El constructor de prompt legacy Codex, un
helper de write-set sin consumidor, el wrapper de effort que era todo su
fichero y los dos wrappers de protocolo durable por locale estaban sustituidos
por rutas activas. No cambian prompts, modelos, perfiles, contratos ni
proveedores. `rg`, `git diff --check` y los cuatro `go test` focales de runtime
quedaron verdes. Commit de codigo: `d53ff9090`.

## Actualizacion Codex 2026-07-11: guards de limpieza R1 y R2

El revisor detecto dos guards rojos antes de continuar. R1 quedo cerrado en
`cec2848f3`: `orquesta_metricas_deuda.sh` exporta `LC_ALL=C` y su prueba usa
shims de `sort`/`comm` con locale sintetico, sin requerir que `es_ES` exista.
R2 quedo cerrado en `31ecba309` sin elevar presupuestos: se retiraron los dos
overrides `*_TIMEOUT_MS` duplicados del guardian, quedando fichero tipado y
variables de duracion existentes; los tres nombres IPC del helper de tests se
renombraron fuera de `ORQUESTA_*`. La metrica vuelve a 425 productivas y 103
solo-fixture. El guard raiz tambien fija `LC_ALL=C` en su hijo porque Bash
puede emitir un warning antes del script con un locale heredado invalido.

Se reejecutaron focales guardian/tool-file, el guard raiz con C y no-C, y el
test de metricas. Todo verde; no se inicio servidor, guardian, agente ni
proveedor. La incidencia cerrada es `BUG-ORQ-20260711-208X`.
