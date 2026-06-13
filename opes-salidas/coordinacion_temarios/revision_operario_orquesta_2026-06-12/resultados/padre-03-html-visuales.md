# Revision padre-03: HTML y visuales Operario

Fecha: 2026-06-12
Curso: `ope-operario`
Unidad Orquesta: `task-ref-app-change-appchange-bf6e6b9cffaec752705906f5534b8fac`

## Veredicto

No apto para cierre como `generate_html_site` publicable.

Apto solo como snapshot interno de revision de 10 bloques generados. El HTML
existe y permite lectura/navegacion simple, pero no cumple el canon local
USO/TCAE exigido para curso operativo: falta estructura `index.html` +
`html_final/`, assets locales, logos USO, capa `#uso-material-watermark`,
`audio/manifests/`, locales/i18n de interfaz, visuales finales y capturas
desktop/movil.

## Evidencia revisada

- `opes-salidas/operarios_temario_nofilters_2026-06-02/manifest.json`: declara
  10 temas y 10 ficheros raw.
- `opes-salidas/operarios_temario_nofilters_2026-06-02/revision_temario_operarios.html`:
  single-file HTML de 2735 lineas con cabecera, indice, buscador y 10
  articulos de tema.
- `external/opes/plan_temario/9784a562f074769a08043707fdd79eb2`: planifica
  visuales `visual-mapa-temario-operario`, `visual-flujo-lavanderia-limpieza`,
  `visual-tablas-alimentos-riesgos` y `visual-plan-repaso-prl`.
- `docs/opes_flujo_temario_operativo_2026-06-02.md`: canon vigente para
  `generate_html_site`, visuales y paquete local previo a produccion.

## Hallazgos

1. Bloqueante: no hay paquete HTML canonico.
   La entrega localizada es `revision_temario_operarios.html`; no hay
   `index.html`, `html_final/`, directorio de assets, `audio/manifests/` ni
   `locales/i18n`. Esto impide marcarla como HTML local operativo del curso.

2. Bloqueante: no hay branding ni proteccion USO/TCAE.
   El HTML no contiene logos USO ni `#uso-material-watermark`. Para material de
   estudio, el canon exige marca de agua diagonal USO visible sin tapar
   contenido ni infografias.

3. Bloqueante: visuales finales ausentes.
   No se detectan `<figure>`, `<img>`, `<svg>` ni assets de imagen dentro del
   paquete revisado. Los visuales previstos en el plan quedan pendientes como
   trabajos reales `generate_visual_asset`, no como arte final.

4. Mayor: funcion didactica visual no cubierta.
   Temas 3, 4, 5, 6, 7, 8, 9 y 10 tienen procedimientos, riesgos, alimentos,
   limpieza, lavado, cocina y PRL que necesitan doble codificacion visual. Hoy
   solo hay texto; no hay diagramas de flujo, tablas comparativas finales,
   cronogramas ni imagenes realistas donde aportarian aprendizaje.

5. Mayor: captura visual desktop/movil pendiente.
   No hay evidencias de render, capturas ni revision responsive. La revision
   queda estatica sobre fichero y busqueda de estructura; falta prueba visual en
   escritorio y movil antes de aceptar UX local.

6. Mayor: el texto visible expone trazabilidad interna.
   La cabecera dice que es una revision generada por Orquesta/OPES y el cuerpo
   incluye bloques JSON/metadata visibles. El canon indica que refs, manifests,
   agentes, staging y trazabilidad tecnica deben quedar fuera del texto visible
   para alumnado.

7. Medio: i18n parcial, no superficie local completa.
   Algunos bloques incluyen metadata `i18n`, pero no existe estructura
   `locales/i18n` de interfaz ni superficie canonica de textos UI. Para cierre
   HTML debe separarse contenido, interfaz e idioma.

## Riesgos

- Publicar este fichero como curso final degradaria imagen USO/TCAE: parece
  pagina de revision interna, no experiencia de aula.
- Sin visuales finales, los temas operativos pierden apoyo procedimental y
  memoria visual.
- Sin capturas responsive, hay riesgo de desbordes en titulos largos, tablas,
  JSON visible y navegacion movil.
- Sin estructura de assets y manifiestos, audio, tutor, tests y juegos no
  pueden integrarse ni trazarse como paquete local completo.

## Rework causal propuesto

1. Abrir `generate_visual_asset` para las refs ya planificadas:
   `visual-mapa-temario-operario`, `visual-flujo-lavanderia-limpieza`,
   `visual-tablas-alimentos-riesgos` y `visual-plan-repaso-prl`.

2. Anadir visuales por tema cuando el contenido lo justifique:
   planchado y simbolos textiles, ciclo de lavado, protocolo de limpieza de
   habitaciones/oficinas/exteriores, limpieza de cocina, clasificacion de
   alimentos/dietas, manipulacion y trazabilidad, conservacion/transporte de
   alimentos, PRL/productos/incendio.

3. Ejecutar `generate_html_site` real con estructura:
   `index.html`, `html_final/`, `html_final/img`, assets locales,
   `audio/manifests/`, locales/i18n, navegacion de curso, modo estudio,
   tests permitidos, tutor/bots, juegos/retos y revision de errores.

4. Integrar branding y proteccion:
   logo USO, formato curso USO/TCAE de promocion interna y
   `#uso-material-watermark` cuando proceda.

5. Hacer QA visual antes de cierre:
   capturas desktop y movil, comprobacion de desbordes, contraste,
   navegacion, carga local de assets, ausencia de JSON visible y compresion de
   imagenes finales.

## Capturas faltantes o bloqueos visuales

- Falta captura desktop del indice, tema largo, visual integrado y pie/branding.
- Falta captura movil del indice, buscador, cambio entre temas y visuales.
- Falta captura de marca de agua USO si el curso queda como material de estudio.
- Falta captura de al menos un flujo procedimental y una tabla comparativa final.
- Bloqueo: no hay assets visuales que renderizar; solo puede revisarse el HTML
  single-file existente.

## Contrato externo de dominio

Contrato conservado. La revision no toca OPES productivo, no crea jobs manuales,
no drena colas, no lee DB externa y no interpreta internals de OPES. Usa refs y
artefactos disponibles dentro del workdir de Orquesta. El resultado queda en el
write-set indicado para que el bridge/Director lo entregue como artefacto de
revision.

## Ordenacion Orquesta

- Backlog: revisar HTML, revisar visuales, contrastar canon USO/TCAE, declarar
  capturas faltantes, proponer rework causal.
- Ola: revision padre-03 acotada al fichero de salida pedido.
- Agentes: sin subagentes; el alcance es un unico informe y la evidencia local
  es suficiente para esta unidad.
- Tests de cierre:
  - `validar criterios de aceptacion del cambio`: pasado; el informe cubre
    canon HTML, visuales, capturas, riesgos y rework.
  - `validar contrato externo de dominio`: pasado; no hay efectos externos y
    solo se escribe el artefacto permitido.

## Contexto ref_only

`contexto_ref_only_pendiente`: el paquete declara una entrada required
`mode=ref_only` con `materialization_missing`; no hay contenido adicional que
leer en el paquete. La revision se completa con evidencia local disponible y
queda trazada esta limitacion.
