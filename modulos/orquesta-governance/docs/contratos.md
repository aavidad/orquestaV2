# Contratos locales: orquesta-governance

Registra puertos, DTOs y eventos que `orquesta-governance` expone o consume.

## Plantilla

```text
Nombre:
Tipo: puerto_entrada | puerto_salida | dto | evento | error
Version:
Propietario:
Consumidores:
Campos:
Invariantes:
Errores:
Pruebas de contrato:
```

## GovernanceCatalogV0

```text
Nombre: GovernanceCatalogV0
Tipo: dto
Version: v0
Propietario: orquesta-governance
Consumidores: orquesta-core, orquesta-mcp, orquesta-cli futuro, orquesta-web futuro, director
Campos:
  contract: const GovernanceCatalogV0.
  catalog_version: const v0.
  owner: const orquesta-governance.
  activation_policy: const historical_entries_require_review.
  source_policy: object. Declara DB v1 como evidencia forense, IDs heredados no canonicos y revision obligatoria antes de activar historicos.
  secret_control_policy: object. Declara controles estructurados obligatorios y prohibe depender solo de busquedas de texto libre para detectar secretos.
  catalogs.effective: GovernanceCatalogEntryV0[]. Reglas/skills/workflows que gobiernan V2; una entrada DB v1 solo puede aparecer aqui si tiene revision y decision de promocion. Consultable por modulo, rol, fase y tags cuando el alcance los declare.
  catalogs.proposed: GovernanceCatalogEntryV0[]. Candidatos documentales con origen forense, no activos.
  catalogs.quarantine: GovernanceCatalogEntryV0[]. Historicos conservados como evidencia, no activos.
Invariantes:
  - Ninguna entrada queda activa por aparecer en DB v1.
  - `catalogs.effective` es el unico bloque ejecutable como regla efectiva por consumidores futuros.
  - Una entrada con `origin.source_type=dbv1` solo puede estar en `catalogs.effective` si declara `status.review_state=approved_effective`, `promotion.decision_ref` y `promotion.promoted_at`.
  - `catalogs.proposed` y `catalogs.quarantine` no gobiernan runtime, tareas ni permisos.
  - Los IDs numericos de DB v1 no son canon ni se conservan como identidad publica.
  - El origen se expresa por tabla, rol, categoria y titulo/nombre; nunca por rowid.
  - Cada entrada mantiene tipo, alcance, version documental, estado, origen forense, criterio de promocion y controles de secretos.
  - Los secretos, tokens, rutas privadas no necesarias y estado operativo viejo no forman parte del catalogo.
  - La ausencia de secretos no se valida solo por texto libre; cada entrada declara marcadores estructurados de revision.
  - Las reglas efectivas V2 deben ser compactas y consultables por modulo, agente, fase y tags.
  - Governance publica catalogos; no ejecuta tareas, no decide runtime y no asigna permisos operativos.
Errores:
  - governance_catalog_source_unavailable: no se puede leer la fuente forense en modo readonly.
  - governance_catalog_invalid_source: la fuente no contiene tablas esperadas o no coincide con el inventario DBV1-000.
  - governance_catalog_entry_without_origin: una entrada no declara origen forense trazable.
  - governance_catalog_entry_without_promotion_criteria: una entrada no declara criterio de promocion.
  - governance_catalog_forbidden_activation: un consumidor intenta tratar entradas propuestas como reglas efectivas.
  - governance_catalog_secret_detected: una entrada contiene material sensible o credenciales.
  - governance_catalog_text_only_secret_control: una entrada o catalogo declara control de secretos solo por busqueda textual.
Pruebas de contrato:
  - Ver `docs/pruebas.md`, casos GOV-001-CT-001 a GOV-001-CT-005 y GOV-004-CT-001 a GOV-004-CT-004.
```

Schema canonico del modulo propietario:

- `docs/schemas/governance_catalog_v0.schema.json`

Fixtures canonicos del modulo propietario:

