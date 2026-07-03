# Pruebas

Comando local:

```sh
go test -count=1 ./modulos/orquesta-run-memory
```

Cobertura v0:

- `PauseRunV0` y `ResumeRunV0` cambian el estado, conservan evidencias
  normalizadas como `evidence-ref-autoprogramming-goal-backend-missing-after-external-cleanup`
  y preservan copias defensivas.
- `StopRunV0` registra `forced` y estado `stop_requested`.
- `CancelRunV0` registra `forced` y estado `cancel_requested`.
- `CompleteRunControlV0` marca `stopped/canceled` y rechaza estados no
  terminales.
- `SetRunPriorityV0` actualiza candidatos en memoria y puede marcar un
  candidato como terminal para que no reaparezca en scheduling.
- `ListRunSchedulingCandidatesV0` devuelve snapshots filtrables y excluye
  aliases terminales legacy como `completed` antes de aplicar `limit`.
- Ranking delegado a `orquesta-run-queue` conserva prioridad antes de aging.
- La arquitectura de produccion no usa E/S, red, procesos ni persistencia.
