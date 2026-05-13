# Integracion OPES-Orquesta

Fecha: 2026-05-13.

## Decision

OPES queda como aplicacion de dominio editorial independiente. Orquesta no
entra en su base de datos, ficheros internos, rutas locales ni worker interno.
La integracion se hace por conectores REST/MCP, tratando cada trabajo OPES como
trabajo externo con refs opacas.

## Contrato Operativo

- Orquesta crea trabajos externos en OPES por `POST /api/jobs` o
  `create_document_job`.
- OPES devuelve `job.id`; Orquesta lo guarda como evidencia externa junto a
  `run_ref`, `task_ref` y `correlation_id`.
- Orquesta envia siempre `correlation_id`, `idempotency_key`,
  `requested_by=orquesta` y `external_refs`.
- Los reintentos reutilizan la misma `idempotency_key`; OPES deduplica jobs y
  artefactos equivalentes.
- Orquesta devuelve resultados por `POST /api/jobs/{id}/artifacts` o
  `submit_job_artifact`.

## Frontera Hexagonal

Dentro de Orquesta deben vivir:

- seleccion de agentes, modelos, capacidad y razonamiento;
- leases, sesiones, tmux/runtime, handoff, reintentos y watchdogs;
- Codex, Claude, Gemini u otros proveedores;
- registro de entregas como `RegisterDelivery` con `evidence_refs` hacia OPES.

Dentro de OPES deben vivir:

- programas, temarios, temas, capitulos, bloques y fuentes;
- validacion y persistencia de artefactos editoriales;
- endpoints de dominio para revisiones, fuentes u objetos que no sean
  `content_block`.

## Restricciones

- No asumir SQLite, Postgres, rutas locales ni estructura interna de OPES.
- No pasar conceptos de agente a OPES; si falta informacion se pide como campo
  de dominio o `external_refs`.
- Para documentacion de temarios OPES, Orquesta debe usar `gpt-5.5` con
  razonamiento `xhigh` o un modelo posterior/superior disponible, salvo
  override explicito del operador. OPES no elige modelo ni lo envia como campo
  de dominio; la seleccion vive en la politica de capacidad/modelos de
  Orquesta.

## Pendiente En Orquesta

Crear un conector `opes_rest_mcp` o equivalente, opt-in, que traduzca jobs OPES
a microtareas/agentes de Orquesta y publique artefactos de vuelta sin acoplarse
al nucleo de OPES.
