# Contratos v0

## RunQueueReaderPortV0

`RunQueueReaderPortV0` es el puerto outbound para obtener candidatos de runs desde una fuente externa:

```go
ListRunSchedulingCandidatesV0(context.Context, RunQueueReadRequestV0) ([]RunSchedulingCandidateV0, error)
```

El puerto no reserva, no bloquea, no reordena y no muta estado. Cualquier adaptador real debe vivir fuera de este microproyecto.

## RunQueuePriorityWriterPortV0

`RunQueuePriorityWriterPortV0` permite cambiar la puntuacion base de una run
sin exponer almacenamiento:

```go
SetRunPriorityV0(context.Context, RunQueuePriorityCommandV0) (RunSchedulingCandidateV0, error)
```

`RunQueuePriorityCommandV0` exige `run_ref` y transporta `priority_score`,
`app_ref`, `status`, `requested_by`, `reason`, `idempotency_key` y
`evidence_refs`.

`status` es opcional: si llega vacio, el writer conserva el estado del candidato
o usa el default del adaptador. Si una composicion lo informa como estado
terminal, el adaptador debe persistirlo para que el ranking no vuelva a exponer
esa run como ejecutable.

## RunSchedulingCandidateV0

Campos minimos:

- `run_ref`: ref opaca de la run.
- `app_ref`: ref opaca de la app propietaria.
- `status`: estado publico de la run.
- `priority_score`: prioridad base, ordenada de mayor a menor.
- `updated_at`: instante de ultima actualizacion usado para aging/fairness.
- `fairness_group_ref`: ref opcional de grupo de fairness. Si falta, el ranking
  deriva un grupo estable `app:<app_ref>` o `run:<run_ref>` y expone
  `fairness_group_missing`.
- `attempt_group`: clave causal opaca opcional para agrupar intentos del mismo
  consumidor/objetivo/item/write-set sin reglas de dominio.
- `parent_run_ref`, `supersedes_run_ref`, `rescue_reason`: enlaces opcionales
  para rescates/reintentos. Son refs/razones opacas; el modulo no interpreta
  contenido OPES ni de ninguna app consumidora.
- `evidence_refs`: refs opacas de evidencia.

## ProjectRunQueueAttemptsV0

Funcion pura:

- recibe candidatos de cola y no consulta servicios externos;
- agrupa por `attempt_group.group_ref` o por clave derivada de
  `consumer_ref`, `objective_ref`, `work_item_ref` y `write_set_refs`;
- si no hay metadata causal, cada run queda en su propio grupo `run:<run_ref>`;
- expone `original_run_ref`, `rescue_run_refs`, `active_attempt_ref`,
  `parent_run_ref`, `supersedes_run_ref`, `rescue_reason`, `status_counts` y
  `evidence_refs`;
- el intento activo se elige de forma determinista priorizando estado
  ejecutable, `updated_at` mas reciente, mayor `priority_score` y desempate por
  `run_ref`.

## RankRunCandidatesV0

Funcion pura:

- recibe candidatos y una `RunQueueRankingPolicyV0`;
- no lee reloj global;
- no consulta servicios externos;
- no muta el slice de entrada.

Reglas:

1. filtra `paused`, `delivered`, `canceled`, `stopped` y `closed`;
2. ordena por `priority_score` descendente;
3. dentro de la misma prioridad aplica `fairness_group_paused` y
   `fairness_group_boosted` segun ventana, limite y reloj inyectado;
4. desempata por `aging_boost` descendente;
5. desempata por `updated_at` ascendente con orden estable.

El `aging_boost` es secundario en v0: no se suma a `priority_score`.
La fairness tambien es secundaria: no eleva un grupo por encima de una prioridad
manual mayor ni sustituye guardas opt-in de smokes, proveedor o efectos externos.
