# Tareas

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
