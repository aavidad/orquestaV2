# Contratos

## `ShutdownServerV0`

Entrada: `ServerShutdownCommandV0`.

Puertos:

- `RunQueueReaderPortV0` para descubrir runs candidatos;
- `RunControlReaderPortV0` para no reabrir terminales;
- `RunControlWriterPortV0` para solicitar `stop`;
- `RunControlCheckpointWriterPortV0` para registrar ACK durable de checkpoint;
- `PrepareAgentShutdownPortV0` para preparar checkpoint antes de stop no
  forzado;
- `RunSupervisorPortV0` para drenar;
- `RunStatsReaderPortV0` para verificar `agents_in_flight`.

Invariantes:

- no toca stores ni procesos directamente;
- solo acepta shutdown solicitado por el Director (`requested_by` con autoridad
  de director); los agentes no pueden iniciar apagado, solo responder con
  checkpoint/ACK por puerto;
- `forced=false` prepara checkpoint por puerto y registra ACK durable antes de
  solicitar `StopRunV0`; si falta ACK, no pide stop ni ejecuta drainer;
- `forced=true` permite drenar agentes sin checkpoint previo, pero no puede
  declarar `shutdown_ready` si el lector de trabajo activo informa un backend
  Goal vivo (`backend_still_running`);
- el resultado expone `goal_actions` tipadas para el handoff operativo de
  goal-first/backend (`observe_active_goal`, `wait_checkpoint`,
  `stop_requested_wait`, `forced_stop_requested`, `cleanup_required`,
  `cleanup_requested`, `cleanup_attempted_wait`, `cleanup_completed`) con refs
  y evidencias compactas;
- `cleanup_goal_backends=true` habilita una limpieza gobernada de backends Goal
  propios antes de decidir `shutdown_ready`: solo se ejecuta si todo el trabajo
  activo observado son backends Goal residuales, nunca si queda un `goal_first`
  activo, y despues siempre relee `ActiveShutdownWorkReaderPortV0`;
- `checkpoint_deadline_at` permite declarar que la espera cooperativa ya vencio;
  si llega vencido junto a `occurred_at`, el caso de uso pide `StopRunV0`
  forzado y devuelve evidencia de deadline, sin ocultar que faltaba checkpoint;
- si falta checkpoint, el resultado expone `pending_checkpoint_agent_refs`,
  `checkpoint_evidence_refs` y `checkpoint_agents_pending` con refs compactas,
  sin rutas ni detalles de runtime;
- si el deadline de checkpoint vence, el resultado expone
  `checkpoint_deadline_expired`, `forced_after_checkpoint_deadline` y
  `checkpoint_deadlines_expired`;
- `shutdown_ready=true` solo cuando todas las runs objetivo estan terminales o
  sin agentes en vuelo segun stats compactas y sin checkpoint pendiente.
- La composicion servidor debe congelar el supervisor residente al aceptar
  `POST /api/v0/server/shutdown`. Mientras `shutdown_in_progress` siga activo,
  el supervisor no puede reactivar runs paradas por shutdown ni lanzar nuevos
  agentes/automejora. Si la peticion se rechaza por autorizacion o contrato, la
  congelacion se libera.
