# Pruebas: orquesta-app-director-intake

```bash
go test -count=1 ./modulos/orquesta-app-director-intake
```

Cobertura esperada:

- `AppSpecV0` validada crea run activo en `brainstorming_arquitectura`;
- el run contiene la solicitud de brainstorm del director;
- con autonomia normal, el provider emite un candidato de capacidad y agente
  director;
- con autonomia alta, el intake prepara un equipo de directores y el provider
  emite una cohorte inicial pequena;
- el loop progresivo puede arrancar el director usando un launcher inyectado y
  queda en `wait_external` esperando su artefacto;
- el loop progresivo puede arrancar el equipo de directores en batch sin dejar
  outbox pendiente y queda en `wait_external`;
- las refs de task/agent se aislan por `spec_id` para evitar colisiones entre
  solicitudes con el mismo nombre de app;
- el summary del director conserva contexto funcional compacto de `AppSpecV0`
  sin transportar runtime/proveedor;
- el camino neutral `PrepareAppDirectorInputV0` prepara el mismo run sin usar
  `AppSpecV0` de factory;
- el adaptador `AppDirectorInputSpecFromFactoryV0` conserva contexto funcional,
  plataformas, datos, i18n, calidad y preferencias de agentes;
- el wizard de intake pide el siguiente campo con pregunta de director compacta;
- un borrador completo crea `AppSpecV0` mediante factory y prepara el run del
  director;
- una peticion humana amplia produce plan revisable del director antes de
  preparar trabajo de codigo;
- el plan distingue ejecutar ahora, estudiar antes, posponer solapes y pedir
  revision;
- refs `worktree_ref` y `branch_ref` se conservan como opacas;
- no hay imports legacy ni bases hardcodeadas.

Evidencia 2026-05-09:

- `TestPrepareAppDirectorIntakeV0CreatesBrainstormRun`;
- `TestPrepareAppDirectorIntakeV0RefsAisladasPorSpecID`;
- `TestPrepareAppDirectorIntakeV0PropagaContextoFuncionalEnSummary`;
- `TestPrepareAppDirectorIntakeV0RejectsSpecNoValidada`;
- `TestAppDirectorCandidateProviderV0EmitsDirectorAgent`;
- `TestAppDirectorCandidateProviderV0DoesNotReemitStartedDirector`;
- `TestAppDirectorIntakeV0ProgressiveLoopStartsDirector`;
- `TestPrepareAppDirectorIntakeV0AutonomiaAltaCreatesDirectorTeam`;
- `TestAppDirectorCandidateProviderV0EmitsDirectorTeam`;
- `TestAppDirectorIntakeV0ProgressiveLoopStartsDirectorTeamInBatch`;
- `TestAdvanceAppDirectorIntakeWizardV0PideSiguienteCampo`;
- `TestAdvanceAppDirectorIntakeWizardV0CreaAppSpecYRunPreparado`;
- `TestAdvanceAppDirectorIntakeWizardV0UsaFactoryParaInvalidarEnums`;
- `TestAppDirectorIntakeArchitectureV0NoImportaLegacyNiDBHardcodeada`.

Evidencia 2026-05-23:

- `TestPrepareAppDirectorInputV0UsesNeutralSpecWithoutFactory`;
- `TestAppDirectorInputSpecFromFactoryV0PreservesDirectorContext`;
- `TestPrepareAppDirectorIntakeV0LegacyFactoryMatchesNeutralInput`.

Evidencia 2026-05-23, intake humano:

- `TestBuildHumanDirectorReviewablePlanV0ExecuteNowConRefsOpacas`;
- `TestBuildHumanDirectorReviewablePlanV0EstudiaAntesSiWriteSetAmplio`;
- `TestBuildHumanDirectorReviewablePlanV0NoRechazaTodoPorSolape`;
- `TestBuildHumanDirectorReviewablePlanV0PideRevisionSinObjetivo`;
- `TestBuildHumanDirectorReviewablePlanV0NormalizaAliasesYGlobsSeguros`;
- `TestBuildHumanDirectorReviewablePlanV0PideRevisionPorWriteSetInseguro`.
