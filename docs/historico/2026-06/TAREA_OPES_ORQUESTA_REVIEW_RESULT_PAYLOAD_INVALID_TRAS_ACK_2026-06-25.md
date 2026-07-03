# TAREA OPES - Orquesta falla al cerrar review tras ACK válido

Fecha: 2026-06-25.

## Contexto

Durante el cierre del temario `Oficial de Servicios Múltiples` C2 se lanzó una
run focal de RAG completo:

`run-external-work-servicios-multiples-c2-rag-full-20260625`.

El agente Codex generado por Orquesta ejecutó el trabajo, escribió artefactos de
producto y dejó `agent_ack.json` con `status: completed`.

## Evidencia

Artefactos creados correctamente:

- `rag/corpus/chunks.jsonl`
- `rag/corpus/summary.json`
- `rag/manifest.json`
- `10_tutor_rag/corpus/chunks.jsonl`
- `10_tutor_rag/corpus/summary.json`
- `10_tutor_rag/manifest.json`
- `09_validacion/informe_rag_full_20260625.md`
- `09_validacion/informe_rag_full_20260625.json`
- `00_control/ACKS/rag_full_20260625_ack.md`

Validación del producto:

- `topic_count=17`.
- `chunk_count=381`.
- variantes `html_final` y `html_ampliado`.
- temas `tema_01` a `tema_17`.
- `rag/corpus/chunks.jsonl` y `10_tutor_rag/corpus/chunks.jsonl` idénticos.
- smoke retrieval pasado para tema común, oficio manual y electricidad/alumbrado.

Pero `POST /api/v0/runs/supervise` devolvió:

```text
estado=error
stop_reason=runtime_error
last.status=failed
diagnostics:
director_supervised_burst_step_input:
field=step_input_builder:
orquestacoreworkflow.ReviewResultErrorV0:
payload_invalido: payload=<redacted>
```

## Problema

Orquesta no debe marcar como fallida una run cuyo agente:

- completó el trabajo;
- dejó `agent_ack.json` válido;
- dejó recibos de tests obligatorios pasados;
- produjo los artefactos esperados.

Si el payload de review es inválido, el error debe aislarse como fallo del
supervisor/reviewer, no como fallo del trabajo de dominio. La run debe quedar en
un estado recuperable como `completed_with_review_error` o debe reintentar la
review con un payload mínimo derivado de `agent_ack.json`.

## Tarea Técnica

1. Añadir validación previa del payload `ReviewResultErrorV0` antes de llamar al
   cierre de workflow.
2. Si el payload de review es inválido, registrar incidencia del reviewer y
   conservar el estado de la entrega de dominio como completado.
3. Permitir cierre causal desde `agent_ack.json` + receipts cuando los tests
   obligatorios están `passed`.
4. Añadir test de regresión con una run OPES cuyo agente deja ACK válido pero la
   review genera payload inválido.
5. Exponer en `/api/v0/runs/supervise` una señal distinguible:
   `domain_work_completed_review_failed`, no `run_supervisor_execute_error`
   genérico.

## Impacto OPES

Sin esta corrección, OPES produce artefactos válidos pero Orquesta informa fallo
global. El director humano tiene que inspeccionar `agent_ack.json` y validar a
mano, lo que rompe la autonomía real de cierre.
