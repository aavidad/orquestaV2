# orquesta-state-file

Adaptador durable basado en ficheros para puertos del nucleo de orquestacion.

Implementa:

- `RunStorePortV0`;
- `EventSinkPortV0`;
- `RunEventReaderPortV0`;
- `WorkflowTaskStorePortV0`;
- `WorkflowTaskWriterPortV0`;
- `WorkflowTaskWaitStateStorePortV0` / `WorkflowTaskWaitStateWriterPortV0`;
- `RequiredTestEvidenceStorePortV0`;
- `OperationalDirectorPlanStateStorePortV0` /
  `OperationalDirectorPlanStateWriterPortV0`;
- `AgentProcessRegistryPortV0`.

El subpaquete `outbox` implementa el ledger durable de outbox usado por el
stack de aplicacion.

El modulo guarda documentos JSON bajo un directorio configurado por el operador.
No usa base de datos, SQL, runtime, proveedor, HOME, OAuth ni servidor. Las
escrituras se hacen con fichero temporal, `sync` y `rename`, y el adaptador
mantiene un mutex interno por instancia.

## Uso

```go
store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
    RootDir: "/ruta/configurada/estado",
})
```
