# orquesta-opes-bridge

Adaptador opt-in para tomar jobs externos pendientes de OPES y convertirlos en
runs de Orquesta mediante `/api/v0/external-work/run`.

No pertenece al nucleo. OPES sigue siendo la app de dominio editorial y
Orquesta sigue siendo el nucleo de orquestacion.

Para `plan_tema`, `plan_temario` y `plan_documento`, el bridge no solo declara
`document_plan`: tambien pasa al director la politica editorial OPES. En
particular, si existe maestro A1/A2 o A1 equivalente, el plan debe crear o
validar primero ese maestro y despues derivar B/C1/C2/AP por resumen,
reduccion editorial y adaptacion de nivel.

Para `plan_temario`, el contrato vigente exige el flujo local completo antes de
produccion: investigacion de examenes y temarios relacionados, redaccion,
infografias, banco de tests, revisiones, ensamblado, audios por tema/apartado,
tutor/bots, HTML local con logos USO y aspecto USO/TCAE promocion interna y
manuales graficos de ayuda USO derivados del HTML local. La secuencia canonica esta en
`docs/opes_flujo_temario_operativo_2026-06-02.md`.

El HTML de temas OPES no queda a criterio de cada agente: el contrato inyecta
`opes_html_topic_template_v1` y el renderer
`modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py`, basado en
`modulos/orquesta-opes-bridge/templates/opes_html_topic_template_v1.html`.
Los agentes aportan Markdown, metadatos y assets locales; el script monta la
estructura web canonica tipo Tema 11 y valida referencias locales.
El cierre mecanico de cada paquete se comprueba con
`modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`: palabras A1, duplicados largos,
HTML offline, banco JSON de 50 preguntas con 4 opciones, SVG y fugas de rutas
internas. El banco es por tema, queda junto a su temario y se reserva para la
parte de afiliados; no hay banco comun ni test de prueba publico.

## Validacion

```sh
go test -count=1 ./modulos/orquesta-opes-bridge
```

## Uso

Dry-run:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_DRY_RUN=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario \
go run ./cmd/orquesta-server opes-drain-once
```

Ejecucion real:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario \
go run ./cmd/orquesta-server opes-drain-once
```

Para abrir o continuar un temario distinto sin que trabajos anteriores ocupen
la cola, acota por programa/correlacion ademas del tipo o secuencia:

```sh
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id> \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=<correlation_id> \
ORQUESTA_OPES_BRIDGE_JOB_TYPE=plan_temario
```

Ese scope solo separa temarios y evita mezclar colas; no es un filtro de
calidad, formato ni palabras exactas de los agentes.

Modo residente por pases:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id> \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=<correlation_id> \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=plan_temario,update_topic_registry,research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site,generate_help_manual_assets,finalize_temario_package \
go run ./cmd/orquesta-server run
```

La secuencia consulta los tipos en orden y solo drena el primer tipo con jobs
`pending`. Si el ledger marca un job como `already_submitted`, no lo reenvia y
no avanza a fases posteriores hasta que OPES deje de mostrar pendientes de ese
tipo.

Modo autonomo acotado para cerrar temario:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=3 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id> \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=<correlation_id> \
ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH=<ruta-ledger-temario>.json \
go run ./cmd/orquesta-server opes-temario-cycle
```

`opes-temario-cycle` usa por defecto la secuencia canonica completa
`OPESFullTemarioJobTypeSequenceV0()`, desde `plan_temario` hasta
`finalize_temario_package`. En cada tick llama al bridge existente, conserva
errores recuperables como evidencia y continua. Solo termina como `completed`
cuando ya ha visto `finalize_temario_package` y despues OPES devuelve la
secuencia completa sin jobs `pending`. Si no llega a ese punto, devuelve
`continuable` o `pending` con `stop_reason`; el trabajo queda en ledger/run y se
puede relanzar el mismo comando.

Variables relevantes del ciclo:

- `ORQUESTA_OPES_BRIDGE_MAX_TICKS`: maximo de ticks antes de declarar bucle o
  trabajo continuable. Por defecto son 1000 para no cortar temarios amplios.
- `ORQUESTA_OPES_BRIDGE_INTERVAL_SECONDS`: pausa entre ticks. Por defecto son 5
  segundos en el comando autonomo.
- `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`: opcional, sustituye la secuencia
  canonica si una composicion OPES concreta necesita una secuencia distinta.

El scope por `program_id`, `topic_id` o `correlation_id` separa trabajos y evita
mezclar colas. No es un rail de contenido: no rechaza entregas por palabras,
alias, formato recuperable ni resultados parciales.

## Estado T12

T12 queda reconciliada como bloqueo verificable para smoke real OPES temporal:
los tests de bridge/conector y el fake `run-until-assemble` validan la ruta
local de derivados; historicamente llegaba a `assemble_topic -> assembled_topic`
y la secuencia vigente anade investigacion externa, tests, audio, tutor/bots,
HTML local y manuales graficos de ayuda. Falta entorno OPES temporal, servidor
Orquesta temporal, confirmacion de efectos y cuota/modelo real. No repetir
implementaciones padre para generar la misma evidencia fake.

## Audio accesible

OPES debe crear tambien jobs `generate_audio_asset` para producir el artefacto
`audio_asset` de cada tema publicable y de sus apartados/secciones. El bridge lo
trata como trabajo de dominio posterior a `assemble_topic`: recibe refs opacas
del tema ensamblado o paquete final y devuelve un manifest de audio con idioma,
formatos, duracion, refs/checksums de artefactos y mapa `section_ref ->
audio_ref`.

