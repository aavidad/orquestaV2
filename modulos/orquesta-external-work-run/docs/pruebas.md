# Pruebas: orquesta-external-work-run

## Unitarias

```bash
go test -count=1 ./modulos/orquesta-external-work-run
```

Cobertura:

- crea run nuevo y abre `programacion`;
- registra el cambio por puerto `AppChange`;
- encola el run por `RunQueuePriorityWriterPortV0`;
- rechaza solicitudes sin `external_work` sin mutar run ni cola.
