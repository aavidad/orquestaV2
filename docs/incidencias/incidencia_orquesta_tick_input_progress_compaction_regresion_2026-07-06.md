# Incidencia: regresion de compactTickInputSnapshotForProgressV0 en supervision progresiva - 2026-07-06

Detectada por Claude (director) al revisar el fix del events budget. Bisecada
y verificada determinista (5 ejecuciones por commit).

## Sintoma

`TestExternalProcessAgentBatchExecutorV0StopsOnlyLoopingProcess`
(`modulos/orquesta-orchestration-core/external_process_agent_batch_supervision_test.go`)
falla en `173b69e41c` (y en `28c562bccf`) y pasa en `885e76b021`:

- Esperado: `ProgressiveLoopStatusWaitExternalV0` (queda un
  `StopRuntimeAgent` pendiente en outbox tras parar al agente en bucle).
- Obtenido: `quiescent` con `FinalAction:stop_quiescent`,
  `FirstPendingCount:1`, `PendingOutboxCount:0`.

## Causa

La mitigacion del scheduler payload (checkpoint `28c562bccf`) anadio
`compactTickInputSnapshotForProgressV0`
(`modulos/orquesta-director-tick-input/tick_input_active_lane_snapshot_v0.go`),
que ademas de filtrar assessments/preguntas por refs causales ANULA por
completo `CapacityRequests`, `CapacityDecisions`, `ConcurrencyGates`,
`PhaseArtifacts`, `Reviews`, `ReviewResults`, `AcceptedReviews`,
`ReworkRequests`, `ExpiredLeaseRefs`, `ReplanRefs` y
`BlockingQualityGateRefs` del snapshot del carril progress.
`orquesta-orchestration-core` consume ese carril en la supervision progresiva
(`RunProgressiveLoopV0`) y, sin esos campos, el runner concluye `quiescent`
en vez de esperar el dispatch externo pendiente.

La mitigacion FUNCIONA para su objetivo (el error
`director_tick_input_build_invalido: field=scheduler_input.payload`
desaparecio en vivo, verificado 2026-07-06 en el servidor), pero recorta
estado causal necesario: sobre-compactacion.

## Nota operativa

El binario desplegado en remoto (`173b69e41c`) incluye esta regresion. El
servidor esta parado (sin cuota/auth), asi que no hay impacto vivo ahora
mismo; NO redeployar hasta corregir esto junto con el fix del events budget.

## Salida esperada (para Codex, mismo frente que events budget)

1. Reducir la anulacion: conservar en el snapshot compactado los campos que
   la supervision progresiva necesita para decidir wait_external
   (como minimo gates/decisiones/pendientes asociados a los agentes y tareas
   del candidato, mismo criterio de filtrado causal que assessments), en vez
   de anular familias enteras.
2. `TestExternalProcessAgentBatchExecutorV0StopsOnlyLoopingProcess` vuelve a
   verde SIN perder el test
   `TestBuildDirectorSchedulerTickInputV0CompactaCarrilProgressConHistorialLargo`.
3. Anadir `./modulos/orquesta-orchestration-core` a la bateria requerida de
   cualquier goal que toque `orquesta-director-tick-input` (leccion: la
   validacion del 2026-07-06 en el servidor no incluia este paquete y la
   regresion se colo; el nightly completo la habria cazado esta noche).

## Refs

- Commit que introduce la regresion: `28c562bccf` (mitigacion payload).
- Fix relacionado en curso: `docs/incidencias/incidencia_orquesta_supervisor_events_budget_2026-07-06.md`.
- Bisect: 885e76b021 ok (5/5), 28c562bccf FAIL, 173b69e41c FAIL (5/5).
