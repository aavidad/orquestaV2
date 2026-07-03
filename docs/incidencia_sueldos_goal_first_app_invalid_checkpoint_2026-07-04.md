# Incidencia: goal-first de nueva app termina con checkpoint invalido

Fecha: 2026-07-04

Inventario: `BUG-ORQ-20260704-166` en
`docs/inventario_bugs_orquesta_2026-06-30.md`.

## Contexto

Se intento crear desde Orquesta una app externa en:

- Proyecto: `/home/alberto/Trabajo/Sueldos`
- Scope: `generated-apps/mapa-de-gasto-publico`
- Ruta: `POST /api/v0/apps/director`
- Modo: `director_execution_mode=goal_first`
- Run: `run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-43c20b877af23ecc450d324a25c1e616`
- Goal: `goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-43c20b877af23ecc450d324a25c1e616`
- External goal: `019f2a37-df1f-7e00-a60b-dab168d79d38`

El servidor temporal estaba aislado con estado/runtime bajo
`/home/alberto/Trabajo/Sueldos/.orquesta`.

## Resultado observado

Orquesta acepto el run con HTTP 200 y lanzo el goal, pero solo materializo:

- `generated-apps/mapa-de-gasto-publico/checkpoint_started.txt`
- `generated-apps/mapa-de-gasto-publico/docs/orquesta_goal_result_goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-4.json`

El resultado durable quedo en:

```json
{
  "schema_version": "orquesta_goal_result.v0",
  "status": "invalid",
  "summary": "resultado durable inicial; pendiente de implementacion, artefactos y pruebas",
  "missing_refs": ["source_tree", "handoff_report", "technical_stack_manifest", "go_app", "tests"]
}
```

No se genero app Go, UI, datos, documentacion tecnica ni tests.

Durante el polling posterior, el servidor dejo de responder. El estado final
quedo como `status=stopped` y el audit log registro:

- `server_shutdown_signal` con `signal_interrupt`
- `runtime_shutdown_hook` con `codex_app_server_tmux_pane_exit_timeout`

El log del app-server de Codex solo mostro un aviso de trust:

```text
Project-local config, hooks, and exec policies are disabled ... /home/alberto/Trabajo/orquesta/.codex
```

## Observacion posterior

Tras el fallo, el app-server de Codex/Orquesta dejo algunos ficheros parciales
en el scope, sin cierre valido de Orquesta:

- `internal/domain/seed.go`
- `internal/domain/validation.go`
- `internal/ports/repository.go`

Esos ficheros no constituyen una entrega cerrada: no venian acompanados de app
ejecutable completa, tests, `source_tree`, `technical_stack_manifest` ni
`handoff_report` aceptado. Ademas, el operador inicio despues un desbloqueo
manual dentro del mismo scope. Por tanto, cualquier verificacion de esta
incidencia debe distinguir:

- artefactos Orquesta previos al fallo;
- artefactos manuales posteriores al fallo;
- ausencia de cierre Orquesta `complete/accepted`.

## Impacto

La ruta vigente de nueva app goal-first puede dejar una entrega inutil marcada
como artefacto durable inicial, sin completar el trabajo ni replanificar hacia
un programador. Para el operador, el run parece lanzado correctamente, pero el
resultado real es un checkpoint invalido.

## Comportamiento esperado

Para un goal de nueva app aceptado:

1. Orquesta debe mantener vivo el servidor/observer hasta estado terminal real,
   o devolver una incidencia operativa accionable antes de cerrar el intento.
2. Un `orquesta_goal_result.v0` con `status=invalid` y `missing_refs` de app,
   fuente, handoff y tests no debe tratarse como entrega util.
3. El observer debe disparar replan/rework o crear una tarea para programador
   Orquesta/Codex cuando solo existe checkpoint.
4. El shutdown del backend tmux debe cerrar limpiamente o registrar reparacion
   especifica si `codex_app_server_tmux_pane_exit_timeout` se repite.
5. La UI/API de Orquesta debe mostrar que la app esta bloqueada y requiere
   programador, en vez de dejar al usuario pensando que el trabajo sigue normal.

## Reproduccion historica resumida

Contexto: incidencia observada en un harness temporal aislado. No es una ruta
operativa vigente ni una instruccion para arrancar runtime manual en sesiones
normales.

1. Arrancar el servidor temporal aislado del harness de reproduccion con:
   - `ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/Sueldos`
   - `ORQUESTA_CODEX_RUNTIME_WORKDIR=/home/alberto/Trabajo/Sueldos/.orquesta/runtime`
   - `ORQUESTA_SERVER_STATE_DIR=/home/alberto/Trabajo/Sueldos/.orquesta/state`
   - `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`
   - `ORQUESTA_CODEX_CODE_HOME=/home/alberto/.codex`
