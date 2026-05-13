# Handoff sesion 2026-05-13: nucleo, stats y parada

Estado: listo para retomar desde otro equipo tras commit de avances.

## Resumen

- Se probo Orquesta con agentes Codex reales desde REST.
- Se corrigio la frontera de estadisticas para que la web/director vean progreso vivo.
- Se corrigio el falso `no_signal` cuando ya existe assessment compacto.
- Se corrigio el coordinador global para drenar runs `stop_requested` forzados.
- Se documentaron las pruebas reales y el hueco pendiente de continuidad hasta delivery.
- Se agregaron pruebas deterministas de stack para cerrar dos huecos de regresion:
  `DeliveryRegistered` desde ACK de workers de programacion y stop forzado por
  API con stats finales coherentes.
- Se ejecuto una prueba real acotada posterior que llego a
  `worker real -> agent_ack.json -> DeliveryRegistered`.
- Se corrigio la perdida de `agent_ref` en entregas persistiendo
  `delivered_agents`.
- Se documento el procedimiento actual de apagado controlado en
  `docs/runbooks/apagado_controlado_servidor_y_agentes.md`.

## Cambios clave

- `orquesta-runtime-codex-delivery`:
  - `FileCodexProgressStateStoreV0` conserva `SampleAccepted` y tiempos reales.
  - `CodexProgressObservationSourceV0` puede emitir progreso normal para stats sin meter ruido en decision.
  - Snapshot perdido se trata como proceso parado observado, no como confirmacion falsa de stop.
- `orquesta-app-codex-stack`:
  - `/api/v0/director/stats` usa `statsProgressSourceV0`.
  - El ciclo de decision mantiene `progressSourceV0` silencioso salvo decision requerida.
- `orquesta-orchestration-core`:
  - Stats proyecta progreso desde `AgentWorkAssessed`.
  - `no_signal_agent_refs` excluye agentes parados, reflejados, stop_requested y assessed.
  - Se conserva progreso compacto para web/director aunque no haya observacion runtime viva.
  - Una entrega real tiene prioridad sobre progreso/assessment obsoleto:
    `delivered_tasks` marca la tarea como `delivered` y `delivered_agents`
    marca el agente como `completed`.
- `orquesta-core-workflow`:
  - `DeliveryRegistered` proyecta `delivery_ref`, `task_id` y `agent_ref` en
    `deliveries`, `delivered_tasks` y `delivered_agents`.
- `orquesta-run-coordinator`:
  - `stop_requested forced=true` ya no se salta por `DispatchAllowed=false`.
  - El coordinador llama al drainer para permitir `StopRuntimeAgent` y confirmaciones.
- Documentacion:
  - `docs/runbooks/self_observability_real_test_2026-05-13.md`
  - `docs/runbooks/stop_runtime_reinicio_process_ref.md`
  - `docs/runbooks/smoke_servidor_rest_director.md`
  - `docs/decision_estadisticas_y_frontera_programacion_2026-05-12.md`

## Prueba real ejecutada

Servidor e2e aislado:

- Puerto: `127.0.0.1:18794`
- Proyecto: `/home/alberto/Trabajo/orquesta-e2e-doc-20260513/project`
- Run: `run-spec-mini-agenda-documental-req-mini-agenda-documental-4ed9fd91`

Resultado observado:

- `POST /api/v0/apps/director` arranco director real.
- Stats vio `observed_agents=1`, `progressing_agents=1`, modelo `gpt-5.5`, effort `xhigh`.
- El director escribio `docs/arquitectura.md` y `docs/plan_microtareas.md` en el proyecto e2e.
- Orquesta detecto estancamiento: `agent_assessments=1`, `director_questions=1`, `needs_attention=true`.
- Tras `POST /api/v0/runs/control action=stop forced=true`, Orquesta dejo:
  - `agents_stop_requested=3`
  - `agents_stop_confirmed=3`
  - `agents_in_flight=0`

No quedaron procesos Orquesta/Codex vivos de la prueba.

## Prueba real posterior con entrega

Servidor e2e aislado:

