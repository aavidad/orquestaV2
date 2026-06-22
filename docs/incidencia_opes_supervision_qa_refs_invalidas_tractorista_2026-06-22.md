# Incidencia OPES Supervision QA Refs Invalidas Tractorista

Fecha: 2026-06-22.

## Contexto

Run OPES:
`run-opes-tractorista-qa-texto-tests-20260622`.

Objetivo: QA textual y de tests por tema para `operario-tractorista-grupo-5`.

## Sintomas

1. La run genero 7 `agent_ack.json` y no quedaban procesos vivos.
2. `autoprogramming/status` devolvio `state=hung`, `completion_percentage=1`,
   `tasks_open=10`, `agents_in_flight=0`, `agents_stalled=6` y
   `agents_stuck=6`.
3. `POST /api/v0/autoprogramming/supervise` devolvio:
   `estado=error`, `stop_reason=runtime_error`.
4. Diagnostico publico:
   `director_supervised_burst_step: field=step: paso del ciclo fallo:
   issue=director_runner_cycle_invalido: step.runner.evidence_refs: refs invalidas`.

## Impacto

- Orquesta no reconcilia entregas ACK validas cuando la run queda en estado
  perdido/hung por contabilidad previa de agentes.
- El supervisor no abre rework ni cierra tareas: falla por validacion interna
  de evidencias.
- El director humano debe parar esa run y lanzar continuacion para los temas
  pendientes, aunque haya trabajo util entregado.

## Evidencia

- Respuesta de supervision:
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/produccion_tractorista_2026-06-22/responses/supervise_qa_after_7_acks_hung.json`.
- Runtime:
  `/tmp/orquesta-opes-tractorista-bases-20260622/runtime/run-opes-tractorista-qa-texto-tests-20260622`.

## Tareas Tecnicas

- `QA-REFS-TASK-001`: normalizar o filtrar `evidence_refs` generadas por el
  supervisor antes de pasarlas al runner; ninguna evidencia invalida debe romper
  la reconciliacion de ACKs.
- `QA-REFS-TASK-002`: si hay `agent_ack.json` validos y no hay procesos vivos,
  el supervisor debe reconciliar entregas aunque agentes previos figuren
  `lost/stalled`.
- `QA-REFS-TASK-003`: el error publico debe incluir la evidencia exacta invalida
  para depurar sin abrir stores internos.

## Criterio De Cierre

Un smoke debe reproducir una run con agentes `lost` pero ACKs presentes y
demostrar que `autoprogramming/supervise` cierra o replanifica sin
`runtime_error`.
