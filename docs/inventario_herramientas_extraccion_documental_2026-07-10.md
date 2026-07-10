# Inventario de herramientas para extracción documental - 2026-07-10

## Alcance y evidencia

Inventario para extraer datos tipados desde PDF, imágenes y documentos sucios,
escaneados o fotografiados. No es una selección de proveedor ni una promesa de
precisión. Las fuentes enlazadas son oficiales/primarias facilitadas para este
corte. «Hecho» describe interfaz o licencia publicada; «claim» es una capacidad
afirmada por proveedor que requiere medición propia. No se inventan precios,
benchmarks ni requisitos de hardware sin fuente confirmada.

[ExtractBench](https://github.com/ContextualAI/extract-bench) y
[OmniDocBench](https://github.com/opendatalab/OmniDocBench) son referencias
independientes. ExtractBench muestra que esquemas muy amplios degradan mucho y
pueden producir 0 % de salida válida: localizar páginas y partir esquema es
requisito de diseño, no optimización opcional.

## Parser, layout y OCR

| Herramienta | Rol, entradas y salidas | Local/API, licencia, privacidad/hardware | Fortalezas (hecho o claim) | Límites y encaje |
| --- | --- | --- | --- | --- |
| [Datalab Chandra](https://huggingface.co/datalab-to/chandra) | OCR documental para imagen/páginas renderizadas; salida según tarjeta. | Local; confirmar licencia, revisión y hardware en tarjeta. Privacidad local. | Claim: OCR multimodal documental. | Candidato OCR; medir idiomas, manuscrito y geometría. |
| [Marker](https://github.com/datalab-to/marker) | Convierte PDF/imagen a Markdown, JSON y layout. | Local/autoalojable; revisar `LICENSE` de revisión fijada. | Hecho: conversión documental estructurada. | Normalizador/parser; no valida campos. |
| [Docling](https://github.com/docling-project/docling) | Ingesta y conversión documental a representación estructurada/exportable. | Local/autoalojable; licencia publicada en repo/release. | Hecho: framework de conversión documental. | Buen candidato IR/layout; medir escaneos y tablas difíciles. |
| [MinerU](https://github.com/opendatalab/MinerU) | Parseo de PDF/documentos complejos a Markdown/JSON/layout. | Local o servicio; confirmar licencia/hardware por release. | Claim: conversión de documentos complejos. | Aislar detrás de `DocumentParser`. |
| [PaddleOCR / PP-StructureV3 / PaddleOCR-VL](https://github.com/PaddlePaddle/PaddleOCR) | OCR, orientación, layout, tablas y modelos visuales para imagen/PDF renderizado. | Local, Apache-2.0; privacidad local. Hardware depende de modelo/backend. | Hecho: suite OCR y estructura; precisión es claim del proyecto. | Base OCR/layout; fijar modelo y versión en adaptador. |
| [olmOCR](https://github.com/allenai/olmocr) | OCR de documentos/PDF renderizado a texto/estructura. | Local; licencia en repo/revisión fijada. | Claim: OCR orientado a documentos. | Candidato de segunda lectura; medir cómputo e idiomas. |
| [Unstructured](https://github.com/Unstructured-IO/unstructured) | Partición de formatos a elementos, texto y metadatos. | Local o API; licencia según repositorio/release. | Hecho: biblioteca de partición documental. | Ingesta/normalización, no garantía semántica. |
| [OCRmyPDF / Tesseract](https://github.com/ocrmypdf/OCRmyPDF) | Añade OCR a PDF escaneados; genera PDF con capa de texto. | Local; OCRmyPDF MPL-2.0; privacidad local; CPU habitual según despliegue. | Hecho: OCR y PDF buscable. | Baseline barato; no resuelve tablas/campos por sí solo. |
| [docTR](https://github.com/mindee/doctr) | Detección y reconocimiento de texto en imágenes/documentos. | Local, Apache-2.0; privacidad local. | Hecho: OCR detección+reconocimiento. | Necesita layout/tabla y validación separadas. |
| [Apache Tika](https://tika.apache.org/) | Detección de tipo y extracción de texto/metadatos multi-formato. | Local, Apache-2.0; privacidad local. | Hecho: extracción multi-formato. | Baseline born-digital; no OCR/tabla fiable. |
| [pdfplumber](https://github.com/jsvine/pdfplumber) | Texto, caracteres, geometría y tablas de PDF digitales. | Local, MIT; privacidad local. | Hecho: acceso fino a contenido/geometría. | Excelente born-digital; no OCR raster. |
| [Camelot](https://github.com/camelot-dev/camelot) | Tablas desde PDF de texto. | Local, MIT; privacidad local. | Hecho: extracción tabular PDF. | Medir tablas escaneadas/sin bordes/complejas. |
| [Tabula](https://github.com/tabulapdf/tabula-java) | Tablas desde PDF, interactivo/programático. | Local; licencia en repo; privacidad local. | Hecho: foco en tablas PDF. | Complemento born-digital; no OCR ni comprensión semántica. |
| [invoice2data](https://github.com/invoice-x/invoice2data) | Extracción de facturas basada en plantillas/reglas. | Local; licencia en repo; privacidad local. | Hecho: enfoque de plantilla de factura. | Adaptador de dominio, no extractor universal. |

## Extractores guiados por esquema locales

| Herramienta | Rol, entradas y salidas | Local/API, licencia, privacidad/hardware | Fortalezas (hecho o claim) | Límites y encaje |
| --- | --- | --- | --- | --- |
| [Datalab Lift / lift-pdf](https://pypi.org/project/lift-pdf/) · [modelo](https://huggingface.co/datalab-to/lift) | Modelo ~10B para PDF/imagen y JSON Schema; salida JSON schema-constrained. | Local. Código Apache-2.0; pesos OpenRAIL modificada con restricciones comerciales según ficha. Privacidad local. Hardware sin perfil fijado. | Claim: extracción restringida por esquema sobre PDF/imagen. | Candidato `SchemaExtractor`; comprobar licencia de pesos y partir esquema/páginas. JSON no es salida final obligatoria. |
| [NuExtract3](https://huggingface.co/numind/NuExtract3) | Modelo ~4B para imagen/texto con template; render multipágina y JSON/Markdown. | Local; confirmar licencia/revisión/hardware en tarjeta antes de adopción. Privacidad local. | Claim: extracción por template multipágina. | Candidato ligero; medir adherencia, texto largo, idiomas y evidencia espacial. |

## Servicios gestionados y APIs

| Servicio | Rol, entradas y salidas | Datos y contrato | Fortalezas (claim de proveedor) | Límites y encaje |
| --- | --- | --- | --- | --- |
| [Datalab](https://pypi.org/project/lift-pdf/) | Servicios asociados a conversión/extracción documental. | API solo por adaptador opt-in; confirmar región, retención, DPA y versión. | Capacidad documental del proveedor. | Normalizar a IR; recibo obligatorio. |
| [Reducto Extract](https://reducto.ai/extract) | Extracción estructurada desde documentos. | Cloud/API; aceptar explícitamente datos, retención y región. | Extracción documental gestionada. | Fallback cloud; evaluar y exigir evidencia. |
| [LandingAI ADE](https://docs.landing.ai/ade/ade-extract-schema-json) | Extracción JSON guiada por esquema. | Cloud/API; revisar contrato/región. | Schema extraction. | Fallback de extractor; validar/exportar neutralmente. |
| [Mistral OCR](https://docs.mistral.ai/studio-api/document-processing/basic_ocr) | OCR documental por API. | Cloud/API; retención/privacidad por contrato. | OCR documental. | Fallback OCR; no presumir tablas/manuscrito. |
| [Azure Document Intelligence](https://github.com/MicrosoftDocs/azure-ai-docs/blob/main/articles/ai-services/document-intelligence/prebuilt/invoice.md) | Análisis documental y modelos prebuilt, incluida factura. | Cloud/API; región, identidad, retención y cifrado en composición. | Extracción prebuilt/análisis documental. | No meter tipos de factura ni credenciales en core. |
| [Google Document AI](https://docs.cloud.google.com/document-ai/docs/ce-with-genai) | Procesamiento documental y extractor generativo según producto. | Cloud/API; región/política explícitas. | Procesadores documentales/GenAI. | Comparar normalización, evidencia y coste real. |
| [AWS Textract](https://docs.aws.amazon.com/textract/latest/dg/how-it-works-analyzing.html) | Texto, formularios, tablas y consultas sobre documentos. | Cloud/API; privacidad/región/retención AWS configurables. | Hecho: análisis documental; precisión es claim. | Adaptar bloques a IR neutral. |
| [Adobe PDF Extract](https://developer.adobe.com/document-services/docs/overview/pdf-extract-api/) | Estructura y elementos de PDF. | Cloud/API; datos según cuenta Adobe. | Extracción de estructura PDF. | Candidato born-digital; no asumir OCR/schema general. |
| [ABBYY Vantage](https://docs.abbyy.com/vantage/introduction) | Plataforma de automatización documental. | Producto comercial; privacidad/despliegue por organización. | Clasificación/extracción empresarial. | Evitar taxonomía propietaria en core. |
| [Mindee](https://docs.mindee.com/) | API de OCR y extracción prebuilt/personalizada. | Cloud/API; revisar región/retención/contrato. | Automatización documental. | Encerrar SDK/formato tras adaptador. |
| [Nanonets](https://docs.nanonets.com/reference/overview) | API de OCR y workflow documental. | Cloud/API; privacidad/residencia/borrado por contrato. | OCR/extracción gestionada. | Fallback; benchmark antes de selección. |

## Encaje inicial

**Recomendación, no resultado de benchmark:** para born-digital empezar con
Tika/pdfplumber y Camelot/Tabula cuando aplique; para escaneos/fotos, OCR/layout
local (OCRmyPDF, PaddleOCR, docTR, Docling, Marker, MinerU u olmOCR) elegido por
corpus y hardware; evaluar Lift y NuExtract3 tras localización y schema slicing;
mantener cloud como fallback opt-in. La decisión final exige corpus propio,
gold humano, privacidad/licencias y métricas de calidad/coste/recursos.

Antes de adoptar cualquier fila, fijar revisión exacta, licencia de código y
pesos, clases de documento reales, capacidad de evidencia por campo, tratamiento
de datos/región/borrado, e identidad que se pueda registrar en recibo.
