# orquesta-run-queue

Mini-proyecto puro para ordenar candidatos de ejecucion en una cola global multi-app.

Incluye:

- contrato `RunQueueReaderPortV0` para leer candidatos desde una fuente externa;
- DTO `RunSchedulingCandidateV0` con `app_ref`, `run_ref`, `status`, `priority_score` y `updated_at`;
- funcion pura `RankRunCandidatesV0` con aging/fairness determinista;
- funcion pura `ProjectRunQueueAttemptsV0` para proyectar intentos/rescates
  enlazados por refs opacas y calcular `active_attempt_ref`;
- pruebas locales del contrato de ordenacion y filtrado.

Fuera de alcance:

- DB, HTTP, MCP, scheduler interno, workflow core y adaptadores reales;
- bloqueos, leases, despacho de runs o mutacion de estado;
- politicas de capacidad, runtime o seleccion de agentes.

Orden de ranking v0:

1. filtra runs con `status` `paused`, `delivered`, `canceled`, `stopped` o
   `closed`;
2. ordena por `priority_score` descendente;
3. dentro de la misma prioridad aplica pausa/boost de `fairness_group_ref`;
4. desempata por `aging_boost` descendente si aplica;
5. desempata por `updated_at` ascendente con orden estable en empates exactos.

Validacion local:

```sh
go test -count=1 ./modulos/orquesta-run-queue
```
