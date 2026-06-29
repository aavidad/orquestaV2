# Handoff OPES derivados reales hasta HTML local

Este runbook fija el siguiente frente real de OPES como consumidor de Orquesta:
ejecutar derivados de un `document_plan` contra una instancia OPES temporal y
cerrar como minimo `generate_html_site -> local_html_site`, sin tocar OPES
productivo ni mover reglas de dominio al nucleo.

## Estado actual

Actualizacion 2026-06-29: este handoff queda historico como plan de ejecucion.
La cadena real temporal goal-first quedo cerrada funcionalmente el 2026-06-28
hasta `finalize_temario_package`/`completed_syllabus_package`; ver
`docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`.
Los pendientes vivos ya no son "ejecutar la cadena", sino reejecutar un smoke
desde cero con el enriquecimiento posterior, automatizar continuaciones largas,
medir coste/tiempo y revisar calidad editorial antes de cualquier promocion
manual.

Cerrado:

- `plan_temario -> document_plan` por REST contra OPES temporal.
- Creacion de derivados desde el plan.
- Wrapper fake/offline de derivados por secuencia hasta
  `finalize_temario_package`.
- Contratos de artefactos para investigacion, contenido, visuales, test,
  revisiones, ensamblado, audio, tutor, HTML, manuales y paquete.
- Smoke real temporal goal-first hasta `completed_syllabus_package`, con
  `24/24` jobs OPES completados y receipts de dominio conservados.

Pendiente residual:

- Reejecutar la cadena desde cero con el enriquecimiento posterior de artefactos
  aceptados en `GoalWorkSpecV0`.
- Automatizar continuaciones largas sin observacion manual entre tramos.
- Revisar calidad editorial/visual/audio del material conservado antes de
  cualquier promocion manual.

## Secuencia canonica

Cadena principal de un temario completo:

```text
plan_temario
update_topic_registry
research_exam_precedents
draft_content_block
generate_visual_asset
generate_question_bank
review_legal
review_pedagogical
review_quality
review_codex
review_gemini
review_claude
review_pair_codex_gemini
review_pair_codex_claude
review_pair_gemini_claude
review_director_consolidation
validate_topic
assemble_topic
generate_audio_asset
generate_tutor_assets
generate_learning_games
generate_html_site
generate_help_manual_assets
finalize_temario_package
```

Los jobs `generate_agent_candidate_*`, `vote_agent_candidates_*` y
`select_agent_candidate_director` son opcionales de rework. Se abren solo cuando
una pieza concreta no da nivel, conservando original y candidatos con refs
causales. No deben ralentizar todos los temarios como pasos fijos.

## Preflight

- OPES debe ser temporal o un entorno de pruebas acotado.
- No usar OPES productivo ni colas amplias.
- Declarar `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`.
- Para efectos reales, declarar `ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1` y
  `ORQUESTA_OPES_DERIVATIVES_EXECUTE=1`.
- Usar `ORQUESTA_OPES_BRIDGE_LIMIT=1` al primer pase real.
- Usar scope operativo por `correlation_id`, `topic_id` o cola temporal
  dedicada. Si solo se usa `program_id`, declarar
  `ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1` despues de comprobar que el
  OPES temporal filtra realmente por `program_id`; el valor documental del
  payload no basta para no tocar jobs ajenos.
- No declarar `ORQUESTA_OPES_BRIDGE_ALLOW_UNFILTERED=1` ni
  `ORQUESTA_OPES_BRIDGE_PRODUCTIVE_CONFIRM=1` en este smoke.
- Arrancar Orquesta temporal con Codex Goal
  (`ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`) o, si ya esta levantada fuera
  del entorno del wrapper, confirmar
  `ORQUESTA_OPES_DERIVATIVES_ORQUESTA_GOAL_FIRST_CONFIRMED=1`.
  `app_server_proxy` no es ruta normal para este smoke; solo debe usarse como
  diagnostico aislado con confirmacion explicita del operador.
- No mezclar el reconciliador independiente de paquetes finales:
  `ORQUESTA_OPES_REGISTRY_FINALPKG_ENABLED` debe quedar desactivado.
- No mezclar `ORQUESTA_OPES_BRIDGE_JOB_TYPE` ni
  `ORQUESTA_OPES_BRIDGE_JOB_REF` con `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`.
- Guardar ledger en una ruta del smoke:
  `ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH=<salida>/external-bridge-input-ledger.json`.

Probe de scope contra OPES temporal, sin efectos:

