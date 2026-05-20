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
  lectura continua, notas de test separadas, visuales utiles, fuentes oficiales
  y umbral A1 de 20.250-22.500 palabras;
- todos los jobs OPES reciben tambien
  `opes_html_publication_policy_2026_05_19` para trabajos que generen HTML:
  patron web tipo Tema 11, barra lateral plegable, primera lectura activa por
  defecto, modo tutor, notas de test ocultables con formato unico, supuestos con
  solucion ocultable, visuales locales responsivos, bancos i18n externos y
  validacion de HTML/Markdown/assets antes de publicar;
- para `expand_topic_from_summary`, el bridge exige paquete editorial
  multiformato: `tema_grande`, `tema_mediano`, `resumen`,
  `esquema_repaso` y `plan_visuales`;
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
- `allowed_write_set` se limita a `external/opes/<work_kind>/<job_id>` para que
  cada job tenga una entrega unica y varios agentes del mismo tipo no se pisen;
- los artefactos esperados se expresan como input fields, no como decisiones de
  OPES.

## Fronteras

No se leen DB, ficheros internos, rutas locales ni workers de OPES. No se pasan
modelos, sesiones ni cuotas a OPES.
