# RTE-B: ExternalAgentProcessBatch v0

Fecha: 2026-05-07

## Decision

Se implementa `LaunchExternalAgentProcessBatchV0` como helper local generico para lanzar varios `ExternalAgentLaunchSpecV0` usando los puertos existentes:

- `ExternalAgentProcessCommandResolverV0`;
- `ExternalAgentProcessRuntimePortV0`;
- `LaunchExternalAgentProcessV0`.

Cada item aporta `{spec, resolver, runtime}` y el resultado conserva `index`, `item_ref`, `request_id`, `correlation_id`, `profile_ref`, `connector_ref` y `runtime_kind`.

## Invariantes

- No conoce proveedor, modelo, HOME, OAuth, DB ni comandos por defecto.
- No decide capacidad, fase, asignacion ni politica de negocio.
- No detiene el batch completo por un fallo parcial: cada item devuelve `started` o `blocked`.
- `maxConcurrency < 1` se normaliza a `1`.
- El orden de salida coincide con el orden de entrada.
- La concurrencia solo limita llamadas al helper individual; cada lanzamiento sigue validando spec, resolver y runtime por el contrato existente.

## Pruebas

Comando:

```bash
go test -count=1 ./modulos/orquesta-runtime
```

Cobertura esperada:

- 3 items con perfiles distintos y resolvers/runtime fake;
- correlacion por indice e `item_ref`;
- fallo parcial bloqueado con issue normalizada por `correlation_id`;
- `maxConcurrency=0` tratado como ejecucion serial.
