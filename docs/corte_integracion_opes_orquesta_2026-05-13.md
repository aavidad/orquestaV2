# Corte de integracion OPES-Orquesta

Fecha: 2026-05-13.

## Propuesta aceptada localmente

Preparar la integracion OPES en Orquesta, sin tocar OPES, creando primero la
frontera comun y un cliente REST opt-in pequeno para jobs y artefactos.

La forma elegida es un miniproyecto documental:

```text
modulos/orquesta-opes-connector/
```

El modulo es opt-in y actua como adaptador REST inicial. No forma parte del
nucleo de Orquesta ni del contrato generico `orquesta-domain-work`.

## Contrato operativo

- Orquesta crea jobs externos en OPES con `POST /api/jobs` o
  `create_document_job`.
- Orquesta envia siempre `correlation_id`, `idempotency_key`,
  `requested_by=orquesta` y `external_refs`.
- OPES devuelve `job.id`; Orquesta lo guarda como evidencia externa.
- Los reintentos reutilizan la misma `idempotency_key`.
- Orquesta devuelve resultados con `POST /api/jobs/{id}/artifacts` o
  `submit_job_artifact`.

## Frontera

Orquesta conserva:

- seleccion de agente, modelo, capacidad y razonamiento;
- leases, sesiones, runtime, handoff, reintentos y watchdogs;
- Codex, Claude, Gemini u otros proveedores;
- `RegisterDelivery` con `evidence_refs` hacia OPES.

OPES conserva:

- programas, temarios, temas, capitulos, bloques y fuentes;
- validacion y persistencia editorial;
- jobs documentales y artefactos de dominio;
- endpoints REST/MCP publicos.

## Restricciones

- No acceder a DB, ficheros internos ni rutas locales de OPES.
- No asumir SQLite, Postgres ni ningun backend.
- No pasar conceptos de agente a OPES como decisiones de dominio.
- No introducir REST/MCP dentro de `orquesta-domain-work`.
- Para documentacion de temarios OPES, Orquesta debe reservar `gpt-5.5` con
  razonamiento `xhigh` o un modelo posterior/superior disponible; esta regla es
  politica operativa de Orquesta, no contrato interno de OPES.
- El codigo productivo queda limitado al cliente REST de jobs/artefactos y sus
  tests con `httptest`; no toca OPES real.

## Documentos creados

- `modulos/orquesta-opes-connector/README.md`
- `modulos/orquesta-opes-connector/AGENTS.md`
- `modulos/orquesta-opes-connector/docs/contrato_v0.md`
- `modulos/orquesta-opes-connector/docs/mapeo_rest_mcp_v0.md`
- `modulos/orquesta-opes-connector/docs/plan_integracion_v0.md`

## Codigo creado

- `modulos/orquesta-domain-work`: contrato generico de trabajos/artefactos de
  dominio.
- `modulos/orquesta-opes-connector`: cliente REST opt-in para crear jobs OPES y
  enviar artefactos desde `DomainWorkJobRequestV0` y
  `DomainWorkArtifactSubmissionV0`.
