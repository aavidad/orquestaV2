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
- Un replay con `CapacityRequested` durable, ledger vacio y candidate de
  recovery reemite `RequestCapacity`, guarda una nueva outbox pendiente y no
  duplica eventos ni refs de capacidad.
- Outbox pendiente previa se lista por ledger inyectado y llega a tick-input
  como `pending_outbox_refs` sin aplicar comandos.
- Candidates reales de programacion con `WorkClaims` compartidos pasan por
  tick-input, scheduler, runner y core-workflow: se registran gates, agentes y
  dos outbox `LaunchRuntimeAgent` en el ledger sin despachar runtime.
- Multi-step presupuestado: continua tras `commands_applied`, refresca snapshot
  por `snapshot_port`, para en `wait_outbox`, `wait_external`, `blocked`,
  `needs_director` y `stop_quiescent`, corta por `stop_max_steps` y propaga
  error de snapshot como error publico.
- Entrada incompleta se rechaza con error publico.

Ultima ejecucion: 2026-06-01, ok, `go test -count=1 ./modulos/orquesta-director-cycle`.
