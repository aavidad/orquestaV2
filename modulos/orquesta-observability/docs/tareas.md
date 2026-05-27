# Tareas locales: orquesta-observability

Cada tarea debe ser pequena y cerrada.

```text
ID: OBS-012
Objetivo: Reconciliar T198 para descriptors MCP de operational-status y workspace timeline.
Write-set: docs locales y backlog T198.
Simbolo foco: OperationalStatusQueryV0; WorkspaceTimelineQueryV0.
Contrato: mcp.resource.descriptor_source.v0 como consumidor de observability.
Validacion: go test -count=1 ./modulos/orquesta-mcp ./modulos/orquesta-operator-mcp ./modulos/orquesta-observability ./modulos/orquesta-governance ./modulos/orquesta-core ./cmd/orquesta-server.
Bloqueos: No habilita sink, DB, runtime ni lectura de transcripts; solo fija la fuente canonica para el descriptor.
Estado: completada_documental 2026-05-27
Revalidacion: `agent-ref-task-autoprogramming-c3678e9bc306-g01` confirma cierre
stale documental sin codigo nuevo.
```

## Backlog inicial desde DB v1

```text
ID: OBS-001
Objetivo: Definir extraccion segura de senales agregadas desde runtime_transcript, runtime_telemetry_samples y audit_log sin cargar contexto masivo.
Write-set: docs/tareas.md, docs/decisiones.md, docs/contratos.md, docs/pruebas.md
Simbolo foco: OrquestaEventV0
Contrato: OrquestaEvent v0
Validacion: solo agregados, filtros, contadores y ejemplos pequenos; nada de transcripts completos en docs ni prompts.
Bloqueos: ninguno vivo; las tablas de DB v1 quedan como evidencia forense historica y no como prerequisito ejecutable de automejora.
Estado: completada_documental
```

```text
ID: OBS-002
Objetivo: Definir envelope de evento minimo para runtime, presupuesto, decision, review y merge.
Write-set: docs/contratos.md, docs/pruebas.md
Simbolo foco: OrquestaEventV0
Contrato: OrquestaEvent v0
Validacion: evento no decide negocio, no contiene secretos y es consumible por MCP/web sin conocer tablas.
Bloqueos: ninguno para el envelope contractual sin DB real; OBS-001 queda como adaptador/lector posterior.
Estado: completada; promovida globalmente por decision del director de 2026-05-04.
```

```text
ID: OBS-003
Objetivo: Anyadir schema draft7 y fixtures de contrato para `OrquestaEventV0`.
Write-set: docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md, docs/schemas/orquesta_event_v0.schema.json, docs/fixtures/orquesta_event_v0/*
Simbolo foco: OrquestaEventV0
Contrato: OrquestaEvent v0
Validacion: `jq` sobre schema/fixtures y `ajv-cli` draft7 con un fixture valido y dos invalidos esperados.
Bloqueos: ninguno.
Estado: completada
```

```text
ID: OBS-004
Objetivo: Cerrar consulta al director tras promocion global de `OrquestaEvent v0`.
Write-set: docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OrquestaEventV0
Contrato: OrquestaEvent v0
Validacion: docs locales reflejan que el contrato ya es compartido; productores autorizados son core/runtime/capacity y review futuro; MCP/web solo leen proyecciones compactas futuras; sink/DB real queda fuera.
Bloqueos: ninguno.
Estado: completada
```

```text
ID: OBS-005
Objetivo: Crear DTOs y validacion pura Go para `OrquestaEventV0` y `PublishOrquestaEventRequestV0`.
Write-set: orquesta_event_v0.go, orquesta_event_v0_test.go, docs/tareas.md, docs/pruebas.md
Simbolo foco: OrquestaEventV0
Contrato: OrquestaEvent v0
Validacion: `go test -count=1 ./modulos/orquesta-observability`; fixtures validos/invalidos, source_area/event_type, privacy false, payload compacto, claves prohibidas y aceptacion sin sink real.
Bloqueos: ninguno; el corte no implementa DB, bus, fichero, memoria, servidor ni sink real.
Estado: completada
```

```text
ID: OBS-006
Objetivo: Responder a la consulta CLI con un candidato documental read-only para diagnostico/progreso compacto.
Write-set: docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OperationalStatusQueryV0 / DiagnosticoCompactoV0
Contrato: promovido globalmente tras decision del director.
Validacion: contrato documentado como propiedad de observability, read-only, sin DB/runtime directo, sin secretos, con referencias opacas y consumidores CLI/MCP/Web autorizados tras promocion.
Bloqueos: ninguno; la consulta al director quedo cerrada el 2026-05-04.
Estado: completada_documental; promocion_global_completada
```

