# TAREA OPES - Checkpoint rechazado y review payload inválido en cierre ASG

Fecha: 2026-06-25.

## Contexto

Durante el cierre local del temario `Auxiliar de Servicios Generales` C2 se
lanzó la run:

`run-external-work-asg-c2-cierre-normalizacion-20260625`.

El agente completó la normalización del curso y escribió:

- `09_tests/course_tests.json`
- `09_tests/question_banks/*.json`
- `09_tests/question_bank_manifest.json`
- `10_tutor_rag/corpus/chunks.jsonl`
- `10_tutor_rag/corpus/summary.json`
- `09_validacion/informe_no_author_notes.*`
- `09_validacion/informe_professional_infographics.*`
- `09_validacion/informe_visual_asset_reuse_qa_cierre.*`
- `09_validacion/informe_cierre_local_post_normalizacion_20260625.*`
- `00_control/ACKS/cierre_local_normalizacion_20260625_ack.md`
- `agent_ack.json` con `status: completed`

Validación de producto:

- `course_manifest.status = apto_local_post_normalizacion_20260625`
- 20 bancos en `09_tests/question_banks`
- 1000 preguntas
- `10_tutor_rag/corpus`: 20 temas, 726 chunks
- `rag/corpus/chunks.jsonl` y `10_tutor_rag/corpus/chunks.jsonl` idénticos

## Fallo 1: review inválida tras ACK válido

`POST /api/v0/runs/supervise` devolvió:

```text
estado=error
stop_reason=runtime_error
last.status=failed
director_supervised_burst_step_input:
field=step_input_builder:
orquestacoreworkflow.ReviewResultErrorV0:
payload_invalido: payload=<redacted>
```

Esto reproduce el fallo documentado en:

`TAREA_OPES_ORQUESTA_REVIEW_RESULT_PAYLOAD_INVALID_TRAS_ACK_2026-06-25.md`

## Fallo 2: checkpoint válido rechazado

Al pedir `/api/v0/server/shutdown`, Orquesta generó
`orquesta_shutdown_request.json` y el agente respondió:

```json
{
  "schema_version": "codex_shutdown_checkpoint_ack.v0",
  "status": "checkpoint_ready"
}
```

Orquesta lo rechazó con:

```text
shutdown-checkpoint-issue-invalid-agent_shutdown_checkpoint_ack.json-forbidden_detail
```

El checkpoint no contiene detalles extensos ni campos de producto; solo los
campos mínimos esperables (`schema_version`, `run_ref`, `agent_ref`,
`checkpoint_ref`, `status`, `occurred_at`). Si eso es inválido, el contrato que
Orquesta pide al agente y el validador del shutdown están desalineados.

## Tarea Técnica

1. Unificar contrato de `codex_shutdown_checkpoint_ack.v0` entre prompt,
   generador de request y validador de shutdown.
2. Añadir test con checkpoint mínimo `status=checkpoint_ready`.
3. Si el checkpoint tiene un campo no permitido, devolver diagnóstico con el
   nombre de campo, no `forbidden_detail` genérico.
4. No mantener `agents_in_flight=1` cuando:
   - existe `agent_ack.json` completed;
   - existe `agent_shutdown_checkpoint_ack.json`;
   - el proceso ya no produce trabajo de dominio.
5. Reutilizar `agent_ack.json` como cierre causal cuando la review posterior
   falle por payload inválido.

## Impacto OPES

El curso quedó correctamente normalizado, pero Orquesta impidió un cierre limpio
y obligó al director humano a interpretar artefactos internos y forzar cierre.
Esto rompe la autonomía de producción OPES aunque el trabajo de dominio esté
terminado.
