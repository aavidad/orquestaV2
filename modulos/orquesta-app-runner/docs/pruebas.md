# Pruebas: orquesta-app-runner

```bash
go test -count=1 ./modulos/orquesta-app-runner
```

Cobertura:

- `AppSpecV0` validada produce run activo en programacion;
- la salida expone progreso inicial para que web/director no reconstruyan el
  estado del plan;
- `AppSpecV0` con persistencia prepara plan grande de 11 microtareas y respeta
  olas por delivery;
- el run preparado materializa una microtarea durable por unidad del plan antes
  de programacion;
- el provider emite la primera microtarea lista;
- el loop progresivo del nucleo solicita capacidad y agente para bootstrap;
- `RunPreparedAppOrchestrationV0` arranca bootstrap y, al observar entrega,
  registra delivery y avanza a arquitectura;
- `RunPreparedAppOrchestrationV0` completa un plan grande con entregas
  observadas por agente arrancado, incluyendo olas paralelas y caps compactos;
- si hay `ExternalWaiter`, `MaxExternalWaits=0` usa el presupuesto por defecto y
  permite cerrar una app grande por entregas externas paso a paso;
- con `UseAutonomousDirectorLoop=true`, el runner usa
  `RunAutonomousDirectorLoopV0`, pasa limites locales a la politica y propaga
  `director_loop_stats`;
- con `UseAutonomousDirectorLoop=true`, las esperas externas y el avance
  multi-fase cierran tambien unidades de documentacion, integracion y revision;
- con `UseAutonomousDirectorLoop=true` y `ProgressSource` inyectado, el runner
  reconsulta el puerto al cerrar stats y expone progreso final util en
  `director_loop_stats.run.progress`: agentes observados, progressing/stalled,
  tareas observadas y `last_progress` por agente;
- un fallo transitorio de `LoadRunV0` no reinicia ni guarda el run inicial;
- el provider de candidatos se reconstruye desde `Prepared.Plan`, aunque el
  provider embebido venga vacio tras cruzar transporte/persistencia;
- no se importa `cmd` ni `db`.

Evidencia 2026-05-09:

- `TestPrepareAppOrchestrationV0CreatesRunAndProvider`;
- `TestPrepareAppOrchestrationV0PreparaPlanGrandeDesdeAppSpec`;
- `TestPrepareAppOrchestrationV0RejectsSpecNoValidada`;
- `TestPrepareAppOrchestrationV0ConservaCampoPlanner`;
- `TestPreparedAppOrchestrationV0ProgressiveLoopRequestsBootstrapAgent`
  verifica `wait_external` tras arrancar el agente y outbox vacio;
- `TestRunPreparedAppOrchestrationV0ArrancaBootstrapGrande`;
- `TestRunPreparedAppOrchestrationV0RegistraEntregaYAvanzaOla`;
- `TestRunPreparedAppOrchestrationV0CompletaPlanGrandeConEntregas`;
- `TestRunPreparedAppOrchestrationV0EsperaExternaPorDefectoHastaCompletar`;
- `TestRunPreparedAppOrchestrationV0UsaDirectorAutonomoOptInYPropagaStats`;
- `TestRunPreparedAppOrchestrationV0DirectorAutonomoMantieneEsperaExterna`;
- `TestRunPreparedAppOrchestrationV0DirectorAutonomoEnriqueceStatsConProgressSource`;
- `TestRunPreparedAppOrchestrationV0NoReiniciaRunSiLoadFalla`;
- `TestRunPreparedAppOrchestrationV0ReconstruyeProviderDesdePlan`;
- `TestAppRunnerArchitectureV0NoImportaLegacyNiDBHardcodeada`.
