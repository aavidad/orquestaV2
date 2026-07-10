# Tarea: tools de ingesta de datos y presentaciones

Estado: contrato local de ingesta implementado; sin adaptadores reales ni apps
temporales.

## Alcance

- `data_ingestion`: CSV, XLSX, ODS, JSON y Parquet; discovery, lectura
  streaming, perfilado, mapping, validacion, provenance y recibo.
- Conectores de solo lectura para SQLite, MySQL y PostgreSQL por puertos/ref
  opacas. Drivers, DSN, credenciales, red y SQL dialecto quedan fuera del core
  y entran solo por adaptadores opt-in.
- `presentation_extraction`: PPTX y ODP como documento paginado por
  diapositivas, con texto, tablas, notas, medios y evidencia espacial. Reutiliza
  la IR documental; no duplica OCR ni el SDK de tools.

## Restricciones

No fijar libreria, proveedor, licencia ni precision sin corpus autorizado,
benchmark reproducible, politica PII y pruebas de compatibilidad. Los tres
frentes se integran mediante `ToolBundle/AttachPlan` y autoridades/snapshots,
no mediante imports directos desde el nucleo.

Las olas `wave-data-ingestion-tool-architecture-20260710` y
`wave-presentation-extraction-tool-20260710` se pararon sin entrega por el
presupuesto de diagnostico. Deben relanzarse desde supervisor goal-first una
vez integrado el residual de 208S, con write-sets nuevos y disjuntos.

Las capabilities son externas al nucleo hexagonal: `core`, `goal` y
`orchestration-core` no pueden depender de ingesta, extraccion documental ni
del SDK de tools. La composicion de una app resuelve esos bundles por puertos,
autoridades y snapshots; los adaptadores reales se montan fuera de esos modulos.

## Corte local posterior

`modulos/orquesta-data-ingestion` ya aporta source/profiler/mapper/validator/
receipt por puertos, conserva source kind, hash, snapshot y provenance, y
rechaza formatos fuera del contrato. Sus pruebas usan fakes sin ficheros ni
drivers. Falta conectar su descriptor al SDK de tools y crear los adaptadores
opt-in reales; no se ha aceptado ningun parser ni conector de BBDD todavia.
