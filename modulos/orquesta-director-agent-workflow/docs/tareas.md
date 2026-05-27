# Tareas locales

## DAW-007

Objetivo: preservar `context_refs` opacas desde `create_microtask` hasta
`WorkflowTaskV0` y `TaskStore`.

Estado: hecho.

Validacion:

- `TestBuildDirectorAgentWorkflowCommandV0TraduceMicrotarea`;
- `TestApplyDirectorAgentDecisionV0PlanificaMicrotareaDesdeCadenaDirector`;
- `go test -count=1 ./modulos/orquesta-director-agent-workflow`.

## DAW-001

Objetivo: traducir decisiones compactas del director a comandos publicos del
workflow.

Estado: hecho.

Validacion:

- `TestBuildDirectorAgentWorkflowCommandV0TraduceBrainstorm`;
- `TestBuildDirectorAgentWorkflowCommandV0TraduceOpenPhase`;
- `TestBuildDirectorAgentWorkflowCommandV0TraduceVote`;
- `TestBuildDirectorAgentWorkflowCommandV0TraduceAcceptDecision`;
- `TestBuildDirectorAgentWorkflowCommandV0TraduceContrato`;
- `TestBuildDirectorAgentWorkflowCommandV0TraduceMicrotarea`;
- `TestBuildDirectorAgentWorkflowCommandV0RechazaDecisionInvalida`;
- `TestBuildDirectorAgentWorkflowCommandV0RequiereOccurredAt`.

## DAW-002

Objetivo: aplicar una decision del director a un run por puertos inyectados.

Estado: hecho.

Validacion:

- `TestApplyDirectorAgentDecisionV0AplicaBrainstormPorPuertos`;
- `TestApplyDirectorAgentDecisionV0PlanificaMicrotareaDesdeCadenaDirector`;
- `TestApplyDirectorAgentDecisionV0AplicaContratoYMicrotareaPorPuertos`;
- `TestApplyDirectorAgentDecisionV0RequiereTaskStoreParaMicrotarea`;
- `TestApplyDirectorAgentDecisionV0ValidaPuertos`.

## DAW-003

Objetivo: publicar un puerto de fuente de decisiones para que servicios de aplicacion consuman salidas del director.

Estado: hecho.

Validacion:

- `go test -count=1 ./modulos/orquesta-director-agent-workflow`.

## DAW-004

Objetivo: documentar y probar que `propose_autonomous_plan_team` es DTO puro
del director hasta que exista comando publico del workflow.

Estado: hecho.

Validacion:

- `TestBuildDirectorAgentWorkflowCommandV0RechazaPlanEquipoComoComandoWorkflow`;
- `go test -count=1 ./modulos/orquesta-director-agent-workflow`.

## DAW-005

Objetivo: traducir y aplicar `close_task` como `CloseTask` publico del workflow.

Estado: hecho.

Validacion:

- `TestBuildDirectorAgentWorkflowCommandV0TraduceCierre`;
- `TestApplyDirectorAgentDecisionV0AplicaCloseTaskPorPuertos`;
- `go test -count=1 ./modulos/orquesta-director-agent-workflow`.

## DAW-006

Objetivo: traducir comandos faltantes para app completa y entregar stats compactas al source port.

Estado: hecho.

Validacion:

- `TestBuildDirectorAgentWorkflowCommandV0TraduceConsultasCapacidadAgenteReworkYReplan`;
- `TestBuildDirectorAgentWorkflowCommandV0AskUserMantieneTargetUser`;
- `TestBuildDirectorAgentCompactStatsV0ResumeRunSinDetallesOperativos`;
- `go test -count=1 ./modulos/orquesta-director-agent-workflow`.

## DAW-008

Objetivo: reconciliar T207 sin abrir comando nuevo del workflow.

Estado: cerrado documentalmente el 2026-05-27.

Validacion:

- `go test -count=1 ./modulos/orquesta-director-agent-workflow`;
- la cobertura funcional de T207 vive en runtime/orchestration.
