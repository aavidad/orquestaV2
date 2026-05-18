# Mapeo REST/MCP V0

Fecha: 2026-05-13.

Este documento fija las superficies publicas de OPES que consume o podria
consumir el conector. REST para jobs, bloques publicos de tema y artefactos ya
tiene cliente local. Orquesta tambien publica MCP generico
`orquesta.domain_work.v0`, que puede delegar en este conector por puertos
inyectados. Lo que queda como extension futura es un cliente MCP especifico
contra tools OPES, si aporta ventaja frente a REST.

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

Implementado para ventanas pequenas de cola externa:

```text
GET /api/jobs?execution_mode=external&status=pending&job_type=<opcional>&limit=<n>
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
topic_summary
topic_expansion_package
```

Para `content_block`, OPES exige integridad editorial: `topic_id` y
`chapter_id` deben existir antes de entregar el artefacto si se quiere crear un
bloque vivo. El smoke completo debe preparar esos IDs con endpoints publicos de
OPES, no por DB ni por ficheros internos.

Para `visual_asset`, Orquesta entrega `artifact_type=visual_asset` con campos
`asset_type`, `format`, `title`, `caption`, `alt_text`, `body`, `placement`,
`language_code` y `source_refs` si aplica. `format=svg` debe ser autocontenido
y sin scripts, eventos JavaScript, `foreignObject` ni URLs remotas.

Para `summarize_topic`, `summarize_chapter`, `summarize_block` y
`create_exam_outline`, Orquesta entrega `artifact_type=topic_summary`. Si el
agente produce JSON estructurado, se envia como `payload_json` sin envolverlo
en `body`.

Para `expand_topic_from_summary`, Orquesta entrega
`artifact_type=topic_expansion_package`. El agente debe producir un JSON con
`topic_id`, `language_code` y `chapters`; OPES materializa capitulos y bloques.

Para `plan_tema`, `plan_temario` y `plan_documento`, Orquesta entrega
`artifact_type=document_plan` con payload compatible con
`DomainDocumentPlanV0`. OPES valida ese plan como `PlanTemaV0` o
`PlanTemarioV0` y decide despues que jobs concretos crear para redaccion,
visuales, revision, ensamblado y exportacion.

Para otros objetos de dominio, el conector debe usar endpoints OPES especificos
cuando OPES los publique. No debe resolverlos por DB ni ficheros.

## Jobs documentales externos

Contrato OPES actual:

```text
research_sources
plan_documento
plan_tema
plan_temario
split_syllabus_topic
draft_topic_outline
draft_content_block
summarize_topic
expand_topic_from_summary
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
