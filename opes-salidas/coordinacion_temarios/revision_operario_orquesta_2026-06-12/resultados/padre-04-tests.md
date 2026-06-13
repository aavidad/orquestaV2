# Revisión padre-04: tests Operario AP

Fecha: 2026-06-12.
Curso: `ope-operario`.
Contrato: revisar banco de tests, alineación con temario, distractores, tutor y estado de importación. No editar banco en esta ola.

## Veredicto

Entrega aceptada como revisión externa completada.

El banco v2 es aprovechable y mejora el patrón de distractores frente a una versión fácil, pero no debe marcarse como cerrado editorialmente sin rework parcial: las preguntas solo declaran procedencia por tema (`tema_01` a `tema_10`), no por apartado/cita del temario final, y una parte relevante del tutor usa explicaciones de plantilla.

## Evidencias revisadas

- Paquete Orquesta: `agent_packet.json` y payload OPES `padre-04-tests.json`.
- Estado conocido del paquete: producción tiene 10 tests y 500 preguntas publicadas; staging local 2026-06-03 tiene `course_tests.json` parcial de 1 pregunta.
- Banco local v2: `/home/alberto/Trabajo/OPES/opes-salidas/operario_ap_tests_2026-06-03-v2/question_banks`.
- Informe v2: `/home/alberto/Trabajo/OPES/opes-salidas/operario_ap_tests_2026-06-03-v2/INFORME_VALIDACION_TESTS_OPERARIO_AP_V2_2026-06-03.md`.
- Despliegue v2: `/home/alberto/Trabajo/OPES/opes-salidas/operario_ap_tests_2026-06-03-v2/DESPLIEGUE_TESTS_OPERARIO_AP_V2_2026-06-03.md`.
- Temario publicado previo: `produccion_externa_2026-05-19/markdown_final` y sitio corregido `operario_ap_audio_2026-06-02/site_operario_ap_20260602_corregido`.
- Manifiestos nuevos de curso: `diputacion_granada/cursos_opes_2026/administracion_especial/operario`.

## Conteo local y remoto

Producción/remoto:

- Según contexto de Orquesta: 10 tests y 500 preguntas publicadas.
- Según informe de despliegue v2: `tests|10`, `questions|500`, `options|2000`, `links|10`, `bad_option_shape|0`, `metaacademic_patterns|0`.
- No he ejecutado consulta live contra producción ni API externa; esta revisión conserva el dato como evidencia documental y del paquete.

Local:

- `question_banks` v2 contiene 10 ficheros.
- Conteo por fichero: 50 preguntas en cada tema.
- Total local v2: 500 preguntas y 2000 opciones.
- Validación estructural focal: 0 preguntas con forma inválida en opciones, correcta única o `source_anchor` fuera del patrón `tema_XX`.
- Staging local indicado por Orquesta: `course_tests.json` parcial de 1 pregunta. No debe usarse como fuente de publicación ni como prueba de sincronización completa.

## Alineación con temario

Resultado: alineación plausible por tema, no cerrada al 100% por falta de ancla fina.

Evidencia positiva:

- Los 10 títulos del banco coinciden con el programa Operario AP: Constitución/Administración local, empleados públicos, plancha, lavado, limpieza, cocinas, alimentos, manipulación/residuos/trazabilidad, conservación/transporte y PRL/incendio.
- Muestra focal del tema 4 encaja con el HTML publicado: clasificación de prendas, tipos de lavado, lavado industrial, lavadoras/secadoras, manchas y tratamiento aparecen en el temario publicado.
- El banco declara `source_anchor` en todas las preguntas.

Riesgo:

- Las 500 preguntas declaran solo `tema_01`...`tema_10`. No hay `section_ref`, cita textual, apartado, línea ni evidencia mínima que permita verificar pregunta por pregunta contra el texto final.
- Los manifests nuevos del curso `diputacion_granada/.../operario` tienen `status: pendiente_redaccion` en temas revisados. Por tanto, no sirven como temario terminado. El cierre debe apuntar al paquete publicado/corregido que sí contiene HTML/Markdown final, o actualizar manifests antes de cerrar.

