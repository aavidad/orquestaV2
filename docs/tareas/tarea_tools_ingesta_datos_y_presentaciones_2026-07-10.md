# Tarea: tools de ingesta de datos y presentaciones

Estado: preparada; sin adaptadores reales ni apps temporales.

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
