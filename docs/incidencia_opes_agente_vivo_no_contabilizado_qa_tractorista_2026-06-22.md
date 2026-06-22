# Incidencia OPES Agente Vivo No Contabilizado QA Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-qa-texto-tests-20260622`.

Objetivo: QA textual y de tests del curso `operario-tractorista-grupo-5`.

## Sintomas

1. La API `/api/v0/autoprogramming/status` informo:
   `agents_total=6`, `agents_in_flight=6`, `tasks_open=10`.
2. El runtime contenia 7 directorios de agente:
   `g01`, `g02`, `g03`, `g05`, `g07`, `g09`, `g10`.
3. El proceso Codex de `g05` estaba vivo y escribiendo, pero la API mantenia
   `task-autoprogramming-86860ef951b1-g05` como `pending`.
4. `g05` no aparecia en la lista de agentes de la respuesta de estado aunque
   existia proceso real:
   `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-qa-texto-tests-20260622/agent-ref-task-autoprogramming-86860ef951b1-g05`.

## Impacto

- El operador puede creer que una tarea esta pendiente cuando ya hay un agente
  real trabajando.
- La contabilidad de concurrencia puede superar el limite declarado sin reflejo
  en `status`.
- Un `stop` o una reconciliacion podria no alcanzar al proceso no registrado.

## Tareas Tecnicas

- `PROC-TASK-001`: al crear runtime de agente, el registro de control debe
  persistirse antes o atomicamente con el arranque del proceso; si falla, no se
  debe dejar un proceso vivo no contabilizado.
- `PROC-TASK-002`: `autoprogramming/status` debe contrastar runstore y runtime
  cuando `include_process_refs` o modo operador este activo, y reportar
  `agent_control_missing` o `orphan_process`.
- `PROC-TASK-003`: el supervisor debe reconciliar directorios/procesos vivos no
  registrados antes de declarar capacidad disponible o tareas `pending`.

## Criterio De Cierre

Un smoke equivalente debe lanzar una run con limite de concurrencia y verificar
que cada directorio/proceso real aparece en `agents[]`, que su tarea no queda
`pending`, y que `agents_in_flight` coincide con procesos vivos controlados.

## Ampliacion 2026-06-22: Launch Reclamado Sin Proyeccion

Run OPES:
`run-opes-tractorista-rework-tests-004-006-008-20260622`.

Sintoma adicional confirmado:

1. `g01` y `g03` arrancaron y dejaron `agent_ack.json`.
2. `g02` quedo con outbox `LaunchRuntimeAgent` en estado `no_ack` reclamado.
3. `ProcessRegistry` tenia registro de `g02` con `process_ref`, `pid`,
   `launch_ref`, `readiness_ref` y `ack-ref-*`.
4. El PID ya no existia y el run no tenia `g02` en `agents[]` ni en
   `started_agents[]`, aunque el evento durable `AgentRequested` si existia.

Impacto: el supervisor no podia tratar `g02` como agente iniciado, perdido,
terminal ni relanzable. La tarea quedaba aparentemente `pending` con un claim de
outbox vivo, rompiendo la autonomia del cierre OPES.

Arreglo aplicado:

- `DrainRunV0` ejecuta una reconciliacion de `LaunchRuntimeAgent` reclamado
  contra `ProcessRegistry` antes de recuperar outbox faltante.
- Si existe `AgentRequested` durable pero no esta proyectado, lo reproyecta por
  comando idempotente.
- Si existe `AgentStarted` durable pero falta en la proyeccion, lo aplica sin
  avanzar cursor historico.
- Si no existe `AgentStarted` pero si hay registro de proceso, registra
  `AgentStarted` desde `ProcessRegistry` y ACKea el outbox de lanzamiento
  supersedido.
- El ciclo normal posterior puede detectar el proceso muerto o sin ACK y
  decidir rework/reintento por las reglas existentes.

Validacion focal:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestReconcile(ClaimedLaunchOutbox|OrphanCapacity)'
```

Criterio de cierre especifico: una run con claim de lanzamiento, registro de
proceso y proyeccion parcial debe dejar de tener outbox pendiente de launch, el
agente debe aparecer en `agents[]` y `started_agents[]`, y la supervision debe
continuar por el flujo normal de ACK, perdida o replan.
