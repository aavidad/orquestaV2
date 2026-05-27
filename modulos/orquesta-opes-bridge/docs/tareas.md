# Tareas

## T12 smoke OPES real opt-in

Estado: bloqueado verificable 2026-05-27 para ejecucion real contra OPES
temporal.

Evidencia ya cerrada:

- `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector`
  paso en los intentos `agent-ref-task-autoprogramming-5373ad36695c-g01` y
  `agent-ref-task-autoprogramming-51f9a01810a0-g01`.
- El smoke fake aislado `run-until-assemble` paso y recorrio
  `draft_content_block`, `generate_visual_asset`, revisiones, `validate_topic`
  y `assemble_topic -> assembled_topic`.

Bloqueo real:

- falta OPES temporal vivo;
- falta servidor Orquesta temporal;
- faltan `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
  `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de efectos;
- falta cuota/modelo confirmado para ejecucion con agente real.

No relanzar otra implementacion padre para T12 sin esas precondiciones. La
reapertura debe ejecutar el runbook con limite bajo, secuencia de derivados y
sin `JOB_TYPE`/`JOB_REF` manual en la ruta de derivados.
