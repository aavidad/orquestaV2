#!/usr/bin/env bash

smoke_write_project_context() {
  mkdir -p "$PROJECT_DIR/docs" "$RUNTIME_DIR"
  cat >"$PROJECT_DIR/AGENTS.md" <<'EOF'
# AGENTS

Reglas locales:
- resolver solo el paquete de dominio externo recibido;
- no leer OPES por DB, ficheros internos ni rutas locales;
- escribir entregables bajo el write-set indicado;
- crear `agent_ack.json` exactamente en el path indicado por Orquesta;
- mantener contenido en espanol, trazable y sin relleno.
EOF
  cat >"$PROJECT_DIR/README.md" <<'EOF'
# Smoke OPES External Work

Proyecto temporal para validar Orquesta -> agente real -> artefacto OPES.
EOF
  cat >"$PROJECT_DIR/docs/contexto.md" <<'EOF'
# Contexto

OPES es la app propietaria del dominio editorial. Orquesta solo orquesta
agentes y devuelve artefactos por el conector publico OPES.
EOF
}

smoke_write_opes_job_request() {
  local run_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local idem="$4"
  local output="$5"
  jq -n \
    --arg run_ref "$run_ref" \
    --arg topic_id "$topic_id" \
    --arg chapter_id "$chapter_id" \
    --arg idem "$idem" \
    '{
      job_type: "draft_content_block",
      input: {
        program_id: "program-smoke-oposiciones",
        topic_id: $topic_id,
        chapter_id: $chapter_id,
        block_stable_id: "block-smoke-intro",
        level: "A1/A2",
        language_code: "es",
        block_type: "technical",
        target_exam: "oposiciones administracion local",
        target_length: "bloque breve de 600 a 1000 palabras para smoke",
        syllabus_full: "Tema de prueba: principios basicos del procedimiento administrativo y relacion con garantias de la ciudadania.",
        outline: "1. Concepto. 2. Principios. 3. Derechos de la ciudadania. 4. Ejemplo aplicado.",
        chapter_objective: "Explicar de forma clara un bloque introductorio util para estudio.",
        neighbor_context: "No hay bloques previos; el bloque siguiente trataria plazos y recursos.",
        source_policy: "Priorizar fuentes espanolas y europeas. No sustituir normativa aplicable por fuentes internacionales.",
        source_refs: ["BOE-A-2015-10565"]
      },
      max_attempts: 1,
      correlation_id: $run_ref,
      idempotency_key: $idem,
      requested_by: "orquesta-smoke",
      external_refs: {
        run_ref: $run_ref,
        smoke_id: $idem
      }
    }' >"$output"
}

smoke_write_external_work_run_request() {
  local run_ref="$1"
  local topic_id="$2"
  local chapter_id="$3"
  local job_ref="$4"
  local output="$5"
  local safe_job
  local safe_topic
  local safe_chapter
  safe_job="$(smoke_compact_ref "$job_ref")"
  safe_topic="$(smoke_compact_ref "$topic_id")"
  safe_chapter="$(smoke_compact_ref "$chapter_id")"

  jq -n \
    --arg run_ref "$run_ref" \
    --arg topic_id "$topic_id" \
    --arg chapter_id "$chapter_id" \
    --arg job_ref "$job_ref" \
    --arg safe_job "$safe_job" \
    --arg safe_topic "$safe_topic" \
    --arg safe_chapter "$safe_chapter" \
    '{
      request_id: ("req-opes-change-" + $safe_job),
      correlation_id: ("corr-opes-change-" + $safe_job),
      app_change_request: {
        schema_version: "app_change_request.v0",
        request_id: ("req-opes-change-" + $safe_job),
        correlation_id: ("corr-opes-change-" + $safe_job),
        run_ref: $run_ref,
        app_ref: "opes",
        change_ref: ("opes-job-" + $safe_job),
        actor_ref: "opes",
        locale: "es-ES",
        user_intent: "Resolver job OPES draft_content_block como unidad editorial coherente y devolver artefacto OPES.",
        target_area: "domain_work",
        current_state_refs: [
          ("opes-job-" + $safe_job),
          ("opes-topic-" + $safe_topic),
          ("opes-chapter-" + $safe_chapter)
        ],
        acceptance_criteria: [
          "markdown valido",
          "sin placeholders",
          "fuentes verificables cuando aplique",
          "entrega en fichero bajo write set externo"
        ],
        constraints: [
          "no inventar normativa",
          "no leer internals de OPES",
          "devolver resultado por artefacto OPES"
        ],
        external_work: {
          project_ref: "opes",
          job_ref: $job_ref,
          interface_refs: ["opes-rest-v0", "opes-mcp-v0"],
          work_kind: "draft_content_block",
          work_refs: [
            ("opes-topic-" + $safe_topic),
            ("opes-chapter-" + $safe_chapter)
          ],
          input_fields: [
            {name:"topic_id", value:$topic_id},
            {name:"chapter_id", value:$chapter_id},
            {name:"block_type", value:"technical"},
            {name:"language_code", value:"es"},
            {name:"target_exam", value:"oposiciones administracion local"},
            {name:"target_length", value:"bloque breve de 600 a 1000 palabras para smoke"},
            {name:"syllabus_full", value:"Tema de prueba: principios basicos del procedimiento administrativo y relacion con garantias de la ciudadania."},
            {name:"outline", value:"1. Concepto. 2. Principios. 3. Derechos de la ciudadania. 4. Ejemplo aplicado."},
            {name:"chapter_objective", value:"Explicar de forma clara un bloque introductorio util para estudio."},
            {name:"neighbor_context", value:"No hay bloques previos; el bloque siguiente trataria plazos y recursos."},
            {name:"source_policy", value:"Priorizar fuentes espanolas y europeas. No sustituir normativa aplicable por fuentes internacionales."},
            {name:"source_refs", values:["BOE-A-2015-10565"]}
          ]
        }
      }
    }' >"$output"
}

smoke_write_stats_request() {
  local run_ref="$1"
  local job_ref="$2"
  local output="$3"
  local safe_job
  safe_job="$(smoke_compact_ref "$job_ref")"
  jq -n \
    --arg run_ref "$run_ref" \
    --arg job_ref "$job_ref" \
    --arg safe_job "$safe_job" \
    '{
      request_id: ("req-opes-agent-stats-" + $safe_job),
      correlation_id: ("corr-opes-agent-stats-" + $safe_job),
      locale: "es-ES",
      run_ref: $run_ref,
      external_job_ref: $job_ref,
      app_ref: "opes",
      include_process_refs: true,
      include_agent_progress: true,
      include_agent_usage: true
    }' >"$output"
}
