# V21: i18n total sobre superficies reales

Fecha: 2026-07-23. Base acreditada:
`8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8`.

Estado: **development_unsealed**. V20 está acreditada; el producto V21 todavía
no existe. Este corte define contrato, pruebas y documentación pública, pero no
acredita `UI-18`, no crea receipt y no promueve el roadmap.

## 1. Decisión

V21 introduce un solo owner de texto humano público. No introduce otro motor,
registro de comandos, lifecycle, DTO, store, daemon, variable de entorno ni
configuración. Reutiliza `api.locale`, con `es` como default y fallback y `en`
como segunda locale instalada.

La afirmación es deliberadamente acotada:

- superficies existentes: registro de comandos, HTTP, MCP, CLI y documentación
  pública V21;
- consumidores futuros con ratchet, no producto fingido: web, Wizard,
  notificaciones y prompts;
- códigos, refs, OIDs, hashes, claves semánticas y demás protocolo máquina
  permanecen invariantes.

V21 no recorre ni traduce todo `docs/`, `modulos/` o material histórico. La
documentación pública está declarada de forma explícita; los documentos de
desarrollo no forman parte de la superficie.

## 2. Dependencia V20

La única base válida es el post-E integrado de V20:

| Estado | OID |
|---|---|
| P producto V20 | `a3da82278a3ce947b4425665d92d961c99b2baaf` |
| S fuente sellada | `7f27685d992c9a8f5f308d94bbfc3e9828009d84` |
| E evidencia | `7b1bbaf475743c16c5749b6e4e039e23b0a8354b` |
| integración post-E | `8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8` |

El receipt `product/evidence/v20_command_registry.json` es V3, PASS y procede
de una fuente `detached_clean`. V21 lo verifica; no lo copia, reescribe ni
convierte en evidencia propia.

La rama histórica `reconstruccion/v21-i18n` partía de `e311a97e4f` y solo sirve
como origen del contrato rojo. El producto se desarrolla exclusivamente sobre
`reconstruccion/v21-i18n-integrated`.

## 3. Manifest canónico

Ruta única prevista: `internal/i18n/manifest.json`.

Campos:

- `schema_version`;
- `default_locale` y `fallback_locale`;
- `enabled_locales`;
- `catalogs`: locale y ruta exacta;
- `surfaces`: identidad, estado, fuentes de claves y política;
- `public_documents`: parejas explícitas `es`/`en`.

Políticas por superficie:

| Superficie | Estado | Política |
|---|---|---|
| `command_registry` | activa | `registry_keys_catalog_values` |
| `mcp` | activa | `catalog_only` |
| `cli` | activa | `catalog_only` |
| `http` | activa | `machine_envelope_catalog_presenter` |
| `public_docs` | activa | `localized_document_bundle` |
| `web`, `wizard`, `notifications`, `prompts` | futura | `catalog_required_before_activation` |

Una superficie futura no declara fuentes ni se presenta como implementada. Al
aparecer en una vertical posterior debe cambiar a activa, declarar sus fuentes
y pasar el mismo extractor.

## 4. Catálogo y extractor

Catálogos instalados:

- `internal/i18n/catalogs/es.json`;
- `internal/i18n/catalogs/en.json`.

Dentro del manifest embebido se declaran como `catalogs/es.json` y
`catalogs/en.json`; las rutas anteriores son su resolución desde la raíz del
repositorio.

El extractor no busca palabras sospechosas ni clasifica contenido con
heurísticas. Solo consume fuentes tipadas declaradas:

1. `description_key` y `error_codes` del registro V20;
2. llamadas tipadas del presenter/catálogo en fuentes activas;
3. parejas documentales declaradas por el manifest.

Gate:

- JSON sin claves duplicadas;
- mismas claves en cada locale;
- claves utilizadas y solo claves utilizadas;
- una clave ausente falla, no se devuelve como supuesto texto;
- variables `{name}` equivalentes por clave y locale;
- documentos públicos no vacíos y pareados;
- ninguna clasificación léxica de contenido ni veto por palabras sueltas.

API mínima esperada en `internal/i18n`:

