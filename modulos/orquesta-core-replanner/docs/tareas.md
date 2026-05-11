# Tareas locales: orquesta-core-replanner

```text
ID: RPL-008
Objetivo: Permitir `split_task` desde `changes_requested` para planes de retrabajo que aportan microtareas por puerto.
Write-set: review_replan_validation_v0.go, review_replan_v0_test.go, docs/*
Contrato: ReviewReworkSignalV0 -> ReplanProposalV0
Validacion: go test -count=1 ./modulos/orquesta-core-replanner
Bloqueos: no crear tareas aqui; la materializacion vive en director/orquestacionnucleoapp.
Estado: completada
```

```text
ID: RPL-000
Objetivo: Crear microproyecto y contratos candidatos de replanificacion.
Write-set: orquesta-core-replanner/**
Contrato: ReplanProposalV0, ReplanAcceptedV0 candidato
Validacion: git diff --check -- modulos/orquesta-core-replanner
Bloqueos: ninguno
Estado: completada documental inicial
```

```text
ID: RPL-001
Objetivo: Implementar DTO puro `ReplanProposalV0` con validacion compacta.
Write-set: replan_proposal_*.go, docs/*
Contrato: ReplanProposalV0
Validacion: go test -count=1 ./modulos/orquesta-core-replanner
Bloqueos: no tocar core-workflow
Estado: completada
```

```text
ID: RPL-002
Objetivo: Implementar `ReviewReworkSignalV0` y traducir `ReviewResultV0 changes_requested/rejected` a `ReplanProposalV0`.
Write-set: review_replan_*.go, tests, docs/*
Contrato: ReviewResultV0/ReviewReworkSignalV0 -> ReplanProposalV0
Validacion: go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow
Bloqueos: solo importar contratos publicos
Estado: completada
```

```text
ID: RPL-003
Objetivo: Implementar `AgentReworkSignalV0` y traducir `AgentWorkAssessed` loop_detected/garbage/needs_revision a `ReplanProposalV0`.
Write-set: assessment_replan_*.go, tests, docs/*
Contrato: AgentReworkSignalV0 -> ReplanProposalV0
Validacion: go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow
Bloqueos: no ejecutar runtime ni parar agentes
Estado: completada
```

```text
ID: RPL-004
Objetivo: Definir promocion minima a core-workflow para aceptar/rechazar una propuesta de replanificacion.
Write-set: docs/promocion_core_workflow.md
Contrato: AcceptReplan candidato
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e.
Bloqueos: requiere RPL-001
Estado: cerrada por promocion equivalente
Salida: la promocion efectiva no usa `AcceptReplan`; queda reemplazada por `RecordReplanDecision -> ReplanDecisionRecorded` y followups explicitos separados.
```

```text
ID: RPL-005
Objetivo: Especificar registro durable minimo `RecordReviewResult`, `RequestRework` y `RecordReplanDecision`.
Write-set: docs/promocion_core_workflow.md
Contrato: ReviewResultRecorded, ReworkRequested, ReplanDecisionRecorded candidatos
Validacion: 2026-05-06, ok, go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e.
Bloqueos: no tocar core-workflow hasta cerrar DTOs puros
Estado: completada tras promocion a workflow
Salida: `RecordReviewResult`, `RequestRework` y `RecordReplanDecision` estan implementados en workflow con identidad fuerte, pruebas unitarias y E2E.
```

```text
ID: RPL-006
Objetivo: Implementar candidato puro `ReplanDecisionV0` para preparar `RecordReplanDecision -> ReplanDecisionRecorded`.
Write-set: replan_decision_*.go, docs/*
Contrato: ReplanDecisionV0
Validacion: go test -count=1 ./modulos/orquesta-core-replanner
Bloqueos: no tocar core-workflow; followups reales se materializan en cortes separados.
Estado: completada
```

```text
ID: RPL-007
Objetivo: Traducir `AgentFailed` a `ReplanProposalV0` sin relanzamiento automatico ni reutilizacion implicita del agente fallido.
Write-set: agent_failed_replan_*.go, tests, docs/*
Contrato: AgentFailedReplanSignalV0 -> ReplanProposalV0
Validacion: go test -count=1 ./modulos/orquesta-core-replanner
Bloqueos: no tocar workflow/director; el reemplazo real se materializa despues con comandos separados y candidates explicitos.
Estado: completada
```
