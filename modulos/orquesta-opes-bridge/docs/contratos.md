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
- `allowed_write_set` se limita a `external/opes/<work_kind>` para que el
  agente produzca una entrega unica;
- los artefactos esperados se expresan como input fields, no como decisiones de
  OPES.

## Fronteras

No se leen DB, ficheros internos, rutas locales ni workers de OPES. No se pasan
modelos, sesiones ni cuotas a OPES.