- Proyecto: `/home/alberto/Trabajo/orquesta-e2e-real-20260512T225636Z/project`
- Run: `run-spec-notas-mini-api-web-req-notas-mini-api-web-2ea80660`
- Modelo: `gpt-5.5`, reasoning `xhigh`

Resultado observado:

- El director real genero `docs/arquitectura.md`,
  `docs/plan_microtareas.md`, `docs/api.md`, `docs/web.md` y
  `docs/persistencia.md`.
- Orquesta abrio `programacion` y creo 6 microtareas.
- Orquesta arranco worker real para `task-programacion-bootstrap-go`.
- El worker entrego `go.mod`, `cmd/server/main.go` e
  `internal/app/bootstrap.go`.
- El worker dejo `agent_ack.json` valido y Orquesta registro
  `DeliveryRegistered`.
- La prueba se corto de forma controlada tras la primera entrega real, por lo
  que valida la frontera minima de entrega pero no una app completa.

Hallazgo corregido:

- Stats mantenia progreso/assessment obsoleto despues de la entrega.
- Causa raiz: `DeliveryRegistered` no proyectaba `agent_ref` en el estado del
  run.
- Solucion: `OrchestrationRunV0.DeliveredAgents` y stats basadas en esa
  proyeccion, sin depender de convenciones de nombres de ACK.

## Subagentes

- Cicero: confirmo que `decision -> microtarea -> worker` ya existe. El hueco real es continuidad hasta `DeliveryRegistered` y ACK valido de workers.
- Dewey: reviso RunControl. Propuso test exacto API stop forced + drain + stats y detecto el hueco del coordinador que saltaba `stop_requested`.
- Curie: documento la prueba real y la decision de frontera stats/progreso.

## Pendiente principal

Orquesta ya arranca director y workers, detecta atasco, registra una primera
entrega real y tiene prueba determinista de continuidad `decision -> microtarea
-> worker ACK -> DeliveryRegistered`. Falta validar esa misma continuidad con
agentes Codex reales hasta completar una app sin cortar el run tras el primer
ACK:

- todos los workers deben escribir ACK valido;
- el supervisor debe seguir drenando sin intervencion hasta review/cierre;
- documentacion ejecutable hoy debe ir como microtarea de `programacion` sobre `docs/...` o crear un provider hexagonal especifico para fase `documentacion`;
- no se deben inventar deliveries si el run se para antes del ACK.

Avance posterior: se implemento la primera version de shutdown de servidor:

- nuevo modulo `modulos/orquesta-server-shutdown`;
- nuevo tool `orquesta.server.shutdown.v0`;
- nueva ruta `POST /api/v0/server/shutdown`;
- `orquesta-server stop` llama al endpoint antes de enviar la senal final;
- el cierre disponible coordina RunQueue, RunControl, supervisor y stats por
  puertos hexagonales;
- si `shutdown_ready=false`, CLI no corta el servidor.

Sigue pendiente el protocolo graceful completo con checkpoint:

- `prepare_shutdown`/ACK durable por agente;
- deadline controlado antes de forzar stop;
- stats de razon de cierre por agente.

## Validacion local

Comandos ejecutados:

```bash
git diff --check
go test -count=1 ./modulos/orquesta-run-coordinator ./modulos/orquesta-run-supervisor ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime-codex-delivery ./cmd/orquesta-server
```

Tambien se ejecutaron pruebas focales previas en `orquesta-run-control`, `orquesta-run-memory`, `orquesta-run-file`, `orquesta-mcp`, `orquesta-app-gateway`, `orquesta-runtime` y `orquesta-orchestration-core`.

Validacion posterior al retomar:

```bash
go test ./modulos/orquesta-app-codex-stack -run 'TestDrainRunV0RegistraEntregasDeProgramacionTrasDecisionDirector|TestCodexStackV0StopForzadoPorAPIDrenaAgentesYActualizaStats' -count=1 -v
go test ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./modulos/orquesta-run-coordinator ./modulos/orquesta-mcp ./modulos/orquesta-web -count=1
go test ./... -count=1
go test ./cmd/orquesta-server ./modulos/orquesta-server-shutdown ./modulos/orquesta-mcp ./modulos/orquesta-http-gateway ./modulos/orquesta-app-gateway ./modulos/orquesta-app-codex-stack -count=1
```
