# Tareas

## ACDS-001

Objetivo: generar decisiones del director desde `AppChangeRecordV0` concreto.

Estado: hecho.

Validacion:

- `go test -count=1 ./modulos/orquesta-app-change-director-source`.
- `TestAppChangeDirectorDecisionSourceV0ConsumeCambioRecibidoComoEvento`.
