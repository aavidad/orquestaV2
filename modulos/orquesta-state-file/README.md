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
- `GoalWorkStateStorePortV0` / `GoalWorkStateListPortV0`;
- `AgentProcessRegistryPortV0`.
- `AutonomyProgramStorePortV0`.

El subpaquete `outbox` implementa el ledger durable de outbox usado por el
stack de aplicacion.

El modulo guarda documentos JSON bajo un directorio configurado por el operador.
No usa base de datos, SQL, runtime, proveedor, HOME, OAuth ni servidor. Las
escrituras se hacen con fichero temporal, `sync` y `rename`, y el adaptador
mantiene un mutex interno por instancia.

Los programas de autonomia se indexan por `project_ref`, `root_ref` y
`program_ref`; otro scope no puede recuperar un programa ajeno. Su creacion es
create-or-identical y toda transicion usa compare-and-swap. Ademas del mutex de
instancia, un lock de fichero por agregado serializa writers de procesos/store
distintos para evitar lost updates y doble claim de frontera.

## Uso

```go
store, err := orquestastatefile.NewStoreV0(orquestastatefile.ConfigV0{
    RootDir: "/ruta/configurada/estado",
})
```
