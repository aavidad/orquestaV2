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
tutor/bots y HTML local con logos USO y aspecto USO/TCAE promocion interna. La
secuencia canonica esta en
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

Modo residente por pases:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_ENABLED=1 \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_html_site \
go run ./cmd/orquesta-server run
```

La secuencia consulta los tipos en orden y solo drena el primer tipo con jobs
`pending`. Si el ledger marca un job como `already_submitted`, no lo reenvia y
no avanza a fases posteriores hasta que OPES deje de mostrar pendientes de ese
tipo.

## Estado T12

T12 queda reconciliada como bloqueo verificable para smoke real OPES temporal:
los tests de bridge/conector y el fake `run-until-assemble` validan la ruta
local de derivados; historicamente llegaba a `assemble_topic -> assembled_topic`
y la secuencia vigente anade investigacion externa, tests, audio, tutor/bots y
HTML local. Falta entorno OPES temporal, servidor Orquesta temporal,
confirmacion de efectos y cuota/modelo real. No repetir implementaciones padre
para generar la misma evidencia fake.

## Audio accesible

OPES debe crear tambien jobs `generate_audio_asset` para producir el artefacto
`audio_asset` de cada tema publicable y de sus apartados/secciones. El bridge lo
trata como trabajo de dominio posterior a `assemble_topic`: recibe refs opacas
del tema ensamblado o paquete final y devuelve un manifest de audio con idioma,
formatos, duracion, refs/checksums de artefactos y mapa `section_ref ->
audio_ref`.

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
