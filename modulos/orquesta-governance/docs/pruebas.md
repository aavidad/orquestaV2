# Pruebas locales: orquesta-governance

Registra pruebas obligatorias del modulo.

```text
Caso: GOV-CT-008 descriptor_source MCP governance
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server
Evidencia esperada: el descriptor MCP de governance referencia owner, fuente
canonica, freshness y errores publicos del catalogo publico sin activar
historicos ni exponer DB v1.
Ultima ejecucion: 2026-05-27, ok en paquete
`agent-ref-task-autoprogramming-c3678e9bc306-g01`.
Riesgos: Valida la frontera documental/contractual; no sustituye pruebas de consumidores web/CLI.
Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` usa el comando
obligatorio ampliado para conservar el cierre T198.
```

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```

## Pruebas previstas GOV-001

```text
Caso: GOV-001-CT-001 - lectura forense readonly de DB v1
Tipo: contract
Comando historico en cuarentena: lectura readonly acotada del snapshot forense
`backups/legacy-sqlite-20260422/orquesta.db` cuando exista localmente; no es
prueba operativa obligatoria ni trabajo programable sin decision del director.
Evidencia esperada: La lectura no escribe en DB y confirma tablas `reglas`, `skills`, `workflows` y sus tablas `_versiones`.
Ultima ejecucion: 2026-05-04; ejecutada con lectura readonly y consultas acotadas.
Riesgos: La DB historica puede estar bloqueada puntualmente; repetir con consultas pequenas y sin cargar tablas masivas.
```

```text
Caso: GOV-001-CT-002 - catalogo separa tipos de entrada
Tipo: contract
Comando: revision documental de `docs/contratos.md`
Evidencia esperada: `GovernanceCatalogV0` separa reglas, skills y workflows en bloques `effective`, `proposed` y `quarantine`; cada entrada declara origen DB v1 cuando aplique, estado y criterio de promocion.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: Es una prueba documental hasta que exista schema automatizable.
```

