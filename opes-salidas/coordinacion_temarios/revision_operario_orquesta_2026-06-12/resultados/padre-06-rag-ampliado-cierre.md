# Revision padre-06: RAG, ampliado y cierre Operario

Fecha: 2026-06-12
Curso: `ope-operario`
Contrato: cierre externo acotado por OrquestaV2
Write-set: `opes-salidas/coordinacion_temarios/revision_operario_orquesta_2026-06-12/resultados/padre-06-rag-ampliado-cierre.md`

## Veredicto

**No cerrado.**

La entrega disponible permite conservar 10 bloques de contenido y un HTML de revision, pero no cumple el paquete local obligatorio para cierre OPES. No hay evidencia local de `html_ampliado`, RAG/tutor, `completed_syllabus_package` ni triple visto bueno Codex/Gemini/Claude. Por tanto, el curso no debe marcarse como terminado ni como publicado con pendientes: queda en estado **no cerrado**, con rework causal acotado.

## Evidencia revisada

- `opes-salidas/operarios_temario_nofilters_2026-06-02/manifest.json`: manifest `opes_review_bundle.v0` con 10 temas y refs a `raw/*/content_block.*`.
- `opes-salidas/operarios_temario_nofilters_2026-06-02/revision_temario_operarios.html`: HTML revisable con 10 temas visibles y estado textual "10/10 aceptados por Orquesta y completados en OPES".
- `opes-salidas/operarios_temario_nofilters_2026-06-02/raw/*/content_block.*`: bloques de contenido por tema; algunos incluyen notas `audio_ready` o politica para TTS, pero no audios generados.
- Busquedas locales por nombre/contenido sobre `opes-salidas`: no aparecen artefactos `*rag*`, `*html*ampliado*`, `*completed*syllabus*`, `*package*` ni informes Gemini/Claude/Codex asociados a este cierre.

## Comprobaciones de cierre

| Criterio | Estado | Evidencia | Decision |
| --- | --- | --- | --- |
| Temario base 10 temas | presente | `manifest.json` lista temas 1-10 | conservar como insumo valido |
| HTML revisable | presente parcial | `revision_temario_operarios.html` | no equivale a `generate_html_site` final |
| `html_ampliado` | ausente | busqueda local sin resultados | bloquea cierre |
| RAG/tutor/bots | ausente | busqueda local sin resultados | bloquea cierre |
| `completed_syllabus_package` | ausente | busqueda local sin resultados | bloquea cierre |
| Triple visto bueno Codex/Gemini/Claude | ausente | sin informes independientes ni pair reviews | bloquea cierre |
| Audios por tema/apartado | ausente | solo notas de preparacion TTS | bloquea cierre |
| Tests/banco de preguntas | ausente en evidencia revisada | no hay `generate_question_bank` para Operario en este paquete | bloquea cierre |

## Hallazgos

1. El material existente es recuperable. Hay 10 bloques de contenido trazados por `job_id` y un HTML de revision que permite lectura local.
2. La evidencia de cierre no existe en el workdir revisado. El canon OPES exige paquete local completo antes de produccion: ampliado separado, HTML final, RAG/tutor, audios, tests, revisiones, pair reviews, matriz del Director y `completed_syllabus_package`.
3. El HTML hallado es una pagina de revision de bloques, no el curso final con estructura USO/TCAE, `index.html`, `html_final/`, assets locales, i18n, audio/manifests y watermark.
4. No hay base para declarar triple visto bueno. La ausencia de informes independientes y por pares impide cerrar, aunque los bloques base esten aceptados.
5. El contexto `ref_only` requerido queda resuelto para esta unidad mediante evidencia filesystem y nota ACK; no aporta documento materializado adicional para cambiar el veredicto.

## Plan operativo y olas

Ola 1 - Inventario y preservacion:
- Registrar los 10 `content_block` como insumo reusable.
- Marcar `revision_temario_operarios.html` como HTML de revision, no como entrega final.
- No rehacer bloques por defecto; revisar contra canon oficial antes de pedir rework.

Ola 2 - Derivados bloqueantes:
- Generar temario ampliado separado y su `html_ampliado`.
- Crear RAG/tutor/bots con manifest de fuentes y refs.
- Generar banco de preguntas por tema y validaciones estructurales.
- Generar audios por tema/apartado y manifests.

Ola 3 - Revision:
- Ejecutar `review_codex`, `review_gemini`, `review_claude`.
- Ejecutar pair reviews Codex-Gemini, Codex-Claude y Gemini-Claude.
- Consolidar discrepancias en matriz del Director.

Ola 4 - Paquete:
- Ensamblar HTML local final con assets, i18n, watermark y rutas relativas.
- Crear `completed_syllabus_package` con manifest, checksums/refs, matriz de revisiones y estado de validacion.
- Solo despues pedir decision de publicacion.

## Rework causal propuesto

- `rework-ref-operario-html-ampliado-20260612`: crear `html_ampliado` desde los bloques existentes, separando resumen/ampliado si procede.
- `rework-ref-operario-rag-tutor-20260612`: generar RAG/tutor/bots con refs a fuentes y contenido local.
- `rework-ref-operario-tests-20260612`: generar banco de preguntas y validarlo antes de publicacion.
- `rework-ref-operario-reviews-20260612`: obtener triple visto bueno y revisiones por pares.
- `rework-ref-operario-package-20260612`: ensamblar `completed_syllabus_package` solo tras superar las revisiones.

## Validacion del contrato externo

- Orquesta se mantiene como orquestador; OPES conserva reglas, validadores, persistencia y ensamblado.
- No se ha creado job manual, no se ha llamado a colas OPES y no se ha tocado OPES productivo.
- Las refs revisadas son artefactos de filesystem del proyecto y se tratan como refs opacas/evidencias opacas; no se infiere estado desde DB interna.
- El cierre usa la regla de conservar trabajo recuperable: los bloques existentes quedan como insumo, no se descartan por ausencia de paquete final.

## Dictamen final

Estado: **no cerrado**.
Motivo: faltan `html_ampliado`, RAG/tutor, `completed_syllabus_package`, triple visto bueno, banco de tests, audios y HTML final publicable.
Siguiente accion: lanzar rework causal por Orquesta con scope `ope-operario`, empezando por inventario reutilizable y generacion de derivados bloqueantes.