- `docs/fixtures/governance_catalog_v0/catalog_minimo_valido.json`
- `docs/fixtures/governance_catalog_v0/historico_efectivo_sin_revision_invalido.json`
- `docs/fixtures/governance_catalog_v0/filtro_texto_libre_secretos_invalido.json`

### GovernanceCatalogEntryV0

```text
Nombre: GovernanceCatalogEntryV0
Tipo: dto
Version: v0
Propietario: orquesta-governance
Consumidores: GovernanceCatalogV0
Campos:
  kind: enum rule | skill | workflow.
  scope: object. Nivel global, modulo, rol, fase, plantilla de dominio o proceso legado; consultable por modulo/agente/fase/tags cuando aplique. `tags` es opcional y solo etiqueta consulta documental compacta; no concede permisos.
  name: string. Titulo de regla o nombre de skill/workflow.
  summary: string. Resumen documental, no instruccion activa.
  status.catalog_state: enum effective | proposed | quarantine.
  status.review_state: enum pending_review | reviewed_candidate | approved_effective | quarantined | rejected_for_v2.
  version.contract_version: const v0.
  version.document_version: version documental V2; nunca ID canonico DB v1.
  version.source_version_evidence: evidencia observada, por ejemplo version_num=1/seed o sin_version_historica.
  origin.source_type: enum dbv1 | local_doc | director_decision.
  origin.source_ref: referencia forense por tabla/rol/categoria/nombre o documento local; nunca rowid.
  origin.forensic_inventory_ref: inventario o tarea que soporta la lectura.
  origin.legacy_public_id_policy: const no_legacy_ids_as_canon.
  promotion.criterion: criterio necesario para promocionar o rescatar la entrada.
  promotion.decision_ref: decision V2 obligatoria si `status.catalog_state=effective`.
  promotion.promoted_at: fecha obligatoria si `status.catalog_state=effective`.
  promotion.historical_review_required: const true.
  secret_controls: marcadores estructurados de ausencia de secretos; `text_scan_only` no es valido.
Invariantes:
  - `name` no se usa como identificador estable entre modulos.
  - `summary` no debe contener secretos ni copiar configuracion viva.
  - `catalog_state=proposed` significa candidato documental, no activo.
  - `catalog_state=quarantine` conserva razonamiento historico pero no debe alimentar catalogos efectivos.
  - `catalog_state=effective` exige decision de promocion y fecha; si procede de DB v1, exige revision explicita.
Errores:
  - governance_entry_unknown_kind
  - governance_entry_unknown_status
  - governance_entry_legacy_id_as_canon
  - governance_entry_missing_review_scope
  - governance_entry_effective_without_decision
  - governance_entry_text_only_secret_control
Pruebas de contrato:
  - Ver `docs/pruebas.md`, casos GOV-001-CT-002, GOV-001-CT-003, GOV-004-CT-002 a GOV-004-CT-004 y GOV-006-UNIT-001 a GOV-006-UNIT-003.
```

## Catalogo efectivo/propuesto/cuarentena v0

Lectura normativa de los bloques en `GovernanceCatalogV0`:

- `effective`: reglas V2 que los consumidores autorizados pueden tratar como efectivas. En v0 no contiene reglas, skills ni workflows de DB v1 promovidos como canon vivo.
- `proposed`: entradas de DB v1 candidatas a revision. Pueden servir como evidencia para una decision posterior, pero no gobiernan tareas, runtime ni permisos.
- `quarantine`: entradas historicas conservadas por trazabilidad. No son candidatas directas a activacion; requieren contrato o decision nueva si se rescata algun concepto.

## DecisionPromotionPolicyV0

