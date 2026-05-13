# Contrato V0: conector OPES

Fecha: 2026-05-13.

## Decision

El conector `opes_rest_mcp` sera opt-in y vivira fuera del nucleo de Orquesta.
Su unica responsabilidad sera traducir el contrato publico de OPES a trabajos y
entregas externas de Orquesta.

OPES sigue siendo la aplicacion de dominio editorial. Orquesta sigue siendo el
plano de orquestacion de agentes.

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

## Mapeo con `orquesta-domain-work`

El conector futuro debe adaptar OPES a:

- `DomainWorkJobRequestV0` al crear o aceptar trabajo externo;
- `DomainWorkArtifactSubmissionV0` al devolver artefactos.

`domain_ref` debe ser `opes`. `interface_refs` debe apuntar a refs publicas
REST/MCP, no a rutas locales ni internals de OPES.

## Estado

Aceptado como contrato local. Implementado primer corte REST para crear jobs y
enviar artefactos; MCP queda pendiente.
