# orquesta-outbox-dispatch

Responsabilidad: seleccionar una o varias entradas pendientes de outbox para un
`target_port` explicito y ejecutar un ciclo unico de dispatch mediante puertos
inyectados.

Este microproyecto no implementa runtime, red, procesos, DB, sleeps, goroutines
ni adaptadores productivos. El flujo real queda expresado por puertos: listar
pendientes, reclamar una entrada, ejecutar el conector abstracto y registrar ACK
solo si la ejecucion termino sin error.

Incluye:

- contratos v0 de entrada pendiente, decision e intent;
- puertos abstractos de lectura, claim, executor y ACK;
- caso de uso puro `ChooseNextDispatchV0`;
- caso de uso puro `ChooseDispatchBatchV0` para preparar varios intents
  ejecutables en paralelo por adaptadores externos;
- planificador puro `PlanOutboxDispatchAckClosureV0` para cerrar solo items con
  ACK correlacionado;
- caso de uso `RunOutboxDispatchOnceV0`;
- tests unitarios para success, fallo de executor, no pending y claimed.

No incluye DB, SQLite, Postgres, filesystem productivo, red, runtime, proveedor,
modelo, HOME, OAuth, daemons, sleeps ni goroutines.

Un dispatch real no se considera cerrado por haber sido seleccionado o ejecutado
en lote: requiere ACK `success` correlacionado por `message_id`, `run_id` y
`target_port`. ACK ausente mantiene el item pendiente; ACK `failed` produce
estado publico `ack_failed`.

## Validacion

```bash
gofmt -w modulos/orquesta-outbox-dispatch/*.go
go test -count=1 ./modulos/orquesta-outbox-dispatch
```
