# Corte de integracion OPES-Orquesta

Fecha: 2026-05-13.

Nota 2026-05-18: documento historico. Para el estado vigente usar
`docs/corte_opes_como_consumidor_orquesta_2026-05-18.md` y
`docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md`. El conector
REST OPES ya existe; la superficie AI-first de Orquesta es el tool MCP generico
`orquesta.domain_work.v0`, con REST OPES como adaptador de composicion.

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

- juicio y planificacion mediante director/agentes;
- seleccion de agente, modelo, capacidad y razonamiento;
- leases, sesiones, runtime, handoff, reintentos y watchdogs;
- Codex, Claude, Gemini u otros proveedores;
- `RegisterDelivery` con `evidence_refs` hacia OPES.

OPES conserva:

- programas, temarios, temas, capitulos, bloques y fuentes;
- validacion y persistencia editorial;
- jobs documentales y artefactos de dominio;
- endpoints REST/MCP publicos.
- ensamblado final de documentos y PDFs.

## Restricciones

- No acceder a DB, ficheros internos ni rutas locales de OPES.
- No asumir SQLite, Postgres ni ningun backend.
- No pasar conceptos de agente a OPES como decisiones de dominio.
- OPES no debe planificar con juicio propio: si necesita decidir orden,
  dependencias, estructura pedagogica o agentes, debe pedir a Orquesta un
  trabajo de planificacion documental para que piense el director.
- No introducir REST/MCP dentro de `orquesta-domain-work`.
- Para documentacion de temarios OPES, Orquesta debe reservar `gpt-5.5` con
  razonamiento `xhigh` o un modelo posterior/superior disponible; esta regla es
  politica operativa de Orquesta, no contrato interno de OPES.
- Los temas largos no deben ejecutarse como una unica sesion opaca. OPES debe
  pedir trabajos de tamano editorial controlado (`draft_content_block` por
  capitulo, subcapitulo o bloque amplio), y Orquesta debe exigir artefacto
  trazable al cierre de cada unidad. Si una unidad larga no produce evidencia
  observable durante el presupuesto configurado, el director debe parar,
  replanificar o dividir el trabajo antes de gastar mas cuota.
- El scheduler de Orquesta debe repartir turno entre runs de igual prioridad.
  Al drenar una run se refresca su posicion temporal en cola para que otro job
  equivalente pueda avanzar en el siguiente tick.
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
