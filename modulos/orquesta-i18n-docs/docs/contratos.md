# Contratos locales: orquesta-i18n-docs

Registra puertos, DTOs y eventos que `orquesta-i18n-docs` expone o consume.

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

## Contratos del flujo nueva app

Estado compartido: `GenerarI18nDocsIniciales v0` fue promovido por direccion a contrato compartido en `../../CONTRATOS.md`. Este archivo mantiene el detalle canonico local del puerto, DTOs, errores e invariantes; los schemas y fixtures listados abajo son los canonicos locales referenciados por el contrato global.

Schemas canonicos locales del primer corte material:

- `docs/schemas/app_i18n_docs_plan_v0.schema.json`
- `docs/schemas/i18n_bundle_v0.schema.json`
- `docs/schemas/docs_bundle_v0.schema.json`

Fixtures locales iniciales:

- `docs/fixtures/i18n_docs_v0/plan_minimo_valido.json`
- `docs/fixtures/i18n_docs_v0/plan_sin_locale_invalido.json`
- `docs/fixtures/i18n_docs_v0/bundle_claves_duplicadas_invalido.json`

Nota de validacion v0: JSON Schema cubre estructura, campos obligatorios, enums, patrones, unicidad de arrays y ausencia de campos no contratados. Las invariantes relacionales dinamicas, como `default_locale` incluido en `locales`, correspondencia exacta entre `locales` y `catalogs`, cobertura de `required_keys` en cada catalogo, y completitud de `required_doc_types` por locale, quedan como validacion de contrato posterior hasta existir harness local.

```text
Nombre: GenerarI18nDocsIniciales
Tipo: puerto_entrada
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: orquesta-factory
Campos:
  - app_spec: AppSpecV0 validada por orquesta-factory mediante SolicitarNuevaApp v0.
  - target: web | factory | docs | all.
  - locales_ui: lista no vacia de codigos BCP-47 normalizados para UI.
  - locales_docs: lista no vacia de codigos BCP-47 normalizados para documentacion.
  - default_locale_ui: codigo incluido en locales_ui.
  - default_locale_docs: codigo incluido en locales_docs.
  - requested_outputs: i18n_web | i18n_factory | docs_user | docs_developer | docs_system.
Invariantes:
  - i18n es default: el puerto no acepta `requested_outputs` sin catalogos i18n cuando hay texto visible de UI, app o docs.
  - No persiste archivos, no escribe filesystem y no invoca LLM; devuelve un plan serializable para adaptadores.
  - No crea apps, tareas, proyectos ni backlog.
  - `app_spec` se trata como contrato publico; este modulo no importa internals de factory.
  - Si `target` es `all`, debe producir catalogos web, catalogos factory y docs iniciales.
Errores:
  - app_spec_requerida
  - app_spec_no_validada
  - idioma_ui_requerido
  - idioma_docs_requerido
  - idioma_invalido
  - default_locale_fuera_de_catalogo
  - salida_requerida_incompatible
  - plantilla_no_disponible
Pruebas de contrato:
  - AppSpecV0 valida con locales UI/docs produce AppI18nDocsPlanV0.
  - AppSpecV0 sin locales docs falla con idioma_docs_requerido.
  - requested_outputs solo docs con textos visibles falla si no incluye catalogo i18n.
```

```text
Nombre: PlanSeedV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: tests locales y adaptadores futuros del puerto GenerarI18nDocsIniciales v0.
Campos:
  - app_id_hint: identificador logico estable opcional; se normaliza como slug si llega vacio o con separadores.
  - app_title: texto base de titulo de app que se materializa dentro de catalogos i18n.
  - primary_action_label: texto base de accion primaria que se materializa dentro de catalogos i18n.
  - ui_locales: lista opcional de locales UI; si falta se usa `es-ES`.
  - docs_locales: lista opcional de locales docs; si falta se usa el default de docs.
  - ui_default_locale: locale default UI; si falta se usa `es-ES`.
  - docs_default_locale: locale default docs; si falta se usa `ui_default_locale`.
Invariantes:
  - Es una semilla local minima, no un alias ni import de AppSpecV0 de factory.
  - El builder puro `BuildAppI18nDocsPlanV0` no escribe filesystem, no invoca LLM, DB, runtime ni plantillas reales.
  - El builder siempre produce web_bundle, factory_bundle y docs_bundle por defecto.
  - Los default locales se incluyen en sus listas y catalogos para que el plan sea validable por `ValidateAppI18nDocsPlanV0`.
Errores:
  - No aplica como DTO local; la validacion de relaciones pertenece a `ValidateAppI18nDocsPlanV0`.
Pruebas de contrato:
  - Semilla vacia produce un AppI18nDocsPlanV0 valido.
  - La misma semilla produce JSON determinista.
  - Una semilla equivalente al fixture valido conserva claves obligatorias, namespaces y tipos documentales.
```

