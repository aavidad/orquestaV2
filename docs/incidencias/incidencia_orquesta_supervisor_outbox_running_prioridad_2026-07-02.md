# Incidencia: outbox pendiente publicado como ejecucion en curso

Fecha: 2026-07-02.

## Sintoma

Un `RunExecutionSummaryV0` con `QueueStatus=running` y diagnostico de
`PendingOutboxCount` podia proyectarse como ejecucion en curso/stalled antes de
publicarse como `waiting_outbox`. En ese estado, el operador veia una cola viva
sin una accion clara de dispatch/capacidad.

## Causa

La prioridad de `supervisorExecutionPublicStatusV0` trataba el `pending_outbox`
diagnostico como espera solo si `Outcome` no era `running`. Eso dejaba fuera el
caso frecuente `QueueStatus=running` con outbox pendiente.

## Cierre

`pending_outbox` diagnostico se proyecta como `waiting_outbox` aunque el resumen
marque `running`, pero no tapa un `process_ref_registered` sin proceso vivo
verificable. Ese caso sigue siendo `stalled`/`external_process_unverified`.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-server -run 'TestSupervisorProjectionV0(WaitUnhandledOutboxNoEsRunning|ProcessRefSinProcesoVerificableEsStalled|QueueRunningSinEvidenciaVivaEsStalled|SnapshotProcesoRunningConOutboxEsRunningLive|NoMezclaEvidenciaVivaDeOtroRun)|TestRuntimeV0SupervisorNoPreparaAutomejoraConWaitUnhandledOutbox'
go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server
```
