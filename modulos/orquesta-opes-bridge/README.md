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
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic \
go run ./cmd/orquesta-server run
```

La secuencia consulta los tipos en orden y solo drena el primer tipo con jobs
`pending`. Si el ledger marca un job como `already_submitted`, no lo reenvia y
no avanza a fases posteriores hasta que OPES deje de mostrar pendientes de ese
tipo.

## Estado T12

T12 queda reconciliada como bloqueo verificable para smoke real OPES temporal:
los tests de bridge/conector y el fake `run-until-assemble` ya validan la ruta
local hasta `assemble_topic -> assembled_topic`, pero falta entorno OPES
temporal, servidor Orquesta temporal, confirmacion de efectos y cuota/modelo
real. No repetir implementaciones padre para generar la misma evidencia fake.
