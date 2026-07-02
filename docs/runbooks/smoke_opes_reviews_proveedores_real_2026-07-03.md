# Smoke OPES revisiones por proveedores reales

Estado: runbook operativo acotado para `scripts/smoke_opes_reviews_providers_real.sh`.

Este smoke comprueba que Orquesta puede despachar revisiones OPES a proveedores
configurados como adaptadores (`codex`, `gemini`, `claude`) y que las entregas
vuelven por `DomainWork` temporal. No toca OPES productivo ni publica nada.

## Requisitos

- Ejecutar solo con confirmacion explicita:
  `ORQUESTA_OPES_REVIEW_PROVIDER_SMOKE_CONFIRM=1`.
- Como el payload usa `director_execution_mode=legacy_director_loop`, tambien
  deben estar activados:
  `ORQUESTA_EXTERNAL_WORK_LEGACY_DIRECTOR_LOOP=1` y
  `ORQUESTA_AUTOPROGRAMMING_LEGACY_DIRECTOR_LOOP=true`.
- `ORQUESTA_OPES_REVIEW_PROJECT_DIR` debe apuntar al repo OPES real que se
  quiere revisar, por ejemplo `/home/alberto/Trabajo/OPES`. Si se prueba dentro
  del repo Orquesta, el contrato de escritura debe quedar acotado a
  `external/opes/`; no uses un directorio temporal arbitrario como si fuera OPES.
- Los proveedores reales deben existir en `PATH` o en
  `ORQUESTA_CODEX_COMMAND`, `ORQUESTA_GEMINI_COMMAND` y
  `ORQUESTA_CLAUDE_COMMAND`.

## Diagnosticos Esperados

- Si el proveedor entrega y el `DomainWork` temporal acepta el artefacto, un
  error posterior del supervisor se publica como
  `delivered_with_post_delivery_supervisor_error`, con accion de conservar la
  evidencia y reintentar supervision sin relanzar el proveedor.
- Si Gemini falla con el cliente/tier actual, el wrapper debe escribir el
  diagnostico estructurado `provider_auth_or_tier_blocked` en
  `orquesta_provider_diagnostic_v0.json`; Orquesta lo consume como bloqueo de
  autenticacion/tier, no como `no_ack` opaco.

## Seguridad

El backend HTTP de dominio del smoke es temporal y local. El script conserva
artefactos si `ORQUESTA_KEEP_SMOKE_DIR=1`; esos artefactos son evidencia de
prueba, no material listo para produccion.
