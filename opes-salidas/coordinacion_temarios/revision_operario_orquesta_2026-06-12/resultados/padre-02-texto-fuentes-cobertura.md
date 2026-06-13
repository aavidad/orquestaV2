# Revision padre-02: texto, fuentes y cobertura - Operario AP

Fecha: 2026-06-12
Curso: `ope-operario`
Contrato Orquesta: `task-ref-app-change-appchange-ab1235dea94533acc4da2111c0f52441`
Write-set: `opes-salidas/coordinacion_temarios/revision_operario_orquesta_2026-06-12/resultados/padre-02-texto-fuentes-cobertura.md`

## Veredicto

`apto_con_rework_obligatorio`.

Los 10 temas del programa oficial de Operario AP estan presentes y cada bloque conserva `official_source_id` o `source_refs` contra `OPES/administracion-especial/Operario/Operario.txt`. La cobertura basica del programa queda resuelta.

No debe cerrarse como paquete publicable todavia. Faltan fuentes externas/normativas completas por tema, normalizacion editorial de tildes en varios bloques, separacion formal de capas OPES completas y evidencias de revision final. El material es aprovechable: no procede rehacerlo desde cero; procede rework dirigido.

## Evidencia revisada

- Programa oficial: `/home/alberto/Trabajo/OPES/OPES/administracion-especial/Operario/Operario.txt`.
- Salida generada: `opes-salidas/operarios_temario_nofilters_2026-06-02/revision_temario_operarios.html`.
- Bloques fuente: `opes-salidas/operarios_temario_nofilters_2026-06-02/raw/*/content_block.*`.
- Manifests puntuales: `opes-salidas/operarios_temario_nofilters_2026-06-02/manifest.json` y manifests de `raw/`.
- Canon operativo usado: `docs/opes_flujo_temario_operativo_2026-06-02.md`, `docs/corte_opes_como_consumidor_orquesta_2026-05-18.md`, `docs/runbooks/resultado_prueba_opes_operario_api_2026-06-02.md`, `/home/alberto/Trabajo/OPES/AGENTS.md`, `/home/alberto/Trabajo/OPES/GUIA_ESTILO_TEMARIOS_OPES_GLOBAL_2026-05-18.md`.

## Cobertura por tema

| Tema | Epigrafe oficial cubierto | Evidencia local | Estado | Rework necesario |
| --- | --- | --- | --- | --- |
| 1 | Constitucion de 1978, derechos/deberes, Administracion local, municipio y provincia | `raw/7a0cf9.../content_block.json`, refs `tema-01` y comun AP | Cubierto | Anadir fuentes normativas exactas y fecha; separar notas de test/tutor si se publica. |
| 2 | Empleados publicos, clases, regimen juridico, derechos y deberes | `raw/5cfcc4.../content_block.md`, manifest tema 2, refs `tema-02` | Cubierto | Completar cita normativa TREBEP/EBEP y limpiar capa de repaso para no mezclar desarrollo ampliado con enfoque de examen. |
| 3 | Plancha, prendas, textiles, ropa plana, simbolos, cadena, maquinaria, tecnicas | `raw/6ccc7a.../content_block.json`, refs `tema-03` | Cubierto | Normalizar referencias tecnicas de simbolos internacionales; valorar infografia final de simbolos y flujo de planchado. |
| 4 | Lavado de prendas, clasificacion, tipos, industrial, lavadoras/secadoras, suciedad y manchas | `raw/0eada4.../content_block.md`, refs `tema-04` | Cubierto | Reforzar tabla de manchas/fibras y productos; anadir fuente tecnica verificable. |
| 5 | Productos, utensilios, maquinaria, limpieza integral/desinfeccion y espacios | `raw/73ff77.../content_block.md`, refs `tema-05` | Cubierto | Buen alcance; falta normalizar fuentes de productos/seguridad y evitar duplicacion con temas 6 y 10 mediante referencias cruzadas. |
| 6 | Limpieza de cocinas: caracteristicas, productos, medios y normas | `raw/1eb10f.../content_block.md`, refs `tema-06` | Cubierto | Reforzar normativa higienico-sanitaria y separar residuos/incendio hacia temas 8/10 cuando sea repeticion. |
| 7 | Alimentos, clasificacion, caracteristicas y dietas basicas | `raw/4dad1e.../content_block.md`, refs `tema-07` | Cubierto | Hay texto sin tildes en titulo/epigrafes (`Clasificacion`, `basicos`); corregir castellano completo. |
| 8 | Manipulacion de alimentos, manipuladores, medio ambiente, residuos, trazabilidad | `raw/ab68a1.../content_block.md`, `artifact.json`, refs `tema-08` | Cubierto | Titulo HTML aparece como `Tema 8` en indice; corregir metadata. Normalizar tildes y fuentes sanitarias/trazabilidad. |
| 9 | Preparacion, conservacion, emplatado y transporte de alimentos | `raw/c58688.../content_block.json`, refs `tema-09` | Cubierto | Bloque breve frente al alcance practico; ampliar conservacion/transporte con temperaturas, separacion, proteccion y trazabilidad sin invadir tema 8. |
| 10 | PRL, riesgos, productos de limpieza, almacenamiento, incendio en cocina y actuaciones | `raw/5ddd69.../content_block.json`, refs `tema-10` | Cubierto | Reforzar fuentes de PRL, fichas de seguridad y planes de emergencia; coordinar con tema 5 para no repetir productos sin valor nuevo. |

