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
- `external_work.work_kind` conserva `job.type`;
- `input_fields` copia el `payload_json` sin interpretar dominio;
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
  final, con audio por tema y por apartado/seccion; cualquier app RTX/GPU queda
  como adaptador OPES externo y no se expone como proveedor, ruta local ni
  proceso en el contrato publico;
- para `research_exam_precedents`, el bridge exige
  `expected_artifact_type=exam_research_report` y el job debe buscar por
  internet examenes, convocatorias, temarios y pruebas de administraciones
  relacionadas con fuentes publicas verificables;
- para `generate_question_bank`, el bridge exige
  `expected_artifact_type=question_bank` y tests por tema con respuesta,
  distractores y explicacion tutor;
- para `generate_tutor_assets`, el bridge exige
  `expected_artifact_type=tutor_bot_package` y debe producir tutor/bots del
  temario por refs opacas;
- para `generate_html_site`, el bridge exige
  `expected_artifact_type=local_html_site` y debe producir un HTML local
  operativo con logos USO, aspecto USO/TCAE promocion interna, assets locales,
  audios, infografias, tests permitidos y tutor/bots;
- para `plan_tema`, `plan_temario` y `plan_documento`, el bridge declara
  `expected_artifact_type=document_plan`, `expected_schema=domain_document_plan.v0`
  y partes minimas del plan: `sections`, `deliverables`, `quality_criteria`,
  `review_steps` y visuales cuando aporten valor;
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
  tutor/bots y HTML local antes de produccion;
- `allowed_write_set` se limita a `external/opes/<work_kind>/<job_id>` para que
  cada job tenga una entrega unica y varios agentes del mismo tipo no se pisen;
- si OPES aporta `worktree_ref` o `branch_ref` en `external_refs`, el bridge los
  conserva como refs opacas en `input_fields`/`work_refs` y rechaza valores con
  forma de ruta; no los interpreta como paths, nombres Git ni write-set;
- los artefactos esperados se expresan como input fields, no como decisiones de
  OPES.

## Fronteras

No se leen DB, ficheros internos, rutas locales ni workers de OPES. No se pasan
modelos, sesiones ni cuotas a OPES.
