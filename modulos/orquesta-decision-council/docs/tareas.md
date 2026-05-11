# Tareas: orquesta-decision-council

## DC-001 cerrada

Objetivo: crear mini-proyecto y contratos de consejo multiagente.

Write-set: README.md, AGENTS.md, docs/*, tipos y tests locales.

Validacion: `go test -count=1 ./modulos/orquesta-decision-council`.

## DC-002 cerrada

Objetivo: planificar rondas de propuesta, critica y voto con quorum de familias.

Write-set: `council_plan_v0.go`, `council_types_v0.go`, tests.

Validacion: tres familias opacas producen propuestas, criticas cruzadas y votos.

## DC-003 cerrada

Objetivo: evaluar votos con regla de no autores, familias, umbral y bloqueos.

Write-set: `council_vote_v0.go`, tests.

Validacion: acepta consenso multi-familia y rechaza bloqueo o quorum insuficiente.

## DC-004 cerrada

Objetivo: impedir deliberacion y votos baratos sin evidencia y conservar source refs en la decision evaluada.

Write-set: contratos/docs locales, validacion de plan, validacion/evaluacion de voto y tests.

Validacion: rechaza plan sin `evidence_refs`, rechaza voto no abstencion sin `evidence_refs` y agrega refs sin transcripts.
