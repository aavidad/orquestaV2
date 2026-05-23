# Contratos locales: orquesta-core-concurrency

## Contrato candidato: WorksetClaimV0

```text
Tipo: DTO puro
Estado: candidato
Campos:
  - claim_ref
  - run_ref
  - task_ref
  - group_ref opcional
  - agent_request_id opcional
  - read_set
  - write_set
  - depends_on
  - evidence_refs opcional
Invariantes:
  - `write_set` no puede estar vacio para trabajo de codigo.
  - No contiene rutas absolutas HOME, secretos, comandos Git, runtime ni DB.
  - Las rutas/refs son scopes logicos normalizados.
```

## Contrato local: WorkflowTaskV0/ContextBundleV0 -> WorksetClaimV0

```text
Tipo: mapper puro
Estado: implementado local en CCY-004
Funcion: BuildWorksetClaimFromWorkflowTaskV0
Entrada:
  - WorkflowTaskV0 publico de orquesta-core-workflow
  - ContextBundleV0 publico de orquesta-context
Salida:
  - WorksetClaimV0 normalizado
  - issues publicos de WorksetClaimV0
Invariantes:
  - No lee filesystem, Git, runtime ni estado real.
  - `write_set` viene de la tarea y de entradas write_ref del contexto.
  - `read_set` viene de entradas read_ref del contexto.
  - `evidence_refs` conserva refs compactas del contexto.
  - Tarea o contexto invalidos producen issues; no se inventan scopes ni defaults.
```

## Contrato candidato: ScopeRefV0

```text
Tipo: valor puro normalizado
Estado: candidato
Formato:
  - refs logicas relativas al workspace o modulo
  - separador `/`
  - sin path absoluto, `..`, HOME ni secretos
Reglas:
  - `modulos/x` solapa con `modulos/x/a.go`.
  - `modulos/x/a.go` no solapa con `modulos/x/b.go`.
  - duplicados se normalizan de forma determinista.
```

## Contrato candidato: WorksetConflictV0

```text
Tipo: DTO/evento candidato
Estado: implementado candidato
Campos:
  - conflict_ref
  - claim_refs
  - conflict_kind: write_write | read_write
  - scope_refs opcional
  - recommended_action: serialize | ask_director | split_task | reject_claim
  - summary
  - evidence_refs opcional
```

Las dependencias invalidas o no cerradas se emiten como `WorksetDependencyIssueV0`,
no como `WorksetConflictV0`.

## Contrato candidato: ParallelGroupPlanV0

```text
Tipo: salida pura de evaluacion
Estado: implementado candidato
Campos:
  - plan_ref
  - run_ref
  - ready_claim_refs
  - blocked_claim_refs
  - hard_blocked_claim_refs opcional
  - conflict_refs
  - repairable_conflict_refs opcional
  - sequence_claim_refs opcional
  - evidence_refs opcional
  - summary
Invariantes:
  - No crea goroutines, locks, procesos ni scheduler real.
  - Solo decide que trabajos logicos pueden convivir; director/runtime materializan despues.
  - `ready_claim_refs` contiene claims sin dependencias pendientes y sin conflictos declarados.
  - `blocked_claim_refs` combina bloqueos por dependencia y claims implicados en conflictos.
  - `conflict_refs` referencia conflictos detectados por read/write-set declarado.
  - Un conflicto seguro de write/read-set no es fallo de claim: queda tambien
    en `repairable_conflict_refs` y `sequence_claim_refs` para que el Director
    pueda serializar o replanificar con evidencia compacta.
  - Alcances inseguros o invalidos siguen en `hard_blocked_claim_refs` y no se
    convierten en conflictos reparables.
```

## Contrato candidato: ConcurrencyGateEvaluationV0

```text
Tipo: DTO/evento candidato
Estado: implementado candidato local
Funcion local: EvaluateConcurrencyGateV0
Campos:
  - schema_version: concurrency_gate.v0
  - gate_ref
  - run_ref opcional
  - plan_ref
  - subject_claim_refs
  - ready_claim_refs
  - blocked_claim_refs
  - conflict_refs
  - decision: allow_request_agent | block_request_agent | ask_director
  - summary
Invariantes:
  - Se basa en `EvaluateParallelGroupsV0`; no reimplementa conflictos ni dependencias.
  - El sujeto del gate son claims ya declarados con `WorksetClaimV0` y `ScopeRefV0`.
  - `allow_request_agent` solo aplica si todos los `subject_claim_refs` estan ready.
  - `block_request_agent` aplica si algun sujeto esta blocked por dependencia, claim invalido o conflicto.
  - `ask_director` cubre sujeto vacio o desconocido para evitar asumir estado de workflow.
  - No crea agentes, goroutines, locks, scheduler ni eventos durables reales.
```
