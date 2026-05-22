# Tareas

## ACDS-005

Objetivo: proyectar `AppChangeRequestV0.metadata_refs` como `context_refs`
opacas en la microtarea del director.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ProyectaMetadataRefsComoContextRefs`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`.

## ACDS-001

Objetivo: generar decisiones del director desde `AppChangeRecordV0` concreto.

Estado: hecho.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-change-director-source`.
- `TestAppChangeDirectorDecisionSourceV0ConsumeCambioRecibidoComoEvento`.

## ACDS-002

Objetivo: proyectar trabajo externo de dominio como contrato/microtarea sin
acoplar la fuente a OPES, REST, MCP ni runtime.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`.

## ACDS-003

Objetivo: abrir revision cuando un cambio de app ya esta entregado.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0AbreRevisionTrasEntrega`;
- `TestCodexStackV0CambioProgresivoPasaReviewGate`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source ./modulos/orquesta-app-codex-stack`.

## ACDS-004

Objetivo: proyectar trabajos documentales externos, especialmente
`draft_content_block` de OPES, como microtareas que declaran paquete de dominio
suficiente y no contexto minimo.

Estado: hecho.

Validacion:

- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExterno`;
- `TestAppChangeDirectorDecisionSourceV0ProyectaTrabajoExternoSinWriteSetLocal`;
- `go test -count=1 ./modulos/orquesta-app-change-director-source`.
