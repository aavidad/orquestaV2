# Handoff sesion 2026-05-13: nucleo, stats y parada

Estado: listo para retomar desde otro equipo tras commit de avances.

## Resumen

- Se probo Orquesta con agentes Codex reales desde REST.
- Se corrigio la frontera de estadisticas para que la web/director vean progreso vivo.
- Se corrigio el falso `no_signal` cuando ya existe assessment compacto.
- Se corrigio el coordinador global para drenar runs `stop_requested` forzados.
- Se documentaron las pruebas reales y el hueco pendiente de continuidad hasta delivery.
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

## Subagentes

- Cicero: confirmo que `decision -> microtarea -> worker` ya existe. El hueco real es continuidad hasta `DeliveryRegistered` y ACK valido de workers.
- Dewey: reviso RunControl. Propuso test exacto API stop forced + drain + stats y detecto el hueco del coordinador que saltaba `stop_requested`.
- Curie: documento la prueba real y la decision de frontera stats/progreso.

## Pendiente principal

Orquesta ya arranca director y workers, y detecta atasco. Falta cerrar la continuidad productiva hasta `DeliveryRegistered`:

- workers deben escribir ACK valido;
- el supervisor debe seguir drenando sin intervencion;
- documentacion ejecutable hoy debe ir como microtarea de `programacion` sobre `docs/...` o crear un provider hexagonal especifico para fase `documentacion`;
- no se deben inventar deliveries si el run se para antes del ACK.

Tambien queda pendiente el shutdown graceful completo:

- hoy `orquesta-server stop` para el servidor, pero no solicita checkpoint a
  cada agente antes de cortar;
- el cierre seguro actual requiere parar cada run por RunControl y esperar
  `agents_in_flight=0`;
- falta implementar `orquesta.server.shutdown.v0` con quiesce, checkpoint,
  stop, confirmacion y cierre final.

## Validacion local

Comandos ejecutados:

```bash
git diff --check
go test -count=1 ./modulos/orquesta-run-coordinator ./modulos/orquesta-run-supervisor ./modulos/orquesta-app-codex-stack ./modulos/orquesta-orchestration-core ./modulos/orquesta-runtime-codex-delivery ./cmd/orquesta-server
```

Tambien se ejecutaron pruebas focales previas en `orquesta-run-control`, `orquesta-run-memory`, `orquesta-run-file`, `orquesta-mcp`, `orquesta-app-gateway`, `orquesta-runtime` y `orquesta-orchestration-core`.
