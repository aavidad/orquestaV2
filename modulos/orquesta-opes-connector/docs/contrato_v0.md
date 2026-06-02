# Contrato V0: conector OPES

Fecha: 2026-05-13.

## Decision

El conector `opes_rest_mcp` es opt-in y vive fuera del nucleo de Orquesta.
Su unica responsabilidad es traducir el contrato publico de OPES a trabajos y
entregas externas de Orquesta.

OPES sigue siendo la aplicacion de dominio editorial. Orquesta sigue siendo el
plano de orquestacion de agentes.

OPES no planifica con juicio propio. Si necesita decidir estructura de un
temario, dependencias entre temas, orden de creacion, visuales, revisiones o
agentes, debe crear un job externo de planificacion para que Orquesta arranque
un director documental y devuelva un plan validable por OPES.

## Frontera

El conector puede conocer:

- nombres publicos de jobs OPES;
- endpoints REST y tools MCP publicados por OPES;
- refs externas de Orquesta: `run_ref`, `task_ref`, `delivery_ref`;
- metadatos de correlacion e idempotencia.

El conector no puede conocer ni asumir:

- base de datos, tablas, dialecto SQL o rutas de OPES;
- workers internos, colas internas o estructura de ficheros de OPES;
- decisiones de agente dentro de OPES;
- estrategia documental decidida sin director de Orquesta;
- sesiones, leases, `tmux`, runtime o proveedor como campos de dominio OPES;
- payloads internos no documentados por OPES.

## Campos obligatorios

Toda creacion de job externo OPES desde Orquesta debe incluir:

```json
{
  "correlation_id": "orquesta-run-or-task-id",
  "idempotency_key": "orquesta-task-id",
  "requested_by": "orquesta",
  "external_refs": {
    "run_ref": "run-id",
    "task_ref": "task-id"
  }
}
```

Los reintentos de la misma intencion reutilizan `idempotency_key`.

## Entregas

Toda entrega hacia OPES debe:

- conservar `job.id` como evidencia externa;
- usar una `idempotency_key` estable por entrega;
- enviar `external_refs` con refs de Orquesta;
- registrar en Orquesta una entrega propia con `evidence_refs` hacia OPES.

Ejemplo documental:

```json
{
  "artifact_type": "content_block",
  "summary": "Borrador del bloque",
  "idempotency_key": "orquesta-delivery-id",
  "payload_json": {
    "topic_id": "TOPIC_ID",
    "chapter_id": "CHAPTER_ID",
    "block_type": "technical",
    "content_type": "text/markdown",
    "title": "Bloque",
    "body": "Contenido producido por Orquesta",
    "source_refs": ["BOE-A-..."]
  },
  "external_refs": {
    "run_ref": "run-id",
    "task_ref": "task-id",
    "delivery_ref": "delivery-id"
  },
  "complete_job": true
}
```

Ejemplo visual:

```json
{
  "artifact_type": "visual_asset",
  "summary": "Vineta tecnica de red en estrella",
  "idempotency_key": "orquesta-visual-delivery-id",
  "payload_json": {
    "topic_id": "TOPIC_ID",
    "chapter_id": "CHAPTER_ID",
    "asset_type": "vignette",
    "format": "svg",
    "content_type": "image/svg+xml",
    "title": "Red en estrella",
    "caption": "Topologia donde todos los equipos se conectan a un nodo central.",
    "alt_text": "Switch central conectado a cinco equipos cliente.",
    "body": "<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 800 420\"></svg>",
    "placement": "after_block",
    "language_code": "es",
    "source_refs": []
  },
  "external_refs": {
    "run_ref": "run-id",
    "task_ref": "task-id",
    "delivery_ref": "delivery-id"
  },
  "complete_job": true
}
```

Ejemplo audio accesible:

```json
{
  "artifact_type": "audio_asset",
  "summary": "Audio accesible del tema",
  "idempotency_key": "orquesta-audio-delivery-id",
  "payload_json": {
    "topic_id": "TOPIC_ID",
    "assembled_topic_artifact_id": "ASSEMBLED_TOPIC_ARTIFACT_ID",
    "language_code": "es",
    "format": "mp3",
    "mime_type": "audio/mpeg",
    "duration_seconds": 1830,
    "audio_ref": "AUDIO_REF",
    "manifest_ref": "AUDIO_MANIFEST_REF",
    "source_artifact_ref": "ASSEMBLED_TOPIC_ARTIFACT_ID"
  },
  "external_refs": {
    "run_ref": "run-id",
    "task_ref": "task-id",
    "delivery_ref": "delivery-id"
  },
  "complete_job": true
}
```

## Confirmacion OPES 2026-05-13

OPES confirma que:

- `POST /api/jobs` acepta jobs documentales externos con
  `execution_mode=external`;
- `cmd/opes-worker` no reclama esos jobs externos;
- `draft_content_block` es job valido para redaccion de bloques;
- `POST /api/jobs/{id}/artifacts` materializa `content_block`,
  `block_revision` y `source` si el `payload_json` trae campos completos;
- OPES publica `generate_visual_asset` y acepta `visual_asset` como artefacto
  preferente para esquemas, vinetas, flujogramas, mapas conceptuales e
  infografias;
- OPES debe publicar `research_exam_precedents` y aceptar
  `exam_research_report` para busqueda externa verificable de examenes,
  convocatorias y temarios relacionados;
