# Tareas: orquesta-app-director-intake

## APP-DIR-001

Objetivo: preparar run inicial para director desde `AppSpecV0`.

Estado: hecho.

Validacion: `TestPrepareAppDirectorIntakeV0CreatesBrainstormRun`.

## APP-DIR-002

Objetivo: emitir candidato de director sin acoplar proveedores.

Estado: hecho.

Validacion: `TestAppDirectorCandidateProviderV0EmitsDirectorAgent`.

## APP-DIR-003

Objetivo: demostrar que el loop progresivo arranca el director con launcher
inyectado.

Estado: hecho con launcher fake de lifecycle para unidad del nucleo.

Validacion: `TestAppDirectorIntakeV0ProgressiveLoopStartsDirector`.

## APP-DIR-004

Objetivo: preparar equipo de directores cuando `AppSpecV0` pide autonomia alta.

Estado: hecho.

Validacion: `TestPrepareAppDirectorIntakeV0AutonomiaAltaCreatesDirectorTeam`.

## APP-DIR-005

Objetivo: emitir el equipo de directores en lotes pequenos por tick.

Estado: hecho.

Validacion: `TestAppDirectorCandidateProviderV0EmitsDirectorTeam`.

## APP-DIR-006

Objetivo: arrancar el equipo de directores con batch dispatcher sin payloads
grandes ni outbox pendiente.

Estado: hecho con launcher fake de lifecycle para unidad del nucleo.

Validacion: `TestAppDirectorIntakeV0ProgressiveLoopStartsDirectorTeamInBatch`.

## APP-DIR-007

Objetivo: permitir intake conversacional por pasos hasta crear `AppSpecV0` y
preparar el run del director.

Estado: hecho.

Validacion:

- `TestAdvanceAppDirectorIntakeWizardV0PideSiguienteCampo`;
- `TestAdvanceAppDirectorIntakeWizardV0CreaAppSpecYRunPreparado`;
- `TestAdvanceAppDirectorIntakeWizardV0UsaFactoryParaInvalidarEnums`.
