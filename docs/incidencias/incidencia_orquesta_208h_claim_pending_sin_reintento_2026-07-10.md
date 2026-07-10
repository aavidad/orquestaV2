# Incidencia: 208J claim de atestacion pendiente sin recuperacion

Fecha: 2026-07-10.
Estado: correccion local integrada en `08a993a3e`; permanece abierta como
evidencia de cierre de `BUG-ORQ-20260710-208H` hasta probar la composicion con
un atestador configurado. No queda un claim `pending` sin recuperacion en el
contrato local.

## Hallazgo original

La rama candidata inicial `wip/attestation-208h-20260710` adquiria un claim
durable antes de lanzar el atestador independiente. Si el atestador devolvia
error, el claim quedaba en estado `pending`. En la siguiente observacion,
`AcquireGoalRequiredTestAttestationClaimV0` encontraba ese claim, devolvia
`Acquired=false` y el ciclo lo saltaba.

La validacion final permanecia bloqueada por falta de receipt, pero el sistema
no podia volver a ejecutar el test ni publicar una causa durable de rework. Un
error transitorio, timeout o muerte del proceso de atestacion dejaba el goal
atascado indefinidamente. Esto contradecia el objetivo de autonomia de 208H.

## Correccion integrada y evidencia verificada

- `08a993a3e` anade `FailGoalRequiredTestAttestationClaimV0` al contrato y
  `lifecycle_v0.go` lo invoca cuando falla el atestador. La observacion publica
  una causa tipada de `blocked/rework`; no acepta el cierre ni vuelve a ejecutar
  el atestador por polling.
- `modulos/orquesta-state-file/goal_required_test_attestation_store_v0.go`
  persiste `failed` con `failure_code`; al recrear el store, `Acquire...`
  devuelve el mismo claim sin readquirirlo.
- `TestGoalRequiredTestAttestationV0ErrorTrasClaimPersisteReworkSinReintento`,
  `TestStoreV0GoalRequiredTestAttestationClaimFailedSurvivesRecreateWithoutReacquire`
  y `TestStoreV0GoalRequiredTestAttestationClaimMultiprocessHasSingleOwner`
  cubren error, persistencia tras recreacion y concurrencia respectivamente.
- Revision local posterior, 2026-07-10: `go test -count=1` para
  `orquesta-goal`, `orquesta-runtime-required-test` y los focales de
  `cmd/orquesta-server` de atestacion pasa en cache aislada.
- La composicion ya se ejerce desde configuracion owner-only por
  `TestBuildStackFromEnvV0GoalRequiredTestAttestationExecutesIndependentReceipt`:
  construye el stack, activa sus cuatro puertos, usa un fixture Git minimo,
  obtiene receipt `passed` e identidad de atestador independiente. Se lanzo por
  una ola Orquesta local con write-set limitado a ese test, sin crear una app.

## Evidencia restante para 208H

1. La ruta exitosa de la composicion queda acreditada. La ruta de error,
   `blocked/rework` durable, recreacion de store y concurrencia quedan
   acreditadas por los focales de `orquesta-goal` y `orquesta-state-file`; no
   se reabre el antiguo claim `pending`.
2. Reejecutar los lotes aislados cuando D3 deje de bloquearlos por el ratchet
   de variables de entorno.

No se creara una aplicacion de ejemplo para esta evidencia. Debe emplear un
fixture minimo o una aplicacion real que el operador solicite construir.

## Criterio de cierre

- Correccion de claim: cerrada localmente por `08a993a3e` y los tres focales
  anteriores.
- Cierre global de 208H: solo faltan los dos pases aislados por lotes tras D3.