- OPES debe publicar `generate_question_bank` y aceptar `question_bank` para
  tests por tema con 4 opciones A-D, una correcta exacta, distractores
  plausibles, explicacion tutor, JSON/HTML revisable, metadata, informe y
  validaciones limpias; no debe mezclar ni sobrescribir bancos anteriores;
- OPES debe publicar `generate_audio_asset` y aceptar `audio_asset` como
  artefacto accesible derivado de `assembled_topic` o refs opacas del paquete
  final;
- OPES debe publicar `generate_tutor_assets` y aceptar `tutor_bot_package` para
  tutor y bots del temario;
- OPES debe publicar `generate_html_site` y aceptar `local_html_site` para el
  HTML local operativo con logos USO y aspecto USO/TCAE promocion interna;
- la deduplicacion funciona por `idempotency_key` en jobs y por
  `(job_id, idempotency_key)` en artefactos;
- replay de job puede devolver HTTP `200` con `created=false`; creacion nueva
  puede devolver HTTP `201` con `created=true`.

Para materializar un `content_block` vivo, `topic_id` y `chapter_id` deben
existir previamente en OPES. Si Orquesta usa refs inventadas, OPES puede
aceptar el job y auditar el artefacto, pero no debe crear un bloque vivo contra
un tema o capitulo inexistente.

Para materializar un `visual_asset`, Orquesta debe enviar `asset_type`,
`format`, `title`, `caption`, `alt_text`, `body`, `placement`,
`language_code` y `source_refs` si aplica. Cuando `format=svg`, el SVG debe
ser autocontenido y sin scripts, eventos JavaScript, `foreignObject` ni URLs
remotas.

Para materializar un `audio_asset`, Orquesta debe enviar un manifest publico con
`topic_id`, `assembled_topic_artifact_id` o `source_artifact_ref`,
`language_code`, `format`, `mime_type`, `duration_seconds`, `audio_ref` y
`manifest_ref` si aplica. La implementacion OPES puede usar `edge-tts` de
Microsoft como adaptador de sintesis por su calidad observada; esa decision no
debe viajar en el payload publico ni en refs de Orquesta. Proveedor, modelo,
GPU, rutas locales y procesos internos quedan en logs privados de OPES.

Respuesta de job aceptada por el conector:

```json
{
  "id": "job-ref",
  "status": "accepted",
  "correlation_id": "corr",
  "idempotency_key": "idem",
  "external_refs": {},
  "job": {
    "id": "job-ref",
    "type": "draft_content_block",
    "status": "pending",
    "execution_mode": "external"
  },
  "created": true
}
```

Respuesta de artefacto aceptada por el conector:

```json
{
  "id": "artifact-ref",
  "artifact_id": "artifact-ref",
  "job_id": "job-ref",
  "correlation_id": "corr",
  "idempotency_key": "idem",
  "external_refs": {},
  "artifact": {
    "id": "artifact-ref",
    "job_id": "job-ref",
    "type": "content_block",
    "reference_id": "block-ref"
  },
  "job": {
    "id": "job-ref",
    "status": "completed",
    "execution_mode": "external"
  },
  "block": {
    "id": "block-ref"
  }
}
```

## Mapeo con `orquesta-domain-work`

El conector futuro debe adaptar OPES a:

- `DomainWorkJobRequestV0` al crear o aceptar trabajo externo;
- `DomainWorkArtifactSubmissionV0` al devolver artefactos.
- `DomainDocumentPlanV0` cuando OPES pida `plan_tema`, `plan_temario` o
  `plan_documento`; OPES lo tratara como `PlanTemaV0`/`PlanTemarioV0`
  validable por sus reglas de dominio.
  Para `plan_temario`, el plan debe poder crear el flujo local completo:
  investigacion de examenes relacionados, redaccion, infografias, banco de
  tests, revisiones, ensamblado, audios por tema/apartado, tutor/bots y HTML
  local USO/TCAE antes de produccion.

`domain_ref` debe ser `opes`. `interface_refs` debe apuntar a refs publicas
REST/MCP, no a rutas locales ni internals de OPES.

`DomainWorkFieldV0.value_json` se proyecta como JSON estructurado dentro de
`input` o `payload_json`. Se usa para campos publicos de OPES como
`topic_outline`, `block_position`, `neighbor_context` o `quality_criteria`
cuando no bastan `value` ni `values`.

## Estado

Corte vigente 2026-06-02: `plan_temario` debe poder materializar el flujo
completo local antes de produccion. El contrato documental ya nombra los
artefactos nuevos (`exam_research_report`, `question_bank`,
`tutor_bot_package`, `local_html_site`) y el adaptador REST debe tratarlos como
artefactos publicos de OPES, sin resolver DB, rutas internas, proveedores ni
marca desde Orquesta.

Aceptado como contrato local. Implementado corte REST para:

- consultar ventanas pequenas de `GET /api/jobs`;
- leer bloques publicos de tema con `GET /api/topics/{id}/blocks`;
- crear jobs externos con `POST /api/jobs`;
- enviar artefactos con `POST /api/jobs/{id}/artifacts`, incluido
  `document_plan` y `audio_asset`.

Orquesta ya expone el adaptador MCP generico `orquesta.domain_work.v0`, que
puede usar este conector cuando la composicion lo inyecta. Queda fuera de este
corte un cliente MCP especifico contra tools OPES.