2. Enviar nueva app por `/api/v0/apps/director` con `director_execution_mode=goal_first`.
3. Observar por `/api/v0/apps/director/goal/observe`.
4. Comprobar que solo existe checkpoint y resultado `invalid`.

## Workaround aplicado

Codex va a implementar manualmente la app dentro del scope creado por Orquesta:

`/home/alberto/Trabajo/Sueldos/generated-apps/mapa-de-gasto-publico`

Este workaround debe considerarse desbloqueo puntual, no sustituto del flujo
normal de Orquesta.

Actualizacion del operador: por instruccion posterior del usuario, el workaround
manual no debe tratarse como entrega final. El estado operativo queda bloqueado
a la espera de reparacion por programador de Orquesta.

## Reparacion Orquesta aplicada

Estado: arreglado funcionalmente en codigo Orquesta con pruebas focales; queda
pendiente relanzar el flujo real de Sueldos/Orquesta si se quiere validar el
caso en campo.

Cambios:

- `missing_refs` a nivel raiz en `orquesta_goal_result.v0` se normaliza a
  `checklist.missing_refs` tanto en el backend `codex app-server` como en el
  scanner de resultados materializados del stack.
- El status terminal explicito del marcador/fichero durable (`invalid`,
  `blocked` o `complete`) gobierna el receipt observado; no queda oculto por un
  estado de thread `complete`.
- Nueva App goal-first permite rework automatico para `WorkKind=new_app` cuando
  el resultado terminal `invalid|blocked` declara faltantes estructurales como
  `source_tree`, `handoff_report`, `technical_stack_manifest`, `go_app` o
  `tests`.
- Si hay `GoalReworkLauncher`, el observer relanza un goal causal `-rework-1`
  y no bloquea la run sin programador. Si no hay launcher/presupuesto, el run
  sigue bloqueado con cierre no aceptado.

Pruebas focales:

- `go test -count=1 ./modulos/orquesta-app-director-service -run 'TestObserveAppDirectorGoalV0(LanzaReworkGoalSiNuevaAppSoloCheckpointInvalid|LanzaReworkGoalPorTimeoutActivo|BloqueaRunSiGoalTerminaInvalid|LanzaReworkGoalSiPolicyYPuertoDisponibles)'`
- `go test -count=1 ./modulos/orquesta-runtime-codex-appserver -run 'TestServerCodexAppServerGoalBackendV0(ObservaResultadoMarcadoMigrado|NormalizaMissingRefsRaizEnChecklist)'`
- `go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestStackGoalMaterializedRefsSourceV0NormalizaMissingRefsRaizEnReceiptDurable'`

## Reintento de campo: resultado durable completo no reconciliado

Fecha: 2026-07-04

Tras la reparacion anterior se reintento observar el mismo run de Sueldos con
el estado/runtime aislado ya existente. El servidor Orquesta arranco en
`http://127.0.0.1:40081` y se llamo:

```text
POST /api/v0/apps/director/goal/observe
request_id=request-ref-sueldos-gasto-publico-osm-observe-retry-20260704-001
run_ref=run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-43c20b877af23ecc450d324a25c1e616
```

Respuesta publica resumida:

```json
{
  "estado": "error",
  "partial": true,
  "run_status": "activa",
  "goal_status": "running",
  "closure_status": "blocked",
  "closure_needs_rework": true,
  "summary": "codex_app_server_goal_status_active",
  "recommended_action": "review_partial_artifacts",
  "closure_issues": [
    {
      "code": "partial_artifacts_written",
      "field": "goal_first.partial_artifacts"
    }
  ],
  "errores_publicos": [
    {
      "code": "codex_goal_observation_rejected"
    }
  ]
}
```

Sin embargo, existe un recibo durable terminal completo en el write-set:

`/home/alberto/Trabajo/Sueldos/generated-apps/mapa-de-gasto-publico/docs/orquesta_goal_result_goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-4.json`

Resumen del fichero:

- `schema_version=orquesta_goal_result.v0`
- `status=complete`
- `goal_ref=goal-ref-app-director-run-spec-mapa-de-gasto-publico-req-mapa-de-gasto-publico-43c20b877af23ecc450d324a25c1e616`
- `artifact_refs=3`
- `artifact_paths=32`
- `materialized_artifacts=3`
- `checklist.missing_refs=[]`
- `required_test_results[0].status=passed`

El estado persistido de Orquesta sigue en:

- `run.status=activa`
- `current_phase=brainstorming_arquitectura`
- `goal_state.status=running`
- `goal_state.last_result.summary=codex_app_server_goal_status_active`

Diagnostico operativo:

El backend `app_server_tmux` ya contiene logica para observar resultados de
fichero durante un goal activo, pero en este caso no reconcilia el
`orquesta_goal_result.v0` terminal `complete`. La API publica termina
rechazando la observacion como `partial_artifacts_written`, pese a que el
recibo durable declara los artefactos requeridos y el test requerido como
`passed`.

