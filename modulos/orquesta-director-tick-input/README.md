# orquesta-director-tick-input

Microproyecto para preparar el input compacto que consume `orquesta-director-scheduler`.

Responsabilidad v0:

- recibir un `OrchestrationRunV0` ya durable;
- construir `RunSchedulingSnapshotV0`;
- convertir proyecciones internas compactas del workflow a refs canonicas del scheduler;
- adjuntar candidates explicitos sin modificarlos semanticamente;
- adjuntar refs de outbox pendiente recibidas desde un coordinador externo;
- devolver `DirectorSchedulerTickInputV0` validado por el scheduler.

Frontera de composicion:

- el puerto que lista outbox pendiente vive fuera de este modulo;
- `orquesta-director-cycle` es el ensamblador superior que consulta el ledger
  inyectado y pasa `pending_outbox_refs` a este builder.

Fuera de alcance:

- evaluar concurrencia, leases, progreso o replan;
- crear candidates;
- leer DB, listar outbox o consultar runtime;
- aplicar workflow o ejecutar el runner;
- conocer proveedores, HOME, OAuth, cuotas o modelos.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-tick-input
```
