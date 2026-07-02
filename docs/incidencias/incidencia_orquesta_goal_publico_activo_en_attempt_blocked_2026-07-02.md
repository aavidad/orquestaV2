# Incidencia: goal publico activo con intento bloqueado

Fecha: 2026-07-02.

## Sintoma

Tras agotar creditos del proveedor Codex, el remoto aislado publicaba
`idle_self_improvement_reason=attempt_blocked`, pero
`idle_self_improvement_goal.active=true` y `receipt_status=running`. Para un
operador o consumidor como OPES, eso parecia trabajo vivo aunque no hubiera
edicion ni progreso posible.

## Causa

La proyeccion `ServerPublicIdleSelfImprovementGoalStateV0` marcaba
`Active=true` para cualquier goal con identidad o estado durable. No distinguia
entre "hay un goal historico que exponer" y "hay trabajo operativo vivo".

## Cierre

La proyeccion publica conserva el objeto y sus refs, pero calcula `active` solo
cuando el estado efectivo es operativo/en curso. Estados como `attempt_blocked`,
`goal_blocked`, `goal_invalid`, cierre aceptado o cierre pendiente ya no se
publican como activos.

Pruebas:

```bash
go test -count=1 ./modulos/orquesta-server -run 'TestServerPublicStatusV0|TestServerReadinessV0ExponeGoalFirstSinCambiarReadyV0'
go test -count=1 ./modulos/orquesta-server ./cmd/orquesta-server
go test -count=1 ./...
```
