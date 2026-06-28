# Resultado probe contrato REST OPES derivados - 2026-06-28

## Alcance

Probe real acotado contra una instancia OPES temporal local, con SQLite y
documentos bajo `/tmp`, para comprobar si la API publica `POST /api/jobs`
acepta la secuencia canonica de derivados que Orquesta ya ejecuta en fake/
offline.

No se toco OPES productivo ni colas de agentes OPES vivos.

## Comandos

OPES temporal:

```bash
OPES_ADDR=127.0.0.1:18191 \
OPES_PERSISTENCE_PROVIDER=sqlite \
OPES_PERSISTENCE_DSN=/tmp/orquesta-opes-temporal-contract-20260628T073801Z/opes.db \
OPES_DOCUMENT_ROOT=/tmp/orquesta-opes-temporal-contract-20260628T073801Z/docs \
OPES_AI_PROVIDER=mock \
go run ./cmd/opes-api
```

Probe de contrato REST:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18191 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_CONTRACT_PROBE_CREATE=1 \
SMOKE_ID=contract-probe-20260628T073900Z \
scripts/probe_opes_derivatives_rest_contract.sh
```

Scope probe sobre la subsecuencia aceptada:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18191 \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=scope-probe \
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_html_site,generate_help_manual_assets \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-orquesta-contract-probe-contract-probe-20260628T073900Z \
ORQUESTA_OPES_BRIDGE_CORRELATION_ID=corr-orquesta-contract-probe-contract-probe-20260628T073900Z \
ORQUESTA_OPES_BRIDGE_REQUESTED_BY=orquesta-contract-probe \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
SMOKE_ID=contract-probe-scope-20260628T074000Z \
scripts/smoke_opes_derivatives_rest.sh
```

## Resultado legado

`scripts/probe_opes_derivatives_rest_contract.sh` devuelve:

```text
contract_probe_status=incomplete
accepted_count=13
rejected_count=10
```

Tipos aceptados por `POST /api/jobs` en OPES temporal:

```text
research_exam_precedents
draft_content_block
generate_visual_asset
generate_question_bank
review_legal
review_pedagogical
review_quality
validate_topic
assemble_topic
generate_audio_asset
generate_tutor_assets
generate_html_site
generate_help_manual_assets
```

Tipos rechazados por `POST /api/jobs` con `{"error":"invalid document job"}`:

```text
update_topic_registry
review_codex
review_gemini
review_claude
review_pair_codex_gemini
review_pair_codex_claude
review_pair_gemini_claude
review_director_consolidation
generate_learning_games
finalize_temario_package
```

El `scope-probe` sobre la subsecuencia aceptada devuelve:

```text
scope_probe_status=ok
scope_probe_job_type=research_exam_precedents
scope_probe_seen=1
scope_probe_negative_checks=program_id,correlation_id
```

Evidencia local:

- `/tmp/opes-salidas/opes-contract-probe-contract-probe-20260628T073900Z/opes_derivatives_rest_contract_probe.json`
- `/tmp/opes-salidas/derivatives-rest-contract-probe-scope-20260628T074000Z/opes_derivatives_scope_probe.json`

## Decision legado

El primer probe mostro que el bloqueo del smoke real completo de
derivados/cierre no era el scope ni el loop goal-first de Orquesta. En modo
legado, el contrato REST OPES temporal no aceptaba 10 de los 23 `work_kind`
canonicos que Orquesta necesitaba para cerrar `finalize_temario_package`.

En ese punto habia que hacer una de estas dos cosas antes de reclamar el smoke:

- ampliar el contrato publico OPES para aceptar esos 10 tipos como jobs
  externos con dedupe y artefactos esperados;
- o publicar una secuencia OPES equivalente, soportada por REST, que cubra los
  mismos entregables y cierre causal sin inyectar datos por DB interna ni por
  filesystem compartido.

## Resultado compatibilidad transporte

Despues del ajuste del bridge/connector de Orquesta, el contrato canonico se
mantiene en `payload_json.work_kind` y el REST OPES recibe `job_type` de
transporte solo cuando hace falta. El probe queda en modo compatible por
defecto; el modo legado se reproduce con
`ORQUESTA_OPES_CONTRACT_PROBE_TRANSPORT_COMPAT=0`.

Comando ejecutado contra OPES temporal local:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:18191 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_CONTRACT_PROBE_CREATE=1 \
ORQUESTA_OPES_CONTRACT_PROBE_TRANSPORT_COMPAT=1 \
ORQUESTA_OPES_CONTRACT_PROBE_EXPECT_FULL_SEQUENCE=1 \
SMOKE_ID=transport-compat-20260628T080441Z \
scripts/probe_opes_derivatives_rest_contract.sh
```

Resultado:

```text
contract_probe_status=ok
transport_compat=true
accepted_count=23
rejected_count=0
```

Los tipos canonicos que antes fallaban se crearon con transporte equivalente:

```text
update_topic_registry -> review_textual
review_codex/review_gemini/review_claude -> review_textual
review_pair_* -> review_textual
review_director_consolidation -> review_textual
generate_learning_games -> generate_tutor_assets
finalize_temario_package -> generate_help_manual_assets
```

El `scope-probe` sobre esa cola temporal encontro el primer trabajo canonico
via transporte:

```text
scope_probe_status=ok
scope_probe_job_type=update_topic_registry
scope_probe_transport_job_type=review_textual
scope_probe_seen=1
scope_probe_negative_checks=program_id,topic_id,correlation_id
```

El `dry-run` del bridge tambien selecciono el primer `work_kind` canonico sin
crear runs:

```text
status=completed
selected_job_type=update_topic_registry
transport_job_type=review_textual
seen=1
submitted=0
```

Evidencia local:

- `/tmp/opes-salidas/opes-contract-probe-transport-compat-20260628T080441Z/opes_derivatives_rest_contract_probe.json`
- `/tmp/opes-salidas/derivatives-rest-transport-compat-scope-20260628T080456Z/opes_derivatives_scope_probe.json`
- `/tmp/opes-salidas/derivatives-rest-transport-compat-dry-run-20260628T080913Z/opes_derivatives_rest_dry_run_summary.json`

Decision actual: queda cerrado el bloqueo de creacion/listado REST para la
secuencia canonica desde Orquesta. Sigue pendiente el smoke real con Orquesta
temporal goal-first creando runs, agentes, receipts, TTS y cierre hasta
`finalize_temario_package`; eso no debe reclamarse solo con este probe.
