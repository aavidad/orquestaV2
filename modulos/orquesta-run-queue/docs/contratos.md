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
- `fairness_group_ref`: ref opcional para evolucion futura de fairness.
- `evidence_refs`: refs opacas de evidencia.

## RankRunCandidatesV0

Funcion pura:

- recibe candidatos y una `RunQueueRankingPolicyV0`;
- no lee reloj global;
- no consulta servicios externos;
- no muta el slice de entrada.

Reglas:

1. filtra `paused`, `delivered`, `canceled`, `stopped` y `closed`;
2. ordena por `priority_score` descendente;
3. desempata por `aging_boost` descendente;
4. desempata por `updated_at` ascendente con orden estable.

El `aging_boost` es secundario en v0: no se suma a `priority_score`.
