# Handoff Codex: diseño de extracción documental - 2026-07-10

## Corte funcional local - 2026-07-10

Implementado solo en este worktree, sin proveedores, red, despliegue ni PII:

- `modulos/orquesta-document-extraction`: IR versionada para documento/página/
  bloque/span/tabla/campo/evidencia, schemas tipados de persona/factura por
  refs opacas, política local por defecto, contratos de conectores y servicio
  de aplicación con source -> normalizer -> parser -> localizer -> slices ->
  extractor -> evidencia -> validador/review -> exportador -> recibo.
- El servicio preserva raw/normalizado/estado/provenance y no exporta candidatos
  sin evidencia. El recibo registra hash, configuración, locales separados,
  versión de contrato/IR y las identidades de adaptador usadas.
- Adaptadores separados y sin filesystem: JSON, CSV y fakes deterministas.
- Capability reutilizable: descriptor neutral, registro opt-in y único caso de
  uso asíncrono para `documents.inspect.v0`, `documents.extract.v0`,
  `documents.status.v0`, `documents.review.v0` y `documents.export.v0`.
  MCP/API/CLI deben delegar en `ExecuteDocumentToolV0`.
- Descriptor provisional aislado de bundle para `embedded_module`,
  `local_sidecar` y `remote_connector`; incluye módulo/conector,
  config+i18n+tests+hashes y recibo, sin duplicar el SDK ToolBundle/AttachPlan
  que se está realizando en otro worktree.

Pendientes concretos:

- Composición que convierta comandos asíncronos en workers y conecte un
  `DocumentToolOperationStorePortV0` durable.
- Adaptadores opt-in reales (parsers, OCR/layout, schema extractors, review y
  destinos) tras corpus autorizado, benchmark, licencia y política de datos.
- Transporte MCP/API/CLI fino que delegue en el caso de uso y wizard que consulte
  el registro, cuando los módulos de composición correspondientes estén listos.
- Integración con el SDK genérico ToolBundle/AttachPlan desde su worktree, sin
  copiar sus tipos ni su lógica aquí.

Verificación local realizada:

```text
go test -count=1 -race ./modulos/orquesta-document-extraction ./modulos/orquesta-document-extraction-fake ./modulos/orquesta-document-extraction-json ./modulos/orquesta-document-extraction-csv
go test -count=1 -run '^TestNeutralOrchestrationPackagesDoNotImportProductAdapters$' .
git diff --check
```

Las tres verificaciones anteriores pasaron. `go test -count=1 .` sigue fallando
fuera de este corte en `TestEnvVarsBudgetMEJ106V0`: contabiliza 521 variables
`ORQUESTA_*` frente al máximo 513. No se modificó ese presupuesto ni sus
variables desde este worktree; clasificar y resolver esa desviación en su frente
propio antes de usarlo como verde raíz.

## Alcance original

Se redactaron, sin código, pruebas, despliegue ni cambios en `uso-app` u otras
apps:

- `docs/inventario_herramientas_extraccion_documental_2026-07-10.md`
- `docs/diseno_subsistema_extraccion_documental_2026-07-10.md`

El inventario separa parser/layout/OCR, extractores guiados por esquema y
servicios gestionados. Para cada candidato aporta enlace oficial/primario, rol,
entradas/salidas, modalidad, licencia/privacidad/hardware cuando están
confirmados, fortalezas, límites y encaje. Distingue evidencia independiente de
claims de proveedor; no contiene precios ni benchmarks inventados.

El diseño fija puertos neutrales, IR con evidencia espacial, pipeline
reproducible, recibos, privacidad, i18n, validación, evaluación empírica,
ejemplo de personas, recomendación inicial y preparación del wizard.

## Decisiones que debe conservar el trabajo posterior

- Orquesta sigue siendo núcleo genérico: no introducir PDF/OCR/modelos,
  proveedores, PII, SDKs o reglas DNI/NIE/NIF dentro del core.
- Usar refs opacas y puertos `DocumentSource`, `DocumentNormalizer`,
  `DocumentParser`, `SchemaExtractor`, `EvidenceLocator`, `Validator`,
  `HumanReview`, `Exporter` y `ReceiptStore`.
- JSON es intercambio, no destino único; exportar desde la IR/campos aceptados.
- Localizar páginas y partir esquema antes de extracción. No diseñar un esquema
  gigante one-shot: ExtractBench documenta degradación severa y posible 0 % de
  output válido con esquemas amplios.
- PII local por defecto; cloud solo opt-in con región, retención, borrado,
  cifrado y auditoría en adaptador/composición.
- Un recibo reproducible exige versiones/revisiones exactas; nunca alias
  `latest` como identidad suficiente.
- La elección de herramienta sigue pendiente de corpus propio versionado, gold
  humano y A/B/ratchet; ninguna capacidad de proveedor se ha aceptado como
  precisión demostrada.

## Excepción operativa

Este lanzamiento directo de Codex es una excepción documentada: el binario vivo
de Orquesta aún no ha superado F3/F5. El trabajo futuro de implementación,
evaluación e integración debe volver a ejecutarse y coordinarse por Orquesta
cuando esa frontera esté resuelta; no convertir esta excepción documental en
flujo normal.

## Siguiente acción del operador

Cuando Orquesta esté disponible, abrir un trabajo goal-first de diseño/plan de
implementación con write-sets separados para: contratos puros e IR, adaptadores
locales, política/recibo, corpus de evaluación y composición/wizard. No iniciar
implementación productiva ni enviar PII a cloud sin política explícita y corpus
de evaluación autorizado.

estado_final: ready_for_operator_pdf_extraction_design
