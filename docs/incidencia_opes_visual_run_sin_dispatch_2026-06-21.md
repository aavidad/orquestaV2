# Incidencia OPES Visual: Run Aceptada Sin Dispatch

Fecha: 2026-06-21

## Contexto

- Curso OPES: Oficial de Servicios Múltiples C2.
- Paquete local: `external/opes/generate_html_site/63441a0f360985ebb1a8fae96fd44539`.
- Trabajo OPES: `generate_visual_asset`.
- Job OPES: `014573703539493276e76c502a541f11`.
- Run Orquesta: `run-ref-opes-servicios-multiples-visual-t15-20260621`.
- Correlation: `servicios-multiples-c2-visual-rework-20260621`.
- Idempotency key: `servicios-multiples-c2-visual-rework-20260621-tema-015-seguridad-raster-v1`.

## Síntoma

Orquesta aceptó el run externo y la supervisión respondió `estado=ok`, pero no
arrancó ningún agente para ejecutar el trabajo visual.

Evidencia observada:

- `/api/v0/external-work/run` devolvió `estado=ok`.
- `/api/v0/runs/supervise` devolvió `estado=ok` y proyectó una tarea abierta.
- Las estadísticas quedaron con `tasks_total=1`, `tasks_open=1`,
  `agents_requested=0` y `agents_started=0`.
- El job OPES sigue en `status=pending`, `attempts=0`, sin `locked_by`.
- `/api/v0/server/readiness` indica `ready=true`, pero
  `external_bridge_status=disabled`.

## Resultado Esperado

Si Orquesta acepta un `generate_visual_asset` externo debe ocurrir una de estas
dos cosas:

1. materializar y despachar agente/ejecutor para reclamar el job OPES; o
2. cerrar el run con bloqueo causal explícito si el puente externo está
   deshabilitado, sin dejar el job en espera silenciosa.

## Riesgo OPES

El director OPES cree que el trabajo visual está en cola, pero ningún agente lo
ejecuta. Esto rompe la autonomía en cursos con visuales pendientes porque el
curso queda esperando entregas que nunca llegan.

## Requisitos De Arreglo

- El estado `external_bridge_status=disabled` debe impedir aceptar runs OPES
  como ejecutables o debe activar el puente antes del dispatch.
- Añadir prueba de regresión: crear run OPES externo con job pendiente y
  verificar que termina con `agents_requested>0` o con bloqueo explícito
  `external_bridge_disabled`.
- En trabajos visuales OPES, respetar el contrato del payload: si pide
  raster/WebP/JPG y prohíbe SVG, Orquesta no debe degradarlo a contrato visual
  de SVG ni marcarlo listo por entregar una maqueta.

## Workaround Aplicado En OPES

Para no bloquear la revisión local del curso, Codex integró manualmente una
primera tanda de apoyos visuales trazados y dejó este fallo documentado. No se
tocó el código de Orquesta porque hay otro agente trabajando en el núcleo.
