# Tareas: orquesta-app-runner

## APP-RUN-001

Objetivo: preparar run y provider desde `AppSpecV0`.

Estado: hecho.

Validacion: `TestPrepareAppOrchestrationV0CreatesRunAndProvider`.

## APP-RUN-002

Objetivo: demostrar que el run preparado entra en el loop progresivo del nucleo.

Estado: hecho con launcher fake de lifecycle para unidad del nucleo.

Validacion: `TestPreparedAppOrchestrationV0ProgressiveLoopRequestsBootstrapAgent`.

Pendiente productivo: sustituir el launcher fake por conector real inyectado.

## APP-RUN-003

Objetivo: conservar campos de error del planner para adaptadores MCP/REST.

Estado: hecho.

Validacion: `TestPrepareAppOrchestrationV0ConservaCampoPlanner`.

## APP-RUN-004

Objetivo: materializar tareas durables del plan antes de programacion.

Estado: hecho.

Validacion: `TestPrepareAppOrchestrationV0CreatesRunAndProvider`.

## APP-RUN-005

Objetivo: ejecutar una app preparada con puertos inyectados y esperas externas.

Estado: hecho.

Validacion: `TestRunPreparedAppOrchestrationV0RegistraEntregaYAvanzaOla`.
