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
- Detalles operativos, HOME, OAuth, tokens, prompts, transcripts, proveedores y
  motores concretos no se usan como veto automatico con rails offline. Si
  aparecen en refs/evidencias, el adaptador o sanitizador los minimiza cuando
  sea un valor sensible efectivo; el Director decide aprovechamiento, rework o
  tarea derivada.
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
- Con rails offline, detalles locales o de control en contenido materializado no
  bloquean por si mismos; el Director o sanitizador decide si se aprovechan,
  reducen o convierten en tarea.
- Si falta una ref requerida, la materializacion falla y el agente debe emitir `CONSULTA AL DIRECTOR` o esperar correccion del director.

Errores publicos:

- `context_materialization_bundle_invalido`
- `context_materialization_reader_requerido`
- `context_materialization_root_invalido`
- `context_materialization_ref_invalida`
- `context_materialization_ref_no_encontrada`
- `context_materialization_tamano_invalido`
- `context_materialization_detalle_prohibido` queda como codigo historico o de
  adaptador explicito, no como veto automatico por palabras con rails offline.

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
- Si no hay sanitizador inyectado, la materializacion no reactiva rails de
  detalle historicos mientras siga vigente `RAILS-D016`.
- Si el sanitizador limpia el contenido, `sanitization_evidence` viaja junto al
  bundle con categorias y contador, nunca con el secreto original.
- Si el sanitizador duda, la entrada requerida se degrada a `ref_only`, se marca
  `truncated=true` y se propaga `CONSULTA_AL_DIRECTOR`.

## CodeContextQueryPortV0

Propietario: `orquesta-context`.

Tipo: puerto neutral de consulta compacta de codigo.

Entrada:

- `CodeContextQueryV0` con `repository_ref`, `worktree_ref` o `commit_ref`,
  `worktree_fingerprint` opcional, `dirty_worktree`, `query_kind`, `query`,
  `scope`, `max_results`, `max_bytes`, `cache_only`,
  `allow_external_indexer` y `requested_by`.

Salida:

- `CodeContextResultV0` con resultados compactos, proveedor usado, cache,
  `query_hash`, diagnostics, issues y refs de evidencia.

Reglas:

- El broker central sirve la consulta; los agentes no arrancan
  `codebase-memory-mcp` por su cuenta.
- `codebase-memory-mcp` solo puede ejecutarse con opt-in central y lease de
  herramienta auxiliar.
- Consultas concurrentes iguales se deduplican por in-flight; las siguientes
  esperan el resultado de la primera.
- La cache incluye fingerprint/dirty-worktree para no ocultar cambios locales
  sin commit limpio.
- Los resultados son snippets compactos; no transportan contexto bruto ni
  secretos.

## CodeContextToolLeaseV0

Propietario: `orquesta-context`.

Tipo: DTO/puerto neutral para gobernar herramientas auxiliares de contexto.

Entrada:

- `CodeContextToolLeaseRequestV0`: repo, query hash, tool ref, provider kind,
  owner ref, started_at y TTL.
- `CodeContextToolLeaseObservationV0`: lease, observed_at, CPU, peticiones
  activas y evidencia.

Salida:

- `CodeContextToolLeaseV0`: lease activo/terminal con `lease_until`.
- `CodeContextToolLeaseAssessmentV0`: decision `continue` o `request_stop`,
  reason code y evidencia.

Reglas:

- No contiene PID, HOME, proveedor real, token ni kill.
- El reloj llega como dato; el evaluador es puro.
- Un lease expirado sin peticiones activas pide parada cooperativa; si hay
  peticiones activas o el lease ya es terminal, no pide parada.
