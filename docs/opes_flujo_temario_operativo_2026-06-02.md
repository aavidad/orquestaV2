# Flujo operativo OPES para crear un temario completo

Este documento fija el flujo que debe seguir OPES cuando el operador pide:
`creame el temario de <OPE>`. OPES es la app de dominio; Orquesta coordina
agentes y artefactos por contratos publicos. El objetivo no es generar solo
texto: el resultado debe quedar totalmente operativo en local antes de subirlo
a produccion.

Canon externo vigente: para reglas editoriales y operativas de temarios, usar
`/home/alberto/Trabajo/OPES/AGENTS.md` y los documentos que obliga a leer:
`ENTRADA_UNICA_SIGUIENTE_AGENTE_OPES.md`,
`GUIA_ESTILO_TEMARIOS_OPES_GLOBAL_2026-05-18.md` y
`USO_ORQUESTA_TEMARIOS_OPES_GUIA_AGENTES_2026-06-04.md`. Orquesta debe pasarlo
como politica de dominio OPES, no convertirlo en regla del nucleo generico.

## Resultado local obligatorio

Antes de produccion debe existir un paquete local revisable con:

- temario resumido completo por temas y apartados;
- temario ampliado completo por temas y apartados, como artefacto separado del
  resumido;
- fuentes oficiales y evidencias de busqueda externa;
- infografias utiles y no excesivas en puntos importantes, dificiles,
  comparativos o procedimentales donde aporten aprendizaje;
- banco de tests por tema;
- revisiones legal, pedagogica, calidad, ortografia y consistencia;
- HTML local con logos USO y el formato real de curso de la web USO/TCAE
  promocion interna: `index.html`, `html_final/`, assets locales,
  `audio/manifests/`, locales/i18n para la interfaz y capa protegida
  `#uso-material-watermark` cuando el curso sea material de estudio;
- audios por tema y por apartado/seccion;
- tutor y bots del temario;
- juegos, retos y revision de errores como modulos reutilizables del curso;
- manuales graficos de ayuda USO con capturas anotadas, HTML canonico, PDF y
  Markdown;
- triple visto bueno de Codex, Gemini y Claude con informes independientes;
- revisiones por pares Codex-Gemini, Codex-Claude y Gemini-Claude sobre curso,
  tests, visuales, audios, tutor, HTML y paquete;
- matriz de decision del Director con acuerdos, discrepancias, rework causal y
  cierre explicito;
- `completed_syllabus_package` con manifest de trazabilidad, checksums/refs,
  matriz de revisiones y estado de validacion.

El operador revisa ese paquete local. Solo despues se crea la tarea de subida a
produccion.

## Secuencia de jobs OPES

La secuencia completa vigente es:

```text
plan_temario
update_topic_registry
research_exam_precedents
draft_content_block
generate_visual_asset
generate_question_bank
review_legal
review_pedagogical
review_quality
review_codex
review_gemini
review_claude
review_pair_codex_gemini
review_pair_codex_claude
review_pair_gemini_claude
review_director_consolidation
validate_topic
assemble_topic
generate_audio_asset
generate_tutor_assets
generate_learning_games
generate_html_site
generate_help_manual_assets
finalize_temario_package
```

`plan_temario` debe devolver un `document_plan` que cree esos derivados. Si una
fase no aplica, el plan debe justificarlo, no omitirla en silencio.
Los jobs `generate_agent_candidate_*`, `vote_agent_candidates_*` y
`select_agent_candidate_director` son trabajos de rework opcionales para piezas
concretas flojas. Siguen soportados por contrato, pero no forman parte de la
cola principal de cada temario salvo que una revision o el Director los abra con
refs causales.

## Que hace cada fase

`plan_temario`: lee el listado oficial, inventaria temas, comunes,
transversales, especificos, dependencias, maestros superiores y derivados por
nivel. Debe planificar todas las fases posteriores, incluidos audios, tests,
infografias, tutor/bots, HTML local y manuales graficos de ayuda.

