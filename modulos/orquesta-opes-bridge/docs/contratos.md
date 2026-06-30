# Contratos

## Entrada

`ExternalJobV0` procede de `GET /api/jobs` de OPES con:

- `execution_mode=external`;
- `status=pending`;
- `payload_json` opaco de dominio.

## Salida

El bridge produce `StartExternalWorkRunRequestV0` para
`/api/v0/external-work/run`.

Reglas:

- `project_ref=opes`;
- `app_ref=opes`;
- `external_work.job_ref` conserva `job.id`;
- `external_work.work_kind` conserva el trabajo canonico OPES: primero
  `payload_json.work_kind` y, solo si falta, `job.type`;
- cuando el REST OPES solo acepta un tipo agregado, `job.type` actua como
  `transport_job_type` y se conserva en `input_fields.transport_job_type`
  junto a `input_fields.job_type`; por ejemplo, `review_codex`,
  `review_pair_*`, `update_topic_registry` y cierres editoriales viajan por
  tipos de transporte como `review_textual`, `generate_tutor_assets` o
  `generate_help_manual_assets` sin perder el `work_kind` canonico;
- `input_fields` copia el `payload_json` y solo normaliza metadatos de contrato
  necesarios para routing, evidencias y compatibilidad de transporte;
- el bridge proyecta el contrato neutral de artefacto como
  `artifact_source_kind`, `artifact_canonicality`, `artifact_stage` y
  `artifact_materialization_target`, para que correctores, QA y generadores
  distingan fuentes canonicas, derivados regenerables, evidencia y artefactos
  materializables sin meter semantica OPES en `orquesta-domain-work`;
- para `summarize_topic`, el bridge hidrata `topic_blocks` desde
  `GET /api/topics/{topic_id}/blocks`; si no puede obtenerlos, no crea el run
  para evitar resumenes pobres;
- todos los jobs OPES reciben `opes_global_editorial_policy_2026_05_18` como
  contexto de dominio priorizado. Esta politica sale del conector OPES y no del
  nucleo: incluye modo tutor completo, tono adulto sin infantilizar, primera
  lectura continua, notas de test separadas, visuales utiles, fuentes oficiales,
  umbral A1 de 20.250-22.500 palabras y regla de correccion incremental: si ya
  existe tema o artefacto previo, se estudia que falla y se modifica lo
  necesario; rehacer completo exige justificacion;
- todos los jobs OPES reciben `opes_temario_agent_rules_2026_06_04` como regla
  operativa de dominio para temarios. Resume y referencia el canon externo
  `/home/alberto/Trabajo/OPES/AGENTS.md`: lectura obligatoria de
  `ENTRADA_UNICA_SIGUIENTE_AGENTE_OPES.md`,
  `GUIA_ESTILO_TEMARIOS_OPES_GLOBAL_2026-05-18.md` y
  `USO_ORQUESTA_TEMARIOS_OPES_GUIA_AGENTES_2026-06-04.md`, direccion editorial,
  reutilizacion antes de rehacer, alcance de busqueda por curso/tema/programa,
  exclusion por defecto de backups, paquetes historicos, snapshots, runtime y
  ficheros de control salvo auditoria global explicita, paralelismo por tema,
  visuales/audios/tests, triple visto bueno e i18n/hexagonal. Los hallazgos
  fuera de alcance son evidencia blanda o nota de revision; no bloquean ni
  descartan trabajo recuperable por si solos. Es politica de la composicion
  OPES, no contrato del nucleo Orquesta;
- todos los jobs OPES reciben tambien
  `opes_html_publication_policy_2026_05_19` para trabajos que generen HTML:
  patron web tipo Tema 11, barra lateral plegable, primera lectura activa por
  defecto, modo tutor, notas de test ocultables con formato unico, supuestos con
  solucion ocultable, visuales locales responsivos, bancos i18n externos y
  validacion de HTML/Markdown/assets antes de publicar;
- todos los jobs OPES reciben `opes_html_topic_template_v1`: el HTML final se
  genera con `modulos/orquesta-opes-bridge/scripts/opes_render_topic_html_v1.py` y la plantilla versionada
  `modulos/orquesta-opes-bridge/templates/opes_html_topic_template_v1.html`;
  esto fija hero, barra de modo, primera lectura, indice plegable, secciones,
  tutor, notas, visuales, supuestos, JavaScript local y validacion de enlaces;
- el paquete final se valida con `modulos/orquesta-opes-bridge/scripts/opes_validate_topic_package_v1.py`
  antes de marcarse listo: rango A1, duplicados largos, HTML offline, JSON,
  SVG y ausencia de rutas internas en entregables finales;
