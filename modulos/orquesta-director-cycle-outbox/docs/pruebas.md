# Pruebas: orquesta-director-cycle-outbox

## Ejecutadas Localmente

```sh
go test -count=1 ./modulos/orquesta-director-cycle-outbox
```

Resultado: `ok` el 2026-05-13.

## Cobertura Contractual

- Guarda outbox generada por el runner y lista refs pendientes.
- Rechaza mensajes de otro run.
- Si el ledger devuelve issues, propaga error publico.
- Integra con tick-input/scheduler/runner: la outbox guardada bloquea el siguiente tick como `waiting/outbox_pending`.