Regla de reutilizacion para todos los temarios: antes de redactar o generar
derivados, OPES debe inventariar materiales existentes compatibles: temas,
comunes, tests, audios, visuales, HTML, tutor/RAG, manifests y paquetes
publicables. Cada material se revisa contra el programa oficial del nuevo curso.
Lo valido se copia o referencia dentro del nuevo trabajo; lo recuperable se
conserva como borrador, insumo o tarea de adaptacion; solo se rehace lo que este
obsoleto, sea incorrecto, no cubra el programa o tenga un fallo estructural real.
Diferencias de titulo, alias, orden, metadatos o formato reparable se normalizan
y no justifican tirar trabajo util.

Regla de comunes para todos los temarios: OPES debe buscar primero los temas
comunes ya creados por grupo, subgrupo, nivel o materia equivalente. La busqueda
incluye contenido, tests, visuales, audios, manifests y refs de paquete. Si
existen refs compatibles, se reutilizan antes de generar de nuevo; si no encajan
exactamente, se conservan como insumo y se adaptan con una derivacion localizada
al nuevo cuerpo/OPE. Solo se rehace un comun cuando haya obsolescencia, error
grave, falta real de encaje o una orden expresa. Diferencias de titulo, alias,
orden, metadatos o formato reparable se normalizan y no justifican tirar trabajo
util.

`update_topic_registry`: actualiza o prepara la actualizacion causal del
registro global de temas OPES para el `course_id`/`topic_id` recibido. No
produce contenido docente; entrega `topic_registry_update` con estado, bloqueo
publico si falta conector y evidencias de lock/registro para que el tema no se
marque listo sin asiento trazable.

`research_exam_precedents`: busca por internet examenes, convocatorias,
temarios, supuestos y pruebas de administraciones relacionadas. Prioriza
boletines, sedes oficiales, tribunales, institutos publicos y fuentes
sindicales verificables. Devuelve URLs, fecha de consulta, administracion, ano,
categoria, epigrafes relacionados y utilidad editorial. No inventa examenes.

`draft_content_block`: redacta el contenido del tema o apartado con fuentes,
nivel y lenguaje claro. OPES debe producir dos capas editoriales separadas:
temario resumido y temario ampliado. El temario resumido puede incluir enfoque
de estudio, test o repaso cuando el curso lo requiera. El temario ampliado debe
desarrollar y explicar cada punto con continuidad didactica, ejemplos utiles y
tablas cuando aporten claridad, pero no debe incluir frases de preparacion de
examen, "para una prueba tipo test", "puntos de examen", preguntas tipo test ni
instrucciones de estudio. Si esas piezas existen, quedan en el resumen, banco de
tests, tutor o material de repaso, no en el texto ampliado.

`generate_visual_asset`: crea infografias o diagramas utiles por tema/apartado,
con `placement_ref`, texto alternativo y objetivo didactico. Debe integrarlas
con criterio editorial: puntos importantes, dificiles, comparativos o
procedimentales, incluidos apartados criticos cuando proceda, pero sin saturar
el tema ni crear visuales de relleno. La colocacion final debe respetar el
`placement_ref`: cada infografia queda junto al apartado o parrafo que explica,
no acumulada al inicio del tema salvo mapa inicial deliberado. OPES no decide si
las hace Gemini, Codex, Claude u otro agente; Orquesta asigna roles y OPES solo
recibe `visual_asset`. Antes de que un visual entre en paquete publicable, OPES
debe pasarlo por compresion JPG/WebP, eliminar metadatos y conservar originales
fuera del paquete. El comando canonico vive en OPES:
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/tools/compress_course_images.py`.
Esto es normalizacion de artefactos, no una razon para descartar trabajos
recuperables.

El flujo debe distinguir `brief_visual`, `maqueta_visual` y
`arte_final_visual`. Un SVG rapido, una figura humana pobre, cajas con texto o
una tabla maquillada puede servir como maqueta para decidir direccion, pero no
debe entrar como infografia final ni contarse como visto bueno de Gemini. El
arte final visual debe venir del rol visual real y pasar revision profesional
antes de integrarse.

Para procedimientos fisicos, OPES debe pedir foto/imagen realista de Gemini o
rol visual equivalente: posturas del paciente, movilizacion, transferencias,
higiene, sondajes, cocina, ergonomia, limpieza, seguridad y utensilios. SVG,
texto o diagramas quedan solo como apoyo puntual para menus, overlays, leyendas
o conceptos abstractos.

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
recuperable: proponen rework, reaprovechamiento o tarea derivada, y el Director
decide.
Reparto vigente de capacidad: Codex Pro asume el trabajo pesado de extraccion,
reutilizacion, redaccion, integracion, HTML, audios, paquetes, API y validacion
local. Gemini y Claude se usan con encargos compactos por tema, apartado, visual
o banco de tests: Gemini para fotos/imagenes realistas, visuales y opinion
visual; Claude para textos, pedagogia, tests, distractores, explicaciones y
tutor. OPES sigue pidiendo roles/artefactos, no proveedores concretos; Orquesta
o el Director asigna esos roles. Regla canonica:
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/REGLA_REPARTO_CAPACIDAD_AGENTES_OPES_2026-06-04.md`.
Por capacidad, Gemini y Claude se usan cuanto menos mejor salvo en tests: antes
de publicar un banco, Codex, Gemini y Claude deben revisar el 100% de preguntas
y respuestas/opciones, por lotes si hace falta. Regla canonica:
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/REGLA_REVISION_TESTS_TRES_MODELOS_OPES_2026-06-04.md`.
Cuando una pieza concreta sea floja, Orquesta puede pedir candidatos
alternativos a Codex, Gemini y Claude, conservar todos los candidatos y pedir
voto de los tres antes de que el Director seleccione, fusione o pida rework. La
regla canonica es:
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/REGLA_CANDIDATOS_Y_VOTACION_TRES_AGENTES_OPES_2026-06-05.md`.

