# Resultado T12: smoke OPES consumer real opt-in

Fecha: 2026-05-24.

## Alcance

Validacion acotada del backlog `T12 opes-consumer-smoke-real-opt-in` dentro de
Orquesta. No se toco OPES productivo ni se dreno una cola real amplia.

La ruta verificada historicamente fue el wrapper opt-in de derivados OPES hasta
`assemble_topic`, usando el fake HTTP local del propio script para demostrar:

- consulta por `execution_mode=external`, `status=pending`, `job_type` y limite;
- secuencia
  `draft_content_block,generate_visual_asset,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic`;
- un run Orquesta por fase;
- supervision de cada `run_ref`;
- mapeo final `assemble_topic -> assembled_topic`;
- parada cuando ya no queda `assemble_topic` pendiente.

Nota 2026-06-02: la secuencia vigente anade
`generate_audio_asset -> audio_asset`. Este resultado del 2026-05-24 no prueba
audio; la revalidacion nueva debe llegar al ultimo tipo configurado.

## Comandos ejecutados

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector

bash -n scripts/smoke_opes_derivatives_rest.sh
bash -n scripts/smoke_opes_derivatives_real.sh
bash -n scripts/smoke_opes_consumer_isolated.sh

ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-assemble \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=8 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
SMOKE_ID=agent-000070 \
scripts/smoke_opes_derivatives_rest.sh
```

Reejecucion focal del agente `000070-000015`:

```bash
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector

bash -n scripts/smoke_opes_derivatives_rest.sh
bash -n scripts/smoke_opes_derivatives_real.sh
bash -n scripts/smoke_opes_consumer_isolated.sh

SMOKE_ID=agent-000070-000015 scripts/smoke_opes_consumer_isolated.sh
```

## Resultado

- Tests obligatorios: pasados.
- Sintaxis de wrappers OPES: pasada.
- Smoke fake `run-until-assemble`: pasado.
- Evidencia local: `/tmp/opes-salidas/derivatives-rest-agent-000070/`.

Revalidaciones acotadas posteriores:

- `agent-ref-assessment-task-autoprogramming-135d8a28b0c3-g01-70d257cc541d`:
  tests obligatorios, sintaxis de wrappers y smoke fake pasados. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-ref-assessment-task-autoprogramming-135d8a28b0c3-g01-70d257cc541d/`.
- `agent-ref-assessment-task-autoprogramming-135d8a28b0c3-g01-783223f0c1c0`:
  tests obligatorios, sintaxis de wrappers y smoke fake pasados. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-ref-assessment-task-autoprogramming-135d8a28b0c3-g01-783223f0c1c0/`.
- `000070-000015`: tests obligatorios, sintaxis de wrappers y smoke fake
  pasados. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-000070-000015/`.
- `agent-ref-task-autoprogramming-7ff37a05dd0e-g01`: tests obligatorios,
  sintaxis de wrappers y smoke fake pasados. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-ref-task-autoprogramming-7ff37a05dd0e-g01/`.
- `agent-ref-task-autoprogramming-aa1d3ec1ddd7-g01`: tests obligatorios,
  sintaxis de wrappers y smoke fake pasados. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-ref-task-autoprogramming-aa1d3ec1ddd7-g01/`.
- `agent-ref-task-autoprogramming-38e1e85b0320-g01`: tests obligatorios
  pasados con `GOCACHE=/tmp/orquesta-gocache`; sintaxis de wrappers pasada. La
  suite del conector OPES usa transporte HTTP in-process para no depender de
  puertos locales cuando el sandbox bloquea listeners.
- `agent-ref-task-autoprogramming-36caec5a22a7-g01`: tests obligatorios
  pasados con `GOCACHE=/tmp/orquesta-gocache`; sintaxis de wrappers pasada. El
  smoke fake aislado no pudo arrancar el HTTP local porque el sandbox devolvio
  `PermissionError: [Errno 1] Operation not permitted` al crear el socket. No se
  ejecuto smoke real OPES por faltar `ORQUESTA_OPES_BASE_URL`,
  `ORQUESTA_BASE_URL`, `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de
  efectos sobre una instancia temporal.
- `agent-ref-task-autoprogramming-371cdd32141a-g01` (2026-05-27): tests
  obligatorios pasados; sintaxis de wrappers pasada. El smoke fake aislado quedo
  bloqueado antes de crear runs porque `go run ./cmd/orquesta-server
  opes-drain-once` no compilo por una referencia fuera del write-set
  (`modulos/orquesta-mcp/workspace_timeline_http_v0.go:68`). No se edito fuera
  del alcance asignado. No se ejecuto smoke real OPES por faltar
  `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
  `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de efectos sobre una
  instancia temporal.
- `agent-ref-task-autoprogramming-5373ad36695c-g01` (2026-05-27): tests
  obligatorios pasados; sintaxis de wrappers pasada; smoke fake aislado
  `run-until-assemble` pasado con
  `SMOKE_ID=agent-ref-task-autoprogramming-5373ad36695c-g01`. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-ref-task-autoprogramming-5373ad36695c-g01/`.
  La secuencia fake creo y superviso una run por fase hasta
  `assemble_topic -> assembled_topic` y cerro con tick final sin pendientes. No
  se ejecuto smoke real OPES por faltar `ORQUESTA_OPES_BASE_URL`,
  `ORQUESTA_BASE_URL`, `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de
  efectos sobre una instancia temporal.
- `agent-ref-task-autoprogramming-51f9a01810a0-g01` (2026-05-27): tests
  obligatorios pasados; sintaxis de wrappers pasada; smoke fake aislado
  `run-until-assemble` pasado con
  `SMOKE_ID=agent-ref-task-autoprogramming-51f9a01810a0-g01`. Evidencia:
  `/tmp/opes-salidas/derivatives-rest-agent-ref-task-autoprogramming-51f9a01810a0-g01/`.
  La secuencia fake creo y superviso una run por fase hasta
  `assemble_topic -> assembled_topic` y cerro con tick final sin pendientes. No
  se ejecuto smoke real OPES por faltar `ORQUESTA_OPES_BASE_URL`,
  `ORQUESTA_BASE_URL`, `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de
  efectos sobre una instancia temporal.

Resumen observado del fake:

- `draft_content_block -> content_block`;
- `generate_visual_asset -> visual_asset`;
- `review_legal -> block_revision`;
- `review_pedagogical -> block_revision`;
- `review_quality -> block_revision`;
- `validate_topic -> block_revision`;
- `assemble_topic -> assembled_topic`;
- tick final sin pendientes tras supervisar `assemble_topic`.

Secuencia vigente posterior: anadir `generate_audio_asset -> audio_asset` antes
del tick final sin pendientes.

## Frontera real opt-in

El smoke real contra OPES temporal no se ejecuto porque el paquete no aportaba
`ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
`ORQUESTA_OPES_TEMPORAL_CONFIRM=1` ni confirmacion explicita de efectos sobre
una instancia temporal. La ruta real permanece acotada por el runbook vigente:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18080 \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-assemble \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=20 \
scripts/smoke_opes_derivatives_real.sh
```

No usar `ORQUESTA_OPES_BRIDGE_JOB_TYPE` ni `ORQUESTA_OPES_BRIDGE_JOB_REF` en
esta ruta de derivados: el wrapper usa secuencia por fases y exige OPES
temporal confirmado.
