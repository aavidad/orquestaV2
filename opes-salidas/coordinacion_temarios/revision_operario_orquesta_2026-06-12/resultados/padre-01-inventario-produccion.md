# Inventario de produccion OPES Operario

Fecha: 2026-06-12
Unidad Orquesta: `task-ref-app-change-appchange-937f91321ee7ddbd922d01e27cf9ebd6`
Curso: `ope-operario`
Veredicto: no hay evidencia suficiente para marcar curso publicado ni listo para produccion.

## Alcance

Este informe resuelve solo el contrato externo asignado: inventariar el estado
materializado del curso Operario dentro del workdir de Orquesta y devolver una
entrega verificable para la app propietaria. No se han creado jobs OPES, no se
ha subido ningun artefacto y no se ha tocado produccion.

Write-set usado:

- `opes-salidas/coordinacion_temarios/revision_operario_orquesta_2026-06-12/resultados/padre-01-inventario-produccion.md`

Contexto ref-only requerido:

- `source-ref-app-stack-task-ref-app-change-appchange-937f91321ee7ddbd922d01e27cf9ebd6` venia como `required=true`, `mode=ref_only`, `ref_only_reason=materialization_missing` y `required_ref_action=ack_evidence_required`.
- Resolucion aplicada: no se invento contenido ausente; el inventario se basa en refs y artefactos verificables dentro del workdir, y el ACK debe dejar nota `contexto_ref_only_resuelto`.

## Plan operativo aplicado

Skill materializada: `skill-ref-orquesta-ordenacion-trabajo-v0`.

Backlog compacto:

1. Inventariar rutas OPES/Orquesta disponibles para `ope-operario`.
2. Identificar `material_path` verificable y contenido real.
3. Comparar estados: salida editorial, staging local activo, staging local con tests y produccion.
4. Registrar evidencias verificables y bloqueos para cierre OPES.
5. Validar criterios de aceptacion y contrato externo de dominio.

Olas:

- Ola 1: lectura de paquete, reglas Orquesta/OPES y refs locales.
- Ola 2: inventario de artefactos bajo `opes-salidas`.
- Ola 3: consolidacion del informe y validacion semantica.

No se lanzaron subagentes nuevos: el contrato ya venia materializado por
Orquesta y el write-set era de un solo fichero.

## Material path vigente

`material_path` verificable dentro del repo:

```text
opes-salidas/operarios_temario_nofilters_2026-06-02
```

Contenido real encontrado:

- `manifest.json`: manifest `opes_review_bundle.v0`, actualizado el `2026-06-02T10:48:27.739477+00:00`, con 10 temas.
- `revision_temario_operarios.html`: HTML de revision local, 464775 bytes, titulo "Revision temario operarios OPES".
- `raw/`: 10 carpetas por `job_id` con `content_block.md` o `content_block.json`.
- Ficheros extra puntuales: `raw/5cfcc4984b3ff1975af10101680454bc/manifest.json`, `raw/6ccc7aa743d30c85edc32309fe4ec4af/content_block_outline_original.json`, `raw/ab68a143853edbb90d76fd1087643891/artifact.json`.

No encontrado dentro de este `material_path`:

- `index.html` de curso final.
- `html_final/`.
- `audio/manifests/`.
- audios `.mp3` o `.wav`.
- banco de tests o `question_bank`.
- tutor/bots.
- juegos.
- paquete `completed_syllabus_package`.
- matriz final cerrada del Director con triple revision Codex/Gemini/Claude y revisiones por pares.

## Inventario por tema

| Tema | Job ref local | Titulo en manifest | Artefacto |
| --- | --- | --- | --- |
| 1 | `7a0cf9c149c9e07a261ece298694a552` | Constitucion Espanola de 1978 y Administracion local | `raw/.../content_block.json` |
| 2 | `5cfcc4984b3ff1975af10101680454bc` | Empleados publicos | `raw/.../content_block.md` + manifest |
| 3 | `6ccc7aa743d30c85edc32309fe4ec4af` | La plancha y el planchado | `raw/.../content_block.json` + outline original |
| 4 | `0eada4101e92d34e0c28d3ed1af181a0` | El lavado de prendas | `raw/.../content_block.md` |
| 5 | `73ff772c5c45254bb990d575b7ed6d7c` | Limpieza, desinfeccion, habitaciones, oficinas, pistas y exteriores | `raw/.../content_block.md` |
| 6 | `1eb10f2e0b35f98a9bb76eb3b99bc304` | Limpieza de cocinas | `raw/.../content_block.md` |
| 7 | `4dad1eb2c567ded031736b65b5084117` | Alimentos y dietas | `raw/.../content_block.md` |
| 8 | `ab68a143853edbb90d76fd1087643891` | Tema 8 | `raw/.../content_block.md` + artifact |
| 9 | `c58688d2ae40b17de2dec2145730655d` | Preparacion, conservacion, emplatado y transporte de alimentos | `raw/.../content_block.json` |
| 10 | `5ddd69cf3d789ee07eab52bfef846097` | Prevencion de riesgos laborales, productos de limpieza e incendio en cocina | `raw/.../content_block.json` |

## Comparacion de estados

### Salida editorial