`review_codex`, `review_gemini`, `review_claude`: generan informes
independientes del curso completo. Codex revisa tecnica, integracion, paquete,
HTML local, audios y trazabilidad; Gemini revisa visuales, infografias,
legibilidad, experiencia local y evidencia visual; Claude revisa editorial,
pedagogia, tests, distractores, explicaciones y tutor. Los tres deben revisar
el 100% de tests publicables/importables, por lotes si hace falta, y conservar
evidencia. Si falta una revision obligatoria, el curso queda pendiente.

`review_pair_codex_gemini`, `review_pair_codex_claude`,
`review_pair_gemini_claude`: comparan las revisiones independientes por pares.
Cada par debe devolver acuerdos, discrepancias, riesgos, decisiones propuestas y
rework causal con refs. Las discrepancias no bloquean por una palabra o formato
recuperable: se normalizan, se convierten en rework o pasan al Director para
decision.

`review_director_consolidation`: consolida las revisiones independientes y por
pares. El Director decide que se acepta, que se reusa, que queda como borrador,
que requiere rework y que impide cierre. Debe dejar una matriz final con
evidencia por tema, tests, visuales, audios, tutor, HTML, manuales y paquete.
Sin matriz cerrada no se puede marcar listo.

`generate_agent_candidate_codex`, `generate_agent_candidate_gemini`,
`generate_agent_candidate_claude`: crean candidatos alternativos para una pieza
concreta cuando el material existente no da nivel o un revisor propone una
version mejor. Se usan para tests, distractores, explicaciones, tutor, visuales,
resumenes o bloques acotados. Cada candidato se guarda como artefacto separado:
no sobrescribe el original ni descarta candidatos de otros agentes.

`vote_agent_candidates_codex`, `vote_agent_candidates_gemini`,
`vote_agent_candidates_claude`: cada agente compara original y candidatos,
ordena las opciones, propone fusion cuando proceda y deja evidencia. La votacion
no es un rail automatico ni una media ciega: informa la decision del Director.

`select_agent_candidate_director`: consolida los votos y selecciona el
candidato ganador, fusiona partes aprovechables, conserva descartados como
borrador/evidencia o abre rework acotado.

`validate_topic`: valida contrato, schema, completitud, enlaces, assets,
ortografia basica, HTML offline, ausencia de rutas internas, banco de tests,
audios, infografias y coherencia con el epigrafe oficial.

`assemble_topic`: ensambla el paquete del tema con contenido, metadatos,
fuentes, visuales, tests, revisiones y manifests.