- `LoadBundled`;
- `Catalog.Resolve`;
- `Catalog.Plural`;
- `Catalog.FormatNumber`;
- `Catalog.FormatCurrency`;
- `Catalog.FormatDate`;
- `Catalog.FormatTimeZone`.

El catálogo cargado es inmutable y seguro para lecturas concurrentes.

## 5. Locales y formato

Tags directos instalados: `es`, `en`.

Variantes regionales compatibles como `es-ES` y `en-US` usan su base. Tags
válidos no instalados, incluidos `ca-ES-valencia` y `zh-Hans`, caen de forma
explícita a español. Formas inválidas como espacios periféricos o `en_US`
fallan; no se normalizan silenciosamente.

La matriz prueba:

- plural singular/other;
- número;
- moneda;
- fecha;
- timezone.

La salida puede variar por locale. Nunca pueden variar `command_id`,
`command_version`, `request_ref`, `project_ref`, `audit_ref`, digests,
`error.code` o `error.message_key`.

## 6. Bindings reales

V21 no mueve la autoridad V20:

- HTTP conserva una respuesta máquina con código y clave;
- MCP obtiene instrucciones/descripciones del catálogo;
- CLI usa el mismo catálogo para ayuda y errores humanos;
- el dispatcher y el registro siguen siendo únicos.

`TestRealHTTPMCPCLIAndI18NParityEndToEnd` debe arrancar la composición real,
invocar HTTP y MCP y ejecutar la CLI como subproceso. Comprueba la misma
identidad semántica/código/clave, fallback español y traducción inglesa. Un
fake, una llamada directa al catálogo o tres respuestas fabricadas no cuentan
como E2E.

## 7. Documentación pública

Pareja inicial:

- `docs/public/es/README.md`;
- `docs/public/en/README.md`.

Ambas describen solo capacidades presentes y la frontera futura. El manifest
las referencia por rutas exactas bajo `orquesta.quickstart`. La paridad es
contractual, no identidad literal entre idiomas.

## 8. Simplicidad

Máximos físicos:

| Superficie | LOC |
|---|---:|
| producto `internal/i18n` | 900 |
| integración bindings añadida | 500 |
| catálogo + manifest | 800 |
| documentación pública | 250 |
| delta vendorizado `golang.org/x/text` | 12.000 |

Además: dos locales instaladas, cinco superficies activas, cuatro ratchets
futuros, máximo 256 claves y un máximo de 40 ficheros nuevos del único módulo
vendorizado permitido. El delta real previo a P es 35 ficheros Go y 11.447
líneas de `x/text`; se conserva porque delegar CLDR, BCP-47 y el registro ISO en
la biblioteca mantenida por Go es más seguro y mantenible que duplicarlos en
código propio. El coste de tercero se mide aparte y no se presenta como código
Orquesta. Si el producto necesita superar un límite se revisa el diseño; no se
eleva el número para obtener verde.

## 9. P/S/E

### P

Implementación completa y subjects exactos ordenados, sin receipt, output o
promoción del roadmap. Receta:

```bash
{ git diff --name-only 8063d8ce3ed75dd5ec92108ef6e7e73ee8e11cb8 --; git ls-files --others --exclude-standard; } | LC_ALL=C sort -u
```

### S

Commit separado que liga base, commit/tree, blob del fixture y digest
`sha256:length-framed-git-blob-set:v1`. Se verifica desde Git, no desde el
working tree.

### E

Se ejecuta el `execution_argv` exacto desde una fuente `detached_clean` de S.
El receipt `product/evidence/v21_i18n.json` y el output se escriben fuera del
candidato. Solo entonces `AC-V21-I18N` pasa a `executable` y `UI-18` a
`accredited`.

## 10. Estado actual y siguiente acción

Estado previo a congelar P:

- V20: PASS acreditado;
- contrato, manifest y producto V21: implementados y contrarrevisados;
- catálogos, extractor tipado, presentadores y E2E real: verdes;
- fixture: aún `development_unsealed`, pendiente de congelar subjects P;
- receipt/output V21: ausentes;
- roadmap V21: planned;
- `UI-18`: declared.

Siguiente acción: congelar P, ligar S, ejecutar el argv exacto desde S
`detached_clean` y publicar E. No abrir V22 antes de E V21.
