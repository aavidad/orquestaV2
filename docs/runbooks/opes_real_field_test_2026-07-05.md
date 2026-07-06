# OPES Real Field Test 2026-07-05

## Checkpoint inicial

- Goal: `goal-ref-task-autoprogramming-72bb10d53162-g04`.
- Tarea origen: `task-remote-opes-real-field-test-after-g5-20260705`.
- Alcance: inventariar OPES real, elegir un temario incompleto y ejecutar la mision solo si existen `required_settings` y guarda explicita para efectos reales.
- Siguiente artefacto: decision de ejecucion y evidencia de bloqueo o ejecucion.
- Evidencia compacta: `checkpoint-started-opes-real-field-test-20260705`.

## Resultado operativo

Estado: `blocked`.

No se ejecuto inventario ni creacion/completado de temario contra OPES real. La
sesion solo expuso `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`; faltaban las
settings requeridas para efectos reales:

- `ORQUESTA_OPES_BASE_URL`.
- `ORQUESTA_BASE_URL`.
- `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` o confirmacion productiva gobernada con
  evidencia externa.
- `ORQUESTA_OPES_BRIDGE_CONFIRM=1`.
- Scope duro por `ORQUESTA_OPES_BRIDGE_JOB_REF` o por
  `ORQUESTA_OPES_BRIDGE_PROGRAM_ID`/`ORQUESTA_OPES_BRIDGE_TOPIC_ID`/
  `ORQUESTA_OPES_BRIDGE_CORRELATION_ID`.

Con esas ausencias, ejecutar el bridge habria incumplido la regla vigente de no
tocar OPES productivo ni colas ajenas sin confirmacion/configuracion explicita.
La decision correcta del goal es `missing_required_settings`.

## Ajuste aplicado

Se actualizo `scripts/smoke_opes_lifecycle_real.sh` para que `--help` sea una
ruta sin efectos y publique las settings minimas requeridas antes de cualquier
ejecucion con efectos. Esto mantiene verde el test requerido de ayuda y evita
que una consulta de uso dispare el smoke completo.

## Evidencias

- `checkpoint-started-opes-real-field-test-20260705`: checkpoint inicial
  materializado en este runbook.
- `evidence-ref-code-broker-unavailable-mcp-empty`: no habia recursos MCP
  disponibles para `orquesta.codebase.query.v0`; se uso lectura acotada.
- `evidence-ref-missing-required-settings-opes-real-20260705`: faltan las
  settings externas listadas arriba.
- `evidence-ref-smoke-help-guard-20260705`: `--help` queda sin efectos.

## Rework necesario

Abrir un goal causal nuevo cuando el operador aporte una instancia OPES
temporal/preproduccion y Orquesta temporal observables, con las settings
requeridas y un scope duro para un programa/temario incompleto concreto.
