# Incidencia: supervisor bloqueado por events_full_history_budget_exceeded - 2026-07-06

Actualizado: 2026-07-06 por Claude (director).

## Resumen

Tras desplegar en remoto el binario `173b69e41c` (que incluye la mitigacion de
compactacion del carril progress en `orquesta-director-tick-input`), el error
`director_tick_input_build_invalido: field=scheduler_input.payload` desaparecio.
El supervisor avanza una capa mas y ahora falla cada tick con:

`nucleo_orquestacion_store: events.budget: events_full_history_budget_exceeded`

## Evidencia observada

- Servidor remoto `127.0.0.1:19071`, binario sha256 `332e6d35...`, arrancado
  2026-07-06T06:54:28Z desde `trabajo/plataforma-agentes` = `173b69e41c`.
- Muestreo `/api/status` 3 tomas en 90s: `supervisor_error_ticks` 4605 -> 4614
  -> 4623 (aprox. +1 cada 5s), `last_supervisor_status=error`,
  `last_supervisor_stop_public=error_tick`.
- El error viejo `scheduler_input.payload` ya no aparece: la mitigacion del
  2026-07-05 esta verificada en vivo.
- Candidato rank 1 sigue siendo el run de T137
  (`request-ref-autoprogramming-backlog-t137-public-http-request-body-bounds-248bf945-retry-f7319198e315`),
  cuyo historial de eventos supera el presupuesto.

## Causa (diagnostico verificado 2026-07-06 con datos reales)

El run de T137 tiene exactamente **2669 eventos** (verificado en el servidor:
`event_records/18410563...` = 2669 ficheros, `event_indexes/18410563....json` =
2669 refs). Composicion: 1332 `AgentWorkAssessed` + 1332
`DirectorQuestionRaised` (una pareja por tick del director mientras el run
estaba atascado) + 5 eventos normales de arranque. Un run sano ronda 20-30
eventos (~22 KB); este ocupa 5 MB.

O sea: NO supera el presupuesto teorico (20000 en store, 10000 en core). El
error viene de un desajuste de limites en
`modulos/orquesta-state-file/event_sink_v0.go`:

- `LoadRunEventsV0` pide UNA pagina con `Limit: store.events.MaxRunEvents`
  (=20000).
- `LoadRunEventsPageV0` recorta con `boundedEventPageLimitV0`, cuyo tope es
  `maxEventLogPageLimitV0 = 1000` (`event_log_budget_v0.go`).
- Con 2669 registros, `end < len(records)` -> `HasMore=true` ->
  `LoadRunEventsV0` devuelve `events_full_history_budget_exceeded` (linea 81,
  codigo `ErrNucleoOrquestacionStoreV0`, que es exactamente el prefijo
  observado `nucleo_orquestacion_store:`).

Conclusion: **cualquier run con mas de 1000 eventos rompe toda carga de
historial completo via `LoadRunEventsV0`**, aunque este muy por debajo del
presupuesto. `orquesta-orchestration-core/run_event_reader_v0.go` sabe paginar
(`loadRunEventsWithPagesV0`), pero el camino del supervisor llega por un lector
que no expone `RunEventPagedReaderPortV0`, cae en `LoadRunEventsV0` y muere.

Hay ademas un segundo defecto que es el que inflo el historial: con el run
atascado, cada ciclo del director anadio una pareja `AgentWorkAssessed` +
`DirectorQuestionRaised` identica a la anterior (1332 veces). Re-evaluar sin
cambio causal no deberia emitir eventos nuevos ilimitadamente.

## Estado operativo decidido (2026-07-06)

- El operador confirma que el servidor remoto no tiene cuota de proveedor
  Codex (ademas del 401 de auth ya documentado). Sin agentes posibles, no se
  desatasca la cola: el atasco actua de congelador y no se pierde estado.
- Se detiene el servidor remoto limpiamente para no quemar ticks en error;
  el trabajo continua en local hasta recuperar cuota/auth
  (orden del operador: "hay que trabajar en local hasta entonces").

## Salida esperada (fix de codigo, para Codex)

1. Fix inmediato del lector: `LoadRunEventsV0` en
   `orquesta-state-file/event_sink_v0.go` debe iterar paginas internamente
   hasta `MaxRunEvents` (o el camino del supervisor debe usar el lector
   paginado `RunEventPagedReaderPortV0` que orchestration-core ya prefiere).
   Un limite de pagina (1000) no puede actuar de presupuesto total.
2. Fix del inflado: no emitir `AgentWorkAssessed`/`DirectorQuestionRaised`
   duplicados en cada tick cuando no hay cambio causal (dedupe por clave
   estable o rate-limit); un run atascado no debe crecer sin limite.
3. Resiliencia: si aun asi un run supera el presupuesto real, apartarlo
   (`run_oversized` parking con evidencia) y seguir con el resto de la cola;
   un run envenenado no debe congelar el supervisor global.
4. Tests: (a) run con 1001-3000 eventos se carga entero sin error; (b) tick
   del director sobre run atascado no anade pareja assess/question duplicada;
   (c) run por encima del presupuesto real queda apartado con reason code y
   los demas candidatos avanzan.
5. Verificacion viva tras redeploy: `supervisor_error_ticks` estable y cola
   drenando.

## Refs

- Incidencia previa (capa 1, cerrada en vivo):
  `docs/incidencias/incidencia_orquesta_supervisor_scheduler_payload_2026-07-05.md`
- Guard: `modulos/orquesta-orchestration-core/run_event_reader_v0.go`
- Inventario: pendiente de alta como BUG de plataforma clase supervisor.
