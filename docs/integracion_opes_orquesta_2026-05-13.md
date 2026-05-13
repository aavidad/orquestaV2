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
- OPES puede devolver HTTP `201` con `created=true` al crear y HTTP `200` con
  `created=false` en replay idempotente; Orquesta debe aceptar ambos como
  respuestas correctas.

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

## Granularidad Editorial

OPES decide la unidad editorial: bloque, subcapitulo o capitulo. Orquesta no
divide por defecto un tema de 50 folios en microtareas minimas; solo pide split
si excede contexto, trazabilidad, capacidad de revision o si falta paquete de
dominio suficiente.

Para trabajos amplios, OPES debe mandar refs y paquete acotado: `topic_id`,
`chapter_id`, posicion de bloque/capitulo, esquema, contexto vecino, fuentes,
criterios y longitud esperada. El temario completo no debe viajar como campo
inline gigante. Si Orquesta detecta contexto requerido truncado, el agente solo
puede completar si justifica `contexto_truncado_resuelto: ...`; si no, debe
bloquear con consulta al director.

## Restricciones

- No asumir SQLite, Postgres, rutas locales ni estructura interna de OPES.
- No pasar conceptos de agente a OPES; si falta informacion se pide como campo
  de dominio o `external_refs`.
- Para documentacion de temarios OPES, Orquesta debe usar `gpt-5.5` con
  razonamiento `xhigh` o un modelo posterior/superior disponible, salvo
  override explicito del operador. OPES no elige modelo ni lo envia como campo
  de dominio; la seleccion vive en la politica de capacidad/modelos de
  Orquesta.
- Para un smoke de `content_block` materializado, no usar `topic_id` ni
  `chapter_id` inventados. Deben crearse o seleccionarse previamente mediante
  API publica de OPES.

## Pendiente En Orquesta

- El conector REST opt-in de jobs/artefactos ya existe.
- Siguiente corte: ejecutar smoke real contra OPES con
  `ORQUESTA_OPES_BASE_URL`, usando `topic_id` y `chapter_id` reales.
- Siguiente corte de producto: traducir jobs OPES a unidades de trabajo/agentes
  de Orquesta y registrar entregas con `evidence_refs` hacia OPES.
