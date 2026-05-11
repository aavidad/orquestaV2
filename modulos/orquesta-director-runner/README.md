# orquesta-director-runner

Microproyecto para ejecutar un ciclo acotado del director de OrquestaV2.

Responsabilidad v0:

- recibir un `DirectorSchedulerTickInputV0` ya preparado;
- pedir al scheduler un plan contractual;
- aplicar comandos al workflow mediante un puerto;
- devolver estado compacto, comandos aplicados y outbox pendiente;
- parar en cuanto haya outbox por defecto;
- acumular varios outbox solo si `max_outbox` se configura explicitamente para
  un ciclo que necesita batch externo.

Fuera de alcance:

- construir candidatos, snapshots o contexto de agentes;
- ejecutar como daemon;
- acceder a DB, runtime, procesos, red o filesystem productivo;
- conocer modelos, proveedores, HOME, OAuth o cuotas reales;
- despachar outbox.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-director-runner
```