```bash
SCOPE_PROBE_OUTPUT=/tmp/opes-salidas/opes-derivatives-scope-temporal/opes_derivatives_scope_probe.json
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:<puerto-opes-temporal> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=scope-probe \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT="$SCOPE_PROBE_OUTPUT" \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
SMOKE_ID=opes-derivatives-scope-temporal \
scripts/smoke_opes_derivatives_rest.sh
```

El probe consulta `GET /api/jobs` con la secuencia, comprueba que los jobs
devueltos respetan `program_id`, `topic_id` o `correlation_id` y ejecuta una
consulta negativa con un valor imposible. Solo si devuelve
`scope_probe_status=ok` debe declararse
`ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1` para el preflight/ejecucion.
Si no hay jobs pendientes, el probe no puede demostrar el filtro: crear un job
temporal de smoke o usar `topic_id`, `correlation_id` o cola temporal dedicada.
El JSON aceptado por el preflight debe incluir `scope_probe_status=ok`,
`job_type`, `seen > 0`, el `program_id` esperado y negative check de
`program_id`. Ademas queda ligado al OPES temporal consultado mediante
`base_url_hash` y debe declarar `fake_server=false`; no reutilizar JSON de fake,
de otro puerto ni de otro entorno.

Probe de contrato REST de creacion de jobs temporales:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:<puerto-opes-temporal> \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_CONTRACT_PROBE_CREATE=1 \
SMOKE_ID=opes-derivatives-contract-temporal \
scripts/probe_opes_derivatives_rest_contract.sh
```

Este probe usa solo la API publica `POST /api/jobs` de OPES temporal y deja
`opes_derivatives_rest_contract_probe.json` bajo `/tmp/opes-salidas`. Por
defecto usa compatibilidad de transporte: `payload_json.work_kind` conserva el
trabajo canonico y `job_type` viaja con el tipo agregado que acepta el REST OPES
cuando difiere. Para reproducir el modo legado, exportar
`ORQUESTA_OPES_CONTRACT_PROBE_TRANSPORT_COMPAT=0`.

Resultado del 2026-06-28: en modo legado OPES temporal aceptaba 13 de 23 tipos
y rechazaba 10 con `invalid document job`. Con compatibilidad de transporte
desde Orquesta, el probe temporal acepta 23 de 23, el `scope-probe` localiza
`update_topic_registry` via `transport_job_type=review_textual` y el `dry-run`
del bridge selecciona ese `work_kind` canonico sin crear runs. Evidencia:
`docs/runbooks/resultado_probe_opes_derivados_rest_contract_2026-06-28.md`.

Preflight ejecutable sin OPES/Codex:

```bash
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:<puerto-opes-temporal> \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta-temporal> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=preflight-only \
ORQUESTA_OPES_DERIVATIVES_PREFLIGHT_TARGET_MODE=run-until-finalize \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT=/tmp/opes-salidas/opes-derivatives-scope-temporal/opes_derivatives_scope_probe.json \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS=evidence-ref-tts-temporal-001 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
scripts/smoke_opes_derivatives_rest.sh
```

Tambien es valido sustituir `program_id` por
`ORQUESTA_OPES_BRIDGE_CORRELATION_ID`, `ORQUESTA_OPES_BRIDGE_TOPIC_ID` o
`ORQUESTA_OPES_BRIDGE_DEDICATED_TEMPORAL_QUEUE=1`. El preflight solo valida
guardas locales y no consulta OPES. Si el objetivo es `drain-once`,
`run-until-finalize` o `run-until-final`, y el unico scope es `program_id`,
debe existir `ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF` o un
`opes_derivatives_scope_probe.json` valido en `SMOKE_OUT_DIR` generado por
`scope-probe` contra el mismo `ORQUESTA_OPES_BASE_URL`.
`ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_EVIDENCE_REF` solo debe usarse si apunta a
una evidencia durable real generada por la composicion temporal; para operadores
es preferible reutilizar `ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT`.
Si la secuencia incluye `generate_audio_asset`, la composicion temporal debe
declarar
`ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available` y
`ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS` con refs compactas del
runner/capacidad TTS temporal; si no, el preflight y `opes-drain-once` bloquean
la fase de audio con `external_capability_missing` o con evidencia de capacidad
insuficiente.

## Pruebas offline/fake

```bash
bash -n scripts/smoke_opes_derivatives_rest.sh
go test -count=1 ./modulos/orquesta-opes-bridge
go test -count=1 ./cmd/orquesta-server \
  -run 'TestSmokeOPESDerivativesRESTWrapperFakeServer|TestOPESTemarioCycle|TestRunOPESDrainOnceV0|TestOPESBridgeLoop'
