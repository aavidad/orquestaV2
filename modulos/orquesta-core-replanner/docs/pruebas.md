# Pruebas locales: orquesta-core-replanner

```text
Caso: review_changes_requested_split
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner -run TestReviewResultToReplanProposalV0ChangesRequestedSupportsSplit
Evidencia esperada: `changes_requested` acepta `split_task` como propuesta compacta; no crea tareas ni followups por si solo.
Estado: completada en RPL-008
```

```text
Caso: replan_proposal_compacto
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner
Evidencia esperada: propuesta valida acepta refs opacas y rechaza DB/provider/model/HOME/OAuth/runtime/prompts/transcripts.
Estado: completada en RPL-001
```

```text
Caso: review_result_to_replan
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow
Evidencia esperada: review `accepted` no propone replan; `changes_requested` propone rework; `rejected` propone split/retry/ask_director segun evidencia compacta.
Estado: completada en RPL-002
```

```text
Caso: agent_work_assessment_to_replan
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow
Evidencia esperada: `acceptable` no propone replan; `needs_revision/request_revision` propone retry_task; `garbage|loop_detected|capacity_limited|timeout/stop_agent` propone replace_agent o abort_task; `ask_director` propone ask_director sin parar agentes ni pedir capacidad.
Estado: completada en RPL-003
```

```text
Caso: replan_decision_compacto
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner
Evidencia esperada: decision valida acepta refs opacas, acciones del catalogo y followups; rechaza refs obligatorias vacias, acciones desconocidas y DB/runtime/provider/HOME/OAuth/modelo/secretos/prompts/transcripts; clona slices y es determinista/idempotente para el mismo payload.
Estado: completada en RPL-006
```

```text
Caso: replan_promocion_workflow
Tipo: integration_contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner ./modulos/orquesta-core-workflow ./modulos/orquesta-e2e
Evidencia esperada: workflow registra ReviewResultRecorded, ReworkRequested y ReplanDecisionRecorded con identidad fuerte; E2E cubre rework/replan y reemplazo de agente fallido sin runtime real ni retry implicito.
Estado: completada en RPL-004/RPL-005 el 2026-05-06
```

```text
Caso: agent_failed_to_replan
Tipo: unit | contract
Comando: go test -count=1 ./modulos/orquesta-core-replanner
Evidencia esperada: `AgentFailed` produce propuesta `replace_agent` o `ask_director` con `source_ref=agent_request_id`; `replace_agent` exige `replacement_role`; retry/reuse implicito y detalles runtime/DB/provider/HOME/OAuth/modelo quedan rechazados.
Estado: completada en RPL-007
```
