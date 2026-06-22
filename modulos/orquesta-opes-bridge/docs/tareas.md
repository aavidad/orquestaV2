# Tareas

## T12 smoke OPES real opt-in

Estado: bloqueado verificable 2026-05-27 para ejecucion real contra OPES
temporal.

Evidencia ya cerrada:

- `go test -count=1 ./modulos/orquesta-opes-bridge ./modulos/orquesta-opes-connector`
  paso en los intentos `agent-ref-task-autoprogramming-5373ad36695c-g01` y
  `agent-ref-task-autoprogramming-51f9a01810a0-g01`.
- El smoke fake aislado `run-until-assemble` paso y recorrio
  `draft_content_block`, `generate_visual_asset`, revisiones, `validate_topic`,
  `assemble_topic -> assembled_topic` y `generate_audio_asset -> audio_asset`.
  La secuencia vigente posterior amplia el cierre de temario con
  `research_exam_precedents -> exam_research_report`,
  `generate_question_bank -> question_bank`,
  `generate_tutor_assets -> tutor_bot_package` y
  `generate_html_site -> local_html_site`,
  `generate_help_manual_assets -> help_manual_package`,
  revisiones independientes y por pares hasta `review_director_consolidation`
  y cierre `finalize_temario_package -> completed_syllabus_package`.

Bloqueo real:

- falta OPES temporal vivo;
- falta servidor Orquesta temporal;
- faltan `ORQUESTA_OPES_BASE_URL`, `ORQUESTA_BASE_URL`,
  `ORQUESTA_OPES_TEMPORAL_CONFIRM=1` y confirmacion de efectos;
- falta cuota/modelo confirmado para ejecucion con agente real.

No relanzar otra implementacion padre para T12 sin esas precondiciones. La
reapertura debe ejecutar el runbook con limite bajo, secuencia de derivados y
sin `JOB_TYPE`/`JOB_REF` manual en la ruta de derivados.

## T13 mínimos editoriales OPES como required tests

Estado: hecho local.

Contexto: OPES detectó un temario `operario-tractorista-grupo-5` demasiado
pequeño tras pasar por Orquesta como entrega estructural aceptada.

Trabajo aplicado:

- `OPESRequiredTestPolicyV0` añade required tests específicos para paquetes
  finales: `opes-extension-minima-nivel-*` y
  `opes-derivacion-comunes-maestro-*`;
- la política editorial inyectada al agente incluye mínimos por nivel:
  A1 20.250, A2 14.400, B 10.800, C1 7.200, C2 4.500 y AP 3.150 palabras en
  ampliado publicable;
- el contrato de `finalize_temario_package` exige informe de extensión y matriz
  de comunes o no aplicabilidad explícita.

Validacion:

- `go test -count=1 ./modulos/orquesta-opes-bridge -run 'TestOPESRequiredTestPolicyV0FinalTemarioExigeMinimosYComunes'`.