```text
Nombre: DecisionPromotionPolicyV0
Tipo: dto
Version: v0
Propietario: orquesta-governance
Consumidores: director, orquesta-core futuro, orquesta-mcp futuro, orquesta-cli futuro
Campos:
  contract: const DecisionPromotionPolicyV0.
  policy_version: const v0.
  owner: const orquesta-governance.
  applies_to: const historical_proposals_votes.
  source_window: const OP-001..OP-124.
  evidence_policy.historical_records_role: const evidence_only.
  evidence_policy.accepted_statuses_as_evidence: propuesta historica con consenso, propuesta cerrada con decision explicita, voto trazable con justificacion.
  evidence_policy.rejected_statuses_not_authoritative: propuesta rechazada, duplicada, obsoleta, supersedida o sin evidencia suficiente.
  thresholds.min_votes_default: entero > 0 configurable por fase o decision.
  thresholds.non_author_votes_recommended: entero >= 0 configurable; recomendado >= 2 en decisiones de arquitectura compartida.
  thresholds.high_risk_requires_director: const true.
  thresholds.tie_requires_director: const true.
  phase_policy: mapa por fase. Cada fase declara si admite promocion historica directa como evidencia reutilizable, el minimo de votos y si exige director para cierre.
  outcome_policy.allowed_outcomes: promote_as_evidence | reject_for_v2 | superseded_by_v2 | escalate_to_director.
  outcome_policy.director_decision_required_for_effective: const true.
  references.accepted_origins: docs/op_*.md, docs/diario_*.md, docs/BIBLIA_APP_ORQUESTA.md, inventario forense DB v1, contratos/documentos V2.
Invariantes:
  - Los historicos OP-001..OP-124 sirven como evidencia y contexto, no como mandato operativo.
  - Una propuesta o voto historico no se convierte en regla efectiva, contrato compartido ni workflow vivo sin decision V2 trazable.
  - Rechazadas, duplicadas, obsoletas o supersedidas no mandan aunque tengan votos; solo documentan por que no seguir esa via.
  - Un empate, riesgo alto, conflicto entre fuentes o falta de evidencia minima obliga a `escalate_to_director`.
  - `min_votes_default` es configurable y puede subir por fase o impacto; nunca baja a 0.
  - La aplicacion por fase no reemplaza validacion tecnica, seguridad, i18n ni hexagonalidad.
Errores:
  - decision_promotion_missing_evidence
  - decision_promotion_insufficient_votes
  - decision_promotion_rejected_historical_source
  - decision_promotion_tie_requires_director
  - decision_promotion_high_risk_requires_director
  - decision_promotion_attempts_effective_without_v2_decision
Pruebas de contrato:
  - Ver `docs/pruebas.md`, casos GOV-002-CT-001 a GOV-002-CT-004.
```

### Regla operativa de promocion historica v0

- `promote_as_evidence`: un historico se puede citar como evidencia reutilizable cuando la propuesta original no este rechazada/duplicada/obsoleta, tenga trazabilidad suficiente y alcance el minimo de votos configurado para la fase.
- `reject_for_v2`: se usa cuando el historico contradice reglas globales V2, depende de CLI/rutas/DB v1, o carece de evidencia tecnica suficiente.
- `superseded_by_v2`: se usa cuando la idea era valida pero ya fue absorbida por una decision o contrato V2 mas reciente.
- `escalate_to_director`: obligatorio ante empate, riesgo alto, conflicto entre alternativas tecnicas, cambio de contrato compartido, o cualquier intento de dar caracter efectivo a una decision historica.

### Politica por fase v0

| Fase | Uso del historico | Minimo de votos | Cierre local permitido | Escalado |
| --- | --- | --- | --- | --- |
| brainstorming | evidencia comparativa y alternativas descartadas | configurable, por defecto 2 | si no afecta contratos compartidos ni arquitectura global | empate, riesgo alto o evidencia debil |
| arquitectura | evidencia comparativa, nunca mandato | configurable, recomendado >= 3 y >= 2 no autores | no cuando afecta modulos compartidos | director obligatorio si impacta contratos compartidos |
| implementacion | solo como contexto de microdecisiones | configurable, por defecto 1 mas validacion tecnica | si el write-set es local y no toca contratos | director si abre excepciones o deuda estructural |
| documentacion | evidencia para trazabilidad y rationale | configurable, por defecto 1 | si no reescribe contrato compartido | director si cambia significado operativo |
| validacion/revision | evidencia de antecedentes o fallos repetidos | configurable, por defecto 2 | si solo ajusta pruebas o criterios locales | director si cambia gate global o criterio de aceptacion |

