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
