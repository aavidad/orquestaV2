# Incidencia: error tecnico de supervisor ocultaba estado operativo vivo

Fecha: 2026-07-02.

## Sintoma

En ejecuciones goal-first y OPES, un tick del supervisor podia devolver error o
timeout tecnico mientras el propio resultado incluia proceso vivo, ACK pendiente
o espera externa recuperable. La proyeccion publica quedaba como `error` o como
otro estado terminal, aunque la accion correcta era esperar/reconciliar sin
relanzar el mismo run.

## Causa

`MarkSupervisorErrorV0` trataba cualquier error del puerto supervisor como
terminal para `LastSupervisorStatus`, `LastSupervisorError` y `LastError`.
Ademas la deteccion de espera externa no leia diagnosticos/evidencias como
`ack_pending`, `submit_pending` o refs equivalentes de recuperacion.

## Cierre

La proyeccion de errores recuperables reutiliza la vista publica del resultado:
si hay `running_live`, `waiting_external` o `waiting_outbox`, el estado publico
queda en esa categoria operativa y el error se conserva como diagnostico
advisory con evidencias causales en el mensaje operativo y `RecentErrors`.

Tambien se bloquea la automejora idle cuando hay ACK/espera externa pendiente,
para no arrancar un nuevo frente mientras sigue vivo trabajo que debe
reconciliarse.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-server -run 'TestSupervisorProjectionV0RuntimeErrorConAckPendienteEsWaitExternal|TestSupervisorProjectionV0ErrorRecuperableConservaEstadoOperativo|TestSupervisorErrorProjectionV0ProcesoVivoOAckPendienteNoEsTerminal|TestRuntimeV0SupervisorNoPreparaAutomejoraConAckPendiente'
go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server
```
