# Pruebas

Comando focal del conector:

```sh
go test -count=1 ./modulos/orquesta-opes-connector
```

Comando obligatorio T12 para revalidar frontera bridge/conector:

```sh
go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector
```

Evidencia reconciliada:

- los intentos T12 `agent-ref-task-autoprogramming-5373ad36695c-g01` y
  `agent-ref-task-autoprogramming-51f9a01810a0-g01` pasaron los tests
  obligatorios;
- el fake `run-until-assemble` paso historicamente hasta
  `assemble_topic -> assembled_topic`; la revalidacion vigente debe cubrir
  tambien `research_exam_precedents -> exam_research_report`,
  `generate_question_bank -> question_bank`,
  `generate_audio_asset -> audio_asset`,
  `generate_tutor_assets -> tutor_bot_package` y
  `generate_html_site -> local_html_site`;
- el DTO de job transporta senales publicas opcionales de proveedor audio
  (`provider_timeout`, `running_no_recent_progress`, `provider_status`,
  `provider_reason`) para que el bridge proyecte bloqueos de jobs ya lanzados
  sin leer internals de OPES;
- el cierre funcional real de derivados/cierre OPES quedo documentado el
  2026-06-28 en
  `docs/runbooks/resultado_smoke_opes_derivados_goal_first_real_2026-06-28.md`:
  flujo `goal_first` con `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`, OPES
  temporal, Orquesta temporal, receipts de dominio, cierre aceptado y
  `completed_syllabus_package`.

Revalidar con efectos solo si hay:

- OPES temporal y Orquesta temporal confirmadas;
- scope duro por `job_ref`, `program_id`, `topic_id`, `correlation_id` o cola
  temporal dedicada;
- `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`, confirmacion explicita de efectos y
  `ORQUESTA_CODEX_GOAL_BACKEND=app_server_tmux`;
- cuota/modelo confirmados.

Sin esas precondiciones, la accion correcta es registrar bloqueo externo de
revalidacion y conservar las pruebas focales/offline; no reabrir T12 como fallo
vigente del conector.
