# Incidencia OPES Assessment Replan No Reintenta Followup Terminal Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-rework-tests-004-006-008-20260622`.

Tras corregir la reconciliacion de launch, Orquesta detecto `g02` como perdido
y genero un replacement. Ese replacement fallo por un lock de arranque Codex
obsoleto. Al supervisar de nuevo, el plan quedo en:

```text
operational-director-plan-state:blocked
wait-subagents-terminal-without-delivery
```

## Sintoma

`AssessmentReplanSourceV0` no generaba un nuevo replacement para la misma tarea
si ya existia un followup terminal fallido. Esa proteccion evitaba bucles, pero
tambien impedia recuperarse de fallos transitorios ya corregidos.

## Impacto

- Una tarea OPES quedaba `stalled` aunque no habia agente vivo ni outbox util.
- El plan operativo esperaba una entrega que no podia llegar.
- La autonomia quedaba rota y exigia intervencion manual.

## Arreglo Aplicado

`AssessmentReplanSourceV0` permite un nuevo replacement si el followup previo
ya esta terminal y no bloquea la tarea. Para evitar bucles indefinidos, se corta
tras `3` followups terminales fallidos para la misma tarea.

Validacion focal:

```bash
go test -count=1 ./modulos/orquesta-app-codex-stack -run 'TestAssessmentReplanSourceV0(ReintentaSiFollowupFalloTerminal|CortaBucleTrasTresFollowupsFallidos|NoEncadenaReemplazosParaMismaTarea)'
```

## Criterio De Cierre

Si una tarea tiene un replacement vivo o pendiente, no se duplica. Si ese
replacement ya termino sin entrega, Orquesta genera otro replacement acotado.
Tras tres fallos terminales de followup para la misma tarea, corta el bucle y
mantiene la atencion operativa.
