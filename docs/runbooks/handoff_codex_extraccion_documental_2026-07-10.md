# Handoff Codex: diseño de extracción documental - 2026-07-10

## Alcance cerrado

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
