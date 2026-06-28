# Pruebas: orquesta-external-work-run

## Unitarias

```bash
go test -count=1 ./modulos/orquesta-external-work-run
```

Cobertura:

- crea run nuevo y abre `programacion`;
- registra el cambio por puerto `AppChange`;
- encola el run por `RunQueuePriorityWriterPortV0`;
- rechaza solicitudes sin `external_work` sin mutar run ni cola;
- `TestBuildExternalWorkGoalWorkSpecV0InlineaInputFieldsOperativosSeguros`
  cubre que los `input_fields` seguros se inlinean de forma acotada, las rutas
  absolutas se reducen a `local_path_ref`/`basename` y cada campo conserva una
  ref `input_field_payload` resoluble desde `AppChangeStore`;
- `TestBuildExternalWorkGoalWorkSpecV0RedactaInputFieldsSensibles` cubre que
  campos sensibles no exponen valor ni `payload_ref` al Goal.
