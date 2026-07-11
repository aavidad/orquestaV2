# Incidencia 208AB: goal spec generado invalido

Fecha: 2026-07-11
Estado: cerrado localmente por `97d1d913a`
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

## Cierre

La causa estaba en `goalSpecWithDependencyRequiredTestsV0`: los tests derivados
del grafo se anadian con `TestRef` y `Command`, pero sin `CommandRef` ni hashes
congelados. El wrapper mutaba un spec valido despues de la validacion inicial y
el launcher Codex lo rechazaba.

`97d1d913a` genera `CommandRef` determinista y aplica
`FreezeGoalRequiredTestV0` antes de entregar el spec al launcher. Los focales de
dependencias y stack pasan. El repro API
`autoprog-attestor-contract-repro-20260711` fue aceptado y lanzo exactamente un
goal real, sin `goal_ref_field_invalid` ni run invalido. El goal de verificacion
se detuvo despues manualmente porque la instancia se arranco deliberadamente
sin director/observador residente y amplio su diagnostico; control confirmo
`stopped` y shutdown termino en el primer intento. Ese cierre manual no acredita
208AA ni el attestor con dependencias externas.
