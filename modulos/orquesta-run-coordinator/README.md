# orquesta-run-coordinator

Coordinador puro para un runner global multiapp.

Responsabilidades:

- pedir candidatos a `orquesta-run-queue`;
- rankearlos con `RankRunCandidatesV0`;
- consultar `orquesta-run-control` antes de despachar;
- saltar runs pausadas, canceladas, paradas o con stop solicitado;
- invocar `RunDrainerPortV0` para drenar la run elegida.

No contiene almacenamiento, transporte, MCP, runtime ni adaptadores de stack.

Contrato principal:

```go
CoordinateRunsTickV0(ctx, deps, command)
```

`command` soporta `QueueLimit`, `MaxRuns`, `OccurredAt` y `CorrelationID`.
La respuesta devuelve listas compactas de ejecuciones, skips y ranked.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-run-coordinator
```