- el banco `banco_preguntas_i18n_es.json` es obligatorio y debe contener al
  menos 50 preguntas por tema, 4 opciones por pregunta, respuesta correcta
  identificable y distractores plausibles que discriminen conocimiento real;
  queda en el mismo paquete del tema, reservado a afiliados, sin banco comun ni
  test de prueba publico en el HTML;
- para `expand_topic_from_summary`, el bridge exige paquete editorial
  multiformato: `tema_grande`, `tema_mediano`, `resumen`,
  `esquema_repaso` y `plan_visuales`;
- para `generate_audio_asset`, el bridge exige `expected_artifact_type=audio_asset`
  y el job debe derivar audio desde `assembled_topic` o refs opacas del paquete
  final, con audio por tema y por apartado/seccion; antes de TTS debe resolver
  `common_topic_ref`, `source_content_ref`, `audio_manifest_ref` y `audio_ref`
  para reutilizar audio comun compatible; cualquier app RTX/GPU queda como
  adaptador OPES externo y no se expone como proveedor, ruta local ni proceso en
  el contrato publico. En pasadas finas de preparacion de audio no debe usar
  `10_tutor_rag/corpus/` ni otros corpus RAG regenerables como fuente primaria
  de rework textual: primero se corrigen HTML final/local, bancos de tests y
  tutor fuente, y el RAG se reconstruye despues desde esas fuentes limpias. El
  bridge no postea audio si faltan `text_public_status=pass` o equivalente,
  si el texto publicable/HTML/RAG/manifiestos declarados por OPES contienen
  mojibake explicito (`mÃ`, `Ã`, `Â`, `�`, `â€`),
  refs vigentes de preparacion (`audio_manifest_ref`/sidecar mas
  `source_content_ref` o hash de texto) y modo de regeneracion selectivo; un
  modo global `all`/`--all` bloquea con `phase_precondition_missing`;
- para `research_exam_precedents`, el bridge exige
  `expected_artifact_type=exam_research_report` y el job debe buscar por
  internet examenes, convocatorias, temarios y pruebas de administraciones
  relacionadas con fuentes publicas verificables. La investigacion empieza por
  `course_id`, `topic_id`, programa oficial, canon OPES y materiales
  reutilizables del temario antes de ampliar alcance; excluye por defecto
  backups, paquetes historicos, snapshots y runtime salvo auditoria global
  explicita;
- para `generate_question_bank`, el bridge exige
  `expected_artifact_type=question_bank` y tests por tema con 4 opciones A-D,
  una correcta exacta, distractores plausibles, explicacion tutor, JSON por
  tema, HTML revisable, metadata, informe, validacion estructural y validacion
  de dificultad/proximidad. El job debe aportar fuente publica suficiente del
  tema (`topic_title`, epigrafe/texto oficial, contenido fuente o bloques de
  tema y plan/secciones); si no existe, el bridge debe bloquear con contexto
  insuficiente antes de consumir un goal editorial. No debe sobrescribir bancos originales. Si el
  adaptador OPES/USO importa en Postgres local, debe exigir backup previo, SQL
  con `DELETE` acotado al banco nuevo y verificacion de conteos;
- para `generate_tutor_assets`, el bridge exige
  `expected_artifact_type=tutor_bot_package` y debe producir tutor/bots del
  temario por refs opacas. Si genera corpus RAG, el corpus es derivado
  regenerable: se construye desde HTML final/local, bancos de tests y tutor
  fuente limpios, se valida contra el temario final aprobado y solo se toca
  directamente si una tarea explicita declara rework de RAG/corpus y no existe
  fuente canonica disponible. El cierre solo acepta la ruta canonica
  `rag/corpus/chunks.jsonl` y `rag/corpus/summary.json` con `rag/manifest.json`
  apuntando a esas rutas; `rag/chunks.jsonl` o `rag/summary.json` sueltos son
  salidas incompatibles con cierre;
- para `generate_html_site`, el bridge exige
  `expected_artifact_type=local_html_site` y debe producir un HTML local
  operativo con logos USO y formato real de curso USO/TCAE promocion interna:
  `index.html`, `html_final/` por tema, assets locales, `audio/manifests/`,
  locales/i18n, audios, infografias, tests permitidos y tutor/bots; no debe
  entregar como salida final una maqueta single-file con estilo propio. Los
  manifiestos de audio esperados se calculan por paginas tematicas
  `html_final/tema_*.html` y `html_ampliado/tema_*.html`; `index.html`,
  portadas y listados no cuentan como manifiestos de tema salvo decision
  explicita de producir audio de indice;
