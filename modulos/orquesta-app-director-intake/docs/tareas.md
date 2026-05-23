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

## APP-DIR-008

Objetivo: permitir preparar el director desde un DTO neutral sin que el calculo
interno dependa de `orquesta-factory`.

Estado: hecho; `PrepareAppDirectorInputV0` usa `AppDirectorInputSpecV0` y
`PrepareAppDirectorIntakeV0` queda como wrapper de compatibilidad.

Validacion:

- `TestPrepareAppDirectorInputV0UsesNeutralSpecWithoutFactory`;
- `TestAppDirectorInputSpecFromFactoryV0PreservesDirectorContext`;
- `TestPrepareAppDirectorIntakeV0LegacyFactoryMatchesNeutralInput`.

## APP-DIR-009

Objetivo: convertir una peticion humana amplia en plan revisable del director
antes de preparar trabajo de codigo.

Estado: hecho; `BuildHumanDirectorReviewablePlanV0` crea un contrato neutral
con refs opacas, objetivo, reglas, limites y pistas opcionales. La salida
distingue ejecutar ahora, estudiar antes, posponer por solape y pedir revision.

Validacion:

- `TestBuildHumanDirectorReviewablePlanV0ExecuteNowConRefsOpacas`;
- `TestBuildHumanDirectorReviewablePlanV0EstudiaAntesSiWriteSetAmplio`;
- `TestBuildHumanDirectorReviewablePlanV0NoRechazaTodoPorSolape`;
- `TestBuildHumanDirectorReviewablePlanV0PideRevisionSinObjetivo`;
- `TestBuildHumanDirectorReviewablePlanV0NormalizaAliasesYGlobsSeguros`;
- `TestBuildHumanDirectorReviewablePlanV0PideRevisionPorWriteSetInseguro`.
