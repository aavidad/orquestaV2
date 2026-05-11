# orquesta-director

Modulo de composicion/director de OrquestaV2.

Su responsabilidad es coordinar contratos publicos entre mini-proyectos sin conocer sus internals. En v0 arranca como flujo puro para convertir una `AppSpecV0` validada y un `BacklogInicialPropuestoV0` en un registro de proyecto aceptado y un comando/evento inicial de workflow durable.

## Limites

- No persiste.
- No arranca agentes.
- No decide capacidad/modelo.
- No ejecuta deploy.
- No lee DB ni filesystem productivo.
- No conoce web, CLI, MCP ni runtime concreto.

## Contratos consumidos

- `SolicitarNuevaApp v0`, propiedad de `orquesta-factory`.
- `RegistrarProyectoDesdeAppSpec v0`, propiedad de `orquesta-core`.
- `StartRunFromAppSpecV0` / `OrchestrationCommandV0`, propiedad de `orquesta-core-workflow` hasta promocion global.

## Salida esperada v0

El flujo v0 devuelve:

- resumen compacto del registro aceptado de core;
- comando `StartRun` de workflow;
- resultado puro de `HandleCommandV0` con evento `RunStarted`;
- referencias compactas para persistencia/outbox futura.