Rework causal:

- Añadir ancla por pregunta: `topic_ref`, `section_ref`, fragmento/cita corta o hash de bloque del HTML/Markdown final.
- Marcar como `fuera_de_temario` cualquier pregunta sin evidencia textual verificable, o abrir rework del temario si el concepto debe estar incluido.

## Distractores

Resultado: mejorados, pero con patrón mecánico todavía visible.

Evidencia positiva:

- La v2 usa distractores del mismo bloque y evita muchas falsas absurdas.
- Distribución de respuestas por tema equilibrada: por cada tema, A=13, B=13, C=12, D=12.
- No se detectaron patrones metaacadémicos principales buscados: `No es la mejor respuesta`, `se justifica porque`, `define mejor` repetido de forma dominante, `todas son correctas`, `ninguna de las anteriores`.

Riesgo:

- Plantillas de enunciado repetidas: al menos 5 moldes aparecen 20 veces cada uno.
- Plantillas de explicación repetidas: 170 explicaciones usan `Las otras opciones pertenecen al mismo bloque...`; 160 usan `Los otros fallos pueden aparecer...`.
- En temas 1 y 2 aparecen distractores con fórmula de comodidad del servicio. No son necesariamente inválidos, pero deben revisarse porque el canon pide evitar distractores obvios o por cambio mecánico.

Rework causal:

- Revisar muestra ampliada o 100% por modelo editorial para romper plantillas en explicaciones.
- En cada pregunta, explicar por qué falla cada distractor concreto, no solo que pertenece a otra situación.
- Mantener distractores cercanos, pero sustituir los que se descarten por sentido común sin conocer el temario.

## Tutor y procedencia

Resultado: tutor funcional como explicación mínima; insuficiente para cierre OPES de calidad alta.

Hallazgos:

- Las explicaciones identifican la correcta y el matiz central.
- Faltan referencias explícitas de repaso: apartado, título de sección o bloque textual exacto.
- Muchas explicaciones no explican el error conceptual de cada opción; agrupan distractores en una frase genérica.

Rework causal:

- Convertir `explanation` en tutor de fallo por opción: correcta, error de A/B/C/D y dónde repasar.
- Añadir `provenance` por pregunta: fuente del temario publicado, sección y evidencia corta.
- Generar informe de procedencia por tema antes de aceptar importación nueva.

## Riesgos de publicación

- No aceptar `course_tests.json` parcial de 1 pregunta como estado local completo.
- No cerrar contra manifests con `pendiente_redaccion`.
- No declarar revisión 100% contra temario final hasta que las 500 preguntas tengan ancla verificable.
- Producción puede estar correcta en conteo, pero el estado documental local debe sincronizar `question_banks`, `course_tests` completo, informe de procedencia y paquete final.

## Decisión recomendada

Estado recomendado: `rework_parcial_requerido`.

Acciones mínimas antes de cierre:

1. Materializar `course_tests.json` completo desde `question_banks` v2 o documentar que producción usa otra fuente canónica.
2. Añadir anclas por apartado/cita para las 500 preguntas.
3. Revisar tutor/distractores al 100% por lotes, con foco en explicaciones de plantilla.
4. Actualizar o excluir los manifests `pendiente_redaccion` del criterio de cierre; usar el paquete HTML/Markdown publicado si es el temario final vigente.

## Validación del contrato externo

Contrato cumplido dentro del write-set:

- Se entrega informe Markdown en la ruta esperada.
- No se edita banco, producción, staging, jobs OPES ni ficheros fuera del write-set.
- Se tratan OPES y Orquesta como dominios separados por refs/artefactos, sin acceso directo a persistencia productiva ni efectos externos.
- Contexto `ref_only` resuelto parcialmente por paquete, payload y evidencias locales; queda documentado que no hubo consulta live remota.
