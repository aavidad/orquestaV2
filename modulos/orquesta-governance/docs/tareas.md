# Tareas locales: orquesta-governance

Cada tarea debe ser pequena y cerrada.

## Backlog inicial desde DB v1

```text
ID: GOV-001
Objetivo: Extraer reglas, skills y workflows de DB v1 a un catalogo documental v0 sin activarlos automaticamente.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md
Simbolo foco: GovernanceCatalogV0
Contrato: GovernanceCatalog v0
Validacion: catalogo separa reglas globales, reglas por rol, workflows y skills; cada entrada conserva origen DB v1, estado propuesto y criterio de promocion.
Bloqueos: DBV1-000 completada en docs/reinicio_orquesta_v2/inventario_db_v1.md.
Estado: completada 2026-05-04; catalogo documental v0 registrado en docs/contratos.md, decisiones en docs/decisiones.md y pruebas previstas en docs/pruebas.md.
```

```text
ID: GOV-002
Objetivo: Definir regla de promocion para propuestas/votos historicos de OP-001..OP-124.
Write-set: docs/decisiones.md, docs/contratos.md
Simbolo foco: DecisionRecordV0
Contrato: GovernanceCatalog v0, DecisionRecord v0 futuro
Validacion: las propuestas con consenso pueden ser evidencia; las rechazadas o duplicadas no mandan sin nueva decision del director.
Bloqueos: GOV-001 completada; `GovernanceCatalog v0` ya esta promovido como contrato compartido, pero `DecisionRecord v0` sigue pendiente.
Estado: completada 2026-05-04; `DecisionPromotionPolicyV0` documenta que OP-001..OP-124 son evidencia historica no autoritativa, con minimos de votos configurables por fase y escalado al director para empate, riesgo alto o intento de promocion efectiva.
```

```text
ID: GOV-003
Objetivo: Definir workflow de votacion V2 para brainstorming y cambios de contrato compartido.
Write-set: docs/contratos.md, docs/pruebas.md
Simbolo foco: ArchitectureVoteV0
Contrato: DecisionRecord v0 futuro
Validacion: minimo de votos configurable, no autor obligatorio cuando aplique, cierre por criterio tecnico y decision del director si hay empate o riesgo alto.
Bloqueos: GOV-002.
Estado: completada 2026-05-04 a nivel documental/contractual; `ArchitectureVoteV0` define workflow V2 minimo para brainstorming y cambios de contrato compartido, sin tocar core ni exponer API todavia.
```

```text
ID: GOV-004
Objetivo: Formalizar `GovernanceCatalogV0` como schema canonico draft7 con catalogos `effective`, `proposed` y `quarantine`.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md, docs/schemas/governance_catalog_v0.schema.json, docs/fixtures/governance_catalog_v0/*
Simbolo foco: GovernanceCatalogV0
Contrato: GovernanceCatalog v0
Validacion: 1 fixture positivo y 2 negativos validados con `jq` y `npx --yes ajv-cli`; los historicos DB v1 no pueden estar efectivos sin decision de promocion, y el control de secretos no puede ser solo texto libre.
Bloqueos: GOV-001 completada; contrato promovido globalmente en `../../CONTRATOS.md`; queda pendiente un harness de consulta compacto antes de conectar core/MCP como consumidores reales.
Estado: completada 2026-05-04; schema/fixtures registrados como canonicos del modulo propietario y referenciados por el contrato compartido global.
```

```text
ID: GOV-005
Objetivo: Alinear documentacion local tras promocion de `GovernanceCatalog v0` a contrato compartido global.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md
Simbolo foco: GovernanceCatalogV0
Contrato: GovernanceCatalog v0 compartido
Validacion: docs locales no mantienen la consulta de promocion como abierta ni tratan `GovernanceCatalogV0` como pendiente/local; `git diff --check -- modulos/orquesta-governance` sin errores.
Bloqueos: Decision del director registrada en `../../CONTRATOS.md`.
Estado: completada 2026-05-04.
```

```text
ID: GOV-006
Objetivo: Crear DTOs Go y consulta pura en memoria para `GovernanceCatalogV0`, filtrando `effective` por modulo, rol, fase y tags.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md, docs/schemas/governance_catalog_v0.schema.json, docs/fixtures/governance_catalog_v0/catalog_minimo_valido.json, governance_catalog_v0.go, governance_catalog_v0_test.go
Simbolo foco: QueryEffectiveGovernanceCatalogV0
Contrato: GovernanceCatalog v0 compartido
Validacion: `go test -count=1 ./modulos/orquesta-governance`; `jq empty docs/schemas/governance_catalog_v0.schema.json docs/fixtures/governance_catalog_v0/catalog_minimo_valido.json`; `git diff --check -- modulos/orquesta-governance`.
Bloqueos: No hay bloqueo; `scope.tags` queda como campo opcional documental de consulta y no activa permisos ni historicos.
Estado: completada 2026-05-04.
```

```text
ID: GOV-007
Objetivo: Abrir un slice pequeno y publico de lectura para `GovernanceCatalog v0` que desbloquee el consumo CLI, sin tocar core.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md, governance_catalog_public_query_v0.go, governance_catalog_public_query_v0_test.go
Simbolo foco: GovernanceCatalogPublicQuery v0
Contrato: GovernanceCatalog v0 compartido; ruta/version publica local recomendada `POST /api/v0/governance/catalog/query`
Validacion: `go test -count=1 ./modulos/orquesta-governance`; `git diff --check -- modulos/orquesta-governance`.
Bloqueos: No hay bloqueo tecnico local; si otro modulo adopta este puerto como contrato compartido, el director debe promover ruta/version y DTOs minimos a `../../CONTRATOS.md`.
Estado: completada 2026-05-04; puerto read-only compacto implementado con proveedor inyectable y adaptador HTTP local fino, devolviendo solo `effective` y contadores de `proposed`/`quarantine`.
```

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```
