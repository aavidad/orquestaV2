# Contratos locales: orquesta-context

Este modulo prepara el contexto que Orquesta entrega a un agente. El contrato es hexagonal: el builder no lee disco, no llama MCP, no conoce proveedores y no arranca runtime.

## ContextBundleV0

Propietario: `orquesta-context`

Consumidores previstos:

- `orquesta-director`, para pedir contexto antes de lanzar agentes;
- `orquesta-runtime`, para recibir un manifiesto junto a la orden de arranque;
- `orquesta-mcp`, para exponer/inspeccionar bundles compactos desde IA directora;
- equipos de modulo, como contexto local de trabajo.

Entrada: `ContextBundleRequestV0`

- `schema_version`: `context_bundle_request.v0`.
- `bundle_ref`: ref opaca del bundle solicitado.
- `work_order_ref`: ref opaca de la orden de trabajo.
- `target_module`: modulo destino, por ejemplo `orquesta-runtime`.
- `phase`: fase del workflow.
- `task_kind`: tipo compacto de tarea.
- `objective`: objetivo breve.
- `capacity_level`: `low`, `medium`, `high` o `xhigh` ya decidido por capacity/director.
- `read_set`, `write_set`, `contract_refs`, `cross_module_refs`, `evidence_refs`: refs opacas o rutas relativas de repo.
- `max_entries`, `max_total_bytes`: limites defensivos.

Salida: `ContextBundleV0`

- `schema_version`: `context_bundle.v0`.
- `bundle_ref`, `work_order_ref`, `target_module`, `phase`, `capacity_level`.
- `summary`: objetivo, tipo de tarea y politica de consulta al director.
- `limits`: limites efectivos.
- `entries`: manifiesto ordenado por capas.

Capas:

- `common_rules`: reglas comunes compactas.
- `module_context`: `AGENTS.md`, `README.md` y docs locales imprescindibles.
- `phase_context`: docs locales requeridos por fase.
- `task_context`: read-set/write-set de la microtarea.
- `contract_context`: contratos publicos y refs cruzadas.
- `evidence_context`: evidencias compactas.

Invariantes:

- El bundle contiene refs, no contenido largo.
- Siempre incluye `AGENTS.md`, `README.md`, `docs/contratos.md` y `docs/tareas.md` del modulo.
- `docs/pruebas.md` entra en fases de programacion, integracion, revision, validacion y cierre.
- `docs/decisiones.md` entra en fases de brainstorming, votacion, integracion, revision y cierre.
- Si un agente necesita informacion externa no incluida, debe emitir `CONSULTA AL DIRECTOR`.
- Los detalles de HOME real, OAuth, tokens, secretos, transcripts, prompts completos, proveedores y motores concretos se rechazan.
- Si el bundle supera limites, se rechaza en vez de crecer sin control.

Regla de evolucion:

- Los adaptadores que materialicen refs deben ser conectores versionados.
- Cualquier campo incompatible crea una version nueva.
- El contexto global solo entra como ref compacta de reglas comunes, no como documento completo.

## ContextMaterializedBundleV0

Propietario: `orquesta-context`

Entrada:

- `ContextBundleV0` valido.
- `ContextRefReaderV0`, puerto que materializa refs autorizadas.

Salida:

- `ContextMaterializedBundleV0` con `schema_version=context_materialized_bundle.v0`.
- `entries` materializadas en modo `content` o `ref_only`.
- `total_bytes` acumulado.
- `director_question_hint` propagado desde el bundle.

Conector inicial:

- `FileContextRefStoreV0`.
- Requiere root explicito.
- Solo lee refs relativas bajo `modulos/`.
- Bloquea rutas absolutas, traversal, URL, `~`, `$HOME` y refs fuera del root.
- Lee con limite `max_bytes + 1` y marca `truncated` si recorta.

Reglas:

- `rule_ref` conocido se materializa como reglas comunes compactas.
- `doc_ref` y `read_ref` se materializan por el reader.
- `write_ref`, `contract_ref` y `evidence_ref` quedan como `ref_only`.
- Toda entrada requerida que quede como `ref_only` declara `ref_only_reason` y
  `required_ref_action` para distinguir refs opacas por diseno de refs que
  debian materializarse, requerir lectura local o requerir consulta al director.
- Si se detectan patrones de secreto o ruta real sensible en contenido materializado, se rechaza.
- Si falta una ref requerida, la materializacion falla y el agente debe emitir `CONSULTA AL DIRECTOR` o esperar correccion del director.

Errores publicos:

- `context_materialization_bundle_invalido`
- `context_materialization_reader_requerido`
- `context_materialization_root_invalido`
- `context_materialization_ref_invalida`
- `context_materialization_ref_no_encontrada`
- `context_materialization_tamano_invalido`
- `context_materialization_detalle_prohibido`

## ContextSanitizerPortV0

Propietario: `orquesta-context`.

Tipo: puerto neutral opcional.

Entrada:

- `ContextSanitizationRequestV0`: refs del bundle/entry y contenido materializado.

Salida:

- `ContextSanitizationResultV0`: `clean`, `sanitized` o `review_required`;
- `ContextSanitizationEvidenceV0`: evidencia durable sin guardar el valor sensible.

Reglas:

- El puerto no conoce IA, proveedor, modelo, transporte, HOME ni runtime real.
- Si no hay sanitizador inyectado, la materializacion conserva el bloqueo
  historico por `context_materialization_detalle_prohibido`.
- Si el sanitizador limpia el contenido, `sanitization_evidence` viaja junto al
  bundle con categorias y contador, nunca con el secreto original.
- Si el sanitizador duda, la entrada requerida se degrada a `ref_only`, se marca
  `truncated=true` y se propaga `CONSULTA_AL_DIRECTOR`.