## ArchitectureVoteV0

```text
Nombre: ArchitectureVoteV0
Tipo: dto
Version: v0
Propietario: orquesta-governance
Consumidores: director, brainstorming, arquitectura, documentacion, consumidores futuros por API/CLI/MCP
Campos:
  contract: const ArchitectureVoteV0.
  vote_version: const v0.
  owner: const orquesta-governance.
  scope.phase: enum brainstorming | arquitectura | implementacion | documentacion | validacion.
  scope.decision_kind: enum architecture_option | shared_contract_change | policy_change.
  scope.module_scope: enum local_module | shared_modules | global_app.
  subject: string. Decision compacta a resolver.
  context_ref: refs opacas a briefing, contrato o evidencia historica.
  options: lista no vacia de opciones explicitas con `option_id`, `summary`, `tradeoffs`, `risks`, `validation_plan`.
  criteria: lista no vacia de criterios tecnicos con peso opcional.
  thresholds.min_votes: entero > 0 configurable.
  thresholds.require_non_author_votes: boolean.
  thresholds.min_non_author_votes: entero >= 0 configurable.
  thresholds.director_required_on_tie: const true.
  thresholds.director_required_on_high_risk: const true.
  votes: lista de votos con `voter_ref`, `role`, `option_id`, `rationale`, `risk_level`, `is_author=false|true`.
  result.status: enum open | closed_with_recommendation | escalated_to_director | superseded.
  result.recommended_option_id: opcion recomendada o vacia si hay empate/escalado.
  result.closure_reason: resumen compacto del cierre.
  result.director_ref: obligatorio si `status=escalated_to_director` o si la opcion recomendada afecta contrato compartido.
Invariantes:
  - Siempre hay opciones explicitas; no se vota sobre texto libre ambiguo.
  - El voto requiere justificacion tecnica compacta y nivel de riesgo declarado.
  - El minimo de votos es configurable por fase y alcance.
  - Si `require_non_author_votes=true`, una decision no cierra sin el minimo de votos no autores configurado.
  - Empate o `risk_level=high` en la opcion lider obliga a `escalated_to_director`.
  - Un cambio de contrato compartido no queda efectivo solo por votacion; exige decision del director o contrato V2 posterior.
Errores:
  - architecture_vote_without_options
  - architecture_vote_without_minimum_votes
  - architecture_vote_without_rationale
  - architecture_vote_tie_requires_director
  - architecture_vote_high_risk_requires_director
  - architecture_vote_shared_contract_requires_director
Pruebas de contrato:
  - Ver `docs/pruebas.md`, casos GOV-003-CT-001 a GOV-003-CT-004.
```

### Workflow de votacion V2 v0

1. Abrir decision con `subject`, `phase`, `decision_kind`, `module_scope` y opciones explicitas.
2. Adjuntar evidencia compacta: briefing, contratos, pruebas previas e historicos solo como contexto.
3. Configurar umbrales para la fase: minimo de votos, votos no autores y criterio de escalado.
4. Recoger votos con justificacion tecnica, riesgo y plan de validacion resumido.
5. Cerrar localmente solo si hay ganador claro, riesgo no alto, evidencia suficiente y el alcance no requiere director.
6. Escalar al director ante empate, riesgo alto, falta de votos no autores, conflicto entre contratos o cambio compartido.
7. Registrar la alternativa descartada y la razon, para que el historico sirva despues como evidencia y no como mandato.

## Consulta in-memory v0

`QueryEffectiveGovernanceCatalogV0` es el harness compacto inicial para consumidores futuros:

