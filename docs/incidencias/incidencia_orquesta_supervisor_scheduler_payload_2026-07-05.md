# Incidencia: supervisor bloqueado por scheduler_input.payload - 2026-07-05

Actualizado: 2026-07-05T21:15:15Z

## Resumen

El supervisor autonomo remoto de Orquesta repite `tick_error` al intentar drenar la cola global. El error publico es `director_tick_input_build_invalido: field=scheduler_input.payload`.

## Evidencia observada

- `/api/status` en `127.0.0.1:19071` devuelve `last_supervisor_status=error` y `last_supervisor_stop=tick_error`.
- Contador observado: `supervisor_error_ticks=4511` a las 2026-07-05T21:14Z, subiendo cada tick.
- `last_supervisor_queue_size=10`.
- Audit `orquesta_server_audit_v0.jsonl` muestra que el run ejecutado como rank 1 es `request-ref-autoprogramming-backlog-t137-public-http-request-body-bounds-248bf945-retry-f7319198e315` y termina en `outcome=error` con el mismo `scheduler_input.payload`.
- La cola persistida contiene 10 runs `ready`, aunque la proyeccion publica de status puede mostrar solo 5 segun limite de salida.

## Codigo relacionado

- `modulos/orquesta-director-cycle/director_cycle_step_usecase_v0.go` llama a `BuildDirectorSchedulerTickInputV0` antes del runner.
- `modulos/orquesta-director-tick-input/tick_input_usecase_v0.go` normaliza y compacta el input, y traduce errores del scheduler a `scheduler_input.<field>`.
- `modulos/orquesta-director-tick-input/tick_input_payload_compaction_v0.go` solo recorta `WorkCandidates` mientras haya mas de uno.
- `modulos/orquesta-director-scheduler/scheduler_tick_validation_v0.go` valida el payload completo con limite real `maxSchedulerTickPayloadBytesV0 = 262144` bytes.

## Hipotesis

El tick real de T137 supera 256 KiB o conserva un detalle operativo rechazado como `payload` despues de la compactacion actual. Como el compactador deja de recortar al llegar a un solo `WorkCandidate`, o no compacta snapshot/outbox/claims suficientemente, el scheduler rechaza la entrada y el supervisor vuelve a intentar el mismo run.

## Impacto

- La autoprogramacion queda atascada en el primer candidato ranked.
- Otros runs listos no avanzan aunque haya cola.
- El estado publico puede dar la impresion de cola viva, pero el supervisor no progresa.

## Salida esperada

1. Reproducir el tick de T137 desde el estado real o fixture reducido.
2. Medir tamano de `DirectorSchedulerTickInputV0` por seccion antes/despues de compactar.
3. Corregir compactacion/seleccion para no rebasar el limite sin subirlo a ciegas.
4. Anadir test con caso realista: cola grande + run T137 o fixture equivalente no debe bloquear todo el supervisor.
5. Verificar `/api/status` sin crecimiento de `supervisor_error_ticks` y cola avanzando.


## Actualizacion 2026-07-05T21:25:39Z: mitigacion en tick-input

Se aplico un arreglo acotado en `modulos/orquesta-director-tick-input`: el carril `progress` compacta el snapshot y elimina historial no causal de assessments/preguntas antes de validar contra el scheduler. Esto cubre la causa observada por el subagente: snapshot de T137 con miles de `AgentWorkAssessed` y `DirectorQuestionRaised`.

Pruebas pasadas:
- `go test -count=1 ./modulos/orquesta-director-tick-input -run TestBuildDirectorSchedulerTickInputV0CompactaCarrilProgressConHistorialLargo`
- `git diff --check`

Pendiente: levantar Orquesta con auth/cuota recuperada y confirmar que `/api/status` deja de incrementar `supervisor_error_ticks`.

## Estado

Cerrada la capa `scheduler_input.payload` (2026-07-06): Claude desplego en el
servidor el binario `173b69e41c` con la mitigacion y verifico en vivo que el
error desaparecio de `/api/status`. El supervisor avanza una capa y ahora falla
por presupuesto de historial de eventos; continua en
`docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`.
