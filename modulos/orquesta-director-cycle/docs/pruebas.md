# Pruebas: orquesta-director-cycle

## Ejecutadas Localmente

```sh
go test -count=1 ./modulos/orquesta-director-cycle
```

Resultado: `ok` el 2026-05-13.

## Cobertura Contractual

- Paso completo: tick-input -> runner -> outbox-recorder.
- Outbox generada por workflow queda guardada como pendiente.
- Un segundo paso con outbox pendiente queda en `waiting/outbox_pending`.
- Outbox pendiente previa se lista por ledger inyectado y llega a tick-input
  como `pending_outbox_refs` sin aplicar comandos.
- Entrada incompleta se rechaza con error publico.
