# orquesta-run-supervisor

Supervisor puro para avanzar la cola global multiapp durante un presupuesto
acotado.

Responsabilidades:

- llamar repetidamente a un puerto de tick global;
- respetar `MaxTicks` y `MaxExecutions`;
- parar si no hay ejecuciones cuando `StopOnNoExecution` esta activo;
- evitar repetir la misma run dentro de una pasada salvo que se permita
  explicitamente.

No contiene almacenamiento, transporte, MCP, runtime ni adaptadores de stack.

Contrato principal:

```go
SuperviseRunsV0(ctx, deps, command)
```

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-run-supervisor
```
