# Pruebas

- Mapper de `summarize_topic` a `topic_summary`.
- Mapper de `expand_topic_from_summary` a `topic_expansion_package`.
- Mapper de `plan_tema` a `document_plan` con schema
  `domain_document_plan.v0`, contexto `large`, partes minimas del plan y
  criterios de aceptacion para no redactar el documento final, incluida la
  politica OPES de derivacion descendente desde maestro A1/A2 o A1 cuando
  exista equivalente superior.
- Mapper de `plan_temario` de operadores a `document_plan` con write-set
  `external/opes/plan_temario/<job>`, contexto `large`, schema
  `domain_document_plan.v0`, contrato de plan documental, flujo editorial OPES,
  metodo de asimilacion, requisitos de calidad y regla de modificacion
  incremental cuando ya exista artefacto previo.
- Inyeccion del contrato `opes_html_topic_template_v1`, que obliga a usar
  `modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py` y la plantilla HTML canonica para no
  dejar la web a criterio de cada agente, y
  `modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py` para cerrar los paquetes con las
  mismas comprobaciones mecanicas, incluido banco de 50 preguntas con 4
  opciones y distractores plausibles, siempre por tema y reservado a afiliados.
- Contrato multiformato de expansion: tema grande, tema mediano, resumen,
  esquema de repaso y plan de visuales.
- Mapper de derivados OPES:
  `research_exam_precedents -> exam_research_report`,
  `draft_content_block -> content_block`,
  `generate_visual_asset -> visual_asset`, revisiones y `validate_topic` a
  `block_revision`, `assemble_topic -> assembled_topic` y
  `generate_question_bank -> question_bank`,
  `generate_audio_asset -> audio_asset`,
  `generate_tutor_assets -> tutor_bot_package` y
  `generate_html_site -> local_html_site`,
  `generate_help_manual_assets -> help_manual_package`,
  revisiones independientes `review_codex|review_gemini|review_claude ->
  agent_review_report`, revisiones por pares
  `review_pair_codex_gemini|review_pair_codex_claude|review_pair_gemini_claude
  -> agent_pair_review_report`, `review_director_consolidation ->
  director_review_matrix` y `finalize_temario_package ->
  completed_syllabus_package`.
- Alcance blando de busqueda OPES: `plan_temario` transporta en
  `opes_temario_agent_rules_2026_06_04` la regla de empezar por
  curso/tema/programa/canon/material reutilizable y excluir por defecto backups,
  paquetes historicos, snapshots y runtime salvo auditoria global explicita;
  `research_exam_precedents` incluye los mismos criterios en sus acceptance
  criteria. Los hallazgos fuera de alcance se conservan como evidencia blanda,
  no como veto automatico.
- Routing de proveedores en Orquesta:
  `go test -count=1 ./modulos/orquesta-runtime-claude ./modulos/orquesta-app-codex-stack ./cmd/orquesta-server -run 'TestClaude|TestProviderLaunchSpecResolverV0RuteaReview|TestProviderAwareAckPathResolverV0UsaRuntimeClaude|TestGeminiRuntimeConfig|TestClaudeRuntimeConfig'`.
  Debe demostrar que `review_gemini` se materializa por Gemini cuando esta
  activado, `review_claude` por Claude cuando esta activado, y que los ACK se
  buscan en el runtime del proveedor correspondiente.
- Preservacion de `worktree_ref` y `branch_ref` de autoprogramacion como refs
  opacas, sin usarlas como ruta, rama Git ni componente del write-set.
- Preservacion de payload JSON como `input_fields`.
- Hidratacion de `topic_blocks` para `summarize_topic`.
- Write-set unico por defecto para evitar entregas multiples innecesarias y
  preservacion del write-set de producto cuando el payload declara
  `allowed_write_set`, `product_write_set` o `topic_dir` con ruta relativa
  segura; rutas absolutas, con `..`, drive o separadores inseguros caen al
  fallback `external/opes/<work_kind>/<job_id>`.
- Contrato goal-first para padres OPES con seis subroles sin write-set de
  producto: el `GoalWorkSpec` conserva el fallback estrecho, transporta
  `product_write_set_status=missing_for_canonical_consolidation` y exige rework
  o `pendiente_continuar` en vez de cierre como Markdown canonico consolidado.

Comando:

```sh
go test -count=1 ./modulos/orquesta-opes-bridge
```

## Reconciliacion T12 2026-05-27

Para el alcance T12, los tests focales de bridge/conector y el smoke fake
aislado ya pasaron en intentos cerrados. La prueba obligatoria vigente para
revalidar este modulo junto al conector es:

```sh
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector
```

El smoke real de derivados/cierre sigue bloqueado hasta tener OPES temporal,
Orquesta temporal, `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`,
`ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`, limite bajo y confirmacion de
efectos. Sin esas precondiciones, el resultado correcto es bloqueo verificable,
no nuevo relanzamiento de implementacion.
