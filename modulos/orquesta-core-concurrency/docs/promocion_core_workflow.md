# Promocion a orquesta-core-workflow

## Corte minimo recomendado

La concurrencia debe entrar como gate compacto antes de asignar agentes o como bloqueo local de tarea/grupo. No debe convertirse en scheduler operativo dentro del workflow.

Estado 2026-05-06: la promocion minima `RecordConcurrencyGate -> ConcurrencyGateRecorded` quedo implementada en `orquesta-core-workflow` como NCW-047. El workflow registra la evaluacion, pero no calcula scopes ni lanza agentes.

## Comandos/eventos candidatos

```text
EvaluateConcurrencyGate -> ConcurrencyGateEvaluated
Uso:
  - Decidir si un `RequestAgent` candidato puede pasar para uno o varios claims.
Regla:
  - usa `ConcurrencyGateEvaluationV0` calculado desde `EvaluateParallelGroupsV0`.
  - `allow_request_agent` exige que todos los `subject_claim_refs` esten ready.
  - `block_request_agent` bloquea solo sujetos implicados en conflicto, dependencia pendiente o claim invalido.
  - `ask_director` cubre sujeto vacio/desconocido; no se permite inferir estado leyendo workflow desde este modulo.
```

```text
RegisterWorksetClaim -> WorksetClaimRegistered
Uso:
  - Guardar claim logico de una tarea/grupo antes de pedir agente.
Regla:
  - contiene scopes normalizados; no lee filesystem ni Git.
```

```text
RegisterWorksetConflict -> WorksetConflictDetected
Uso:
  - Bloquear solo los claims implicados cuando hay conflicto.
Regla:
  - el run completo no debe bloquearse si quedan claims seguros ejecutables.
```

```text
RecordParallelGroupPlan -> ParallelGroupPlanRecorded
Uso:
  - Registrar que claims estan listos, bloqueados por dependencia o bloqueados por conflicto.
Regla:
  - no crea goroutines ni workers; director/runtime materializan luego.
```

## Criterios de promocion

- Normalizacion de scopes probada: duplicados, prefijos, absolutos, traversal y refs invalidas.
- Detector puro de write-write/read-write sin imports a filesystem/Git/runtime/persistence.
- Semantica clara de bloqueo local por tarea/grupo.
- NCW-047 validado con tests de workflow; E2E de uso previo a `RequestAgent` queda como cierre de integracion.
