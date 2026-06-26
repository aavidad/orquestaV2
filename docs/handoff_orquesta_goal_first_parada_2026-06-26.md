# Handoff Orquesta goal-first - parada 2026-06-26

Motivo de parada: el operador pide parar para reservar cuota mientras OPES
trabaja en otra sesion.

## Estado del ultimo tramo

- Rama: `trabajo/plataforma-agentes`.
- Ultimo commit ya publicado antes de este handoff: `623ef40c Expone agentes vivos en autoprogramacion`.
- Cambio preparado para el siguiente commit:
  - Web `/autoprogramming` transporta y muestra `queue_health.agents_live`.
  - `modulos/orquesta-goal/docs/tareas.md` y
    `modulos/orquesta-runtime-codex-goal/docs/tareas.md` dejan de decir que el
    smoke real goal-first esta pendiente; el cierre local con
    `app_server_stdio` queda documentado como cerrado el 2026-06-26.
  - `cmd/orquesta-server/director_cycle_resident_restart_smoke_v0_test.go`
    estabiliza el intervalo del test residente para evitar ticks extra antes
    de cancelar.

## Verificacion ejecutada

- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp`
- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server -run TestRuntimeResidentDirectorConduceDirectorCycleStepTrasRestartV0 -v`
- `git diff --check`
- `go test -count=1 -p=1 ./...`

La primera ejecucion amplia fallo en
`TestRuntimeResidentDirectorConduceDirectorCycleStepTrasRestartV0` por carrera
del propio test: el tick residente estaba a `10ms` y podia ejecutar mas de un
tick antes de cancelar. Aislado pasaba. Se subio el intervalo a `100ms` y la
bateria focal + `go test -count=1 -p=1 ./...` pasaron.

## Estado por frente

- `/nueva-app` goal-first: alrededor de 90%. La ruta real con Codex
  `app_server_stdio` esta cerrada por smoke con `goal_status=complete`,
  `run_status=cerrada`, `closure_status=accepted`, artefactos y evidencias.
  Falta restart/resume de goal en vuelo, negativos completos y rework automatico
  cuando el cierre no es aceptado.
- Web/operador: alrededor de 80%. El wizard de `/nueva-app` existe, con ayuda y
  opciones expertas parciales. Queda mejorar accesibilidad real de tooltips,
  i18n de textos expertos hardcodeados y alinear guia/opciones exactas.
- `autoprogramming/status` y `supervise`: alrededor de 75-80%. HTTP colgado
  esta mitigado con `202 accepted_background`; `running_stale` con procesos
  vivos ya consulta stats/procesos y expone `agents_live`. Falta test de
  frontera gateway/stack completo y mas seniales humanas para casos OPES finos.
- Orquesta nucleo goal-first: alrededor de 82-85%.
- Orquesta incluyendo OPES temporal completo hasta derivados/HTML: alrededor de
  70-75%.

## Pendientes concretos siguientes

1. Commit/push de este lote si aun no esta publicado.
2. Reiniciar solo el servidor web local `127.0.0.1:8787` con el binario nuevo
   cuando se retome; no tocar servidores OPES.
3. Mejorar `/nueva-app`:
   - sustituir tooltips CSS por ayuda accesible con `aria-describedby`;
   - mover textos expertos hardcodeados a i18n;
   - completar campos expertos de datos/storage/integraciones;
   - actualizar `guia_nueva_app_opciones_2026-06-25.md`.
4. Demotar visiblemente herramientas MCP legacy de programacion frente a rutas
   goal-first/estado, sin borrar handlers ni smokes historicos.
5. Anadir contrato/test para backend Goal configurado pero degradado: no debe
   caer silenciosamente a `legacy_director_loop` si el operador pidio goal.
6. Anadir test de frontera para `/api/v0/autoprogramming/status` con cola
   `running`, process refs vivos y salida `running_live=1`, `agents_live>0`,
   `stale_running=[]`.
7. OPES sigue en otra sesion: no pisar procesos ni documentos que escriba ese
   agente. El fichero
   `TAREA_OPES_ORQUESTA_EXTERNAL_WORK_VALIDACION_Y_STREAM_2026-06-26.md` estaba
   modificado por OPES y debe quedar fuera del commit salvo orden expresa.

## No reabrir salvo regresion

- `WaitAgentRefs` P1.
- `CODEX-WAVE-REAL`.
- `CODEX-RECURSION-REAL`.
- `CODEX-GOAL-FIRST-APP-SERVER-REAL` con `app_server_stdio`.
- Cierre causal offline ya cubierto en matriz vigente.

