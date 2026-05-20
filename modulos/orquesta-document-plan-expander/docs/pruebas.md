# Pruebas: orquesta-document-plan-expander

Comando:

```bash
go test -count=1 ./modulos/orquesta-document-plan-expander
```

Cobertura:

- plan valido con seccion, visual, reviews y ensamblado produce jobs derivados
  compatibles con `DomainWorkJobRequestV0`;
- aliases de `DomainDocumentPlanV0` se normalizan antes de expandir;
- `assemble_topic` produce `expected_artifact_type=assembled_topic`;
- plan invalido no produce jobs parciales;
- refs invalidas que el plan proyecta a jobs derivados se detectan antes de
  normalizar el plan, para no compactar y ocultar refs opacas defectuosas;
- caso de uso hexagonal: crea jobs por `DomainWorkJobCreatorPortV0`;
- puerto nil: devuelve issue sin panic;
- plan o job derivado invalido: no llama al puerto;
- error del puerto: devuelve error Go conservando progreso;
- resultado `invalid` o `accepted` sin `job_ref`: devuelve issues y detiene la
  secuencia;
- idempotencia: reintentos con la misma entrada generan la misma secuencia de
  `RequestID`/`IdempotencyKey` y `JobRef` del fake;
- guard arquitectonico local contra adaptadores concretos, Codex, runtime, web,
  MCP, DB, red y filesystem productivo.

Pruebas relacionadas fuera del paquete:

```bash
go test -count=1 ./modulos/orquesta-domain-work-file
```

Ese adaptador prueba que `CreateDomainDocumentPlanDerivedJobsV0` puede crear
jobs derivados por puerto inyectado, persistirlos y repetirlos tras reinicio
sin que el expander importe filesystem.