```text
ID: OBS-007
Objetivo: Crear DTOs y validacion pura Go para `OperationalStatusQueryV0` y `DiagnosticoCompactoV0`.
Write-set: operational_status_v0.go, operational_status_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OperationalStatusQueryV0 / DiagnosticoCompactoV0
Contrato: OperationalStatusQuery v0
Validacion: `go test -count=1 .` y `git diff --check -- .`; valida read-only por JSON estricto, consumidores autorizados, scopes/secciones acotadas, referencias opacas, privacy false y ausencia de secretos/transcripts/prompts/completions/SQL/DSN/HOME.
Bloqueos: ninguno; el corte no implementa DB, sink, servidor, filesystem, runtime ni recuperacion activa.
Estado: completada
```

```text
ID: OBS-008
Objetivo: Implementar adaptador/query handler puro en memoria para `OperationalStatusQueryV0` sobre diagnosticos compactos suministrados al constructor.
Write-set: operational_status_memory_adapter_v0.go, operational_status_memory_adapter_v0_test.go, docs/contratos.md, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OperationalStatusQueryV0 / DiagnosticoCompactoV0
Contrato: OperationalStatusQuery v0
Validacion: valida query con `ValidateOperationalStatusQueryV0`, diagnosticos con `ValidateDiagnosticoCompactoV0`, lookup opaco por scope+subject_ref+correlation_id, errores publicos de disponibilidad/frescura, sin DB, sink, runtime, filesystem, bus, procesos ni exposicion de secretos/transcripts/prompts/completions/SQL/DSN/HOME.
Bloqueos: ninguno; el adaptador es solo memoria para contract tests de consumidores y no backend productivo.
Estado: completada
```

```text
ID: OBS-009
Objetivo: Sanear tamano de `operational_status_v0.go` sin cambiar comportamiento ni contratos publicos.
Write-set: operational_status_v0.go, operational_status_types_v0.go, operational_status_query_validation_v0.go, operational_status_diagnostic_validation_v0.go, operational_status_helpers_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OperationalStatusQueryV0 / DiagnosticoCompactoV0
Contrato: OperationalStatusQuery v0
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-observability`; `git diff --check -- modulos/orquesta-observability`; `wc -l` de los Go tocados.
Bloqueos: ninguno; no toca `orquesta_event_v0.go` ni otros modulos.
Estado: completada
```

```text
ID: OBS-010
Objetivo: Sanear tamano de `orquesta_event_v0.go` dividiendolo por responsabilidades sin cambiar contratos publicos, JSON tags, errores, fixtures ni comportamiento.
Write-set: orquesta_event_v0.go, orquesta_event_types_v0.go, orquesta_event_validation_v0.go, orquesta_event_payload_validation_v0.go, orquesta_event_helpers_v0.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OrquestaEventV0 / PublishOrquestaEventRequestV0
Contrato: OrquestaEvent v0
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-observability`; `git diff --check -- modulos/orquesta-observability`; `wc -l` de los Go tocados.
Bloqueos: ninguno; no toca `operational_status_*`, DB, runtime, proveedor real, fixtures ni otros modulos.
Estado: completada
```

```text
ID: OBS-011
Objetivo: Dividir `operational_status_v0_test.go` por contrato o escenario para que los tests sean manejables sin cambiar expectations ni comportamiento.
Write-set: operational_status_v0_test.go, operational_status_diagnostic_v0_test.go, operational_status_helpers_v0_test.go, docs/tareas.md, docs/pruebas.md, docs/decisiones.md
Simbolo foco: OperationalStatusQueryV0 / DiagnosticoCompactoV0
Contrato: OperationalStatusQuery v0
Validacion: `gofmt`; `go test -count=1 ./modulos/orquesta-observability`; `git diff --check -- modulos/orquesta-observability`; `wc -l` de los tests tocados.
Bloqueos: ninguno; no toca codigo productivo, `orquesta_event_*`, runtime, web, mcp, factory ni core.
Estado: completada
```

## Plantilla

```text
ID:
Objetivo:
Write-set:
Simbolo foco:
Contrato:
Validacion:
Bloqueos:
Estado:
```
