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
