# Resultado prueba OPES-Orquesta plan_temario Operario - 2026-05-18

## Alcance

Prueba real acotada de la ruta REST:

```text
OPES temporal -> job externo plan_temario -> Orquesta bridge REST
-> run externa -> supervisor residente -> Codex real xhigh
-> artifact document_plan -> OPES valida, completa job y crea derivados
```

No se uso MCP/MCPO. REST queda operativo como adaptador valido para este
smoke historico. Nota posterior 2026-06-29: el transporte MCP real existe como
opt-in de composicion en `cmd/orquesta-server`; este resultado no lo ejercito
y no debe leerse como estado vigente del frente MCP.

## Entorno

- OPES temporal: `http://127.0.0.1:18084`
- Orquesta temporal: `http://127.0.0.1:18787`
- Raiz temporal: `/tmp/opes-orquesta-operario-e2e-FUZcGH`
- Programa fuente: `/home/alberto/Trabajo/OPES/OPES/administracion-especial/Operario/Operario.txt`
- Codex: `gpt-5.5`, `model_reasoning_effort=xhigh`
- Write-set del agente:
  `external/opes/plan_temario/9784a562f074769a08043707fdd79eb2`

## Resultado

- `program_id`: `1f897b2c119df92d371fea28043d7daf`
- `job_ref`: `9784a562f074769a08043707fdd79eb2`
- `run_ref`: `run-external-work-opes-9784a562f074769a08043707fdd79eb2-opes-job-9784a562f074769a08043707fdd79eb2`
- `agent_ref`: `agent-ref-task-ref-app-change-appchange-140novx`
- `artifact_id`: `576d0a93563c2c40e7a6dc41c74261e8`

Estado final OPES del job: `completed`.

Orquesta registro una entrega Codex con ACK `completed`, valido
`document_plan`, envio `submit_artifact` por `DomainWork` y dejo ledger
idempotente:

```text
idem-domain-work-artifact-opes-9784a562f074769a08043707fdd79eb2-document_plan
```

## Derivados Creados

OPES creo 20 jobs externos pendientes desde el `document_plan`:

- `draft_content_block`: 10
- `generate_visual_asset`: 4
- `review_quality`: 2
- `review_legal`: 1
- `review_pedagogical`: 1
- `validate_topic`: 1
- `assemble_topic`: 1

No se ejecutaron esos derivados en esta prueba. El alcance cerrado es
`plan_temario -> document_plan -> derivados pendientes`.

## Evidencias

- Artefacto local del agente:
  `external/opes/plan_temario/9784a562f074769a08043707fdd79eb2`
- ACK:
  `/tmp/opes-orquesta-operario-e2e-FUZcGH/orquesta_runtime/run-external-work-opes-9784a562f074769a08043707fdd79eb2-opes-job-9784a562f074769a08043707fdd79eb2/agent-ref-task-ref-app-change-appchange-140novx/agent_ack.json`
- Ledger de entrega:
  `/tmp/opes-orquesta-operario-e2e-FUZcGH/orquesta_state/domain-work-artifact-ledger.json`
- DB OPES temporal:
  `/tmp/opes-orquesta-operario-e2e-FUZcGH/opes.db`

## Limites

- Esta prueba acotada uso SQLite temporal para aislar el smoke. La regla
  vigente de OPES queda corregida despues de la prueba: SQLite es solo
  historico/compatibilidad. Las ejecuciones nuevas de Operario y Orquesta deben
  usar Postgres, preferiblemente una base aislada por run si se quiere acotar.
- La run Orquesta quedo activa en fase `revision`; el objetivo OPES quedo
  cerrado porque el artefacto se entrego y OPES completo el job.
- MCP/MCPO no se probo ni se requiere para este camino REST.
- Nota posterior 2026-05-23: la recursion real padre-hijo-nieto del Director
  Operativo ya quedo cubierta por `CODEX-RECURSION-REAL` en la composicion
  Codex. Este documento sigue siendo historico de OPES y no cubre derivados ni
  cierre OPES real.
