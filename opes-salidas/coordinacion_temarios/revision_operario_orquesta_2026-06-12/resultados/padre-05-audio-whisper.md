# Revision padre-05: audio y Whisper Operario

Fecha: 2026-06-12
Curso: `ope-operario`
Contrato: revision acotada de audio/Whisper con refs de dominio opacas.
Write-set: `opes-salidas/coordinacion_temarios/revision_operario_orquesta_2026-06-12/resultados/padre-05-audio-whisper.md`

## Veredicto

Pendiente de cierre audio. En el workdir revisado no hay MP3, directorios `audio/`,
`audio/manifests/` ni informe Whisper materializado para Operario. Si el paquete
quiere cerrar `generate_audio_asset`, debe generar o adjuntar el `audio_asset`
segmentado por tema/apartado y su validacion de transcripcion.

No hay audios generados que hayan quedado obsoletos por cambios de HTML, porque
no existe artefacto MP3 local. El HTML encontrado es una revision single-file de
contenidos (`revision_temario_operarios.html`), no el HTML final de curso con
`audio/manifests/`. Cuando exista HTML final, cualquier audio previo debe
regenerarse o revalidarse contra el DOM/contenido visible definitivo.

## Evidencia revisada

- `opes-salidas/operarios_temario_nofilters_2026-06-02/manifest.json`: 10 temas
  de Operario inventariados.
- `opes-salidas/operarios_temario_nofilters_2026-06-02/revision_temario_operarios.html`:
  HTML de revision con estado `10/10 aceptados por Orquesta y completados en OPES`.
- `opes-salidas/operarios_temario_nofilters_2026-06-02/raw/...`: contenidos con
  notas `audio_ready`, `audio_notes` o politica de numeros romanos para TTS en
  algunos temas.
- `docs/runbooks/resultado_prueba_opes_operario_api_2026-06-02.md`: el plan
  creo 1 derivado `generate_audio_asset`, pero no aporta MP3 ni informe Whisper.
- `docs/opes_flujo_temario_operativo_2026-06-02.md`: audio debe salir despues
  del texto ensamblado aprobado, con un MP3 por apartado/seccion narrable,
  manifest final y validacion por Whisper u otro transcriptor cuando aplique.

## Conteo por variante

| Variante | MP3 | Manifiestos audio | Otros manifest.json | Informe Whisper |
|---|---:|---:|---:|---|
| `operarios_temario_nofilters_2026-06-02` | 0 | 0 | 2 | No encontrado |

Notas de conteo:

- Busqueda acotada en repo sin salir del workdir antes de crear este informe:
  no aparecian `*.mp3`, `*.wav`, `*.m4a`, rutas `audio/`, rutas
  `audio/manifests/` ni informes Whisper fuente asociados a audio.
- Los 2 `manifest.json` encontrados son manifiestos de bundle/contenido, no
  manifiestos de audio.
- Se excluye `opes-salidas/codex_directo/tcae...` por no pertenecer al
  `course_slug=ope-operario`.

## Hallazgos

1. Falta artefacto audio final. No hay MP3 por tema ni por apartado, por lo que
   no se puede validar la regla de "un unico MP3 final por `section_ref`".
2. Falta manifiesto de audio. No existe `audio/manifests/` ni `audio_manifest_ref`
   materializado para enlazar secciones, duraciones, hashes/refs de texto y
   estado de reutilizacion o generacion.
3. Falta informe Whisper obligatorio para cierre de calidad audio. No hay
   evidencia de transcripcion automatica ni muestreo de tablas/listas/esquemas.
4. El contenido si trae preparacion parcial para TTS: varias piezas declaran
   `audio_ready`, `audio_notes` o politica para numeros romanos. Eso es insumo
   aprovechable, no audio cerrado.
5. El HTML disponible no integra audio. Es una revision de temario y no la
   estructura final exigida (`index.html`, `html_final/`, assets,
   `audio/manifests/`, locales/i18n).

## Riesgos

- Si se genera audio antes de cerrar HTML/DOM final, puede quedar desalineado
  con titulos, listas, tablas, textos alternativos o notas visibles.
- Si se usa solo texto plano como fuente TTS, las tablas y esquemas pueden
  narrarse mal. La validacion Whisper debe incluir muestras con estructuras no
  triviales.
- Si se reutiliza audio comun sin comparar hash/ref de texto final, el paquete
  puede publicar audio obsoleto aunque el contenido parezca equivalente.

## Rework causal propuesto

1. `generate_audio_asset`: producir `audio_asset` segmentado para Operario con
   una entrada por `section_ref`, `audio_ref`, duracion, hash/ref de texto y
   estado de reutilizacion/generacion.
2. `generate_audio_asset`: crear `audio/manifests/` o manifest equivalente del
   paquete, enlazado a refs opacas de tema/apartado.
3. `review_audio_whisper`: ejecutar transcripcion Whisper u otro transcriptor
   sobre muestras obligatorias, incluyendo al menos una tabla/lista/esquema y
   un caso con posibles numeros romanos o abreviaturas.
4. `generate_html_site`: integrar solo audios validados en HTML final; si el DOM
   cambia despues, marcar audio como pendiente de regeneracion o revalidacion.

## Validacion de contrato externo

- No se han tocado DB, API OPES, colas, jobs productivos ni rutas externas.
- La revision conserva refs y rutas como evidencia local del repositorio; OPES
  sigue siendo propietario de validadores, persistencia, ensamblado y publicacion.
- La entrega es un artefacto Markdown dentro del write-set autorizado.
- El contexto `ref_only` obligatorio no estaba materializado; se resolvio de
  forma parcial con evidencia local disponible y queda anotado para ACK.
