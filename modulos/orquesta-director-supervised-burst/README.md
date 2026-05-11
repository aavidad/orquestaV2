# orquesta-director-supervised-burst

Microproyecto para ejecutar una rafaga acotada de pasos del director.

Responsabilidad v0:

- pedir el input fresco del siguiente `ExecuteDirectorCycleStepV0` por puerto;
- ejecutar el paso por puerto;
- consultar la politica `DecideDirectorSupervisorNextActionV0` por puerto;
- repetir solo si la decision es `continue`;
- parar en outbox, espera externa, director, bloqueo, quietud, error o `max_steps`.

Fuera de alcance:

- daemon, worker residente, timers, sleeps o polling;
- dispatch de outbox o ACK;
- elegir DB, runtime, proveedor, HOME, OAuth, cuotas o modelos;
- crear candidates de scheduler;
- leer event-store o persistence productiva.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-supervised-burst
```
