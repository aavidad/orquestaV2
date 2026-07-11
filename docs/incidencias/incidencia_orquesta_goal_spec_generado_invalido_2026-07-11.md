# Incidencia 208AB: goal spec generado invalido

Fecha: 2026-07-11
Estado: abierto
Area: autoprogramacion goal-first / binder de atestacion / lanzamiento

## Resumen

Tres requests de autoprogramacion pasaron la validacion publica, materializaron
un `GoalWorkSpecV0` con refs y hashes no vacios, pero el launcher los rechazo con
`goal_ref_field_invalid`. No se lanzo Codex y se persistieron tres runs `invalid`.

## Evidencia

- Runtime retenido: `/tmp/orquesta-208aa-goal-runtime/state/`.
- Runs:
  - `autoprog-208aa-goal-contract-20260711`;
  - `autoprog-attestor-infra-contract-20260711`;
  - `autoprog-attestor-infra-contract-two-tests-20260711`.
- El cambio de task ref numerica a alfabetica no altero el resultado.
- El cambio de una a dos pruebas requeridas tampoco altero el resultado.
- Issues repetidos en `launch_receipt.issues`:
  `goal_ref_field_invalid` sobre `required_tests.command_ref` y
  `goal_required_test_attestation_mismatch` sobre `command_sha256` y
  `definition_sha256`.
- Los valores persistidos parecen bien formados y el `command_sha256` coincide
  con el SHA-256 del comando. Esto apunta a una mutacion/doble binding o a una
  validacion no idempotente entre materializacion y launcher, no al texto del
  operador.
- El servidor cerro cooperativamente sin goals ni procesos residuales.

## Criterio de cierre

1. El spec generado y enlazado por el attestor es idempotente bajo un segundo
   normalize/bind y conserva refs/hashes congelados.
2. La misma instancia de spec que se persiste pasa `ValidateGoalWorkSpecV0` y
   `ValidateGoalRequiredTestAttestationBindingV0` antes de invocar el launcher.
3. Si el generador produce un spec invalido, `prepare-run` falla en validacion
   con field preciso y no persiste un run/marker `invalid` como trabajo vivo.
4. Pruebas cubren una y dos required tests, task refs numericas y alfabeticas,
   y doble binding.
5. Repro real por API lanza exactamente un goal y no consume reintentos para
   corregir refs generadas internamente.

