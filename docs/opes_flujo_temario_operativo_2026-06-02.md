# Flujo operativo OPES para crear un temario completo

Este documento fija el flujo que debe seguir OPES cuando el operador pide:
`creame el temario de <OPE>`. OPES es la app de dominio; Orquesta coordina
agentes y artefactos por contratos publicos. El objetivo no es generar solo
texto: el resultado debe quedar totalmente operativo en local antes de subirlo
a produccion.

## Resultado local obligatorio

Antes de produccion debe existir un paquete local revisable con:

- temario completo por temas y apartados;
- fuentes oficiales y evidencias de busqueda externa;
- infografias utiles y no excesivas en puntos importantes, dificiles,
  comparativos o procedimentales donde aporten aprendizaje;
- banco de tests por tema;
- revisiones legal, pedagogica, calidad, ortografia y consistencia;
- HTML local con logos USO y aspecto coherente con la web USO/TCAE promocion
  interna;
- audios por tema y por apartado/seccion;
- tutor y bots del temario;
- manifest de trazabilidad, checksums/refs y estado de validacion.

El operador revisa ese paquete local. Solo despues se crea la tarea de subida a
produccion.

## Secuencia de jobs OPES

La secuencia completa vigente es:

```text
plan_temario
research_exam_precedents
draft_content_block
generate_visual_asset
generate_question_bank
review_legal
review_pedagogical
review_quality
validate_topic
assemble_topic
generate_audio_asset
generate_tutor_assets
generate_html_site
```

`plan_temario` debe devolver un `document_plan` que cree esos derivados. Si una
fase no aplica, el plan debe justificarlo, no omitirla en silencio.

## Que hace cada fase

`plan_temario`: lee el listado oficial, inventaria temas, comunes,
transversales, especificos, dependencias, maestros superiores y derivados por
nivel. Debe planificar todas las fases posteriores, incluidos audios, tests,
infografias, tutor/bots y HTML local.

`research_exam_precedents`: busca por internet examenes, convocatorias,
temarios, supuestos y pruebas de administraciones relacionadas. Prioriza
boletines, sedes oficiales, tribunales, institutos publicos y fuentes
sindicales verificables. Devuelve URLs, fecha de consulta, administracion, ano,
categoria, epigrafes relacionados y utilidad editorial. No inventa examenes.

`draft_content_block`: redacta el contenido del tema o apartado con fuentes,
nivel, lenguaje claro y enfoque de examen.

`generate_visual_asset`: crea infografias o diagramas utiles por tema/apartado,
con `placement_ref`, texto alternativo y objetivo didactico. Debe integrarlas
con criterio editorial: puntos importantes, dificiles, comparativos o
procedimentales, incluidos apartados criticos cuando proceda, pero sin saturar
el tema ni crear visuales de relleno. OPES no decide si las hace Gemini, Codex,
Claude u otro agente; Orquesta asigna roles y OPES solo recibe `visual_asset`.

`generate_question_bank`: crea tests por tema. Por defecto: minimo 50 preguntas,
4 opciones A/B/C/D, una sola respuesta correcta exacta, distractores plausibles
y explicacion tutor de por que cada opcion falla o acierta y donde repasar. El
banco completo queda para afiliados; no se publica como test abierto completo.
Debe crear salida nueva, no sobrescribir bancos originales, y entregar JSON por
tema, HTML revisable por tema, `index.html`, `metadata.json`, informe Markdown y
validaciones estructural/dificultad limpias. Si OPES/USO importa en Postgres
local, antes debe existir backup, SQL con borrado limitado al banco nuevo y
verificacion de conteos. Para TCAE, seguir tambien la guia externa:
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/GUIA_AGENTES_CREACION_TESTS_TEMARIOS_TCAE_OPES_2026-06-02.md`.

`review_legal`, `review_pedagogical`, `review_quality`: revisan legalidad,
pedagogia, calidad editorial, ortografia, consistencia, nivel, trazabilidad,
fuentes, i18n, accesibilidad y aprovechamiento de artefactos. No tiran trabajo
recuperable: proponen rework, reaprovechamiento o tarea derivada.

`validate_topic`: valida contrato, schema, completitud, enlaces, assets,
ortografia basica, HTML offline, ausencia de rutas internas, banco de tests,
audios, infografias y coherencia con el epigrafe oficial.

`assemble_topic`: ensambla el paquete del tema con contenido, metadatos,
fuentes, visuales, tests, revisiones y manifests.

`generate_audio_asset`: crea audios del tema y de cada apartado/seccion. El
adaptador OPES puede usar RTX4090, `edge-tts` de Microsoft u otro motor. Hacia
Orquesta solo viajan refs opacas y `audio_asset`. Antes de generar TTS se
revisan numeros romanos para que se lean como numeros. Despues se debe escuchar
o validar con transcripcion automatica cuando el adaptador lo soporte.

`generate_tutor_assets`: crea tutor y bots del temario: intents, alcance por
tema/apartado, fuentes permitidas, diagnostico de errores de test, propuestas de
repaso y limites para no inventar fuera del paquete.

`generate_html_site`: crea el HTML local operativo del temario completo con
logos USO y aspecto USO/TCAE promocion interna, navegacion, modo estudio,
temas, infografias, audios, tutor/bots, tests permitidos y assets locales. Debe
funcionar en local antes de produccion.

## Frontera Orquesta/OPES

OPES decide producto, fuentes de dominio, validadores, UI, marca USO,
ensamblado, publicacion y adaptadores de audio/visual/tutor. Orquesta decide
orquestacion, agentes, roles, revision, rework, trazabilidad y entrega de
artefactos por refs opacas.

Los rails blandos no deciden. El Director o el agente orquestador decide si una
senal se ignora, se conserva como evidencia, se convierte en tarea o se retira
del flujo. Solo seguridad real, causalidad rota, refs imposibles, datos
sensibles efectivos o efectos externos no autorizados cortan fuerte.