`generate_audio_asset`: crea audios del tema y de cada apartado/seccion. El
adaptador OPES puede usar RTX4090, `edge-tts` de Microsoft u otro motor. Hacia
Orquesta solo viajan refs opacas y `audio_asset`. Esta fase va despues del
texto ensamblado publicable aprobado, corregido y cerrado; no se ejecuta sobre
borradores ni sobre temas con rework editorial pendiente. Se ejecuta antes de
`generate_html_site` para que el HTML integre el manifest de audios final.
Antes de generar TTS, OPES debe resolver `common_topic_ref`, `source_content_ref`,
`audio_manifest_ref` y `audio_ref` existentes y reutilizar audio comun
compatible. Si el texto final no coincide, el comun se conserva como candidato y
se crea una derivacion localizada. Antes de TTS tambien se revisan numeros
romanos para que se lean como numeros. Despues se debe escuchar o validar con
transcripcion automatica cuando el adaptador lo soporte. Si el tema tiene
apartados, el artefacto no queda cerrado con un unico MP3: debe entregar
`segments` con una entrada por
apartado/seccion narrable (`section_ref`, `audio_ref`, duracion, hash/ref de
texto y estado de reutilizacion o generacion).

Regla de audio por apartado: cada `section_ref` debe apuntar a un unico MP3
final del apartado, no a varios audios visibles ni a capas superpuestas. Ese MP3
se genera desde el contenido final real que vera el alumnado, preferiblemente
desde el modelo HTML/DOM que usara `generate_html_site`: titulo del apartado,
parrafos, listas, listas
anidadas, tablas, esquemas, texto alternativo/descripcion de infografias y notas
didacticas visibles. Si una tabla, esquema o lista no se puede narrar de forma
clara, se normaliza el texto narrable y se regenera el MP3 del apartado; no se
parchea anadiendo varios audios al mismo apartado. La validacion por Whisper u
otro transcriptor debe comprobar muestras con tablas/listas/esquemas, no solo
parrafos planos.

`generate_tutor_assets`: crea tutor y bots del temario: intents, alcance por
tema/apartado, fuentes permitidas, diagnostico de errores de test, propuestas de
repaso, tips del modo tutor, ayudas a tests y limites para no inventar fuera
del paquete. Los tips deben incluir ejemplos cuando aclaren el criterio. Ejemplo
de tutor para un fallo de test: "Has confundido competencia propia con
competencia delegada. Revisa el apartado de competencias municipales y fijate en
quien conserva la responsabilidad final". Ejemplo de ayuda previa a responder:
"Busca en la pregunta si se habla de titularidad, ejecucion o supervision; no
son la misma cosa".

`generate_learning_games`: crea juegos, retos y revision de errores como paquete
modular (`learning_games_package`) que pueda pegarse a cualquier curso sin
tocar el nucleo web ni reescribir temarios. Debe incluir al menos los juegos
vigentes cuando el curso tenga datos suficientes: parejas, completa la frase,
retos por tema, repaso de errores fallados y auditoria del tutor. Cada juego
debe consumir manifest/test/tema por refs, no por rutas internas ni SQL directo.
Si manana se anade otro juego, entra como nuevo modulo con manifest propio,
assets propios, i18n propio y registro en el paquete final, sin modificar el
contrato de contenidos del temario.

`generate_html_site`: crea el HTML local operativo del temario completo con
logos USO y el formato real de curso USO/TCAE promocion interna, navegacion,
modo estudio, temas, infografias, audios, tutor/bots, tests permitidos, juegos,
retos, revision de errores y assets locales. La salida final debe ser
estructura de curso (`index.html`,
`html_final/`, assets, `audio/manifests/`, locales/i18n), no una maqueta
single-file con estilo propio. El texto visible para alumnado no debe explicar
como se genero el temario ni mostrar reutilizacion de comunes, refs, manifests,
OPES, agentes, backend, staging, rutas o trazabilidad tecnica; eso queda en
metadata o informes internos. Si el curso es material de estudio, debe incluir la
capa protegida de la web TCAE/afiliados: marca de agua diagonal USO visible
`#uso-material-watermark`, fondo de agua coherente, sin tapar contenido ni
infografias. Antes de empaquetar, ejecutar la compresion de imagenes del curso y
verificar que `html_final/img` no contiene PNG/JPEG brutos enormes ni carpetas de
revision internas. Debe funcionar en local antes de produccion.

