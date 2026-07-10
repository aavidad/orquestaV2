# Diseño del subsistema reutilizable de extracción documental - 2026-07-10

## Decisión y alcance

**Decisión de diseño:** Orquesta debe dirigir trabajos de extracción de datos
tipados desde documentos sin que el núcleo conozca PDF, OCR, proveedor, modelo,
SDK, PII, formato de salida ni reglas de negocio. Se expone mediante puertos y
refs opacas; las apps externas aportan fuente, esquema de dominio, validadores,
políticas y destinos. Orquesta aporta juicio, planificación y evidencia de
cierre. Este documento no declara implementado ningún puerto ni adaptador.

Cubre PDF born-digital, mixto, escaneos, fotografías, documentos sucios, tablas
y manuscrito. JSON es solo intercambio: CSV, XLSX, Parquet, SQL, XML,
Markdown/HTML, eventos y DTOs son destinos igualmente válidos.

## Principios vinculantes

- Núcleo puro: sin SDK, endpoint, clave, ruta, PII ni tipo de documento de
  negocio; esos datos son de adaptador o composición.
- Refs opacas: `document_ref`, `source_ref`, `schema_ref`, `policy_ref`,
  `receipt_ref`, `artifact_ref` y `destination_ref` no revelan almacenamiento
  interno al núcleo.
- Parser descubre contenido/geometría; extractor propone; validador de dominio
  decide consistencia; persona resuelve ambigüedad; exportador materializa.
- Reproducibilidad: un alias mutable o `latest` nunca identifica por sí solo una
  corrida sensible.
- Incertidumbre preservada: `null`, ausente, ilegible, no aplicable y propuesto
  son distintos. JSON sintácticamente válido no demuestra dato verdadero.
- En flujos sensibles, cada campo aceptado debe tener evidencia de página y
  región/span, salvo excepción explícita de política.

## Puertos del núcleo

| Puerto | Responsabilidad | No decide |
| --- | --- | --- |
| `DocumentSource` | Resuelve fuente opaca, bytes/stream y metadata mínima. | Política o exportación. |
| `DocumentNormalizer` | Rotación, deskew, denoise, renderizado y derivados. | Campos de negocio/modelo. |
| `DocumentParser` | Páginas, texto, layout, tablas y artefactos OCR. | Validación de dominio. |
| `SchemaExtractor` | Candidatos para schema slice sobre contexto localizado. | Aceptación o destino. |
| `EvidenceLocator` | Página, bloque, span y bbox/polígono de un candidato. | Inventar evidencia. |
| `Validator` | Reglas deterministas y normalización de dominio. | OCR/proveedor/estrategia. |
| `HumanReview` | Decisión, corrección y motivo humano auditable. | Persistencia interna de la app. |
| `Exporter` | IR/campos a CSV, XLSX, Parquet, SQL, XML, Markdown/HTML, evento o DTO. | Reextraer o alterar silenciosamente. |
| `ReceiptStore` | Recibos, hashes, versiones y refs de artefactos. | Contener documentos/PII por defecto. |

Los adaptadores concretos (por ejemplo Tika/pdfplumber, OCRmyPDF/PaddleOCR,
Marker/Docling/MinerU, Lift/NuExtract3 o API cloud) proyectan primero a la IR
canónica. La composición, no el core, selecciona adaptador, modelo, revisión,
hardware, credenciales, región, timeout y presupuesto.

## IR canónica neutral

La IR modela observaciones, no una tabla de negocio. Es intercambiable y
versionada; no se identifica con JSON como única persistencia o salida.

```text
Document(document_ref, content_hash, media_kind, detected_kind, page_count, language_hints)
  Page(page_ref, index, dimensions, image_hash, transform_chain)
    Block(block_ref, kind, reading_order, bbox/polygon, confidence, source_refs)
      Span(span_ref, text_raw, bbox/polygon, confidence, source_refs)
      Table(table_ref, bbox/polygon, rows, cells, headers, confidence, source_refs)
  FieldCandidate(field_path, occurrence_ref, value_raw, value_normalized?,
    value_state, datatype_hint, confidence?, evidence_refs, source_refs, validation_refs)
```

Cada entidad conserva, cuando exista, `bbox` o polígono, hashes,
transformaciones, `source_refs` y versión de IR. `value_raw` nunca se sobrescribe
por `value_normalized`, que declara locale, regla y versión de validador.
`confidence` es opcional y no se compara entre proveedores sin calibración.

Estados mínimos: `present`, `null_explicit`, `absent`, `unreadable`,
`not_applicable`, `proposed` y `human_corrected`. Nunca se inventa un valor para
satisfacer un esquema.

## Pipeline reproducible

```text
DocumentSource -> hash + detección born-digital/scan/foto/mixto
 -> normalización (rotación, deskew, denoise)
 -> OCR/layout -> localización de páginas/bloques/tablas
 -> schema slicing -> extracción -> evidencia + validación determinista
 -> segunda lectura independiente según riesgo/confianza
 -> revisión humana -> exportadores + recibo durable
```