Comportamiento esperado adicional:

1. Si existe un `orquesta_goal_result.v0` terminal `complete` dentro del
   write-set y pasa el contrato, `/goal/observe` debe fusionarlo y cerrar el run
   con `goal_status=complete`, `run_status=cerrada` y
   `closure_status=accepted`.
2. Si el guard de write-set rechaza el resultado por cambios fuera del scope,
   debe devolver `codex_app_server_runtime_write_set_violation` con paths
   accionables, no `partial_artifacts_written` generico.
3. Tras un `codex_app_server_tmux_pane_exit_timeout`, el observer debe poder
   reconciliar recibos durables terminales en disco aunque el estado runtime del
   goal-server siga declarando `active/running`.

Estado actual del caso Sueldos:

Bloqueado para cierre Orquesta. No se debe marcar esta entrega como aceptada por
Orquesta hasta que el programador repare la reconciliacion de resultado durable
terminal frente a estado runtime `running`.

## Intento de reparacion usando Orquesta

Fecha: 2026-07-04

Por instruccion operativa se intento que Orquesta programara el arreglo de este
bug antes de aplicar un parche directo con Codex.

Servidor temporal aislado:

- Project workdir: `/home/alberto/Trabajo/orquesta`
- State/runtime: `/home/alberto/Trabajo/Sueldos/.orquesta-orquesta-fix`
- Backend: `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`
- Endpoint: `http://127.0.0.1:39417`

Entrada usada:

```text
POST /api/v0/autoprogramming/prepare-run
request_id=request-ref-orquesta-fix-bug-166-20260704-001
director_execution_mode=goal_first
run_ref=request-ref-orquesta-fix-bug-166-20260704-001
goal_ref=goal-ref-task-autoprogramming-9ffe40481e2c-g01
external_goal_ref=019f2a51-9bb5-7cf3-b37b-9af8d7a9efd3
```

Objetivo enviado a Orquesta:

- reproducir `GetGoal active/running + orquesta_goal_result.v0 complete en
  workspace`;
- fusionar ese recibo terminal y cerrar como `complete/accepted`;
- mantener `codex_app_server_runtime_write_set_violation` si el guard de
  write-set bloquea;
- anadir tests focales en `orquesta-runtime-codex-appserver` y proyecciones
  MCP/app-codex-stack.

Resultado:

- `prepare-run` fue aceptado y lanzo goal-first real.
- Tras 9 observaciones por `/api/v0/autoprogramming/goal/observe`, el goal
  siguio en `goal_status=running`.
- Las observaciones publicaron `recommended_action=review_partial_artifacts`,
  `partial_artifacts_written` y `goal_first_materialized_checkpoint_detected`.
- A partir de la observacion 6 Orquesta publico:
  `codex_app_server_goal_status_active_high_token_usage`, con evidencias
  `goal-observer-no-checkpoint-high-consumption`,
  `goal-observer-high-consumption-stop-requested` y
  `goal-cooperative-stop-requested-run-control`.
- El estado final persistido del goal quedo `status=blocked`,
  `last_result.summary=goal_active_no_checkpoint_high_consumption` y
  `rework_plan_refs=[rework-plan-ref-goal-active-no-checkpoint-high-consumption]`.
- No hubo cambios de codigo; solo seguian modificados los documentos de esta
  incidencia e inventario.

El servidor temporal se cerro por `/api/v0/server/shutdown` con
`cleanup_goal_backends=true` y devolvio `shutdown_ready=true`, sin procesos
`orquesta-server`, `codex app-server` ni `codebase-memory-mcp` vivos al final.

Decision operativa:

Orquesta fue usada primero para programar el arreglo, pero quedo bloqueada por
alto consumo sin checkpoint ni patch. Se permite parche directo con Codex como
desbloqueo puntual documentado para reparar Orquesta.

## Checkpoint de reparacion BUG-ORQ-20260704-166

Fecha: 2026-07-04

Goal: `goal-ref-task-autoprogramming-9ffe40481e2c-g01`.

Alcance autorizado:

- `orquesta-runtime-codex-appserver`: resultado durable y guard runtime de
  write-set.
- `orquesta-app-codex-stack`: extraccion/proyeccion de refs materializadas.
- `orquesta-mcp`: enriquecimiento de `observe_goal` y reparacion de recibos.
- Este documento y el inventario de bugs.

Siguiente artefacto verificable: test focal que reproduce un
`orquesta_goal_result.v0` terminal `complete` bajo write-set mientras el backend
sigue publicando `running`, y fix que lo reconcilia o devuelve
`codex_app_server_runtime_write_set_violation` con rutas accionables si el guard
lo rechaza.