`generate_help_manual_assets`: crea los manuales graficos de ayuda para USO
cuando ya existe HTML local revisable. La guia operativa es
`/home/alberto/Trabajo/USO/web/docs/SCREENSHOT_HELP_MANUALS.md` y la regla de
marca es
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/REGLA_ARTEFACTOS_USO_BRANDING_2026-06-02.md`.
El flujo correcto es YAML de escenario -> capturas/anotaciones -> `index.html`
canonico -> `manual.pdf` exportado desde HTML -> `manual.md`, con carpetas
`img/` para capturas anotadas y `raw/` para capturas originales. El manual debe
incluir logo USO, marca de agua cuando proceda, pie con web/correo/telefono,
comentarios fuera de la captura mediante notas numeradas, imagenes ajustadas a
A4 y revision visual de HTML/PDF antes de entregar.

`finalize_temario_package`: cierra el temario al 100% como
`completed_syllabus_package`. Debe reunir temario resumido, temario ampliado,
fuentes, tests, visuales finales comprimidos, audios por tema/apartado, tutor,
bots, juegos/retos, revision de errores, HTML local, manuales, manifests,
checksums, validacion visual y matriz de revision cerrada. La salida es
candidato local para revision del operador, no subida automatica a produccion.

## Ciclo autonomo Orquesta

La via operativa para continuar un temario hasta cierre es:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=3 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id> \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF=<scope-probe-evidence-ref> \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=<correlation_id> \
ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH=<ledger-temario>.json \
go run ./cmd/orquesta-server opes-temario-cycle
```

`127.0.0.1` no prueba por si solo que OPES sea temporal: el operador debe
confirmar instancia aislada con `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y scope
acotado por `correlation_id`, `topic_id`, cola temporal dedicada o
`program_id` probado con evidencia. Si la instancia OPES no es loopback, debe
existir ademas una evidencia de destino temporal/no productivo antes de ejecutar
efectos.

Para OPES, `/healthz` es solo liveness del proceso. Antes de lanzar
`domain-work`, `external-work/run` o un ciclo de temario hay que comprobar
`GET /api/v0/server/readiness` y exigir `ready=true`. Si readiness devuelve
`external_work_goal_backend_required`, la accion unica es reiniciar Orquesta con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`; OPES debe marcar
`orquesta_degraded_not_ready` y no lanzar jobs ni hacer fallback silencioso.

Si `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE` no esta definido, el comando usa
`OPESFullTemarioJobTypeSequenceV0()`, que empieza en `plan_temario` e incluye
revisiones Codex/Gemini/Claude, revisiones por pares, consolidacion del
Director, HTML, manuales y `finalize_temario_package`. El ciclo no corta por
palabras, alias ni formatos
recuperables: conserva errores o entregas parciales como evidencia y sigue hasta
que OPES no tenga pendientes tras el cierre, o hasta
`ORQUESTA_OPES_BRIDGE_MAX_TICKS` para detectar un bucle real.

## Frontera Orquesta/OPES

OPES decide producto, fuentes de dominio, validadores, UI, marca USO,
ensamblado, publicacion y adaptadores de audio/visual/tutor. Orquesta decide
orquestacion, agentes, roles, revision, rework, trazabilidad y entrega de
artefactos por refs opacas.

### Nota operativa 2026-06-04: refs compactas y rutas

En `external-work/run`, `metadata_refs`, `current_state_refs`,
`interface_refs` y `work_refs` no deben transportar rutas reales. Las rutas
pertenecen a `allowed_write_set` cuando son destino de escritura o a
`external_work.input_fields` cuando son contexto del agente.

El incidente Auxiliar Administrativo C2 de 2026-06-04 ya cerro este caso: una
ola de 20 temas fallo inicialmente con `app_change_metadata_ref_invalid` /
`app_change_external_work_ref_invalid` por rutas recuperables en refs. Orquesta
debe normalizarlas antes de validar; si este mismo caso vuelve a bloquear una
ola, se considera regresion grave.

### Nota operativa 2026-06-04: snapshots no bloqueantes

Los artefactos locales del servidor temporal (`bin/`, `orquesta_state/`,
`runtime_orquesta/` u otros equivalentes de runtime/estado) no forman parte del
contenido del temario y no deben entrar en snapshots Codex del stack.

La captura de baseline de worktree es evidencia auxiliar. Si falla por tamano,
lectura o artefacto local recuperable, el agente debe lanzarse igualmente y el
descriptor queda sin `WorktreeBaselineRef`. La revision posterior puede dejar
`worktree_baseline_missing` como evidencia blanda, pero no bloquear el trabajo.
El incidente Auxiliar Administrativo C2 de 2026-06-04 ya cerro este caso con
`worktree_snapshot_file_too_large` sobre `bin/orquesta-server`; si vuelve a
parar un `LaunchRuntimeAgent`, se considera regresion grave.

