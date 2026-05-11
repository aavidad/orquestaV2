# Pruebas locales: orquesta-i18n-docs

Registra pruebas obligatorias del modulo.

## Plantilla

```text
Caso:
Tipo: unit | contract | integration | smoke
Comando:
Evidencia esperada:
Ultima ejecucion:
Riesgos:
```

## Pruebas previstas

```text
Caso: contract:generar-i18n-docs-iniciales-v0
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-i18n-docs`
Evidencia esperada: `BuildAppI18nDocsPlanV0` desde `PlanSeedV0` minima produce AppI18nDocsPlanV0 con web_bundle, factory_bundle y docs_bundle; el plan valida con `ValidateAppI18nDocsPlanV0`.
Ultima ejecucion: 2026-05-04; paquete `orquesta/modulos/orquesta-i18n-docs` OK.
Riesgos: El puerto completo desde AppSpecV0 real queda para adaptador futuro; el corte actual no importa internals de factory y cubre la semilla local minima.
```

```text
Caso: contract:i18n-bundle-v0
Tipo: contract
Comando: `npx --yes ajv-cli test --spec=draft2019 --strict=false --validate-formats=false -s docs/schemas/i18n_bundle_v0.schema.json -d docs/fixtures/i18n_docs_v0/bundle_claves_duplicadas_invalido.json --invalid`
Evidencia esperada: El schema rechaza `required_keys` duplicadas y mantiene una estructura unica para `scope=web` y `scope=factory`.
Ultima ejecucion: 2026-05-04; fixture invalido rechazado por `uniqueItems` en `/required_keys`.
Riesgos: Correspondencia dinamica entre `locales`, `catalogs` y cobertura de `required_keys` requiere harness posterior.
```

```text
Caso: contract:i18n-catalog-v0
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-i18n-docs`
Evidencia esperada: `plan_minimo_valido.json` valida y un catalogo incompleto falla con `clave_requerida_faltante`; la paridad entre `locales` y `catalogs` falla con `locale_sin_catalogo`.
Ultima ejecucion: 2026-05-04; paquete `orquesta/modulos/orquesta-i18n-docs` OK.
Riesgos: El harness no valida mezcla semantica de idiomas ni calidad de traduccion; solo relaciones estructurales entre DTOs.
```

```text
Caso: contract:i18n-skeleton-loader-shape-v0
Tipo: contract
Comando: pendiente hasta existir comparador; objetivo `npm test -- contract:i18n-skeleton-loader-shape-v0` o equivalente del modulo.
Evidencia esperada: Skeleton y loader producen el mismo `structure_version`, `catalog_path_pattern`, namespaces y `required_keys_hash`; cualquier divergencia falla con `skeleton_loader_desalineado`.
Ultima ejecucion: no ejecutada; no hay implementacion ni harness en este slice.
Riesgos: Adaptadores de skeleton podrian acoplarse a detalles de framework si el shape no se mantiene portable.
```

```text
Caso: contract:docs-bundle-v0
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-i18n-docs`
Evidencia esperada: `docs_bundle.default_locale` debe pertenecer a `locales`; cada locale debe cubrir todos los `required_doc_types` o falla con `tipo_documento_faltante`.
Ultima ejecucion: 2026-05-04; paquete `orquesta/modulos/orquesta-i18n-docs` OK.
Riesgos: El harness no valida contenido markdown ni disponibilidad real de plantillas; eso queda para generador/adaptador futuro.
```

```text
Caso: contract:app-i18n-docs-plan-v0-relacional
Tipo: contract
Comando: `go test -count=1 ./modulos/orquesta-i18n-docs`
Evidencia esperada: `plan_minimo_valido.json` valida; fixtures inline cubren `default_locale_fuera_de_catalogo`, `locale_sin_catalogo`, `clave_requerida_faltante`, `tipo_documento_faltante` y `fallback_locale_invalido`.
Ultima ejecucion: 2026-05-04; paquete `orquesta/modulos/orquesta-i18n-docs` OK.
Riesgos: No ejecuta generador, no escribe filesystem productivo y no invoca LLM; valida solo DTOs serializables ya construidos.
```

```text
Caso: unit:generated-doc-v0
Tipo: unit
Comando: pendiente hasta existir implementacion; objetivo `npm test -- unit:generated-doc-v0` o equivalente del modulo.
Evidencia esperada: Documento sin `locale`, sin `title_key`, con seccion sin clave o con orden duplicado falla con error estable.
Ultima ejecucion: no ejecutada; no hay codigo en este slice documental.
Riesgos: Markdown generado podria incluir texto visible fuera de claves i18n si no se valida por seccion.
```

```text
Caso: integration:app-spec-v0-to-plan-v0
Tipo: integration
Comando: pendiente hasta existir conector de prueba; corte actual cubierto por `go test -count=1 ./modulos/orquesta-i18n-docs`.
Evidencia esperada: Un adaptador futuro traduce AppSpecV0 publico a PlanSeedV0 y obtiene plan serializable sin filesystem, DB, runtime, LLM ni proveedor concreto.
Ultima ejecucion: 2026-05-04; builder local desde PlanSeedV0 ejecutado, integracion AppSpecV0 real pendiente.
Riesgos: Puede requerir fixtures canonicos de factory; no deben copiarse internals ni schemas privados.
```

```text
Caso: smoke:docs-locales-declared
Tipo: smoke
Comando: `go test -count=1 ./modulos/orquesta-i18n-docs`
Evidencia esperada: El builder genera un documento por locale y por `required_doc_types`; cada documento declara `locale`, `doc_type`, claves de titulo/contenido y formato markdown.
Ultima ejecucion: 2026-05-04; paquete `orquesta/modulos/orquesta-i18n-docs` OK.
Riesgos: Plantillas monolingues podrian ocultar faltantes de catalogo cuando exista adaptador de materializacion.
```

```text
Caso: unit:plan-seed-v0-builder
Tipo: unit
Comando: `go test -count=1 ./modulos/orquesta-i18n-docs`
Evidencia esperada: Semilla vacia produce plan valido; la misma semilla produce salida determinista; default locales se incluyen en catalogos; una semilla equivalente al fixture valido conserva required_keys, namespaces y required_doc_types.
Ultima ejecucion: 2026-05-04; paquete `orquesta/modulos/orquesta-i18n-docs` OK.
Riesgos: No genera traducciones completas para todos los idiomas; usa mensajes iniciales contratados dentro de catalogos i18n.
```