## Hallazgos

### Correcciones necesarias

1. Fuentes insuficientes para cierre publicable. La mayoria de bloques especificos solo declaran la fuente oficial del programa (`tema-03` a `tema-10`). Para OPES publicable deben anadirse fuentes externas/normativas o tecnicas verificables: CE/LBRL para tema 1, TREBEP para tema 2, PRL, seguridad alimentaria, manipulacion, residuos, fichas de seguridad y documentacion tecnica de lavanderia/limpieza donde proceda.
2. Normalizacion linguistica pendiente. Hay titulos y epigrafes sin tildes en varios temas, visible especialmente en temas 7 y 8. El canon OPES exige castellano correcto con tildes, `ñ` y signos completos en texto visible, tests, tutor, HTML y audios.
3. Metadata irregular. El indice HTML muestra `Tema 8` sin titulo completo, aunque el contenido interno si lo tiene. Tambien conviven schemas `opes_content_block.v0`, `v1` y bloques `sin schema`. Para cierre, el ensamblado debe normalizar metadata y schema.
4. Capas OPES incompletas. El material revisado cubre texto base, pero no evidencia de resumido/ampliado separados, modo tutor completo, banco de tests por tema, visuales finales, audios, RAG/tutor, HTML local canonico USO, revisiones triples y paquete final.
5. Repeticion tematica entre temas 5, 6, 8 y 10. Es esperable por programa, pero debe gestionarse con referencias cruzadas y foco diferencial: tema 5 limpieza general, tema 6 cocina, tema 8 alimentos/residuos/trazabilidad, tema 10 PRL/incendio/productos.
6. Tema 9 queda mas corto y menos denso que otros temas especificos. Cubre epigrafe, pero necesita ampliacion practica para equilibrar preparacion, conservacion, emplatado y transporte.

### Mejoras opcionales

1. Crear infografias finales, no maquetas, para simbolos de prendas, ciclo de lavado, codificacion de utiles, flujo de limpieza de cocina, cadena alimentaria y actuacion ante incendio.
2. Anadir tablas comparativas: tipos de lavado, manchas/fibras, productos de limpieza, dietas, clases de riesgos y residuos.
3. Crear mini-casos practicos por tema con contexto de operario AP: incidencia en lavanderia, limpieza de office, conservacion en transporte, derrame de producto, conato de incendio.
4. Vincular temas comunes 1 y 2 a maestros AP/A1 ya existentes para evitar divergencias futuras.

## Contrato externo de dominio

- Orquesta no accedio a DB OPES ni modifico jobs. Esta entrega es un artefacto local en el write-set.
- Las refs cruzadas se mantienen opacas (`tema-01`...`tema-10`, `official_source_id`, `source_refs`).
- No se descarta material por formato recuperable: los problemas detectados se convierten en rework editorial o normalizacion.
- El contexto `ref_only` del paquete se resolvio por accion de evidencia: se uso el objetivo, `course_slug`, `expected_output`, programa oficial y salidas locales disponibles; no hizo falta inventar contexto.

## Cierre operativo propuesto

Aceptar la entrega como revision de cobertura y calidad, y abrir rework acotado:

1. Normalizar metadata/schema de los 10 bloques y corregir tildes.
2. Completar fuentes normativas/tecnicas por tema.
3. Separar resumido, ampliado, tutor y notas de test.
4. Ampliar tema 9 y deduplicar temas 5/6/8/10 por foco.
5. Pasar a validacion OPES de texto antes de generar audios, visuales finales, tests y HTML publicable.