Los rails blandos no deciden. El Director o el agente orquestador decide si una
senal se ignora, se conserva como evidencia, se convierte en tarea o se retira
del flujo. Solo seguridad real, causalidad rota, refs imposibles, datos
sensibles efectivos o efectos externos no autorizados cortan fuerte.

### Nota operativa 2026-06-04: prompts sin rails bloqueantes

Los prompts Codex/OPES no deben convertir alcance recuperable, contexto
truncado, rutas fuera de write-set, pruebas fallidas o diagnosticos de formato
en instrucciones de `failed`, invalidacion de entrega o consulta obligatoria al
Director. El agente debe continuar con lo permitido, conservar avance parcial y
dejar notas/tareas derivadas para que el Director revise o replantee.

El incidente Auxiliar Administrativo C2 de 2026-06-04 ya cerro este caso:
aparecieron instrucciones duras heredadas en prompts de runtime (`RAIL
ESTRICTO`, `ACK failed`, invalidacion por protocolo y `write_set_closed` como
precedencia bloqueante). Esos textos son advisory o diagnostico como maximo. Si
vuelven a parar una entrega recuperable o a marcar strict por defecto, se
considera regresion grave.

### Nota operativa 2026-06-04: logs libres no bloquean proveedor

Un texto libre de log, ejemplo, diagnostico o nota documental no puede
convertirse por si solo en `provider_auth_blocked`, `capacity_limited`, `failed`
ni paro automatico de agentes. Las senales reales de proveedor/capacidad deben
venir de estado estructurado del runtime, ACK causal o frontera externa
confirmada; si solo hay texto ambiguo, se conserva como evidencia blanda y el
Director decide.

El incidente Auxiliar Administrativo C2 de 2026-06-04 ya cerro este caso: un
log mencionaba rutas, middleware, `401` o autenticacion como contexto y Orquesta
lo interpreto como `provider_auth_blocked`. Si vuelve a parar un agente o run
por texto libre recuperable, se considera regresion grave.

### Nota operativa 2026-06-04: cola residente alineada con capacidad amplia

La composicion Codex debe arrancar con capacidad amplia coherente: 70 padres o
ejecuciones por tick y `ORQUESTA_SERVER_QUEUE_LIMIT=70` por defecto. No debe
quedar un cuello silencioso menor que haga creer que Orquesta esta llena cuando
solo esta mirando una cola corta.

El incidente Auxiliar Administrativo C2 de 2026-06-04 ya cerro este caso: el
servidor temporal se lanzo con capacidad 70, pero el default de cola residente
seguia en 20 y dejo recuperaciones posteriores fuera del supervisor global. Si
vuelve a bloquear una ola amplia con `capacity_full` por un limite interno
inferior no declarado, se considera regresion grave.

### Nota operativa 2026-06-04: supervision dirigida secuencial

Cuando haya que recuperar runs external-work legacy o no migrados con
`POST /api/v0/runs/supervise`, debe existir opt-in operativo explicito
(`ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1` y
`director_execution_mode=legacy_director_loop` en el payload que cree o empuje
la run legacy) y las llamadas dirigidas
contra la misma instancia temporal deben hacerse secuencialmente salvo evidencia
de que el ledger/dispatcher de esa composicion soporta esa concurrencia real.
Para runs `goal_first`, la ruta normal es observar/cerrar por
`/api/v0/apps/director/goal/observe`, no empujar el loop historico. No es una
regla para parar agentes: evita chocar dos ciclos de supervision sobre el mismo
estado durable.

El incidente Auxiliar Administrativo C2 de 2026-06-04 ya dejo este rastro: dos
supervisiones dirigidas paralelas sobre recoveries 13/15 provocaron un error
`outbox ledger fallo` en una llamada, aunque los agentes terminaron despues y
la ingesta secuencial cerro ambos en `done/quiescent`. Si vuelve a aparecer en
supervision dirigida secuencial, o si una llamada paralela corrompe/para trabajo
en vez de conservarlo, se considera regresion grave.
