# Handoff OPES derivados reales hasta HTML local

Este runbook fija el siguiente frente real de OPES como consumidor de Orquesta:
ejecutar derivados de un `document_plan` contra una instancia OPES temporal y
cerrar como minimo `generate_html_site -> local_html_site`, sin tocar OPES
productivo ni mover reglas de dominio al nucleo.

## Estado actual

Cerrado:

- `plan_temario -> document_plan` por REST contra OPES temporal.
- Creacion de derivados desde el plan.
- Wrapper fake/offline de derivados por secuencia hasta
  `finalize_temario_package`.
- Contratos de artefactos para investigacion, contenido, visuales, test,
  revisiones, ensamblado, audio, tutor, HTML, manuales y paquete.

Pendiente real:

- Ejecutar la cadena de derivados sobre OPES temporal con agentes reales.
- Demostrar que cada job crea run, entrega artefacto valido y OPES acepta o
  deduplica por `external_job_ref`.
- Validar que el resultado local contiene HTML revisable con formato USO/TCAE,
  audios, tests, visuales, tutor/bots, manuales y manifest de trazabilidad.

## Secuencia canonica

Cadena principal de un temario completo:

```text
plan_temario
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
- Usar scope por `program_id`, `correlation_id` o cola temporal dedicada.
- No mezclar `ORQUESTA_OPES_BRIDGE_JOB_TYPE` ni
  `ORQUESTA_OPES_BRIDGE_JOB_REF` con `ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE`.
- Guardar ledger en una ruta del smoke:
  `ORQUESTA_OPES_BRIDGE_INPUT_LEDGER_PATH=<salida>/external-bridge-input-ledger.json`.

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
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-assemble \
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
ORQUESTA_OPES_BASE_URL=http://127.0.0.1:<puerto-opes-temporal> \
ORQUESTA_BASE_URL=http://127.0.0.1:<puerto-orquesta-temporal> \
ORQUESTA_OPES_DERIVATIVES_REST_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_OPES_DERIVATIVES_EXECUTE=1 \
ORQUESTA_OPES_DERIVATIVES_SMOKE_MODE=run-until-assemble \
ORQUESTA_OPES_BRIDGE_PROGRAM_ID=<program_id-temporal> \
ORQUESTA_OPES_BRIDGE_LIMIT=1 \
ORQUESTA_OPES_BRIDGE_MAX_TICKS=30 \
ORQUESTA_OPES_DERIVATIVES_TICK_SLEEP_SECONDS=5 \
SMOKE_ID=opes-derivatives-real-$(date -u +%Y%m%dT%H%M%SZ) \
SMOKE_OUT_DIR=/tmp/opes-salidas/derivatives-real-$SMOKE_ID \
scripts/smoke_opes_derivatives_rest.sh
```

Si solo se quiere parar al HTML local para revision humana, sobrescribir la
secuencia:

```bash
ORQUESTA_OPES_BRIDGE_JOB_TYPE_SEQUENCE=research_exam_precedents,draft_content_block,generate_visual_asset,generate_question_bank,review_legal,review_pedagogical,review_quality,review_codex,review_gemini,review_claude,review_pair_codex_gemini,review_pair_codex_claude,review_pair_gemini_claude,review_director_consolidation,validate_topic,assemble_topic,generate_audio_asset,generate_tutor_assets,generate_learning_games,generate_html_site
```

## Evidencia exigida

- `metadata.txt` del smoke.
- Resumen JSON de cada tick.
- Ledger de idempotencia.
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
- Smoke real hasta `finalize_temario_package` con manuales graficos incluidos,
  no solo hasta HTML local.
- Fuente real de cierre causal OPES por puerto: receipts OPES, dedupe,
  validacion/rechazo, replan causal por task y `OperationalClosureSource`
  inyectado en composicion, no una afirmacion textual de que OPES acepto.
- Preflight ejecutable que demuestre que OPES filtra realmente por
  `program_id` o, si no lo hace, obligue a usar cola temporal dedicada o
  `job_ref` exactos. `program_id` documental no basta para no tocar jobs ajenos.
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
