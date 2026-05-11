# Tareas

## Hecho v0

- Crear microproyecto `orquesta-run-queue`.
- Definir `RunQueueReaderPortV0`.
- Definir `RunSchedulingCandidateV0` con `priority_score` y `updated_at`.
- Implementar `RankRunCandidatesV0` puro y determinista.
- Filtrar estados no ejecutables.
- Cubrir ranking, aging/fairness y estabilidad con tests.

## Pendiente futuro

- Adaptadores reales de lectura fuera de este modulo.
- Integracion con scheduler externo sin mover logica a este paquete.
- Fairness por grupo/app si el contrato v1 lo requiere.
