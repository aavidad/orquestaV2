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
  absolutas se reducen a `local_path_ref`/`basename` y el spec conserva un
  `input_fields_summary` auditable sin refs de payload redundantes;
- `TestBuildExternalWorkGoalWorkSpecV0RedactaInputFieldsSensibles` cubre que
  campos sensibles no exponen valor ni `payload_ref` al Goal.
- `TestBuildExternalWorkGoalWorkSpecV0NoInlineaInputFieldsMasivos` cubre que
  campos masivos no prioritarios quedan fuera del contexto salvo el resumen por
  conteo.
- `TestBuildExternalWorkGoalWorkSpecV0LimitaPayloadRefsDeInputFields` cubre que
  campos prioritarios enormes no inflan el contexto: como maximo se publican
  cuatro refs de payload y el resto queda omitido por resumen.