```text
Nombre: AppI18nDocsPlanV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: orquesta-factory, adaptadores futuros de filesystem o preview.
Campos:
  - contract_version: literal "v0".
  - app_id_hint: identificador logico derivado de AppSpecV0, no persistente.
  - source_contract: literal "AppSpecV0".
  - ui_default_locale: codigo BCP-47.
  - docs_default_locale: codigo BCP-47.
  - web_bundle: I18nBundleV0 opcional segun target/requested_outputs.
  - factory_bundle: I18nBundleV0 opcional segun target/requested_outputs.
  - docs_bundle: DocsBundleV0 opcional segun target/requested_outputs.
  - skeleton_loader_shape: I18nSkeletonLoaderShapeV0.
  - warnings: lista de advertencias no bloqueantes con codigo estable.
Invariantes:
  - Debe incluir al menos un bundle i18n cuando exista cualquier salida con texto visible.
  - `web_bundle.structure_version`, `factory_bundle.structure_version` y `skeleton_loader_shape.structure_version` coinciden cuando existen.
  - No contiene rutas absolutas, secretos, credenciales ni detalles de proveedor.
Errores:
  - No aplica como DTO; los errores pertenecen al puerto.
Pruebas de contrato:
  - Serializa de forma determinista con claves ordenables.
  - Rechaza plan con skeleton_loader_shape distinto a los bundles.
```

```text
Nombre: I18nBundleV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: orquesta-web, orquesta-factory, adaptadores de generacion.
Campos:
  - structure_version: literal "i18n-bundle/v0".
  - scope: web | factory.
  - default_locale: codigo BCP-47 incluido en locales.
  - locales: lista no vacia de codigos BCP-47.
  - catalogs: mapa locale -> I18nCatalogV0.
  - required_keys: lista ordenada de claves obligatorias.
  - namespace: identificador estable del dominio de claves.
Invariantes:
  - La estructura del bundle es unica para web y factory; solo cambia `scope`.
  - Cada locale declarado tiene un catalogo.
  - Cada catalogo contiene todas las `required_keys`.
  - Las claves son estables, legibles y testeables; no incluyen IDs aleatorios ni texto traducido.
  - No hay texto visible de UI fuera de `catalogs`.
Errores:
  - locale_sin_catalogo
  - clave_requerida_faltante
  - clave_inestable
  - estructura_bundle_incompatible
Pruebas de contrato:
  - Mismo schema valida bundles con scope web y factory.
  - Catalogo incompleto falla por clave_requerida_faltante.
```

```text
Nombre: I18nCatalogV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: I18nBundleV0.
Campos:
  - locale: codigo BCP-47.
  - messages: mapa key -> mensaje localizado.
  - metadata: mapa opcional con source, generated_at omitible en fixtures y notes.
Invariantes:
  - `locale` coincide con la clave del mapa `catalogs`.
  - `messages` no contiene claves vacias.
  - Los mensajes pueden estar vacios solo si la clave se marca como pendiente en metadata.
  - No se mezclan idiomas dentro del mismo catalogo salvo nombres propios o terminos tecnicos.
Errores:
  - locale_no_coincide
  - mensaje_faltante
  - mensaje_vacio_sin_pendiente
Pruebas de contrato:
  - Catalogo minimo con todas las claves obligatorias valida.
  - Locale del catalogo distinto al contenedor falla.
```