1. Se hashea el original; el recibo registra hash, no copia PII.
2. La clasificación documental guía adaptadores, es revisable y queda como
   evidencia. Transformaciones registran versión, parámetros, hashes y páginas.
3. Parser/layout precede a extracción semántica: la localización reduce contexto
   y hace posible evidencia espacial.
4. El esquema se parte por entidad, página o bloque. [ExtractBench](https://github.com/ContextualAI/extract-bench)
   demuestra degradación severa e incluso 0 % de salida válida con esquemas
   amplios; un one-shot gigante no es diseño base.
5. El extractor devuelve candidatos, no verdad. El validador acepta, rechaza o
   remite a revisión; exportar se repite desde campos aceptados sin repetir OCR.
6. Riesgo/confianza baja exige segunda lectura independiente o revisión humana;
   adaptadores y grado de independencia quedan en recibo.

## Recibo durable, privacidad e i18n

El recibo contiene versión de contrato/IR, refs opacas, hash original y derivados,
adaptador/contenedor/binario/modelo/revisión exactos, prompt/template/config,
idioma y locale, páginas/regiones, transformaciones, schema slices, validadores,
decisiones humanas, evidencias/exportaciones con hash, y latencia/coste agregado
sin PII. Si proveedor no expone versión exacta, se registra limitación y máxima
identidad disponible; nunca se finge reproducibilidad.

PII es local por defecto. Cloud es opt-in y requiere política de retención,
borrado, región, cifrado y auditoría en adaptador/composición. Credenciales,
bytes, identificadores personales y logs textuales no entran al núcleo ni al
recibo general.

Idioma del documento, locale de normalización y locale de salida son distintos.
Fechas, nombres, decimales, moneda y DNI/NIE/NIF son validadores inyectados de
dominio. El resultado del validador explica regla/versión, estado, motivo,
normalización propuesta y evidencia; forma válida no prueba semántica correcta.

## Evaluación empírica y ratchet

Mantener corpus real, versionado y autorizado de PDF limpios/sucios, escaneos,
fotos, tablas y manuscritos, con gold humano y redacción de PII. Medir por
adaptador, revisión, configuración y schema slice: exact/fuzzy/normalized,
precision/recall/F1 de arrays, falsos positivos/ausentes/inventados, provenance,
métricas de tabla, latencia, coste, VRAM/RAM/CPU y tasa de revisión humana.

Los A/B usan mismo corpus, gold, política y presupuesto. Un ratchet impide
promoción que empeore calidad/seguridad acordada sin excepción explícita.
[OmniDocBench](https://github.com/opendatalab/OmniDocBench) y ExtractBench son
referencias de método, no sustitutos del corpus propio.

## Ejemplo: personas en PDF mixto

Caso: texto digital, fichas fotografiadas y tabla de asistentes. Schema slice
por tabla/ficha:

```json
{"entity":"person","fields":{"full_name":{"type":"string"},"document_id":{"type":"string"},"email":{"type":"string"},"role":{"type":"string"}},"rules":["No inferir datos no visibles","Evidencia por campo","Usar absent/unreadable/null_explicit"]}
```

1. Se hashea PDF; pdfplumber/Tika procesan texto y el normalizador renderiza las
   páginas rasterizadas. OCR/layout local crea bloques y bboxes.
2. Se localizan tabla/fichas y se crean schema slices. Lift y NuExtract3 son
   candidatos a benchmark, no elección declarada.
3. `full_name` apunta a página, tabla/fila, bbox y span raw. Un documento no
   visible queda `absent`, no vacío ni inferido.
4. Validador de dominio normaliza email/DNI/NIE/NIF sin borrar raw/evidencia.
   Conflicto o bajo riesgo/confianza dispara segunda lectura o `HumanReview`.
5. Los campos aceptados exportan a CSV/XLSX/Parquet, SQL o DTO; Markdown/HTML
   puede incluir evidencia según política de acceso. Todos proceden de mismos
   campos y recibo.

## Recomendación inicial e integración wizard

**Recomendación provisional:** baseline local barato para born-digital
(Tika/pdfplumber y Camelot/Tabula); OCR/layout local para escaneo/foto; Lift y
NuExtract3 como candidatos schema; cloud como fallback explícito. La decisión
final solo sigue benchmark propio de licencia, privacidad, hardware, calidad y
coste.

El wizard de Orquesta preguntará privacidad/local-cloud, hardware, coste,
idiomas, tipos, tablas/manuscrito, exportadores, sensibilidad y revisión humana.
Antes de crear app resumirá candidatos, posible salida de datos, recibos,
validadores pendientes y riesgos. Configuración única y opt-in en composición;
el núcleo permanece neutral.
