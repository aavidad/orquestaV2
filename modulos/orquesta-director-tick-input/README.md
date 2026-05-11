# orquesta-director-tick-input

Microproyecto para preparar el input compacto que consume `orquesta-director-scheduler`.

Responsabilidad v0:

- recibir un `OrchestrationRunV0` ya durable;
- construir `RunSchedulingSnapshotV0`;
- convertir proyecciones internas compactas del workflow a refs canonicas del scheduler;
- adjuntar candidates explicitos sin modificarlos semanticamente;
- devolver `DirectorSchedulerTickInputV0` validado por el scheduler.

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
