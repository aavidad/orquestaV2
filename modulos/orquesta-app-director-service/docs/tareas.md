# Tareas: orquesta-app-director-service

## APP-DIR-SVC-001

Objetivo: convertir `AppSpecRequestV0` en arranque de director.

Estado: hecho.

Validacion: `TestStartAppDirectorV0StartsDirectorThroughInjectedPorts`.

## APP-DIR-SVC-002

Objetivo: rechazar requests invalidas como errores publicos de factory.

Estado: hecho.

Validacion: `TestStartAppDirectorV0ReturnsFactoryValidationIssues`.

## APP-DIR-SVC-003

Objetivo: impedir dependencias legacy o persistencia hardcodeada.

Estado: hecho.

Validacion: `TestAppDirectorServiceArchitectureV0NoImportaLegacyNiDBHardcodeada`.

## APP-DIR-SVC-004

Objetivo: devolver el equipo de directores preparado por el intake sin exponer
detalles internos del scheduler.

Estado: hecho.

Validacion: `go test -count=1 ./modulos/orquesta-app-director-service`.

## APP-DIR-SVC-005

Objetivo: demostrar que el servicio arranca el equipo de directores con batch
dispatcher inyectado cuando la solicitud pide autonomia alta.

Estado: hecho.

Validacion: `TestStartAppDirectorV0AutonomiaAltaStartsDirectorTeamThroughBatch`.

## APP-DIR-SVC-006

Objetivo: consumir decisiones compactas producidas por el director sin intervencion manual.

Estado: hecho.

Validacion: `TestStartAppDirectorV0ConsumesDirectorDecisionSource`.

## APP-DIR-SVC-007

Objetivo: reentrar automaticamente al loop tras decisiones del director y lanzar agentes de programacion para microtareas creadas.

Estado: hecho.

Validacion: `TestStartAppDirectorV0ConsumesDecisionsAndRerunsProgrammingLoop`.

## APP-DIR-SVC-008

Objetivo: continuar un run existente sin recrear AppSpec/intake para consumir
ACKs y decisiones tardias del director.

Estado: hecho.

Validacion: `go test -count=1 ./modulos/orquesta-app-director-service` y
`TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion` desde el stack.

## APP-DIR-SVC-009

Objetivo: impedir que el director cierre una solicitud normal sin evidencias
minimas del `request_kind`.

Estado: hecho.

Validacion: `TestGuardStartAppDirectorClosurePolicyV0BloqueaAppCompletaSinCodigo`,
`TestGuardStartAppDirectorClosurePolicyV0PermiteDebugReducido` y
`TestGuardStartAppDirectorClosurePolicyV0PermiteAppCompletaConEvidencia`.

## APP-DIR-SVC-010

Objetivo: probar la politica de cierre desde `ContinueAppDirectorV0`, no solo
desde helpers unitarios.

Estado: hecho.

Validacion: `TestContinueAppDirectorV0BloqueaCierreAppCompletaNormalSinEvidencias`
y `TestContinueAppDirectorV0PermiteCierreAppCompletaNormalConEvidencias`.
