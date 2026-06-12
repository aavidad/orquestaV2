# Pruebas

Comando local:

```sh
go test -count=1 ./modulos/orquesta-run-queue
```

Cobertura v0:

- filtrado de `paused`, `delivered`, `canceled`, `stopped` y `closed`;
- prioridad antes de aging;
- prioridad antes de fairness entre grupos;
- reason codes `fairness_group_paused`, `fairness_group_boosted` y
  `fairness_group_missing`;
- aging como desempate entre misma prioridad;
- `updated_at` ascendente con estabilidad en empates exactos;
- normalizacion de `RunQueuePriorityCommandV0.status`;
- proyeccion causal de rescates con `active_attempt_ref`, `parent_run_ref`,
  `supersedes_run_ref` y `rescue_reason`;
- pureza basica: no mutar referencias de evidencia del input;
- gate de arquitectura contra imports/terminos de adaptadores y persistencia.