Si el tema tiene apartados, un unico `audio_ref` global no cierra el trabajo:
debe entregarse tambien `segments` con una entrada por apartado/seccion narrable,
incluyendo `section_ref`, `audio_ref`, duracion, hash/ref de texto y estado de
reutilizacion (`reused_common`) o generacion. Los temas comunes, como
Constitucion ya creado en TCAE, se resuelven primero contra `audio_manifest_ref`
y solo se sintetiza lo que falte o no encaje.

La app local que use la RTX4090 o una integracion `edge-tts` de Microsoft es un
adaptador de composicion OPES. Por calidad observada, `edge-tts` puede ser la
opcion preferente del adaptador de audio OPES, pero no forma parte del nucleo
Orquesta ni del contrato `orquesta-domain-work`: hacia Orquesta solo deben
viajar `work_kind=generate_audio_asset`, `artifact_type=audio_asset`,
`audio_profile_ref` y refs opacas. Alias como `generate_topic_audio`,
`tts_topic` o `audio_tema` se normalizan sin tirar el trabajo.

Antes de TTS, el adaptador OPES debe revisar titulos con numeros romanos para
que se lean como numeros. Tras generar audio, debe escuchar o validar con
transcripcion automatica cuando el adaptador lo soporte.

Antes de generar TTS, OPES debe resolver refs de comunes ya existentes:
`common_topic_ref`, `source_content_ref`, `audio_manifest_ref` y `audio_ref`.
Si hay equivalencia compatible, se reutiliza el audio comun. Si el texto final
no coincide exactamente, el comun queda como candidato y se crea una derivacion
localizada; no se rehace desde cero por alias, orden, titulo o metadatos
reparables.

## HTML local USO

`generate_html_site` no debe entregar un visor single-file con estilo propio si
el adaptador OPES/USO dispone del formato de curso de la web. La salida final
debe materializar `index.html`, `html_final/` por tema, assets locales,
`audio/manifests/` por tema y locales/i18n para controles visibles, siguiendo
la web USO/TCAE promocion interna. En material de estudio debe incluir tambien
la capa protegida `#uso-material-watermark` con marca de agua diagonal USO. El
HTML local debe funcionar sin backend productivo antes de subir a produccion.

## Manuales graficos de ayuda

OPES debe crear tambien jobs `generate_help_manual_assets` para producir el
artefacto `help_manual_package` cuando el temario o artefacto USO tenga HTML
local revisable. Este paso va despues de `generate_html_site`, porque el manual
usa pantallas/capturas del HTML o de la web local.

La guia operativa vigente esta en
`/home/alberto/Trabajo/USO/web/docs/SCREENSHOT_HELP_MANUALS.md` y la regla de
marca en
`/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/REGLA_ARTEFACTOS_USO_BRANDING_2026-06-02.md`.
El paquete debe conservar el YAML de escenario, `index.html` como fuente
canonica, `manual.pdf` exportado desde HTML, `manual.md`, capturas anotadas
`img/`, capturas originales `raw/` y revision visual. El payload publico solo
debe exponer refs opacas de esos outputs, no rutas locales ni perfiles privados.

## Revision cruzada y cierre 100%

Antes de cerrar un temario OPES, la secuencia debe materializar:

- `review_codex`, `review_gemini` y `review_claude` como revisiones
  independientes sobre contenido, tests, visuales, audios, tutor, HTML,
  manuales y paquete.
- `review_pair_codex_gemini`, `review_pair_codex_claude` y
  `review_pair_gemini_claude` como revisiones por pares con acuerdos,
  discrepancias y rework causal. En bancos publicables, las revisiones cubren el
  100% de preguntas, opciones, respuesta correcta, distractores y explicaciones.
- `review_director_consolidation` como matriz final del Director: acepta,
  replanifica, conserva material recuperable o bloquea solo por causa real.
- `finalize_temario_package` como cierre de `completed_syllabus_package`. Ese
  paquete es la unidad que permite decir que el temario esta terminado al 100%
  para revision local del operador; sin ese artefacto el estado correcto es
  `pendiente_continuar`, no `ready`.

OPES no llama proveedores ni decide sesiones. La composicion Orquesta enruta
los jobs por adaptadores opt-in:

- `review_gemini` y `review_pair_codex_gemini` usan Gemini CLI cuando
  `ORQUESTA_GEMINI_ENABLED=1`; si no, caen por el runtime Codex configurado.
- `review_claude`, `review_pair_codex_claude` y
  `review_pair_gemini_claude` usan Claude CLI cuando
  `ORQUESTA_CLAUDE_ENABLED=1`; si no, caen por el runtime Codex configurado.
- `review_codex`, `review_director_consolidation` y el cierre quedan en Codex
  salvo que una composicion futura declare otro adaptador.

Variables principales para revisiones reales:

```bash
ORQUESTA_GEMINI_ENABLED=1
ORQUESTA_GEMINI_COMMAND=gemini
ORQUESTA_GEMINI_APPROVAL_MODE=auto_edit
ORQUESTA_GEMINI_OUTPUT_FORMAT=text
ORQUESTA_GEMINI_EXTRA_ARGS=--skip-trust

ORQUESTA_CLAUDE_ENABLED=1
ORQUESTA_CLAUDE_COMMAND=claude
ORQUESTA_CLAUDE_PERMISSION_MODE=bypassPermissions
ORQUESTA_CLAUDE_OUTPUT_FORMAT=text
```