- Entrada: `GovernanceCatalogV0` ya materializado en memoria y filtros opcionales `module`, `role`, `phase` y `tags`.
- Salida: entradas de `catalogs.effective` que coinciden con el alcance y contadores filtrados `effective`, `proposed` y `quarantine`.
- `catalogs.proposed` y `catalogs.quarantine` nunca se devuelven como reglas efectivas; solo se cuentan para auditoria compacta si coinciden con el mismo filtro.
- Si una entrada del bloque `effective` declara estado `proposed` o `quarantine`, la consulta falla con `governance_catalog_forbidden_activation`.
- Si una entrada del bloque `proposed` no declara estado `proposed`, o una entrada del bloque `quarantine` no declara estado `quarantine`, la consulta falla con `governance_catalog_forbidden_activation`.
- Si una entrada efectiva carece de `review_state=approved_effective`, `promotion.decision_ref` o `promotion.promoted_at`, la consulta falla como activacion invalida.
- La consulta no lee DB v1, no lee filesystem productivo y no activa historicos por origen forense.

## GovernanceCatalogPublicQuery v0

```text
Nombre: GovernanceCatalogPublicQuery v0
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-governance
Consumidores: orquesta-cli futuro, orquesta-mcp futuro, orquesta-web futuro, director
Ruta/version publica recomendada: POST /api/v0/governance/catalog/query
Campos:
  request.request_id: string opcional para trazabilidad compacta.
  request.correlation_id: string opcional para correlacion entre capas.
  request.filters.module: string opcional. Filtra por modulo exacto.
  request.filters.role: string opcional. Filtra por rol exacto.
  request.filters.phase: string opcional. Filtra por fase exacta.
  request.filters.tags: string[] opcional. Todas las tags pedidas deben existir en la entrada para que coincida.
  request.output_budget.max_entries: entero opcional. Default seguro 50; maximo canonico 200.
  request.output_budget.max_bytes: entero opcional. Default seguro 64 KiB; maximo canonico 256 KiB.
  response.schema_version: const governance_catalog_public_query.v0.
  response.request_id: eco compacto del request cuando existe.
  response.correlation_id: eco compacto para trazabilidad.
  response.catalog_version: version publica del catalogo o `v0` si la fuente no la declaro.
  response.current_block: const effective.
  response.freshness: estado derivado de refs del catalogo o razon estable `catalog_source_refs_unavailable`.
  response.source_refs: refs opacas acotadas a las fuentes/decisiones usadas para frescura.
  response.effective: GovernanceCatalogEntryV0[] filtrado; solo entradas vigentes.
  response.counters: GovernanceCatalogCountersV0. Incluye totales filtrados `effective`, `proposed` y `quarantine`.
  response.inactive_summary: conteos y refs acotadas de `proposed`/`quarantine`, sin payload completo.
  response.output_budget: presupuesto normalizado y estado `complete|truncated`.
  error.request_id: eco compacto del request cuando se pudo parsear.
  error.correlation_id: eco compacto cuando se pudo parsear.
  error.errors[].code: codigo publico estable.
  error.errors[].field: campo publico opcional.
Invariantes:
  - Es un puerto read-only y compacto; no crea, promueve, rescata ni muta reglas.
  - Solo `response.effective` se puede tratar como vigente.
  - `proposed` y `quarantine` nunca se devuelven como reglas activas; solo aparecen como conteos y refs opacas acotadas.
  - No se publican `inactive_blocks`, rowids DB v1, rutas HOME, prompts, transcripts ni catalogos completos de cuarentena.
  - Si se excede `output_budget`, la respuesta conserva contadores totales filtrados y devuelve un prefijo acotado con razon estable.
  - Reutiliza `GovernanceCatalogV0` y `QueryEffectiveGovernanceCatalogV0` como fuente semantica.
  - El proveedor del catalogo es inyectable; el contrato no exige DB real, runtime ni filesystem productivo.
Errores:
  - governance_catalog_invalid_request
  - governance_catalog_method_not_allowed
  - governance_catalog_source_unavailable
  - governance_catalog_forbidden_activation
  - governance_entry_effective_without_decision
Pruebas de contrato:
  - Ver `docs/pruebas.md`, casos GOV-007-UNIT-001 a GOV-007-UNIT-004.
```

