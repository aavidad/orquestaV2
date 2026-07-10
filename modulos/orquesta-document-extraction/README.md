# orquesta-document-extraction

Contrato neutral y servicio de aplicacion para extraer campos tipados desde
documentos. Define una IR observacional de documento, pagina, bloque, span,
tabla, campo y evidencia; no implementa PDF, OCR, proveedores ni persistencia.

El servicio `ExtractDocumentV0` coordina `DocumentSource`, normalizador,
parser, localizador, extractor por schema, evidencia, validador, revision,
exportadores y recibos. Los schemas de persona y factura solo fijan el tipo de
entidad: sus campos son refs opacas de la app externa.

Los adaptadores de salida viven fuera del modulo:

- `orquesta-document-extraction-json`
- `orquesta-document-extraction-csv`
- `orquesta-document-extraction-fake`

La politica normalizada es local. Cualquier uso cloud exige opt-in explicito y
refs de region, retencion, borrado, cifrado y auditoria.

## Capability reutilizable

`DefaultDocumentToolCapabilityDescriptorV0` declara las surfaces
`documents.inspect.v0`, `documents.extract.v0`, `documents.status.v0`,
`documents.review.v0` y `documents.export.v0`. Todas delegan en
`ExecuteDocumentToolV0`: un caso de uso asíncrono que persiste comandos y
estado por un puerto opaco e idempotente. MCP, HTTP y CLI son adaptadores que
llaman a ese caso de uso, sin lógica de negocio propia; el worker de la
composición ejecuta `ExtractDocumentV0` cuando corresponda.

El descriptor declara perfiles de efectos/permisos, presupuesto, capacidades
requeridas, errores públicos y versión de recibo. `DocumentToolCapabilityRegistryPortV0`
permite a una composición opt-in descubrir conectores y a un wizard proponer la
tool sin acoplarla al proceso de Orquesta.

`DocumentExtractionBundleDescriptorV0` es un contrato fino y provisional para
adjuntar el subsistema a una app en `embedded_module`, `local_sidecar` o
`remote_connector`. Declara módulo o conector, configuración, i18n, pruebas,
hashes y recibo. No sustituye ni duplica el SDK genérico `ToolBundle/AttachPlan`;
ese SDK podrá consumir este descriptor cuando esté disponible.