Estado: parcial aprovechable.

Evidencia:

- `revision_temario_operarios.html` declara "10/10 aceptados por Orquesta y completados en OPES".
- `manifest.json` lista 10 temas con refs a bloques crudos.
- Hay `source_refs` por tema, incluido `OPES/administracion-especial/Operario/Operario.txt#tema-*` y refs de comunes para temas 1 y 2.

Limitacion:

- El flujo OPES vigente exige temario resumido y temario ampliado separados. En el material local no aparece esa separacion como paquete final.
- No hay evidencia materializada de revisiones legal, pedagogica, calidad, Codex, Gemini, Claude, pares ni consolidacion final.
- Hay refs a material comun previo `opes-salidas/codex_directo/operario/AP/produccion_externa_2026-05-19/...`, pero esos ficheros no estan presentes bajo `opes-salidas/codex_directo` del workdir actual.

### Staging local activo

Estado: existe revision local, no curso local final.

Evidencia:

- `revision_temario_operarios.html` funciona como HTML revisable de contenido.
- `raw/` conserva los bloques fuente por `job_id`.

Limitacion:

- El contrato vigente de `generate_html_site` exige estructura de curso: `index.html`, `html_final/`, assets locales, `audio/manifests/`, locales/i18n y capa `#uso-material-watermark` si aplica.
- Esa estructura no existe en `material_path`.

### Staging local con tests

Estado: no existe evidencia.

Evidencia negativa:

- No se encontraron ficheros de banco de preguntas, `question_bank`, `course_tests`, HTML de tests, metadatos de tests ni validador asociado dentro de `material_path`.

Impacto:

- No se puede declarar paquete didactico completo ni listo para publicacion.
- Falta revision 100% de tests por Codex, Gemini y Claude exigida por el flujo OPES vigente.

### Produccion

Estado: no publicado / publicacion no verificable desde este workdir.

Evidencia:

- El paquete materializado es una salida local de revision bajo `opes-salidas`.
- El runbook OPES indica que la subida a produccion solo procede despues de paquete local revisable completo y revision del operador.
- No hay artifact de produccion, registro de subida, URL productiva, paquete final ni confirmacion de operador en el material revisado.

Decision:

- No marcar curso como publicado.
- No promocionar ni subir a produccion desde esta unidad.

## Evidencias verificables

- `docs/opes_flujo_temario_operativo_2026-06-02.md`: define el flujo completo hasta `finalize_temario_package` y requisitos de HTML local, tests, audios, tutor, juegos, manuales y cierre.
- `docs/corte_opes_como_consumidor_orquesta_2026-05-18.md`: fija OPES como consumidor de Orquesta por refs opacas y documenta smoke Operario de `plan_temario` con derivados iniciales.
- `docs/runbooks/smoke_opes_plan_temario_operadores_2026-05-18.md`: fija guardas de instancia temporal, scope y no produccion sin confirmacion.
- `opes-salidas/operarios_temario_nofilters_2026-06-02/manifest.json`: evidencia 10 bloques locales.
- `opes-salidas/operarios_temario_nofilters_2026-06-02/revision_temario_operarios.html`: evidencia HTML de revision y estado editorial declarado 10/10.

## Bloqueos para cierre OPES

1. Falta paquete final `completed_syllabus_package`.
2. Falta `generate_html_site` con estructura de curso OPES/USO real.
3. Faltan tests por tema y validacion/revision completa del banco.
4. Faltan audios por tema y apartado, con manifest.
5. Faltan tutor/bots y juegos.
6. Faltan manuales graficos de ayuda.
7. Faltan revisiones independientes y por pares Codex/Gemini/Claude.
8. Falta matriz de decision final del Director.
9. Falta confirmacion de operador para produccion.
10. Faltan refs materializadas externas redirigidas por `main_paths`, recibidas redacted en el paquete.

## Rework causal propuesto

- Relanzar Orquesta sobre OPES temporal/Postgres aislado con scope duro para `ope-operario`, no productivo.
- Continuar desde material reutilizable de `opes-salidas/operarios_temario_nofilters_2026-06-02`, no rehacer por defecto.
- Materializar o resolver refs de comunes `produccion_externa_2026-05-19` antes de decidir reutilizacion definitiva.
- Ejecutar fases pendientes del flujo vigente: `generate_question_bank`, revisiones, `assemble_topic`, `generate_audio_asset`, `generate_tutor_assets`, `generate_learning_games`, `generate_html_site`, `generate_help_manual_assets` y `finalize_temario_package`.
- Mantener refs opacas y submit por bridge/domain-work; no abrir DB ni filesystem interno de OPES desde Orquesta.

## Validacion local

`validar criterios de aceptacion del cambio`: passed.

Comprobado:

- informe existe en el write-set exacto;
- indica curso, `material_path`, estado de publicacion, comparacion de estados, evidencias y bloqueos;
- no crea app nueva ni modifica fuera del alcance.

`validar contrato externo de dominio`: passed.

Comprobado:

- OPES queda como app propietaria;
- Orquesta conserva frontera por refs opacas;
- no hay efectos externos, DB, API productiva ni submit manual;
- artefactos recuperables se conservan como insumo y no se descartan por formato.
