# Decisiones

## Arquitectura

Se usa hexagonal ligera: el modulo define contratos y politica pura, pero no implementa adaptadores.

## Reloj inyectado

`RankRunCandidatesV0` recibe `RunQueueRankingPolicyV0.Now`. Esto mantiene el ranking determinista y testeable.

## Aging secundario

El aging/fairness no se suma a `priority_score` en v0. La prioridad base sigue siendo la primera clave de ordenacion, y el aging solo desempata dentro de la misma prioridad.

## Fairness por grupo secundaria

`fairness_group_ref` tiene semantica ejecutable dentro de la misma prioridad:
una politica con reloj inyectado puede pausar un grupo que ya consumio su ventana
o impulsar un grupo hambriento. Si falta grupo, el ranking deriva uno estable por
app/run y expone `fairness_group_missing`. Esto no reemplaza leases, claims de
write-set ni confirmaciones opt-in para efectos externos.

## Sin superficies externas

No se implementan DB, HTTP ni MCP. Tampoco se toca workflow core ni scheduler interno.

## `updated_at` estable

Cuando prioridad y aging empatan, gana el `updated_at` mas antiguo. Si tambien empata, se conserva el orden de entrada mediante ordenacion estable.

## Alias terminales legacy

La cola trata `completed`, `complete` y `done` como estados no ejecutables. No
son nuevos estados canonicos del contrato, sino compatibilidad defensiva para
runs historicas o adaptadores externos que escribieron esos nombres antes de
normalizar a `closed`, `delivered`, `stopped` o `canceled`.
