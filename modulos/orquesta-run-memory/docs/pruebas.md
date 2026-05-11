# Pruebas

Comando local:

```sh
go test -count=1 ./modulos/orquesta-run-memory
```

Cobertura v0:

- `PauseRunV0` y `ResumeRunV0` cambian el estado y preservan copias defensivas.
- `StopRunV0` registra `forced` y estado `stop_requested`.
- `CancelRunV0` registra `forced` y estado `cancel_requested`.
- `CompleteRunControlV0` marca `stopped/canceled` y rechaza estados no
  terminales.
- `SetRunPriorityV0` actualiza candidatos en memoria.
- `ListRunSchedulingCandidatesV0` devuelve snapshots filtrables.
- Ranking delegado a `orquesta-run-queue` conserva prioridad antes de aging.
- La arquitectura de produccion no usa E/S, red, procesos ni persistencia.
