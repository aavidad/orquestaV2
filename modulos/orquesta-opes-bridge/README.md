# orquesta-opes-bridge

Adaptador opt-in para tomar jobs externos pendientes de OPES y convertirlos en
runs de Orquesta mediante `/api/v0/external-work/run`.

No pertenece al nucleo. OPES sigue siendo la app de dominio editorial y
Orquesta sigue siendo el nucleo de orquestacion.

## Validacion

```sh
go test -count=1 ./modulos/orquesta-opes-bridge
```

## Uso

Dry-run:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_OPES_BRIDGE_DRY_RUN=1 \
go run ./cmd/orquesta-server opes-drain-once
```

Ejecucion real:

```sh
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_BRIDGE_CONFIRM=1 \
go run ./cmd/orquesta-server opes-drain-once
```
