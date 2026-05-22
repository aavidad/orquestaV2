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
- el wizard de intake pide el siguiente campo con pregunta de director compacta;
- un borrador completo crea `AppSpecV0` mediante factory y prepara el run del
  director;
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
