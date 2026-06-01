# Contratos de artefactos de fase v0

## `RegisterPhaseArtifact`

```text
Nombre: RegisterPhaseArtifact
Tipo: comando
Version: v0
Propietario: orquesta-core-workflow
Consumidores: adaptadores de observacion de agentes y director
Campos:
  - artifact_ref
  - phase_id
  - agent_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Requiere run activo.
  - `phase_id` debe ser una fase soportada, actual y activa.
  - `phase_id` no puede ser `programacion`; las entregas de codigo usan `RegisterDelivery`.
  - `agent_ref` debe existir en `agents` y en `started_agents`.
  - `agent_ref` no puede estar fallido ni con parada confirmada antes de
    registrar el artefacto. Una parada solicitada sin confirmacion aun permite
    artefacto tardio causal.
  - Si `artifact_ref` ya esta reflejado, solo se acepta retry exacto por `CommandEffects`.
  - No emite outbox.
  - No transporta codigo, transcript/prompt crudo ni valores reales de runtime, DB, proveedor, modelo, HOME, OAuth, Docker, tmux, Git o secretos.
Errores:
  - payload_invalido
  - fase_no_soportada
  - transicion_invalida
  - detalle_prohibido
Estado: implementado local en NCW-068.
```

## `PhaseArtifactRegistered`

```text
Nombre: PhaseArtifactRegistered
Tipo: evento
Version: v0
Propietario: orquesta-core-workflow
Consumidores: reducer, replay durable, proyecciones de progreso
Campos:
  - artifact_ref
  - phase_id
  - agent_ref
  - summary
  - evidence_refs opcional
Invariantes:
  - Evento compacto y append-only.
  - Solo proyecta una ref compacta en `phase_artifacts`.
  - La proyeccion conserva `artifact_ref`, `phase_id` y `agent_ref` para validar orfandad sin guardar contenido largo.
  - Requiere que el agente citado exista y haya arrancado antes de aplicar.
  - Rechaza `programacion` para no mezclar documentacion/brainstorming con entregas de codigo.
  - Registra/verifica `CommandEffects` por `(PhaseArtifactRegistered, artifact_ref)`.
Errores:
  - evento_invalido
  - payload_invalido
  - secuencia_invalida
  - detalle_prohibido
Estado: implementado local en NCW-068.
```

## Uso previsto

```text
Caso: director en brainstorming/documentacion
Flujo:
  - RequestCapacity
  - RegisterCapacityDecision
  - RequestAgent
  - RegisterAgentStarted
  - RegisterPhaseArtifact
Resultado:
  - Orquesta conserva evidencia durable del artefacto del director sin tratarlo como entrega de programacion.
  - La siguiente fase puede consultar `phase_artifacts` como evidencia compacta.
```
