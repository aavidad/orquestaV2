# Tarea Orquesta: runs OPES quedan `running` tras ACK y sin procesos

Fecha: 2026-06-25

## Contexto

En una sesión OPES para el temario `ope-operario`, Orquesta lanzó 10 padres reales,
varios subroles y entregó artefactos útiles por tema. Al final no quedaban agentes
del trabajo vivos en el sistema operativo; solo quedaba `./orquesta-server run`.

Sin embargo, `POST /api/v0/autoprogramming/status` seguía mostrando las 10 runs
como `running` y recomendaba `supervise:queue`.

## Evidencia observada

- `pgrep -af 'run-external-work-opes-job-opes-operario|codex.*operario|orquesta-server run'`
  devolvía solo el proceso del servidor.
- Los 10 temas tenían ACK en `00_control/ACKS/`.
- Los 10 temas tenían HTML final, HTML ampliado, banco de test y auditoría.
- La cola seguía publicando las 10 runs como activas:
  `run-external-work-opes-job-opes-operario-integracion-t001-20260625...` hasta `t010`.
- El resumen de eficiencia seguía indicando:
  `state=queued`, `completion_percentage=0`, `queue_candidates=10`, `active_runs=10`.

## Impacto

El director OPES no puede distinguir de forma fiable entre:

- trabajo realmente vivo;
- trabajo terminado pero no reconciliado;
- trabajo colgado;
- cola vieja que debe cerrarse.

Esto obliga a intervención manual para revisar procesos y puede provocar bucles,
reseteos innecesarios o que el usuario crea que siguen trabajando agentes que ya
terminaron.

## Cambio requerido

Programar reconciliación causal de runs externas:

1. Si el agente padre deja ACK final y no quedan procesos hijos asociados al
   `run_ref`, Orquesta debe pasar la run a `ready`, `done`, `closed` o el estado
   canónico correspondiente.
2. El estado debe distinguir `running_process_alive` de `running_state_unreconciled`.
3. `/api/v0/autoprogramming/status` debe incluir:
   - `process_alive_count`;
   - `last_output_at`;
   - `last_artifact_at`;
   - `ack_detected`;
   - `recommended_action` real: `close_reconciled`, `wait`, `inspect_stalled` o
     `supervise`.
4. Si hay ACK y artefactos completos pero falta cierre de cola, el supervisor debe
   materializar cierre sin relanzar trabajo.
5. La métrica de eficiencia no debe reportar `completion_percentage=0` cuando hay
   ACKs y entregas completas.

## Criterio de aceptación

Repetir una tanda OPES pequeña con 2-3 runs:

- al terminar los procesos, `status` no debe mostrar esas runs como `running`;
- si hay ACK pero falta algún artefacto, debe quedar `needs_rework` o equivalente;
- si no hay proceso ni ACK, debe quedar `stalled_or_failed`;
- la cola no debe recomendar `supervise:queue` para runs ya cerrables.

## Avance Orquesta 2026-06-28

Estado: mitigada parcialmente; no cerrada al 100%.

Cambios verificados:

- `autoprogramming/status` ya distingue `running_live`,
  `running_without_recent_stats` y `running_stale_no_process` usando stats de
  director y `process.status`, de modo que una run sin proceso vivo verificable
  no se presenta como trabajo vivo opaco.
- `TestMCPAutoprogrammingStatusExecutorV0ProcesoParadoConAckCleanupEsperaACKV0`
  añade la regresión del caso intermedio: run `running`, proceso `stopped` y
  progreso en `ack_registered_cleanup`. La acción pública pasa a
  `wait_for_ack_before_reconcile`, no a relanzar ni a reconciliar de forma
  agresiva.
- `TestClassifyRunLivenessV0AckPendienteNoEsSeguroReconciliar` mantiene la
  regla neutral de liveness: si hay ACK/cleanup pendiente, el stale no es seguro
  para reconciliación automática.
- La familia `running_stale_no_process` ya tiene reconciliación de cola en el
  stack cuando no hay ACK/cleanup pendiente y no hay proceso vivo verificable.

Pendiente real:

- Agregar a la superficie pública de run campos agregados del estilo
  `ack_detected`, `last_output_at` y `last_artifact_at` cuando existan datos
  causales suficientes, sin inventarlos desde strings ni rutas OPES.
- Probar una tanda OPES temporal pequeña con ACKs reales ya escritos y procesos
  terminados, verificando que `autoprogramming/status` no queda en
  `supervise:queue` y que el cierre causal se materializa sin relanzar trabajo.
- Ajustar la métrica de eficiencia para el caso exacto "ACKs y entregas
  completas pero cola vieja" sólo cuando el run tenga evidencia causal de
  completitud, no por presencia suelta de ACK.
