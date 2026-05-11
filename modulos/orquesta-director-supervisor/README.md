# orquesta-director-supervisor

Microproyecto para decidir si el director puede pedir otro paso de orquestacion.

Responsabilidad v0:

- recibir el resultado de `ExecuteDirectorCycleStepV0`;
- clasificar outbox pendiente, espera externa, bloqueo, pregunta al director o quietud;
- cortar la ejecucion cuando se alcanza `max_steps`;
- convertir errores del ultimo paso en parada explicita;
- exponer una recomendacion autonoma `continue`/`wait`/terminal para el siguiente tick;
- devolver una decision compacta para que un adaptador externo gobierne el siguiente tick.

Fuera de alcance:

- ejecutar ciclos o bucles;
- temporizadores, daemon, workers o goroutines;
- elegir DB, runtime, proveedor, HOME, OAuth, cuotas o modelos;
- llamar scheduler, workflow, ledger o dispatch de outbox.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-supervisor
```