### Adaptador HTTP local fino v0

- Implementacion local del modulo propietario: `GovernanceCatalogQueryHTTPHandlerV0`.
- Metodo recomendado: `POST`.
- Content-Type: `application/json`.
- Fuente del catalogo: proveedor inyectable `GovernanceCatalogProviderV0`.
- Si otro modulo quiere consumir esto por REST, el director debe promover la ruta/version y los DTOs minimos a `../../CONTRATOS.md`.

## Catalogo documental v0 extraido de DB v1

Fuente readonly forense en cuarentena: ref relativo opaco
`backups/legacy-sqlite-20260422/orquesta.db` desde la raiz del repositorio
cuando exista el snapshot local. Estado documental: `forense`/`historico`/
`quarantine`; no es prerequisito vivo, persistencia global ni entrada
programable sin decision explicita del director.

Tablas consultadas:

| Tabla | Filas DBV1-000 | Versiones observadas | Lectura v0 |
| --- | ---: | --- | --- |
| `reglas` | 40 | 36 con `version_num=1`, actor `orquesta`, accion `seed`; 4 sin version historica | Candidatas documentales; ninguna activa. |
| `skills` | 10 | 9 con `version_num=1`, actor `orquesta`, accion `seed`; 1 sin version historica | Candidatas documentales; ninguna activa. |
| `workflows` | 5 | 5 con `version_num=1`, actor `orquesta`, accion `seed` | Candidatos documentales; ninguno activo. |

### Reglas globales candidatas

| Nombre | Origen | Estado propuesto | Criterio de promocion |
| --- | --- | --- | --- |
| Conectores independientes del nucleo | DBv1.reglas(role=programador, category=arquitectura, name=Conectores independientes del nucleo) | promote_candidate | Convertir a regla efectiva V2 si queda alineada con hexagonalidad global y puertos publicos. Sin version historica en DB v1. |
| Propiedad exclusiva de ficheros | DBv1.reglas(role=programador, category=arquitectura, name=Propiedad exclusiva de ficheros) | promote_candidate | Mantener solo como regla de write-set por microtarea, sin nombres de modulos v1. |
| Multilenguaje por defecto donde aplique | DBv1.reglas(role=programador/documentador, category=arquitectura/general, name=Multilenguaje por defecto donde aplique) | promote_candidate | Delegar detalle a `orquesta-i18n-docs`; governance solo conserva obligatoriedad y excepcion tecnica. |
| i18n obligatorio antes de programar textos o UI | DBv1.reglas(role=programador, category=arquitectura, name=i18n obligatorio antes de programar textos o UI) | needs_review | Promocionar si `orquesta-i18n-docs` valida alcance e idiomas iniciales. Sin version historica en DB v1. |
| Sin secretos hardcodeados | DBv1.reglas(role=programador, category=seguridad, name=Sin secretos hardcodeados) | promote_candidate | Expresar como regla general de seguridad; no copiar valores ni proveedores. |
| Autonomia operativa segura | DBv1.reglas(role=programador, category=sesion, name=Autonomia operativa segura) | needs_review | Adaptar a permisos V2; acciones destructivas requieren consulta/director segun contrato de permisos futuro. |
| Tests coherentes, rapidos y mantenibles | DBv1.reglas(role=programador, category=calidad, name=Tests coherentes, rapidos y mantenibles) | promote_candidate | Promocionar como criterio general de pruebas por modulo. Sin version historica en DB v1. |
| Tests DB orquestados | DBv1.reglas(role=programador, category=calidad, name=Tests DB orquestados) | needs_review | Reescribir para persistencia como conector V2; no imponer helpers v1. Sin version historica en DB v1. |

### Reglas por rol candidatas

