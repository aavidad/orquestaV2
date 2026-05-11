# Decisiones locales

## 2026-05-06: builder fino

Decision: el modulo importa solo contratos publicos de workflow, concurrencia, director y scheduler.

Motivo: el scheduler no debe inventar candidates, pero el builder tampoco debe conocer estado durable ni adaptadores. La frontera v0 es un DTO compacto con comandos, refs e idempotency keys explicitos.

Impacto: la validacion se apoya en constructores publicos existentes y no duplica reglas de workflow.

## 2026-05-06: plan compacto sin derivacion de refs

Decision: `BuildSchedulableWorkCandidatesFromPlanV0` acepta microtareas ya enriquecidas con refs de candidate, tarea, claims, comandos, idempotency, capacidad, agente y evidencias.

Motivo: el modulo puede conectar backlog/planificacion con scheduler sin conocer el origen durable del plan ni crear identificadores.

Impacto: la conversion preserva orden y delega cada microtarea en `BuildSchedulableWorkCandidateV0`; cualquier dato ausente o scope invalido aborta la conversion completa.

## 2026-05-07: consejo por rondas con refs neutrales

Decision: `BuildSchedulableWorkCandidatesFromCouncilPlanV0` convierte una sola ronda del consejo en candidates y deriva refs neutrales por ordinal.

Motivo: el core prohibe detalles de proveedor y no debe recibir nombres de familias ni agentes reales. El director conserva la relacion con la asignacion del consejo fuera del payload durable.

Impacto: Codex/Gemini/Claude pueden participar como familias en `orquesta-decision-council`, pero al workflow solo cruzan refs neutrales compactas como `c:p:001`, `c:c:001` o `c:v:001`.

## 2026-05-10: equipo neutral por complejidad

Decision: `BuildDecisionCouncilTeamPlanFromComplexityV0` deriva quorum, capacidad y slots neutrales desde una complejidad compacta.

Motivo: el director necesita un punto local y puro para pasar de una senal pequena de complejidad a un plan multiagente sin acoplarse a infraestructura ni cuentas reales.

Impacto: `low`, `medium`, `high` y `xhigh` producen planes de 2, 3, 4 y 5 slots respectivamente; el plan final se valida con el builder publico del consejo.
