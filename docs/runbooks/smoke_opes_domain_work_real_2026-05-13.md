# Smoke real OPES-Orquesta: domain_work

Fecha: 2026-05-13.

## Objetivo

Validar el puente real:

```text
Orquesta /api/v0/domain-work -> conector OPES REST -> OPES /api/jobs y /api/jobs/{id}/artifacts
```

La prueba debe materializar un `content_block` vivo en OPES. No vale usar
`topic_id` ni `chapter_id` inventados.

## Arranque usado

OPES:

```bash
OPES_ADDR=127.0.0.1:18082 \
OPES_PERSISTENCE_PROVIDER=sqlite \
OPES_PERSISTENCE_DSN=/tmp/opes-orquesta-smoke.db \
OPES_DOCUMENT_ROOT=/tmp/opes-orquesta-docs \
OPES_AI_PROVIDER=mock \
go run ./cmd/opes-api
```

Orquesta:

```bash
ORQUESTA_SERVER_ADDR=127.0.0.1:18787 \
ORQUESTA_SERVER_STATE_DIR=/tmp/orquesta-opes-smoke-server \
ORQUESTA_CODEX_PROJECT_WORKDIR=/home/alberto/Trabajo/orquesta \
ORQUESTA_CODEX_RUNTIME_WORKDIR=/home/alberto/Trabajo/orquesta/.orquesta-smoke-work \
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18082 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_CODEX_MODEL=gpt-5.5 \
ORQUESTA_CODEX_REASONING_EFFORT=xhigh \
go run ./cmd/orquesta-server run
```

Nota: `ORQUESTA_CODEX_RUNTIME_WORKDIR` debe vivir dentro del
`ORQUESTA_CODEX_PROJECT_WORKDIR` cuando el sandbox es `workspace-write`.

## Script permanente

El smoke queda automatizado en:

```bash
scripts/smoke_opes_domain_work_real.sh
```

Uso:

```bash
ORQUESTA_OPES_DOMAIN_SMOKE_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_BASE_URL=http://127.0.0.1:18787 \
OPES_BASE_URL=http://127.0.0.1:18082 \
SMOKE_ID=opes-real-001 \
SMOKE_OUT_DIR=/tmp/orquesta-opes-smoke-results/opes-real-001 \
scripts/smoke_opes_domain_work_real.sh
```

`127.0.0.1` no sustituye la confirmacion temporal: no ejecutar contra OPES
productivo ni contra una instancia con temarios reales en curso.

El script:

- comprueba health de OPES y Orquesta;
- crea un topic real por `POST /api/topics`;
- crea un chapter real por `POST /api/topics/{id}/chapters`;
- crea job OPES externo desde Orquesta con `action=create_job`;
- entrega artefacto `content_block` desde Orquesta con `action=submit_artifact`;
- comprueba que OPES materializa exactamente un bloque;
- repite job y artefacto con la misma `idempotency_key`;
- comprueba que el replay no crea un segundo bloque.

Guarda operativa: desde el 2026-05-13 el script exige
`ORQUESTA_OPES_DOMAIN_SMOKE_CONFIRM=1` para evitar crear topics, chapters y
jobs en una instancia OPES que este generando un temario real.

## Resultado observado

Smoke manual ejecutado correctamente antes de fijar el script permanente.

```text
topic_id=b15694af111aaa68ca4546463045156a
chapter_id=568ccc0654972a186b0242fa6cf02231
job_ref=67d1e0dc271740357256fba88d993dd6
receipt_ref=2c9e9b6024f39baa19f68ce48965c329
block_id=95c1913b8497fdf3fd28f588316a010c
blocks_after_replay=1
```

Bloque materializado en OPES:

```text
title=Bloque de prueba Orquesta
type=technical
status=pendiente_revision
language_code=es
```

Markdown materializado:

```markdown
# Bloque de prueba Orquesta

Contenido de prueba en markdown producido mediante el puente Orquesta -> OPES.
```

## Resultado automatizado

El script permanente tambien fue ejecutado correctamente:

```text
smoke_id=opes-real-script-20260513T002
topic_id=969288a55c8fd19b33265a950b2128ba
chapter_id=1d42cb549bc016270347992eb7743b2f
job_ref=268a768f2cbf1afbd1552136c5343917
receipt_ref=7978f5aa2dab99e9e7a686061b5ae7f0
block_id=13a22bed440a877c94452d953a8c41e3
block_count_after_replay=1
output_dir=/tmp/orquesta-opes-smoke-results/opes-real-script-20260513T002
```

## Criterios cerrados

- OPES acepta `draft_content_block` creado desde Orquesta.
- OPES materializa `content_block` con topic/chapter reales.
- Orquesta conserva `job_ref` como evidencia `opes-job-ref-*`.
- Orquesta conserva `receipt_ref` como evidencia `opes-artifact-ref-*`.
- El replay idempotente no duplica bloque.
- No se accede a DB ni ficheros internos de OPES desde Orquesta.
- OPES no recibe instrucciones de agente/modelo/sesion/runtime.

## Artefactos que no se deben borrar

La base temporal de esta ejecucion queda en:

```text
/tmp/opes-orquesta-smoke.db
```

Los documentos y runbooks creados para esta prueba no deben eliminarse; sirven
como evidencia de integracion y como punto de partida para el siguiente corte.

## Siguiente corte

Integrar un job OPES real en el flujo de agentes:

```text
job OPES externo -> microtareas Orquesta -> agentes -> entrega artefacto OPES -> RegisterDelivery
```
