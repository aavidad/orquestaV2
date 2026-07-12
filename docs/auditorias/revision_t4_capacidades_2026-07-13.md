# Revisión T4 — capacidades no ejecutadas

Fecha: 2026-07-13  
Revisor: Codex

## Corrección del alcance

La fuente `docs/auditorias/capacidades_no_ejecutadas_2026-07-12.md` enumera
estas tres capacidades sin consumidores productivos:

1. `orquesta-document-plan-expander`
2. `orquesta-presentation-extraction`
3. `orquesta-autonomy-program`

`orquesta-domain-work-memory` no sustituye a la segunda: es un adaptador volátil
de infraestructura, con consumidores en tests. No debe promoverse como backend
productivo cuando ya existe `orquesta-domain-work-file` durable.

## Corte A — document-plan-expander

- Exponer `orquesta.document_plan.expand.v0` con `preview` y `create_jobs`.
- `create_jobs` reutiliza `serverDomainWorkMCPJobCreatorV0` sobre el mismo
  executor de `orquesta.domain_work.v0`; no abre un segundo store.
- Replay idéntico conserva job refs mediante la idempotencia del backend file.
- La creación es secuencial, no transaccional: un fallo intermedio debe publicar
  jobs ya creados e issues; no afirmar atomicidad.
- Límites de plan, jobs y salida en la frontera MCP.

Run inicial local:

```text
request-ref-h5-document-plan-expander-t4a-20260713-001
resultado: backend perdido antes del primer diff
evidencia: cuatro codex_goal_observation_rejected, sin artefactos
acción: stop gobernado por /api/v0/runs/control; blocked terminal
```

Rework causal:

```text
request-ref-h5-document-plan-expander-t4a-20260713-002
rework_of: request-ref-h5-document-plan-expander-t4a-20260713-001
```

## Corte B — presentation-extraction

- Crear adaptador real OpenXML para PPTX con `archive/zip` y `encoding/xml`.
- `presentation_ref` opaco resuelto por catálogo bajo raíz obligatoria.
- `os.OpenRoot`, rechazo de traversal/symlinks/no-regulares y lectura acotada.
- No extraer el ZIP; limitar bytes, entradas, XML, total descomprimido, slides,
  texto y profundidad.
- Orden real desde `ppt/presentation.xml` y sus relationships, no `slideN`.
- Fixture PPTX real y smoke de `ExtractPresentationV0` extremo a extremo.
- Soportar solo PPTX en este corte; no anunciar ODP hasta implementar su parser.

Run local:

```text
request-ref-h5-presentation-extraction-t4b-20260713-001
estado al corte: running / observe_later
```

## Corte C — autonomy-program

- El core y el store file/CAS ya existen; falta consumo productivo.
- Tool propuesta: `orquesta.autonomy.program.v0` con `create`, `observe` y
  `claim_frontier`, reutilizando el mismo `*orquestastatefile.StoreV0` del stack.
- No exponer `PrepareAutonomyProgramFrontierV0`: puede producir acciones antes
  de persistir. `claim_frontier` debe pasar por
  `ClaimAutonomyProgramFrontierV0` (persist-before-act).
- `observe` debe devolver `RecoverAutonomyProgramActionsV0`; un replay de claim
  tras perder la respuesta devuelve cero acciones nuevas.
- `create` acepta una topología inicial, no un agregado runtime arbitrario:
  todos los nodos pendientes, sin launches, cierres, receipts ni rework.
- La tool completa se clasifica como mutación de control-plane, aunque
  `observe` sea una acción de lectura.
- Límites y paginación para no cruzar 64 KiB de salida MCP.

Este corte se ejecutará en serie: comparte registries MCP y `stack.go` con el
expander y con el trabajo T5 de Claude.

## Decisión de concurrencia

Paralelizar solo write-sets disjuntos:

- expander MCP y adaptador PPTX pueden correr en paralelo;
- autonomy-program espera al expander;
- ningún corte T4 toca `orquesta-council` ni el gate T5 de Claude.