- para `generate_help_manual_assets`, el bridge exige
  `expected_artifact_type=help_manual_package` y debe producir manuales
  graficos de ayuda USO derivados del HTML local: YAML de escenario, capturas
  anotadas, `index.html` canonico, `manual.pdf` exportado desde HTML,
  `manual.md`, capturas `img/`/`raw/` y revision visual. La guia vigente es
  `/home/alberto/Trabajo/USO/web/docs/SCREENSHOT_HELP_MANUALS.md` y la regla de
  marca USO es
  `/home/alberto/Trabajo/OPES/opes-salidas/coordinacion_temarios/REGLA_ARTEFACTOS_USO_BRANDING_2026-06-02.md`;
- para `review_codex`, `review_gemini` y `review_claude`, el bridge exige
  `expected_artifact_type=agent_review_report` y cada job debe emitir una
  revision independiente del curso o artefacto asignado. En bancos publicables
  la revision cubre el 100% de preguntas/opciones/respuestas/distractores y
  explicaciones tutor, por lotes si hace falta. En correctores y QA textual,
  `10_tutor_rag/corpus/` y otros corpus RAG regenerables quedan excluidos por
  defecto como fuente primaria: los hallazgos deben corregirse en HTML
  final/local, tests/bancos o tutor fuente y regenerar RAG al final;
- para `review_pair_codex_gemini`, `review_pair_codex_claude` y
  `review_pair_gemini_claude`, el bridge exige
  `expected_artifact_type=agent_pair_review_report` y debe producir matriz de
  acuerdos, discrepancias, accepted_refs, rework_refs, blocked_refs y
  evidence_refs. No descarta trabajo recuperable por alias, formato reparable o
  texto blando y aplica la misma prioridad de fuentes canonicas frente a corpus
  RAG regenerables;
- para `generate_agent_candidate_codex`, `generate_agent_candidate_gemini` y
  `generate_agent_candidate_claude`, el bridge exige
  `expected_artifact_type=agent_candidate_artifact`. Cada job propone una
  version alternativa acotada para una pieza floja y conserva el original y los
  demas candidatos;
- para `vote_agent_candidates_codex`, `vote_agent_candidates_gemini` y
  `vote_agent_candidates_claude`, el bridge exige
  `expected_artifact_type=agent_candidate_vote_report`. Cada agente compara
  original y candidatos, ordena opciones, propone fusion si procede y deja
  evidencia;
- para `select_agent_candidate_director`, el bridge exige
  `expected_artifact_type=agent_candidate_selection_matrix`. El Director elige
  ganador, fusiona partes aprovechables o pide rework acotado; los candidatos no
  ganadores se conservan como borrador/evidencia;
- para `visual_asset_reuse`, el bridge exige
  `expected_artifact_type=visual_reuse_manifest`. Debe inventariar assets
  visuales comunes/reutilizables, copiar o registrar los validos, conservar los
  rechazados con motivo, declarar `placement_ref`/anclas para el HTML y aportar
  justificacion explicita cuando no apliquen visuales reutilizables;
- para `review_director_consolidation`, el bridge exige
  `expected_artifact_type=director_review_matrix` y debe consolidar revisiones
  independientes y por pares, aceptar artefactos, pedir rework localizado,
  conservar material recuperable o bloquear solo por causa real;
- para `finalize_temario_package`, el bridge exige
  `expected_artifact_type=completed_syllabus_package` y debe entregar el
  temario terminado al 100% para revision local: resumido, ampliado, fuentes,
  investigacion, tests, visuales finales, audios por apartado, tutor/bots, HTML
  local, manuales graficos, manifest, checksums, matriz de revisiones y
  validacion visual. El cierre reconstruye RAG/corpus desde HTML/tests/tutor
  limpios y valida el RAG reconstruido contra el temario final aprobado. Sin
  este artefacto el curso no se marca `ready`;
- el cierre `completed_syllabus_package` declara el required test
  `opes-final-package-manifest-*`: debe existir `manifest_cierre.json` con
  schema `opes_final_package_evidence_manifest.v0` y evidencias obligatorias de
  HTML, RAG, audio, tests, visual y QA final. Si falta una evidencia, el estado
  debe ser `pendiente_continuar` con `followup_refs` causales, no
  `listo_para_revision_operador`;
