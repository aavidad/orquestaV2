# Tareas locales: orquesta-i18n-docs

Cada tarea debe ser pequena y cerrada.

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

## Microtareas iniciales

```text
ID: I18N-011
Objetivo: Publicar `ActiveI18nDocsCompositionOwnerV0` como proyeccion activa T75 para factory, web y MCP.
Write-set: i18n_docs_owner_v0.go, i18n_docs_owner_v0_test.go, docs/contratos.md, docs/decisiones.md, docs/pruebas.md
Simbolo foco: ActiveI18nDocsCompositionOwnerV0
Contrato: AppI18nDocsPlanV0 -> owner activo de bundles, loader shape, fallback locale, required keys y docs generadas.
Validacion: `go test -count=1 ./modulos/orquesta-i18n-docs ./modulos/orquesta-factory ./modulos/orquesta-web ./modulos/orquesta-mcp`.
Bloqueos: Ninguno; los adaptadores de filesystem/materializacion real siguen fuera del owner puro.
Estado: completada
```

```text
ID: I18N-000
Objetivo: Alinear alcance del mini-proyecto con SolicitarNuevaApp v0 y registrar que i18n/docs son default en nueva app.
Write-set: docs/decisiones.md, docs/contratos.md, docs/tareas.md, docs/pruebas.md
Simbolo foco: GenerarI18nDocsIniciales v0
Contrato: SolicitarNuevaApp v0 -> AppSpecV0 -> AppI18nDocsPlanV0
Validacion: Lectura de AGENTS.md, README.md, docs locales y fragmento autorizado de ../../CONTRATOS.md.
Bloqueos: Ninguno para documentacion local.
Estado: completada
```

```text
ID: I18N-001
Objetivo: Definir contrato local del puerto de entrada para generar catalogos i18n y docs iniciales desde AppSpecV0.
Write-set: docs/contratos.md
Simbolo foco: GenerarI18nDocsIniciales
Contrato: puerto_entrada v0
Validacion: Contrato promovido en `../../CONTRATOS.md`; prueba ejecutable pendiente `contract:generar-i18n-docs-iniciales-v0` hasta existir harness local.
Bloqueos: Ninguno. Direccion promovio `GenerarI18nDocsIniciales v0` a contrato compartido factory -> i18n-docs.
Estado: completada/promovida
```

```text
ID: I18N-002
Objetivo: Definir DTO de salida unico para el plan serializable de i18n/docs sin escribir filesystem.
Write-set: docs/contratos.md
Simbolo foco: AppI18nDocsPlanV0
Contrato: dto v0
Validacion: Prueba de contrato pendiente `contract:app-i18n-docs-plan-v0`.
Bloqueos: Ninguno.
Estado: documentada
```

```text
ID: I18N-003
Objetivo: Definir estructura unica de bundle i18n para web y factory.
Write-set: docs/contratos.md, docs/decisiones.md
Simbolo foco: I18nBundleV0
Contrato: dto v0
Validacion: Prueba de contrato pendiente `contract:i18n-bundle-v0`.
Bloqueos: Ninguno.
Estado: documentada
```

```text
ID: I18N-004
Objetivo: Definir catalogo localizable con claves estables y mensajes testeables.
Write-set: docs/contratos.md
Simbolo foco: I18nCatalogV0
Contrato: dto v0
Validacion: Prueba de contrato pendiente `contract:i18n-catalog-v0`.
Bloqueos: Ninguno.
Estado: documentada
```

```text
ID: I18N-005
Objetivo: Definir huella comun para garantizar que skeleton y loader comparten estructura unica.
Write-set: docs/contratos.md, docs/decisiones.md
Simbolo foco: I18nSkeletonLoaderShapeV0
Contrato: dto v0
Validacion: Prueba de contrato pendiente `contract:i18n-skeleton-loader-shape-v0`.
Bloqueos: Ninguno.
Estado: documentada
```

```text
ID: I18N-006
Objetivo: Definir bundle de documentacion generada con idioma declarado por documento.
Write-set: docs/contratos.md, docs/decisiones.md
Simbolo foco: DocsBundleV0
Contrato: dto v0
Validacion: Prueba de contrato pendiente `contract:docs-bundle-v0`.
Bloqueos: Ninguno.
Estado: documentada
```

```text
ID: I18N-007
Objetivo: Definir documento generado inicial para manual de usuario, desarrollador y sistemas.
Write-set: docs/contratos.md
Simbolo foco: GeneratedDocV0
Contrato: dto v0, DocSectionV0
Validacion: Prueba de contrato pendiente `contract:generated-doc-v0`.
Bloqueos: Ninguno.
Estado: documentada
```

```text
ID: I18N-008
Objetivo: Registrar pruebas previstas de contrato, integracion simulada y smoke documental para el primer slice.
Write-set: docs/pruebas.md
Simbolo foco: pruebas i18n-docs v0
Contrato: GenerarI18nDocsIniciales v0, I18nBundleV0, DocsBundleV0
Validacion: docs/pruebas.md contiene casos unit, contract, integration y smoke.
Bloqueos: Comandos reales pendientes hasta que exista implementacion.
Estado: documentada
```

```text
ID: I18N-009
Objetivo: Crear schemas y fixtures locales para los DTO v0 documentados.
Write-set: docs/schemas/i18n_bundle_v0.schema.json, docs/schemas/docs_bundle_v0.schema.json, docs/schemas/app_i18n_docs_plan_v0.schema.json, docs/fixtures/i18n_docs_v0/*.json, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: docs/schemas y docs/fixtures futuros.
Contrato: I18nBundleV0, DocsBundleV0, AppI18nDocsPlanV0
Validacion: `jq` para sintaxis JSON; JSON Schema contra fixture valido e invalidos si `ajv` o `python -m jsonschema` esta disponible; `git diff --check`.
Bloqueos: Ninguno en este corte. Las invariantes relacionales dinamicas requieren harness posterior.
Estado: completada
```

```text
ID: I18N-010
Objetivo: Ejecutar harness relacional inicial y builder puro determinista para AppI18nDocsPlanV0 desde PlanSeedV0 minima, sin filesystem productivo ni LLM.
Write-set: i18n_docs_contract_v0.go, i18n_docs_contract_v0_test.go, i18n_docs_plan_builder_v0.go, i18n_docs_plan_builder_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/contratos.md, docs/decisiones.md
Simbolo foco: BuildAppI18nDocsPlanV0, PlanSeedV0, ValidateAppI18nDocsPlanV0
Contrato: PlanSeedV0, AppI18nDocsPlanV0, I18nBundleV0, DocsBundleV0, I18nSkeletonLoaderShapeV0
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-i18n-docs`; `git diff --check -- modulos/orquesta-i18n-docs`.
Bloqueos: Ninguno. El builder produce DTOs serializables en memoria y no importa internals de factory.
Estado: completada
```
