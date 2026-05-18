# Contratos

## STF-001: StoreV0

`StoreV0` implementa los puertos de estado que hoy pueden montarse en memoria:

- `RunStorePortV0`
- `EventSinkPortV0`
- `RunEventReaderPortV0`
- `WorkflowTaskStorePortV0`
- `WorkflowTaskWriterPortV0`
- `WorkflowTaskWaitStateStorePortV0`
- `WorkflowTaskWaitStateWriterPortV0`
- `RequiredTestEvidenceStorePortV0`
- `OperationalDirectorPlanStateStorePortV0`
- `OperationalDirectorPlanStateWriterPortV0`
- `AgentProcessRegistryPortV0`

## STF-002: FileOutboxLedgerV0

El subpaquete `outbox` implementa:

- `DirectorCycleOutboxLedgerPortV0`
- `PendingOutboxReaderPortV0`
- `OutboxDispatchClaimerPortV0`
- `OutboxDispatchAckPortV0`

Mantiene pending, claim y ack en JSON atomico. Es un conector independiente del
store de runs/tareas para evitar mezclar responsabilidades.

## Persistencia

- `runs/`: un documento por `run_id`.
- `events/`: un documento por `run_id`.
- `workflow_tasks/`: un documento por `run_id + task_id`.
- `workflow_waits/`: un documento por `run_id + wait_ref`.
- `required_test_evidence/`: un documento por `run_id + evidence_ref`.
- `operational_director_plan_states/`: un documento mutable por
  `run_id + plan_ref`; conserva el estado vivo del Director Operativo,
  incluyendo `accepted_review_refs` en pasos `review_deliveries`.
- `agent_processes/`: un documento por `run_id + agent_request_id`.
- `outbox_ledger_v0.json`: ledger durable del subpaquete `outbox`.

Los nombres de fichero derivan de hashes de refs normalizadas para que refs
opacas no se conviertan en rutas.
