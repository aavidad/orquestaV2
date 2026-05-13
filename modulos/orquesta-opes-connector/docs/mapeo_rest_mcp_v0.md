# Mapeo REST/MCP V0

Fecha: 2026-05-13.

Este documento fija las superficies publicas de OPES que consume o podria
consumir el conector. REST para jobs y artefactos ya tiene primer cliente local;
MCP queda como extension futura.

## Crear jobs externos

REST:

```text
POST /api/jobs
```

MCP:

```text
create_document_job
```

Uso esperado:

- crear jobs con `execution_mode=external`;
- enviar siempre `correlation_id`, `idempotency_key`,
  `requested_by=orquesta` y `external_refs`;
- guardar `job.id` como evidencia externa en Orquesta.

## Consultar jobs

REST:

```text
GET /api/jobs
GET /api/jobs/{id}
```

MCP:

```text
list_jobs
get_job
```

Mapeo de estados recomendado:

```text
pending/retrying -> espera externa
running          -> trabajo en curso
completed        -> RegisterDelivery con evidence_ref OPES
failed           -> RequestRework o bloqueo
cancelled        -> tarea cancelada
```

## Reintentar o cancelar

REST:

```text
POST /api/jobs/{id}/retry
POST /api/jobs/{id}/cancel
```

MCP:

```text
retry_job
cancel_job
```

Regla: los reintentos de la misma intencion conservan la misma
`idempotency_key`. Las cancelaciones pueden enviar `reason` para auditoria.

## Entregar artefactos

REST:

```text
POST /api/jobs/{id}/artifacts
GET /api/jobs/{id}/artifacts
```

MCP:

```text
submit_job_artifact
list_job_artifacts
```

Artefactos materializables documentados:

```text
content_block
block_revision
source
visual_asset
```

Para `content_block`, OPES exige integridad editorial: `topic_id` y
`chapter_id` deben existir antes de entregar el artefacto si se quiere crear un
bloque vivo. El smoke completo debe preparar esos IDs con endpoints publicos de
OPES, no por DB ni por ficheros internos.

Para `visual_asset`, Orquesta entrega `artifact_type=visual_asset` con campos
`asset_type`, `format`, `title`, `caption`, `alt_text`, `body`, `placement`,
`language_code` y `source_refs` si aplica. `format=svg` debe ser autocontenido
y sin scripts, eventos JavaScript, `foreignObject` ni URLs remotas.

Para otros objetos de dominio, el conector debe usar endpoints OPES especificos
cuando OPES los publique. No debe resolverlos por DB ni ficheros.

## Jobs documentales externos

Contrato OPES actual:

```text
research_sources
split_syllabus_topic
draft_topic_outline
draft_content_block
generate_visual_asset
review_legal
review_pedagogical
review_quality
validate_topic
assemble_topic
export_topic
verify_sources
```

La unidad de redaccion recomendada es `draft_content_block`, no el tema
completo.
