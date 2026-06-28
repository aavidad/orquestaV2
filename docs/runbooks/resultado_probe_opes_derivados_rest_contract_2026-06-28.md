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

## Resultado

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

## Decision

El bloqueo actual del smoke real completo de derivados/cierre no es el scope ni
el loop goal-first de Orquesta. El contrato REST OPES temporal actual no acepta
10 de los 23 `work_kind` canonicos que Orquesta necesita para cerrar
`finalize_temario_package`.

Para cerrar el 100% real hay que hacer una de estas dos cosas antes de reclamar
el smoke:

- ampliar el contrato publico OPES para aceptar esos 10 tipos como jobs
  externos con dedupe y artefactos esperados;
- o publicar una secuencia OPES equivalente, soportada por REST, que cubra los
  mismos entregables y cierre causal sin inyectar datos por DB interna ni por
  filesystem compartido.