- el cierre final tambien declara `opes-visual-reuse-manifest-*`: si hay
  comunes o assets visuales reutilizables, debe existir `visual_reuse_manifest`
  con contadores de reutilizados/copied/inserted, refs opacas y placement; si
  `visual_count=0`, exige `visual_requirement_status=not_applicable` o
  `visual_zero_justification_ref`. El cierre goal-first bloquea HTML/final
  `ready`/`html_validado` con `domain_work_opes_visual_reuse_missing` cuando
  declara visuales pendientes sin importarlos;
- la asignacion a Codex, Gemini o Claude no pertenece a OPES ni al bridge. La
  composicion Orquesta puede enrutar `review_gemini` y
  `review_pair_codex_gemini` al adaptador Gemini CLI opt-in, y
  `review_claude`, `review_pair_codex_claude` y
  `review_pair_gemini_claude` al adaptador Claude CLI opt-in. Si esos
  adaptadores no estan activados, el trabajo conserva el contrato y cae por el
  runtime Codex configurado, sin descartar entregas recuperables;
- para `plan_tema`, `plan_temario` y `plan_documento`, el bridge declara
  `expected_artifact_type=document_plan`, `expected_schema=domain_document_plan.v0`
  y partes minimas del plan: `sections`, `deliverables`, `quality_criteria`,
  `review_steps` y visuales cuando aporten valor;
- si un payload declara `subroles_required=6`, el bridge no permite cerrar el
  contrato como producto consolidado sin write-set de producto seguro. Sin
  `topic_dir`, `product_write_set` o `allowed_write_set` relativo y seguro,
  transporta `opes_subroles_materialization_status=blocked_missing_product_write_set`;
  con write-set seguro transporta seis roles, task refs y write-sets hijos
  deterministas, pero mantiene
  `opes_subroles_materialization_status=blocked_workflow_task_store_materialization_required`.
  Ese estado no es materializacion real: el materializado real en
  `WorkflowTaskStore`/wait/review pertenece a la composicion y sigue pendiente
  hasta que exista wiring causal;
- esos trabajos documentales tambien reciben la metodologia editorial OPES como
  campos de dominio: `opes_editorial_workflow`,
  `opes_level_derivation_policy`, `opes_assimilation_method` y
  `opes_quality_requirements`. La regla clave es descendente: si existe maestro
  A1/A2 o A1 equivalente, se planifica primero ese maestro y despues se derivan
  B/C1/C2/AP por resumen, reduccion editorial y adaptacion de nivel; si no hay
  equivalente superior, el plan debe marcar `creacion_directa_nivel`;
- para temas publicables OPES, el plan debe incluir una fase
  `generate_audio_asset` y un deliverable `audio_asset`, de forma que personas
  invidentes o cualquier alumno puedan escuchar el contenido del tema y de sus
  apartados. Tambien debe incluir investigacion externa, banco de tests,
  tutor/bots, HTML local y manuales graficos de ayuda antes de produccion;
- `allowed_write_set` usa por defecto `external/opes/<work_kind>/<job_id>` para
  que cada job tenga una entrega unica y varios agentes del mismo tipo no se
  pisen; si OPES declara explicitamente `allowed_write_set`,
  `product_write_set` o `topic_dir` como ruta relativa segura, el bridge
  preserva ese write-set de producto para que el agente pueda consolidar el
  artefacto canonico;
- cuando el payload declara `subroles_required=6` o el contrato
  `opes.padre-tema-6-subroles.v1` y no hay `allowed_write_set`,
  `product_write_set` ni `topic_dir` seguro, el bridge conserva el fallback
  estrecho de coordinacion pero anade `product_write_set_status=
  missing_for_canonical_consolidation` y criterio de aceptacion de rework. En
  esa situacion el goal puede producir borradores/subentregas, pero no debe
  cerrar como producto canonico consolidado hasta que OPES aporte un write-set
  de producto seguro;
- cuando ese write-set de producto seguro existe, el bridge solo transporta el
  contrato determinista de seis subroles; no marca `ready`. La composicion debe
  crear seis `WorkflowTaskV0` hijos reales con parent refs, waits acotados y
  review causal, o conservar bloqueo operativo explicito;
- si OPES aporta `worktree_ref` o `branch_ref` en `external_refs`, el bridge los
  conserva como refs opacas en `input_fields`/`work_refs` y rechaza valores con
  forma de ruta; no los interpreta como paths, nombres Git ni write-set;
- los artefactos esperados se expresan como input fields, no como decisiones de
  OPES.

## Fronteras

No se leen DB, ficheros internos, rutas locales ni workers de OPES. No se pasan
modelos, sesiones ni cuotas a OPES.
