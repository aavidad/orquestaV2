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
