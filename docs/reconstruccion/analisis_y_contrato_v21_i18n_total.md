# V21 — i18n total: contrato rojo

Fecha: 2026-07-22. Base: `e311a97e4f`.

Estado: **awaiting_dependency** de V20 (`command_registry`) y producto V21
ausente. Este corte no acredita `UI-18`, no modifica el catálogo y no fija APIs
internas futuras.

## Autoridad única

V21 es dueño de `UI-18`: un único catálogo es owner de todo texto humano
público de web, Wizard, CLI, notificaciones, errores, prompts y documentación
pública. Ningún handler, provider, adapter o binding conserva su propia copia
de texto público ni otro catálogo. Español (`es`) es default y fallback; los
códigos máquina son invariantes y nunca se traducen.

La futura implementación recibe las claves semánticas producidas por V20. V21
no crea comandos, lifecycle, DTOs autoritativos ni bindings HTTP/MCP/CLI. El
E2E de bindings queda diferido hasta que V20 esté acreditada; no se finge una
superficie ni una API de registro durante este corte.

## Contrato de catálogo y formato

Las claves son estables y las locales son tags BCP-47 válidos. Para cada clave
pública, cada catálogo de locale habilitada tiene paridad; una locale ausente o
no soportada cae de forma explícita al valor español. Plurales, fecha, número,
moneda y timezone se formatean por locale y nunca alteran códigos máquina,
refs, digests ni otros datos de protocolo.

La extracción futura recorre las superficies públicas declaradas, compara claves
extraídas con el catálogo y falla por clave faltante, locale sin paridad, tag
inválido, clave no usada o texto humano fuera del catálogo. La documentación
pública usa el mismo owner; los documentos internos de desarrollo no se
presentan como superficie localizada.

## Recovery, seguridad y límites

Una elección de locale no cambia autorización, identidad, lifecycle, receipt ni
efecto. Fallo de catálogo o formatter no revela secretos ni sustituye un código
máquina. No se añade store, scheduler, daemon, configuración paralela o lógica
de i18n específica de proveedor.

## Gate rojo y E2E futuro

El fixture declara el catálogo owner, `es` default/fallback, ejemplos BCP-47,
formatters, extracción/paridad/missing/unused y la dependencia de bindings V20.
`TestAcceptanceV21I18N` valida ese contrato y falla exclusivamente como
`V21_PRODUCT_PENDING` hasta que existan V20 acreditada y producto V21.

El E2E posterior crea una misma respuesta pública por bindings HTTP, MCP y CLI
de V20, comprueba claves y códigos iguales, renderiza las locales habilitadas y
verifica fallback español, plural/formato y rechazo de clave missing/unused. No
hay E2E simulado antes de los bindings reales.

## P/S/E y write-set

`P` contendrá catálogo, extractor, formatters y tests, sin autoacreditación.
`S` sellará árbol, binario, catálogo y configuración efectiva redactada. `E`
ejecutará la matriz de bindings desde `detached_clean` de `S` y escribirá un
receipt V3 fuera del candidato.

Este corte solo puede tocar:

- `docs/reconstruccion/analisis_y_contrato_v21_*.md`;
- `docs/reconstruccion/worksets/v21_*.json`;
- `acceptance/v21_*.go` y `acceptance/fixtures/v21_*.json`.

Producto, roadmap, evidence, trace, configuración y superficies V20 quedan
fuera. Tras V20 sellada, la implementación V21 define sus límites concretos en
un write-set nuevo y separado.
