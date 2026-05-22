# orquesta-director

Modulo de composicion/director de OrquestaV2.

Su responsabilidad es coordinar contratos publicos entre mini-proyectos sin conocer sus internals. En v0 arranca como flujo puro para convertir un `RegistrarAppSpecV0` validado y un `RegistrarBacklogInicialV0` compatible en un registro de proyecto aceptado y un comando/evento inicial de workflow durable.

## Limites

- No persiste.
- No arranca agentes.
- No decide capacidad/modelo.
- No ejecuta deploy.
- No lee DB ni filesystem productivo.
- No conoce web, CLI, MCP ni runtime concreto.

## Contratos consumidos

- `RegistrarProyectoDesdeAppSpec v0`, propiedad de `orquesta-core`.
- `StartRunFromAppSpecV0` / `OrchestrationCommandV0`, propiedad de `orquesta-core-workflow` hasta promocion global.

La adaptacion desde salidas de `orquesta-factory` queda fuera de este modulo.

## Salida esperada v0

El flujo v0 devuelve:

- resumen compacto del registro aceptado de core;
- comando `StartRun` de workflow;
- resultado puro de `HandleCommandV0` con evento `RunStarted`;
- referencias compactas para persistencia/outbox futura.