```

Dry-run fake:

```bash
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
SMOKE_ID=derivatives-fake-$(date -u +%Y%m%dT%H%M%SZ) \
scripts/smoke_opes_derivatives_rest.sh
```

Run fake hasta paquete:

```bash
ORQUESTA_OPES_DERIVATIVES_FAKE_SERVER=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=program-ref-fake-operario-001 \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=30 \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=0 \
scripts/smoke_opes_derivatives_rest.sh
```

## Ejecucion real temporal

```bash
SMOKE_ID=opes-derivatives-real-$(date -u +%Y%m%dT%H%M%SZ)
SMOKE_OUT_DIR=/tmp/opes-salidas/derivatives-real-$SMOKE_ID
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:<puerto-opes-temporal> \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta-temporal> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-finalize \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_SCOPE_FILTER_CONFIRMED=1 \
ORQUESTA_OPES_BRIDGE_SCOPE_PROBE_OUTPUT=/tmp/opes-salidas/opes-derivatives-scope-temporal/opes_derivatives_scope_probe.json \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_CAPABILITY=available \
ORQUESTA_OPES_BRIDGE_SPEECH_SYNTHESIS_EVIDENCE_REFS=evidence-ref-tts-temporal-001 \
ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=30 \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=5 \
SMOKE_ID="$SMOKE_ID" \
SMOKE_OUT_DIR="$SMOKE_OUT_DIR" \
scripts/smoke_opes_derivatives_rest.sh
```

Si solo se quiere parar al HTML local para revision humana, sobrescribir la
secuencia:

```bash
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=update_topic_registry,research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site
```

## Evidencia exigida

- `metadata.txt` del smoke.
- Resumen JSON de cada tick.
- Ledger de idempotencia.
- `goal_receipts_manifest.json` con `artifact_refs`, `domain_receipt_refs` y
  cierre aceptado por cada run goal-first.
- Tick final vacio despues del ultimo tipo configurado, con toda la secuencia en
  `empty_job_types`.
- `run_ref`, `job_ref`, `work_kind` y `artifact_type` por derivado.
- Receipts OPES de `POST /api/jobs/<job_ref>/artifacts`.
- Para HTML: ruta o ref de `local_html_site`, capturas visuales y manifest.
- Para audios: manifest por tema/apartado, hash/ref de texto y muestra validada
  con escucha o transcripcion.
- Para tests: informe de revision de Codex, Gemini y Claude sobre todas las
  preguntas publicables/importables.

## Criterio de cierre

El smoke real queda cerrado cuando OPES temporal demuestra:

- no hay jobs pendientes de la secuencia despues del final configurado;
- cada derivado aceptado tiene artefacto del tipo esperado;
- los reintentos no duplican artefactos;
- `generate_audio_asset` entrega `audio_asset` segmentado por apartado;
- `generate_tutor_assets` entrega `tutor_bot_package`;
- `generate_learning_games` entrega `learning_games_package`;
- `generate_html_site` entrega `local_html_site` revisable con formato USO/TCAE;
- si se llega al final completo, `finalize_temario_package` entrega
  `completed_syllabus_package`.

## Pendientes futuros

- Arranque reproducible de OPES temporal con datos de prueba, sin depender de
  estado manual de una sesion previa.
- Contrato OPES REST para los 23 `work_kind` canonicos o secuencia equivalente
  publicada: el probe del 2026-06-28 ya acepta los 23 tipos canonicos; queda
  ejecutar la secuencia temporal completa con goals reales hasta
  `finalize_temario_package`.
- Smoke real hasta `finalize_temario_package` con manuales graficos incluidos,
  no solo hasta HTML local.
- Fuente real de cierre causal OPES por puerto: receipts OPES, dedupe,
  validacion/rechazo, replan causal por task y `OperationalClosureSource`
  inyectado en composicion, no una afirmacion textual de que OPES acepto.
- Smokes parciales aun utiles: `draft_content_block` con agente real y
  `generate_visual_asset` con agente real quedan como validaciones acotadas si
  la cadena completa falla o consume demasiada cuota.
- Transporte MCPO/MCP real opt-in para OPES; REST sigue siendo la via funcional
  actual.
- Checklist visual automatizado con capturas del HTML local, tablas,
  infografias, audios y juegos antes de publicar.
- Evidencia real de RAG/tutor por curso y verificacion de que cada paquete
  contiene markdown/fuentes suficientes para reconstruir el indice.
- Conciliar documentos historicos que aun digan que candidatos/votos son pasos
  fijos o que audio va despues del HTML final.
