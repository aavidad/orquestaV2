# Tareas

## STF-001

Crear `StoreV0` con JSON atomico y tests de recuperacion.

## STF-002

Crear `outbox.FileOutboxLedgerV0` y cablear `cmd/orquesta-server` para usar los
conectores file-based como estado operativo durable.

## STF-003

Persistir `WorkflowTaskWaitStateV0` para que el Director pueda explicar y
recuperar waits por cohorte, ola o parent task.

Estado: hecho primer corte.

Validacion:

```sh
go test -count=1 ./modulos/orquesta-state-file -run TestStoreV0RecuperaWorkflowTaskWaitStateV0
```