| Rol | Nombre | Origen | Estado propuesto | Criterio de promocion |
| --- | --- | --- | --- | --- |
| programador | Clean Architecture + DDD | DBv1.reglas(role=programador, category=arquitectura, name=Clean Architecture + DDD) | needs_review | Adaptar terminologia a hexagonal V2; no fijar carpetas v1. |
| programador | Modulos opcionales | DBv1.reglas(role=programador, category=arquitectura, name=Modulos opcionales) | needs_review | Promocionar solo si V2 adopta modulo opcional por contrato, no por variable `MODULES` heredada. |
| programador | main.go intocable | DBv1.reglas(role=programador, category=arquitectura, name=main.go intocable) | quarantine | Replantear como propiedad de integracion por modulo; `main.go` es detalle v1. |
| programador | Gate antes de commit | DBv1.reglas(role=programador, category=calidad, name=Gate antes de commit) | needs_review | Cambiar comandos por suite propia del modulo; no imponer `go test ./internal/...` global. |
| programador | Gate de cierre de modulo | DBv1.reglas(role=programador, category=calidad, name=Gate de cierre de modulo) | needs_review | Convertir a pruebas locales por modulo. |
| programador | Commits frecuentes | DBv1.reglas(role=programador, category=calidad, name=Commits frecuentes) | needs_review | No promocionar como automatismo; puede quedar como guia operativa si el director lo confirma. |
| programador | Propuesta antes de codigo | DBv1.reglas(role=programador, category=calidad, name=Propuesta antes de codigo) | needs_review | Depende de GOV-002/GOV-003 para nuevo workflow de decisiones. |
| programador | Idioma castellano | DBv1.reglas(role=programador, category=calidad, name=Idioma castellano) | needs_review | Compatibilizar con i18n por defecto; no contradecir documentacion multilenguaje. |
| programador | Protocolos de inicio, fin, tareas y votacion CLI | DBv1.reglas(role=programador, categories=sesion/comandos) | quarantine | CLI v1 no es canal principal V2; extraer conceptos, no comandos. |
| documentador | Solo documentacion | DBv1.reglas(role=documentador, category=general, name=Solo documentacion) | promote_candidate | Mantener como restriccion de rol cuando exista agente documentador; write-set explicito por tarea prevalece. |
| documentador | Activacion por notificacion | DBv1.reglas(role=documentador, category=general, name=Activacion por notificacion) | needs_review | Requiere workflow V2 entre programador/documentador. |
| documentador | Ficheros asignados e indice maestro | DBv1.reglas(role=documentador, category=general, name=Ficheros asignados) | quarantine | Rutas `docs/modulos/MXX_*.md` y `docs/00_INDICE.md` son legado. |
| documentador | Referencia obligatoria de documentacion externa | DBv1.reglas(role=documentador, category=general, name=Referencia obligatoria de documentacion externa) | needs_review | Promocionar como trazabilidad documental, sin acoplar a BD. |
| documentador | Protocolos de inicio y fin CLI | DBv1.reglas(role=documentador, category=sesion) | quarantine | Reescribir para MCP/director/API si se conserva el concepto. |

### Plantillas de dominio en cuarentena

| Nombre | Origen | Estado propuesto | Motivo |
| --- | --- | --- | --- |
| Multi-tenant aislado | DBv1.reglas(role=programador, category=arquitectura, name=Multi-tenant aislado) | quarantine | Regla municipal/tenant especifica; no doctrina global de OrquestaV2. |
| decimal.Decimal obligatorio | DBv1.reglas(role=programador, category=financiero, name=decimal.Decimal obligatorio) | quarantine | Valida para dominios monetarios, no para governance global. |
| Redondeo HALF_UP | DBv1.reglas(role=programador, category=financiero, name=Redondeo HALF_UP) | quarantine | Plantilla financiera; requiere contrato de dominio consumidor. |
| Auditoria obligatoria | DBv1.reglas(role=programador, category=seguridad, name=Auditoria obligatoria) | needs_review | El principio es reutilizable; parametros de retencion/cifrado no son canon. |
| JWT RS256 | DBv1.reglas(role=programador, category=seguridad, name=JWT RS256) | quarantine | Algoritmo y parametros de password son decision de seguridad por producto. |
| RBAC completo | DBv1.reglas(role=programador, category=seguridad, name=RBAC completo) | needs_review | Promocionar solo como modelo de permisos futuro, sin AD groups obligatorios. |
| Bloqueo tras 5 intentos | DBv1.reglas(role=programador, category=seguridad, name=Bloqueo tras 5 intentos) | quarantine | Umbral de producto, no regla global. |
| Filtrado por tenant_id | DBv1.reglas(role=programador, category=seguridad, name=Filtrado por tenant_id) | quarantine | Depende de multi-tenant municipal v1. |

