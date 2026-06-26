# Handoff Orquesta goal-first - parada 2026-06-26

Motivo de parada: el operador pide parar para reservar cuota mientras OPES
trabaja en otra sesion.

## Estado del ultimo tramo

- Rama: `trabajo/plataforma-agentes`.
- Ultimo commit ya publicado antes del corte final de parada: `213eba6d Documenta parada y salud web goal-first`.
- Cambio publicado en el corte final de `/nueva-app`:
  - `/nueva-app` mantiene los tooltips visuales y anade ayuda accesible por
    `aria-describedby` para cada nodo con `data-help`.
  - Las pestanas del wizard exponen `tablist`/`tab`/`tabpanel`,
    `aria-selected` vivo y navegacion con respeto a `prefers-reduced-motion`.
  - Los textos expertos de datos, almacenamiento e integraciones pasan a i18n
    ES/EN; la ayuda de tipo de integracion ya enumera todas las opciones
    visibles.
  - `modulos/orquesta-web/docs/guia_nueva_app_opciones_2026-06-25.md` queda
    alineada con el bloque experto ya visible en HTML.

## Verificacion ejecutada

- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp`
- `go test -count=1 ./modulos/orquesta-web ./modulos/orquesta-mcp ./modulos/orquesta-goal ./modulos/orquesta-runtime-codex-goal ./modulos/orquesta-app-director-service ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server`
- `go test -count=1 ./cmd/orquesta-server -run TestRuntimeResidentDirectorConduceDirectorCycleStepTrasRestartV0 -v`
- `git diff --check`
- `go test -count=1 -p=1 ./...`
- `go test -count=1 ./modulos/orquesta-web`
- `git diff --check`

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
- Web/operador: alrededor de 85%. El wizard de `/nueva-app` existe, con ayuda
  visible, ayuda accesible por `aria-describedby`, pestanas ARIA, i18n de
  textos expertos y guia actualizada. Queda QA manual en navegador, capturas
  responsive y completar seniales humanas si aparecen mas opciones ambiguas.
- `autoprogramming/status` y `supervise`: alrededor de 80-85%. HTTP colgado
  esta mitigado con `202 accepted_background`; `running_stale` con procesos
  vivos ya consulta stats/procesos y expone `agents_live`; los runs
  external-work con agentes pedidos pero no arrancados publican
  `external_work_agent_requested_not_started`. Falta test de frontera
  gateway/stack completo y mas seniales humanas para casos OPES finos.
- Orquesta nucleo goal-first: alrededor de 82-85%.
- Orquesta incluyendo OPES temporal completo hasta derivados/HTML: alrededor de
  70-75%.

## Pendientes concretos siguientes

1. Reiniciar solo el servidor web local `127.0.0.1:8787` con el binario nuevo
   cuando se retome; no tocar servidores OPES.
2. Mejorar `/nueva-app` en el siguiente corte:
   - validar en navegador real que tooltips, foco, tabs y scroll funcionan bien
     en desktop/movil;
   - revisar si hacen falta mas filas o campos expertos antes de congelar el
     contrato UI;
   - completar documentacion larga si se anaden nuevas opciones;
   - mantener cualquier opcion de proveedor/DB como adaptador opt-in, no como
     decision de la web.
3. OPES sigue en otra sesion: no pisar procesos ni documentos que escriba ese
   agente. El fichero
   `TAREA_OPES_ORQUESTA_EXTERNAL_WORK_VALIDACION_Y_STREAM_2026-06-26.md` estaba
   modificado por OPES y debe quedar fuera del commit salvo orden expresa.

## Cierre local posterior

- 2026-06-26: anadido test HTTP de frontera para
  `/api/v0/autoprogramming/status` con cola `running`, process refs vivos y
  salida `queue_health.running_live=1`, `queue_health.agents_live=1`,
  `stale_running=[]`. Esto cubre el hueco gateway/adapter sin tocar OPES ni
  arrancar servidores.
- 2026-06-26: anadido contrato de servicio para backend Goal configurado pero
  degradado: si `GoalLauncher` falla, `StartAppDirectorV0` propaga el error y
  no ejecuta el loop legacy ni genera eventos `AgentRequested`/`AgentStarted`.
- 2026-06-26: `orquesta-mcp` demota explicitamente la supervision legacy frente
  a goal-first en descriptores/README: `prepare_run` prefiere `goal`/`goal_specs`,
  `supervise` queda como compatibilidad legacy/resident y las runs goal-first se
  observan por `orquesta.autoprogramming.observe_goal.v0`.
- 2026-06-26: `director.stats` diagnostica
  `external_work_agent_requested_not_started` si hay agentes solicitados,
  ninguno arrancado y nada en vuelo; `autoprogramming/status` propaga ese issue
  como diagnostico publico accionable.
- 2026-06-26: `orquesta-runtime-codex-delivery` clasifica
  `Failed to create stream fd` como `evidence-ref-warning-stream-fd` advisory:
  no lo convierte en capacidad/autenticacion ni tapa `no_ack` cuando falta ACK.
- 2026-06-26: `orquesta-runtime-codex` acepta
  `agent_shutdown_checkpoint_ack.json` minimo con `schema_version` y
  `status=checkpoint_ready`, hidratando refs ausentes desde la request
  correlada; refs explicitas incorrectas siguen bloqueando por causalidad.
- 2026-06-26: `orquesta-orchestration-core` compacta observaciones de review
  demasiado verbosas antes de construir `RecordReviewResult`, conservando refs
  causales y `evidence-ref-review-gate-payload-compacted` para evitar
  `payload_invalido` tras ACK valido.
- 2026-06-26: `orquesta-app-codex-stack` distingue un `runtime_error` de
  supervisor cuando el ultimo snapshot conserva agente/proceso vivo: publica
  `supervisor_transition_error_but_agents_live` y `next_actions` de
  esperar/reintentar, sin relanzar la misma `run_ref` mientras el proceso siga
  vivo.
- 2026-06-26: `orquesta-app-codex-stack` anade diagnostico de frontera para
  `runs/supervise` en external-work/OPES con agente pedido pero no arrancado:
  `external_work_agent_requested_not_started`, causa publica y acciones de
  reintento/revision de capacidad, auth, runtime, cola, outbox o politica.

## No reabrir salvo regresion

- `WaitAgentRefs` P1.
- `CODEX-WAVE-REAL`.
- `CODEX-RECURSION-REAL`.
- `CODEX-GOAL-FIRST-APP-SERVER-REAL` con `app_server_stdio`.
- Cierre causal offline ya cubierto en matriz vigente.
