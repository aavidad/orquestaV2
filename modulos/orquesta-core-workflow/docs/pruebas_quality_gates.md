# Pruebas de quality gates v0

```text
Caso: record_quality_gate_handler_evento_sin_outbox
Tipo: contract | replay
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run TestRecordQualityGateCommandV0ProjectsGateAndNoOutbox
Evidencia esperada: `RecordQualityGate` produce solo `QualityGateRecorded`, outbox vacio y proyeccion compacta en `quality_gates`.
Ultima ejecucion: 2026-05-07, ok.
Riesgos: El gate no ejecuta rework, bloqueo ni pregunta al director; esos pasos siguen por comandos separados.
```

```text
Caso: record_quality_gate_idempotencia_conflicto
Tipo: idempotency | regression
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run TestRecordQualityGateCommandV0IdempotenteYConflicto
Evidencia esperada: repetir el mismo gate ya reflejado devuelve no-op; reutilizar `gate_ref` con decision o payload distinto devuelve `transicion_invalida`.
Ultima ejecucion: 2026-05-07, ok.
Riesgos: La identidad fuerte depende de `CommandEffects`; no guardar payload completo es intencional.
```

```text
Caso: record_quality_gate_validaciones
Tipo: contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run TestRecordQualityGateCommandV0RejectsIssueRefsMissingForRework
Evidencia esperada: `rework_required`, `blocked` y `ask_director` requieren `issue_refs`; refs vacias, textos largos o detalles prohibidos se rechazan por validacion de payload.
Ultima ejecucion: 2026-05-07, ok.
Riesgos: La validacion no interpreta semantica de calidad; solo protege forma durable y detalles prohibidos.
```

```text
Caso: quality_gate_replay_y_estado
Tipo: replay | contract
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestReplayDurableEventsV0AcceptsQualityGateRecordedV0|TestValidateOrchestrationRunV0RejectsInvalidQualityGateProjection'
Evidencia esperada: replay acepta duplicado exacto de `QualityGateRecorded`, reconstruye `quality_gates` y la validacion del run rechaza proyecciones mal formadas.
Ultima ejecucion: 2026-05-07, ok.
Riesgos: `quality_gates` conserva refs compactas, no el resultado completo del analisis de calidad.
```

```text
Caso: quality_gate_blocked_source_replan
Tipo: contract | invariant
Comando: go test -count=1 ./modulos/orquesta-core-workflow -run 'TestRecordReplanDecisionCommandV0AcceptsBlockedQualityGateSourceInProgramming|TestRecordReplanDecisionCommandV0RejectsMissingOrNonBlockedQualityGateSource'
Evidencia esperada: en programacion, `RecordQualityGate(decision=blocked)` puede alimentar `RecordReplanDecision(source_ref=gate_ref)` y producir `ReplanDecisionRecorded`; una gate inexistente o no bloqueante se rechaza.
Ultima ejecucion: 2026-05-07, ok.
Riesgos: El gate bloqueante solo habilita la decision durable de replan; no ejecuta scheduler, runtime, cierre ni followups.
```
