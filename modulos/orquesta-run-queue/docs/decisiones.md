# Decisiones

## Arquitectura

Se usa hexagonal ligera: el modulo define contratos y politica pura, pero no implementa adaptadores.

## Reloj inyectado

`RankRunCandidatesV0` recibe `RunQueueRankingPolicyV0.Now`. Esto mantiene el ranking determinista y testeable.

## Aging secundario

El aging/fairness no se suma a `priority_score` en v0. La prioridad base sigue siendo la primera clave de ordenacion, y el aging solo desempata dentro de la misma prioridad.

## Sin superficies externas

No se implementan DB, HTTP ni MCP. Tampoco se toca workflow core ni scheduler interno.

## `updated_at` estable

Cuando prioridad y aging empatan, gana el `updated_at` mas antiguo. Si tambien empata, se conserva el orden de entrada mediante ordenacion estable.
