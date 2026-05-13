# Pruebas: orquesta-director-tick-input

## Ejecutadas Localmente

```sh
go test -count=1 ./modulos/orquesta-director-tick-input
```

Resultado: `ok` el 2026-05-13.

## Cobertura Contractual

- Construye snapshot desde run durable.
- Extrae refs canonicas de capacidad, gates, leases, phase_artifacts y replan.
- Adjunta outbox pendiente externo y provoca `waiting` en scheduler.
- Rechaza run invalido o campos obligatorios ausentes.
- Integra con scheduler real y runner real usando workflow en memoria.
- Ciclo progresivo de programacion: RequestCapacity, RegisterCapacityDecision externo, RecordConcurrencyGate + RequestAgent, RegisterAgentStarted externo y cierre quiescent sin duplicar agente.
- Ciclo de ACK no-programacion: receipt externo entra como candidate de artefacto,
  el scheduler registra `RegisterPhaseArtifact` y el snapshot posterior deduplica
  por `phase_artifacts`.
- Snapshot de revision: `run.Reviews`, `run.ReviewResults`,
  `run.AcceptedReviews` y `run.ReworkRequests` quedan expuestos como refs
  compactas para que el scheduler deduplique `RequestReview`,
  `RecordReviewResult`, `AcceptReview` y `RequestRework`.
- `ReviewGateCandidates` se propagan desde el request al scheduler sin crear ni
  reinterpretar candidates en tick-input.
