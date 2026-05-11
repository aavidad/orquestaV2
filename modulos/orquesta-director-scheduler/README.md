# orquesta-director-scheduler

Microproyecto para automatizar el siguiente paso de la orquestacion sin contaminar `orquesta-core-workflow`.

Responsabilidad v0:

- recibir un snapshot compacto del run;
- recibir candidates explicitos de lease, capacidad, concurrencia y agente;
- devolver comandos publicos de `orquesta-core-workflow` en orden seguro;
- detenerse cuando falte evidencia, haya outbox pendiente o exista riesgo de duplicar trabajo.

Fuera de alcance:

- reloj interno, goroutines, sleeps o daemon;
- DB, colas, filesystem productivo o red;
- runtime real, proveedor, modelo, HOME, OAuth, cuotas reales o credenciales;
- evaluar politicas de lease, replan o revision por cuenta propia.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-scheduler
```
