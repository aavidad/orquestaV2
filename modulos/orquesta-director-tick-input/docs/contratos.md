# Contratos: orquesta-director-tick-input

## BuildDirectorSchedulerTickInputV0

Entrada canonica: `DirectorTickInputBuildRequestV0`.

Campos obligatorios:

- `tick_ref`
- `occurred_at`
- `run`

Campos opcionales:

- `pending_outbox_refs`
- `work_claims`
- `work_candidates`
- `lease_action_candidates`
- `progress_supervision_candidates`
- `delivery_candidates`
- `review_gate_candidates`
- `replan_followup_candidates`
- `evidence_refs`

Salida:

- `DirectorSchedulerTickInputV0` validado por `orquesta-director-scheduler`.

Invariantes:

- No crea candidates.
- No consulta DB, runtime ni outbox.
- No aplica comandos.
- No despacha outbox.
- La salida solo contiene refs compactas.
- Las refs duplicadas o con espacios se compactan.
- `progress_supervision_candidates` sin `command_meta.run_id` y `report.run_id`
  del run actual se descartan.

Frontera DTI-005/DTI-006:

- Este modulo no define ni invoca un puerto de outbox.
- El caller entrega `pending_outbox_refs` ya calculadas.
- El ensamblador superior vive en `orquesta-director-cycle`, que usa el puerto
  de ledger de `orquesta-director-cycle-outbox` y pasa las refs a este builder.

Normalizacion de proyecciones:

- `capacity_request#capacity_decision:decision_ref` -> `capacity_request`
- `gate_ref#decision:decision#plan:plan_ref` -> `gate_ref`
- `lease_ref#agent:agent_ref#action:...` -> `lease_ref`
- `replan_ref#source:...` -> `replan_ref`
- `quality_gate#decision:<decision>#subject:<subject_ref>` alimenta `blocking_quality_gate_refs`
- `run.tasks` alimenta `snapshot.tasks`
- `run.deliveries` alimenta `snapshot.deliveries`
- `run.reviews` alimenta `snapshot.reviews`
- `run.review_results` alimenta `snapshot.review_results`
- `run.accepted_reviews` alimenta `snapshot.accepted_reviews`
- `run.rework_requests` alimenta `snapshot.rework_requests`

Quality gates:

- El builder deriva `snapshot.blocking_quality_gate_refs` desde `run.quality_gates`.
- Solo proyecta refs opacas de gates `rework_required`, `blocked` o `ask_director` pendientes por `subject_ref`.
- Si el ultimo gate observado de un subject es `accepted`, ese subject no bloquea al scheduler.
- No decide replan, rework ni cierre; solo prepara input compacto para el scheduler.

Review gate candidates:

- El builder acepta `review_gate_candidates` ya calculados y los clona hacia el
  scheduler sin cambiar su semantica.
- No evalua el resultado de revision, no decide aceptacion y no decide rework.
- La decision `accepted`, `changes_requested` o `rejected` debe venir en el
  candidate externo; el scheduler y el workflow validan despues si puede
  traducirse a `AcceptReview` o `RequestRework`.
- `ReworkRequests` viaja solo en el snapshot como refs compactas durables para
  dedupe.

## No Contratos

No forman parte de este modulo:

- candidatos generados desde backlog;
- politica de concurrencia;
- politica de leases;
- evaluacion de progreso;
- lectura de ACKs o receipts de agentes;
- replanificacion;
- event-store o persistencia.
