# orquesta-director-cycle

Microproyecto para ejecutar un paso completo y acotado del director.

Responsabilidad v0:

- consultar outbox pendiente por puerto;
- construir el tick compacto con `orquesta-director-tick-input`;
- ejecutar scheduler/workflow con `orquesta-director-runner`;
- registrar outbox nueva con `orquesta-director-cycle-outbox`;
- devolver estado compacto para que un supervisor externo decida si repetir.

Fuera de alcance:

- daemon, bucles o temporizadores;
- elegir DB, runtime, proveedor, HOME, OAuth, cuotas o modelos;
- generar candidates;
- despachar outbox;
- aplicar politicas de lease/progreso/replan.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-cycle
```
