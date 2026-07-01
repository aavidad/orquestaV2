# Incidencia: cola vacia bloquea automejora por capacidad

Fecha: 2026-07-02.

ID inventario: `BUG-ORQ-20260701-094`.

## Sintoma

En el remoto aislado `/srv/orquesta-self/runtime/audit-13611445`, Orquesta
preparaba automejora por capacidad solo si el supervisor observaba al menos un
item en cola o refs retryables. Con cola vacia, bajo
`IdleSelfImprovementTargetQueue`, la decision salia como:

```text
AuditStatus=skipped
Reason=capacity_queue_unknown
```

Eso impedia lanzar el scanner de backlog cuando precisamente habia capacidad
libre y ningun trabajo en vuelo.

## Causa

`idleSelfImprovementCapacityDecisionV0` trataba `QueueSize <= 0` y ausencia de
refs retryables como desconocido, aunque la cola vacia es un estado valido. En
modo goal-first, con `target_queue` configurado, esa condicion debe contar como
capacidad disponible para planificar automejora segura.

## Cierre

Se elimina el veto `capacity_queue_unknown`. Si existe planner, no hay intento
en vuelo y `QueueSize=0` queda por debajo de `TargetQueue`, Orquesta calcula:

- `Trigger=capacity_free`
- `FreeCapacity=TargetQueue`
- `MaxRequests=min(max_requests, free_capacity)`

La regresion queda cubierta por
`TestRuntimeV0SupervisorPreparaAutomejoraConColaVaciaBajoObjetivoV0`.

## Evidencia

Pruebas locales:

```bash
go test -count=1 ./modulos/orquesta-server -run 'TestRuntimeV0SupervisorPreparaAutomejoraConColaVaciaBajoObjetivoV0|TestRuntimeV0SupervisorPreparaAutomejoraConCapacidadLibreV0|TestRuntimeV0SupervisorFiltraRequestsDeAutomejoraPorPuertoV0'
go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server
```

No se integran los JSON/borradores generados por el remoto porque son receipts
colaterales de goals y no forman parte del parche funcional.