### Skills candidatos

| Rol | Skill | Origen | Estado propuesto | Criterio de promocion |
| --- | --- | --- | --- | --- |
| programador | develop-feature | DBv1.skills(role=programador, name=develop-feature) | promote_candidate | Definir como skill documental para cambios acotados; no asignar automaticamente. |
| programador | fix-bug | DBv1.skills(role=programador, name=fix-bug) | promote_candidate | Promocionar con reproduccion, minimo impacto y pruebas. |
| programador | create-module | DBv1.skills(role=programador, name=create-module) | needs_review | Reescribir el flujo de 14 pasos a contratos V2; no usar carpetas v1. |
| programador | db-test-harness | DBv1.skills(role=programador, name=db-test-harness) | needs_review | Adaptar a persistencia como conector; sin version historica en DB v1. |
| programador | security-review | DBv1.skills(role=programador, name=security-review) | needs_review | Separar checklist general de requisitos de dominio regulado. |
| programador | administracion-publica-segura | DBv1.skills(role=programador, name=administracion-publica-segura) | quarantine | Plantilla de dominio regulado, no skill global. |
| programador | autofirma-integration | DBv1.skills(role=programador, name=autofirma-integration) | quarantine | Integracion especifica de administracion publica. |
| documentador | document-module | DBv1.skills(role=documentador, name=document-module) | needs_review | Requiere workflow V2 y rutas documentales actuales. |
| documentador | review-docs | DBv1.skills(role=documentador, name=review-docs) | promote_candidate | Promocionar como revision documental con evidencia contra codigo/contratos. |
| documentador | update-index | DBv1.skills(role=documentador, name=update-index) | quarantine | El indice maestro v1 no es canon V2. |

### Workflows candidatos

| Rol | Workflow | Origen | Estado propuesto | Criterio de promocion |
| --- | --- | --- | --- | --- |
| programador | inicio-sesion | DBv1.workflows(role=programador, name=inicio-sesion) | quarantine | CLI v1 y tareas en app vieja; rescatar concepto de briefing si se redefine por MCP/director. |
| programador | fin-sesion | DBv1.workflows(role=programador, name=fin-sesion) | needs_review | Reescribir cierre por write-set, pruebas y estado de tarea V2. |
| programador | votar-propuesta | DBv1.workflows(role=programador, name=votar-propuesta) | needs_review | Bloqueado por GOV-002/GOV-003. |
| programador | crear-modulo | DBv1.workflows(role=programador, name=crear-modulo) | needs_review | Reexpresar con AppSpec/ProyectoPlan/FunctionContract; no usar layout v1. |
| documentador | documentar-modulo | DBv1.workflows(role=documentador, name=documentar-modulo) | needs_review | Requiere contrato de handoff y rutas documentales V2. |

## CONSULTA AL DIRECTOR cerrada

```text
Modulo origen: orquesta-governance
Modulo afectado: modulos/CONTRATOS.md, orquesta-core, orquesta-mcp
Bloqueo inicial: GovernanceCatalog no estaba aun promovido globalmente, pero GOV-001 solo permitia escribir docs locales de governance.
Pregunta resuelta: Promocion de `GovernanceCatalogV0` a contrato compartido global.
Decision del director: `GovernanceCatalog v0` queda promovido a contrato compartido en `../../CONTRATOS.md`.
Impacto: El detalle canonico sigue en este modulo; `../../CONTRATOS.md` mantiene el contrato compartido minimo para core/MCP y demas consumidores autorizados.
Estado: cerrada 2026-05-04.
```
