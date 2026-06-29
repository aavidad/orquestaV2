# Smoke real OPES-Orquesta: visual_asset

Fecha: 2026-05-13.

## Objetivo

Validar el contrato publico para recursos visuales:

```text
Orquesta /api/v0/domain-work -> conector OPES REST -> OPES /api/jobs
Orquesta /api/v0/domain-work -> conector OPES REST -> OPES /api/jobs/{id}/artifacts
```

La prueba debe crear un `generate_visual_asset`, entregar un
`artifact_type=visual_asset` con SVG autocontenido y comprobar que OPES lo
materializa como bloque `visual_asset`.

## Guardas

- No ejecutar sobre una instancia OPES que este creando un temario real.
- El script exige `ORQUESTA_OPES_VISUAL_SMOKE_CONFIRM=1`.
- El operador debe declarar `ORQUESTA_OPES_TEMPORAL_CONFIRM=1`; loopback no
  basta como prueba de temporalidad.
- Usar una instancia OPES actualizada. Una instancia antigua puede responder
  `invalid document job` ante `generate_visual_asset`.
- Orquesta debe estar arrancada con `ORQUESTA_OPES_BASE_URL` apuntando a esa
  misma instancia OPES.
- El smoke usa solo REST publico; no lee DB, ficheros internos ni rutas de
  OPES.

## Script

```bash
ORQUESTA_OPES_VISUAL_SMOKE_CONFIRM=1 \
ORQUESTA_OPES_TEMPORAL_CONFIRM=1 \
ORQUESTA_BASE_URL=http://127.0.0.1:18787 \
OPES_BASE_URL=http://127.0.0.1:18082 \
SMOKE_ID=opes-visual-001 \
scripts/smoke_opes_visual_asset_real.sh
```

El script:

- comprueba health de Orquesta y OPES;
- crea `topic` y `chapter` reales de smoke;
- crea el job externo `generate_visual_asset` desde Orquesta;
- entrega `visual_asset` con `format=svg`, `caption`, `alt_text`, `body` y
  trazas externas;
- comprueba que OPES materializa exactamente un bloque `visual_asset`;
- conserva evidencias en `SMOKE_OUT_DIR`.

## Resultado Observado

Prueba manual validada el 2026-05-13 contra OPES actualizado en
`http://127.0.0.1:18082`:

```text
smoke_id=visual-20260513T205255Z
topic_id=a5b9b31690c03cafb537c5b12b9fbcc9
chapter_id=40a09f1d0e0965ab41bd5a7c4bc2ee1d
job_ref=7a1a608582ec790fec9fcbbb0252af03
receipt_ref=2304f6fc5ff6de21e6f51f655017b2a7
artifacts_count=1
blocks_count=1
```

Bloque materializado:

```text
type=visual_asset
title=Red en estrella
status=pendiente_revision
```

Evidencias locales:

```text
/tmp/orquesta-opes-visual-smoke/visual-20260513T205255Z/out
```

## Incidencias

Durante la verificacion se detecto una instancia OPES antigua en
`http://127.0.0.1:18080` que rechazaba `generate_visual_asset` con:

```text
invalid document job
```

Ese resultado no invalida Orquesta; indica que OPES debe ejecutar una version
que incluya `job.TypeGenerateVisualAsset` en su contrato de jobs documentales.
