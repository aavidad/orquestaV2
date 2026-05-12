# Pruebas: orquesta-app-director-service

```bash
go test -count=1 ./modulos/orquesta-app-director-service
```

Cobertura esperada:

- arranque completo con puertos inyectados;
- resultado expone `director_tasks` sin que el transporte conozca scheduler ni
  outbox;
- autonomia alta arranca el equipo de directores mediante batch dispatcher
  inyectado;
- `DeliverySource` inyectado se compone dentro del servicio y registra un
  `PhaseArtifactRegistered` sin que REST/MCP/web conozcan scheduler;
- `DirectorDecisionSource` inyectado permite aplicar una decision del director
  desde el propio servicio;
- `ExternalWaiter` inyectado permite esperar decisiones tardias del director y
  reentrar al loop gestionado para arrancar agentes derivados;
- `ContinueAppDirectorV0` permite reentrar sobre un run existente sin recrear
  AppSpec/intake y usando solo puertos inyectados;
- politica de cierre: `crear_app_completa` normal no puede cerrarse solo con
  documentacion; `debug` permite alcance reducido;
- reentrada automatica tras decisiones del director crea microtarea y arranca
  el agente de programacion sin pasos manuales intermedios;
- errores de validacion de factory sin transporte;
- sin imports legacy ni DB hardcodeada.

Evidencia 2026-05-09:

- `TestStartAppDirectorV0StartsDirectorThroughInjectedPorts`;
- `TestStartAppDirectorV0AutonomiaAltaStartsDirectorTeamThroughBatch`;
- `TestStartAppDirectorV0ConsumesDirectorDecisionSource`;
- `TestStartAppDirectorV0IgnoresPersistedDirectorDecisionsAlreadyApplied`;
- `TestStartAppDirectorV0NoSaltaDecisionPendienteDelDirector`;
- `TestStartAppDirectorDecisionDeferredV0RetieneProgramacionHastaMicrotareas`;
- `TestStartAppDirectorV0ConsumesDecisionsAndRerunsProgrammingLoop`;
- `TestStartAppDirectorV0WaitsExternalBeforeConsumingDirectorDecisions`;
- `TestGuardStartAppDirectorClosurePolicyV0BloqueaAppCompletaSinCodigo`;
- `TestGuardStartAppDirectorClosurePolicyV0PermiteDebugReducido`;
- `TestGuardStartAppDirectorClosurePolicyV0PermiteAppCompletaConEvidencia`;
- `TestContinueAppDirectorV0BloqueaCierreAppCompletaNormalSinEvidencias`;
- `TestContinueAppDirectorV0PermiteCierreAppCompletaNormalConEvidencias`;
- cobertura indirecta desde `orquesta-app-codex-stack`:
  `TestDrainRunV0ConsumeDecisionFileTardioYArrancaProgramacion`;
- `TestStartAppDirectorV0ConsumesDirectorDeliverySource`;
- `TestComposeStartAppDirectorProviderV0IncludesReviewGateSource`;
- `TestContinueAppDirectorV0ProcessesReviewGateSource`;
- `TestStartAppDirectorV0ReturnsFactoryValidationIssues`;
- `TestStartAppDirectorV0RequiresInjectedPorts`;
- `TestAppDirectorServiceArchitectureV0NoImportaLegacyNiDBHardcodeada`.

Evidencia real 2026-05-09:

- Harness MCP real en `/tmp/orquesta_director_driven_real_run`.
- Resultado: `estado=ok`, `loop_status=quiescent`, fase `programacion`.
- Agentes arrancados por Orquesta: director, REST y web.
- Entregas registradas: `ack-ref-agenda-rest`, `ack-ref-agenda-web`.
- Proyecto generado en `/tmp/orquesta-director-driven-20260509190441/project`.
- Validacion externa: `go test ./...` en `apps/agenda-rest`, `node --check` en
  modulos JS de `apps/agenda-web`, sin ficheros de codigo sobre 300 lineas.
