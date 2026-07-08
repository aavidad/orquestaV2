# Incidencia: supervisor bloqueado por scheduler_input.payload - 2026-07-05

Actualizado: 2026-07-08T16:19:30+02:00

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

## Cierre 2026-07-06

Cerrada la capa `scheduler_input.payload` (2026-07-06): Claude desplego en el
servidor el binario `173b69e41c` con la mitigacion y verifico en vivo que el
error desaparecio de `/api/status`. El supervisor avanza una capa y ahora falla
por presupuesto de historial de eventos; continua en
`docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`.

## Actualizacion 2026-07-08: regresion en snapshot sin carril activo

Reabierta por evidencia remota en el servidor activo de Orquesta:

- Proceso vivo: `/srv/orquesta-self/worktrees/pilot-remoto-1`, commit
  `c68929696c550756f1c09fa4588e0206da385eee`, binario con hash
  `f784628154cd404cac6ee12b81faa20a5d103a45ca38093e9f065c5b000a8cbd`.
- State dir real del proceso:
  `/srv/orquesta-self/claude-director-20260705/state`.
- `/api/status` volvio a publicar `last_supervisor_status=error`,
  `last_supervisor_stop_public=error_tick` y
  `director_tick_input_build_invalido: field=scheduler_input.payload`.
- El run T137
  `request-ref-autoprogramming-backlog-t137-public-http-request-body-bounds-248bf945-retry-f7319198e315`
  esta en `programacion`, pesa unos 2.34 MB y acumula 1333
  `agent_assessments` y 1333 `director_questions`.
- El ledger de outbox asociado tiene 1334 registros despachados; no es un
  atasco por outbox pendiente.

Causa nueva: la mitigacion anterior compactaba carriles activos (`progress`,
`review`, `delivery`, etc.), pero el builder podia quedar sin carril activo y
seguir enviando al scheduler un snapshot historico demasiado grande. El
scheduler fallaba antes de poder devolver `waiting` o aparcar el run.

Fix local aplicado el 2026-07-08:

- `modulos/orquesta-director-tick-input/tick_input_payload_compaction_v0.go`
  aplica una compactacion final por presion de payload si, tras recortar
  `work_candidates`, el input sigue superando 256 KiB.
- La compactacion conserva refs causales de candidatos y agentes en vuelo, y
  recorta historico no causal de assessments/preguntas/listas de snapshot a una
  cola acotada.
- Test nuevo:
  `TestBuildDirectorSchedulerTickInputV0CompactaSnapshotSobredimensionadoSinCarrilActivo`.

## Estado actual

2026-07-08T16:19:30+02:00: cerrado y verificado en remoto.

Evidencia de cierre:

- Commit desplegado: `d650d30e977380fc7c69771c83fc1d04cd023de8`
  (`fix: compacta snapshot de tick sobredimensionado`), push realizado a
  `origin/trabajo/plataforma-agentes`.
- Pruebas pasadas antes de desplegar:
  `go test -count=1 ./modulos/orquesta-director-tick-input
  ./modulos/orquesta-director-scheduler ./modulos/orquesta-director-cycle`,
  `go test -count=1 ./...` y build remoto de `./cmd/orquesta-server`.
- Binario remoto instalado:
  `/srv/orquesta-self/runtime/orquesta-server-claude`, hash
  `3a402a9dc1ac78db19d1c3641b70c5f280b6d7675b658576521079d9afd7386e`.
- El arranque remoto requirio limpieza logica explicita de cola
  (`ORQUESTA_STARTUP_CLEANUP_MODE=forced_stop`) porque no habia procesos vivos
  y el startup check detecto `runs transitorios activos=0
  cola_desincronizada=1`.
- `/api/status` remoto tras el arranque publica `status=running`,
  `availability_status=running`, `startup_status=startup_ready`,
  `last_supervisor_status=ok`, `last_supervisor_error=null` y el mismo hash de
  binario `3a402a9dc1ac...`.
- `supervisor_error_ticks` quedo en `19682` durante cuatro muestras separadas
  por 3 segundos; se interpreta como contador acumulado historico, no como error
  activo, porque `last_supervisor_error=null` y no crece.