```text
Nombre: I18nSkeletonLoaderShapeV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: orquesta-web, orquesta-factory, adaptadores de skeleton/loader.
Campos:
  - structure_version: literal "i18n-bundle/v0".
  - loader_contract: literal "i18n-loader/v0".
  - catalog_path_pattern: patron relativo y portable para resolver catalogos.
  - default_locale_strategy: app_spec | explicit.
  - fallback_locale: codigo BCP-47 incluido en el bundle correspondiente.
  - required_namespaces: lista ordenada de namespaces esperados por skeleton y loader.
  - required_keys_hash: hash determinista de required_keys normalizadas.
Invariantes:
  - Skeleton y loader comparten `structure_version`, `catalog_path_pattern`, namespaces y hash de claves.
  - No usa rutas absolutas ni detalles de framework como contrato.
  - El fallback locale pertenece al bundle que se cargara.
Errores:
  - skeleton_loader_desalineado
  - patron_catalogo_invalido
  - fallback_locale_invalido
Pruebas de contrato:
  - Shape calculado desde skeleton y loader produce el mismo required_keys_hash.
  - Diferente namespace entre skeleton y loader falla.
```

```text
Nombre: DocsBundleV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: orquesta-factory, adaptadores de generacion documental.
Campos:
  - structure_version: literal "docs-bundle/v0".
  - default_locale: codigo BCP-47 incluido en locales.
  - locales: lista no vacia de codigos BCP-47.
  - docs: lista de GeneratedDocV0.
  - required_doc_types: user_manual | developer_manual | systems_manual.
  - template_set: identificador versionado de plantillas.
Invariantes:
  - Cada documento declara locale y tipo.
  - Para cada locale existe al menos un documento por cada required_doc_types solicitado.
  - Titulos y contenidos visibles se referencian por claves i18n o por plantilla localizada, no por texto hardcodeado sin contrato.
  - No contiene rutas absolutas ni decisiones de persistencia.
Errores:
  - documento_sin_idioma
  - tipo_documento_faltante
  - plantilla_no_disponible
  - contenido_sin_clave_i18n
Pruebas de contrato:
  - Bundle con manual de usuario, desarrollador y sistemas por locale valida.
  - Documento sin locale falla.
```

```text
Nombre: GeneratedDocV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: DocsBundleV0.
Campos:
  - doc_id: identificador estable dentro del bundle.
  - doc_type: user_manual | developer_manual | systems_manual.
  - locale: codigo BCP-47.
  - title_key: clave i18n estable.
  - content_key: clave i18n estable o referencia de plantilla localizada.
  - format: markdown.
  - sections: lista ordenada de DocSectionV0.
Invariantes:
  - `format` inicial es markdown.
  - Cada seccion tiene clave estable para titulo y contenido.
  - No hay idioma implicito.
Errores:
  - doc_id_duplicado
  - seccion_sin_clave
  - formato_no_soportado
Pruebas de contrato:
  - Documento markdown con secciones localizadas valida.
  - Documento con idioma omitido falla por documento_sin_idioma.
```

```text
Nombre: DocSectionV0
Tipo: dto
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: GeneratedDocV0.
Campos:
  - section_id: identificador estable dentro del documento.
  - title_key: clave i18n estable.
  - content_key: clave i18n estable o referencia de plantilla localizada.
  - order: entero >= 0.
  - required: boolean.
Invariantes:
  - `section_id` no se deriva del texto traducido.
  - `order` es unico dentro del documento.
  - Una seccion requerida no puede omitirse en ningun locale generado para el mismo doc_type.
Errores:
  - section_id_duplicado
  - orden_duplicado
  - seccion_requerida_faltante
Pruebas de contrato:
  - Secciones con orden unico y claves estables validan.
  - Seccion requerida faltante en un locale falla.
```

```text
Nombre: I18nDocsPlanGeneradoV0
Tipo: evento
Version: v0
Propietario: orquesta-i18n-docs
Consumidores: observability futuro, factory si necesita preview.
Campos:
  - event_version: literal "v0".
  - app_id_hint: identificador logico no persistente.
  - source_contract: literal "AppSpecV0".
  - locales_ui: lista de codigos BCP-47.
  - locales_docs: lista de codigos BCP-47.
  - outputs: lista de salidas generadas en el plan.
  - warnings_count: entero >= 0.
Invariantes:
  - Evento describe el plan generado; no implica escritura de archivos ni persistencia.
  - No contiene textos completos ni secretos.
Errores:
  - No aplica como evento.
Pruebas de contrato:
  - Evento se emite solo despues de construir AppI18nDocsPlanV0 valido.
```
