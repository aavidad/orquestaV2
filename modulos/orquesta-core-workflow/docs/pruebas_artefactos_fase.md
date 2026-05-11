# Pruebas de artefactos de fase v0

```text
Caso: register_phase_artifact_director
Tipo: contrato
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRegisterPhaseArtifactCommandV0ProjectsDirectorArtifact'
Evidencia esperada: un agente director ya arrancado en `brainstorming_arquitectura` puede registrar un artefacto compacto; no hay outbox y el retry exacto es no-op.
Ultima ejecucion: 2026-05-09, ok.
Riesgos: El contenido del artefacto no se guarda en el core; debe vivir en el conector/proyeccion externa que corresponda.
```

```text
Caso: replay_phase_artifact
Tipo: replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestReplayDurableEventsV0AcceptsPhaseArtifactRegistered'
Evidencia esperada: replay durable reconstruye `phase_artifacts` y acepta duplicado exacto sin avanzar secuencia ni duplicar refs.
Ultima ejecucion: 2026-05-09, ok.
Riesgos: El evento exige `AgentStarted` previo; no reemplaza el flujo de lanzamiento de agentes.
```

```text
Caso: phase_artifact_no_programacion
Tipo: invariant
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRegisterPhaseArtifactCommandV0RejectsProgrammingPhase'
Evidencia esperada: `RegisterPhaseArtifact` rechaza `programacion`; para codigo se mantiene `RegisterDelivery`.
Ultima ejecucion: 2026-05-09, ok.
Riesgos: Si se necesitan artefactos auxiliares durante programacion, deben modelarse como evidencia de `DeliveryRegistered` o abrir contrato nuevo.
```

```text
Caso: phase_artifact_agent_started
Tipo: invariant
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRegisterPhaseArtifactCommandV0RequiresStartedAgent|TestValidateOrchestrationRunV0RejectsInvalidPhaseArtifactProjection'
Evidencia esperada: no se aceptan artefactos de agentes no arrancados y el validador rechaza proyecciones huerfanas.
Ultima ejecucion: 2026-05-09, ok.
Riesgos: Parar un agente despues de registrar su artefacto no invalida historico; solo se rechaza registrar si ya estaba parado antes.
```
