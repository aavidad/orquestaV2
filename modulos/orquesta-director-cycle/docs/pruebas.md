# Pruebas: orquesta-director-cycle

## Ejecutadas Localmente

```sh
go test -count=1 ./modulos/orquesta-director-cycle
```

Resultado: `ok` el 2026-05-06.

## Cobertura Contractual

- Paso completo: tick-input -> runner -> outbox-recorder.
- Outbox generada por workflow queda guardada como pendiente.
- Un segundo paso con outbox pendiente queda en `waiting/outbox_pending`.
- Entrada incompleta se rechaza con error publico.
