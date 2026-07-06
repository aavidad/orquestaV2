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

## Causa

`modulos/orquesta-orchestration-core/run_event_reader_v0.go` impone
`runEventsFullReadMaxV0 = 10000` eventos al cargar historial completo de un run
(`loadRunEventsWithBudgetV0` / `loadRunEventsWithPagesV0`). El run de T137
acumulo mas de 10000 eventos (miles de `AgentWorkAssessed` y
`DirectorQuestionRaised`, ver incidencia del scheduler payload). El camino del
supervisor pide historial completo, el guard corta con error y el tick entero
falla, dejando la cola congelada en el mismo candidato.

Es la misma clase que `scheduler_input.payload`: el supervisor consume historia
completa no causal donde deberia consumir una proyeccion acotada. El guard de
presupuesto es correcto; lo incorrecto es que el camino del supervisor necesite
historial completo.

## Estado operativo decidido (2026-07-06)

- El operador confirma que el servidor remoto no tiene cuota de proveedor
  Codex (ademas del 401 de auth ya documentado). Sin agentes posibles, no se
  desatasca la cola: el atasco actua de congelador y no se pierde estado.
- Se detiene el servidor remoto limpiamente para no quemar ticks en error;
  el trabajo continua en local hasta recuperar cuota/auth
  (orden del operador: "hay que trabajar en local hasta entonces").

## Salida esperada (fix de codigo, para Codex)

1. En el camino del supervisor/tick, sustituir la carga de historial completo
   por proyeccion causal acotada (mismo patron que
   `compactTickInputProgressLaneV0`): ultimo estado por agente/tarea mas refs
   causales del candidato, con limite duro.
2. Si un run supera el presupuesto igualmente, degradar a `run_oversized`
   parking (apartar el run con evidencia y seguir con el resto de la cola) en
   vez de fallar el tick entero: un run envenenado no debe congelar el
   supervisor global.
3. Test con fixture realista: run con >10000 eventos en cola no bloquea el
   tick; queda apartado con reason code y el resto de candidatos avanza.
4. Verificacion viva: `supervisor_error_ticks` estable y cola drenando tras
   redeploy.

## Refs

- Incidencia previa (capa 1, cerrada en vivo):
  `docs/incidencias/incidencia_orquesta_supervisor_scheduler_payload_2026-07-05.md`
- Guard: `modulos/orquesta-orchestration-core/run_event_reader_v0.go`
- Inventario: pendiente de alta como BUG de plataforma clase supervisor.