```text
Caso: GOV-001-CT-003 - no activacion automatica
Tipo: contract
Comando: revision documental de `activation_policy` y estados propuestos
Evidencia esperada: El catalogo usa `activation_policy=historical_entries_require_review`; `proposed` y `quarantine` se definen como no activos.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md` y `docs/decisiones.md`.
Riesgos: Un consumidor futuro podria confundir candidato con efectivo si no valida `activation_policy`.
```

```text
Caso: GOV-001-CT-004 - no IDs canonicos ni secretos
Tipo: contract
Comando: revision documental y `rg -n "rowid|_id|token|secret|password|clave" docs/contratos.md docs/decisiones.md docs/pruebas.md docs/tareas.md`
Evidencia esperada: No hay IDs numericos de DB v1 como identidad publica ni secretos copiados; las menciones a `_id` solo aparecen en textos de origen/cuarentena o pruebas.
Ultima ejecucion: 2026-05-04; ejecutada. Solo encontro menciones documentales a la prohibicion, al caso de prueba y a parametros historicos puestos en cuarentena.
Riesgos: Los nombres historicos pueden contener rutas o parametros de dominio que deben mantenerse en cuarentena.
```

```text
Caso: GOV-001-CT-005 - reglas de cuarentena no gobiernan V2
Tipo: contract
Comando: revision documental de estados `quarantine` y `needs_review`
Evidencia esperada: Reglas municipales, financieras, CLI y rutas v1 aparecen como cuarentena o revision, no como reglas efectivas.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: Una microtarea posterior debe elevar decision si quiere convertirlas en plantilla de dominio.
```

```text
Caso: GOV-001-SM-001 - formato Markdown y whitespace
Tipo: smoke
Comando: git diff --check -- modulos/orquesta-governance
Evidencia esperada: Sin errores de whitespace en el write-set permitido.
Ultima ejecucion: 2026-05-04; ejecutada sin errores.
Riesgos: Solo valida formato basico de diff, no semantica del catalogo.
```

## Pruebas documentales GOV-005

```text
Caso: GOV-005-SM-001 - alineacion con contrato compartido
Tipo: smoke
Comando: rg -n "contrato global pendiente|sigue local hasta|No promover todavia|documental_only|CONSULTA AL DIRECTOR$" docs/contratos.md docs/decisiones.md docs/tareas.md
Evidencia esperada: Sin menciones abiertas u operativas que traten `GovernanceCatalogV0` como pendiente/local; las menciones historicas deben estar marcadas como superadas o cerradas.
Ultima ejecucion: 2026-05-04; documentado tras la promocion global en `../../CONTRATOS.md`.
Riesgos: Es una comprobacion textual; no sustituye la revision del contrato compartido global.
```

## Pruebas automatizables GOV-004

```text
Caso: GOV-004-CT-001 - JSON valido para schema y fixtures
Tipo: contract
Comando: jq empty docs/schemas/governance_catalog_v0.schema.json docs/fixtures/governance_catalog_v0/catalog_minimo_valido.json docs/fixtures/governance_catalog_v0/historico_efectivo_sin_revision_invalido.json docs/fixtures/governance_catalog_v0/filtro_texto_libre_secretos_invalido.json
Evidencia esperada: `jq` termina con exit code 0 para schema y fixtures.
Ultima ejecucion: 2026-05-04; ejecutada sin errores.
Riesgos: Valida sintaxis JSON, no semantica del contrato.
```

```text
Caso: GOV-004-CT-002 - fixture positivo GovernanceCatalogV0
Tipo: contract
Comando: npx --yes ajv-cli validate -s docs/schemas/governance_catalog_v0.schema.json -d docs/fixtures/governance_catalog_v0/catalog_minimo_valido.json --spec=draft7 --all-errors
Evidencia esperada: `catalog_minimo_valido.json valid`.
Ultima ejecucion: 2026-05-04; ejecutada con AJV draft7 y exit code 0.
Riesgos: Fixture minimo; no cubre todas las combinaciones de scope por modulo/agente/fase.
```

```text
Caso: GOV-004-CT-003 - historico DB v1 no efectivo sin revision
Tipo: contract
Comando: npx --yes ajv-cli validate -s docs/schemas/governance_catalog_v0.schema.json -d docs/fixtures/governance_catalog_v0/historico_efectivo_sin_revision_invalido.json --spec=draft7 --all-errors
Evidencia esperada: AJV devuelve exit code 1; errores en `status.review_state`, `promotion.decision_ref` y `promotion.promoted_at`.
Ultima ejecucion: 2026-05-04; ejecutada y fallo como invalido por intentar poner un origen DB v1 en `effective` sin revision/promocion.
Riesgos: AJV informa tambien el fallo del `then` condicional; es esperado para origen `dbv1`.
```

```text
Caso: GOV-004-CT-004 - control de secretos no solo texto libre
Tipo: contract
Comando: npx --yes ajv-cli validate -s docs/schemas/governance_catalog_v0.schema.json -d docs/fixtures/governance_catalog_v0/filtro_texto_libre_secretos_invalido.json --spec=draft7 --all-errors
Evidencia esperada: AJV devuelve exit code 1; errores en `secret_control_policy.free_text_scan_only_allowed` y `secret_controls.review_method`.
Ultima ejecucion: 2026-05-04; ejecutada y fallo como invalido por declarar control de secretos basado solo en texto libre.
Riesgos: El schema impide la declaracion contractual debil; no sustituye revisiones de seguridad de consumidores futuros.
```

## Pruebas automatizables GOV-006

```text
Caso: GOV-006-UNIT-001 - consulta efectiva por alcance
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: La consulta devuelve solo reglas `effective` que coinciden por modulo, rol, fase y tags; tambien devuelve contadores filtrados de `proposed` y `quarantine`.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores.
Riesgos: La prueba usa DTOs en memoria; no sustituye pruebas de contrato de consumidores core/MCP.
```

```text
Caso: GOV-006-UNIT-002 - proposed/quarantine ignorados como efectivos
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: Entradas `proposed` y `quarantine` que coinciden con el filtro no aparecen en la salida efectiva y solo se cuentan.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores.
Riesgos: No valida UI ni MCP; solo la regla pura del modulo propietario.
```

```text
Caso: GOV-006-UNIT-003 - rechazo de entradas mal ubicadas por bloque
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: Una entrada colocada en `catalogs.effective` con estado `proposed` falla con `governance_catalog_forbidden_activation`; una entrada `effective` en `proposed`, una `proposed` en `quarantine` o una `quarantine` en `proposed` fallan con el mismo error. Una efectiva sin decision falla con `governance_entry_effective_without_decision`.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores tras revision de direccion.
Riesgos: El harness no valida todo el schema JSON; solo invariantes necesarias para la consulta efectiva.
```

```text
Caso: GOV-006-CT-001 - schema y fixture positivo aceptan tags opcionales
Tipo: contract
Comando: jq empty docs/schemas/governance_catalog_v0.schema.json docs/fixtures/governance_catalog_v0/catalog_minimo_valido.json
Evidencia esperada: `jq` termina con exit code 0 para schema y fixture positivo.
Ultima ejecucion: 2026-05-04; ejecutada sin errores. Tambien se valido el fixture positivo con AJV y se confirmo que los dos fixtures negativos siguen fallando como invalidos.
Riesgos: `jq` valida sintaxis, no semantica JSON Schema completa.
```

## Pruebas documentales GOV-002

```text
Caso: GOV-002-CT-001 - historicos OP sirven como evidencia, no como mandato
Tipo: contract
Comando: revision documental de `docs/contratos.md` y `docs/decisiones.md`
Evidencia esperada: `DecisionPromotionPolicyV0` declara `historical_records_role=evidence_only` y las decisiones locales reiteran que OP-001..OP-124 no activan reglas ni contratos por si solos.
Ultima ejecucion: 2026-05-04; documentado en contratos y decisiones.
Riesgos: Es validacion documental hasta que exista un consumidor real por API/MCP/CLI.
```

```text
Caso: GOV-002-CT-002 - rechazadas, duplicadas y supersedidas no mandan
Tipo: contract
Comando: revision documental de `DecisionPromotionPolicyV0`
Evidencia esperada: La politica marca propuestas rechazadas, duplicadas, obsoletas o supersedidas como no autoritativas y solo utiles para trazabilidad negativa.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: Requiere que futuros consumidores respeten `allowed_outcomes` y no reintroduzcan atajos.
```

```text
Caso: GOV-002-CT-003 - empate o riesgo alto escala al director
Tipo: contract
Comando: rg -n "tie_requires_director|high_risk_requires_director|escalate_to_director" modulos/orquesta-governance/docs/contratos.md modulos/orquesta-governance/docs/decisiones.md
Evidencia esperada: La politica de promocion y la votacion V2 obligan a escalado ante empate, riesgo alto o conflicto de evidencia.
Ultima ejecucion: 2026-05-04; prevista para este write-set.
Riesgos: La comprobacion textual no sustituye pruebas ejecutables de un motor futuro.
```

```text
Caso: GOV-002-CT-004 - minimo de votos configurable por fase
Tipo: contract
Comando: revision documental de tablas de fase y umbrales
Evidencia esperada: `DecisionPromotionPolicyV0` y `ArchitectureVoteV0` declaran minimo de votos configurable y aplicable por fase.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: La configuracion concreta por fase queda abierta al director o al consumidor futuro.
```

## Pruebas documentales GOV-003

```text
Caso: GOV-003-CT-001 - voto con opciones explicitas y justificacion
Tipo: contract
Comando: revision documental de `ArchitectureVoteV0`
Evidencia esperada: La votacion exige opciones explicitas, justificacion tecnica compacta, riesgo y plan de validacion.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: No hay serializer ni endpoint todavia; es contrato minimo.
```

```text
Caso: GOV-003-CT-002 - contrato compartido exige director
Tipo: contract
Comando: rg -n "shared_contract_change|shared_contract_requires_director|director_ref" modulos/orquesta-governance/docs/contratos.md
Evidencia esperada: El workflow de votacion marca los cambios de contrato compartido como alcance que exige director para cierre efectivo.
Ultima ejecucion: 2026-05-04; prevista para este write-set.
Riesgos: La regla documental debe reflejarse despues en REST/MCP/CLI.
```

```text
Caso: GOV-003-CT-003 - cierre local solo con ganador claro
Tipo: contract
Comando: revision documental del workflow V2 v0
Evidencia esperada: El cierre local solo se permite con evidencia suficiente, ganador claro, riesgo no alto y alcance no compartido.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: La nocion de ganador claro puede necesitar formalizacion numerica en una version posterior.
```

```text
Caso: GOV-003-CT-004 - historico queda registrado como alternativa descartada
Tipo: contract
Comando: revision documental del workflow V2 v0
Evidencia esperada: El paso final conserva alternativas descartadas y rationale para reuso futuro como evidencia.
Ultima ejecucion: 2026-05-04; documentado en `docs/contratos.md`.
Riesgos: Sin almacenamiento comun todavia, la persistencia queda a cargo del consumidor futuro.
```

## Pruebas automatizables GOV-007

```text
Caso: GOV-007-UNIT-001 - puerto publico devuelve solo effective
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: `QueryGovernanceCatalogPublicV0` devuelve solo entradas `effective` filtradas y contadores compactos de `proposed`/`quarantine`; el resultado explicita `current_block=effective`.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores.
Riesgos: Usa proveedor inyectable en memoria; no valida un servidor externo.
```

```text
Caso: GOV-007-UNIT-002 - fallo del proveedor no expone detalle interno
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: Un proveedor fallido se traduce a `governance_catalog_source_unavailable`.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores.
Riesgos: La causa real del proveedor no se serializa; queda para logs del adaptador externo si existieran.
```

```text
Caso: GOV-007-UNIT-003 - handler HTTP read-only responde JSON canonico
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: `GovernanceCatalogQueryHTTPHandlerV0` acepta `POST /api/v0/governance/catalog/query`, deserializa el request compacto y devuelve JSON con `effective` y contadores.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores.
Riesgos: El test usa `httptest`; no cubre routing ni middleware de otro modulo.
```

```text
Caso: GOV-007-UNIT-004 - handler rechaza metodo o JSON invalido
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: `GET` devuelve `governance_catalog_method_not_allowed` y payloads con campos desconocidos o multiple JSON devuelven `governance_catalog_invalid_request`.
Ultima ejecucion: 2026-05-04; ejecutada con `go test -count=1 ./modulos/orquesta-governance` sin errores.
Riesgos: Solo valida errores publicos compactos del slice local.
```

```text
Caso: GOV-007-UNIT-005 - proyeccion publica aplica budget y frescura
Tipo: unit
Comando: go test -count=1 ./modulos/orquesta-governance
Evidencia esperada: `QueryGovernanceCatalogPublicV0` normaliza
`output_budget`, trunca `effective` por entradas/bytes con razon estable,
preserva contadores totales filtrados, publica `schema_version`,
`catalog_version`, `freshness`/`source_refs` y resume `proposed`/`quarantine`
por conteo/refs sin payload completo.
Ultima ejecucion: 2026-05-26; pendiente de reejecucion en este corte.
Riesgos: La frescura se deriva de refs declaradas en catalogo; si una fuente
externa no las aporta, el contrato publica razon estable en vez de inventarlas.
```
