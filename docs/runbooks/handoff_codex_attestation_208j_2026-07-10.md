# Handoff: recuperacion durable de claim 208J

Fecha: 2026-07-10. Rama: `wip/attestation-208j-20260710`, apilada sobre
`wip/attestation-208h-20260710`.

## Cambio

Si el atestador independiente falla despues de adquirir un claim, el claim
queda durablemente en estado `failed` con codigo y fecha. La observacion
persiste un cierre `blocked` con `needs_rework=true` y evidencia del claim.
Una observacion posterior no vuelve a ejecutar el atestador para ese goal: el
Director debe crear un rework causal con nuevo goal/claim.

No se borra el claim, no se cae a resultados autodeclarados y no existe bucle
de reintentos por polling.

## Frontera

`orquesta-goal` solo define contrato y lifecycle; la transicion durable se
implementa bajo lock/CAS de `orquesta-state-file`. No se usa runtime, shell,
proveedor ni filesystem desde el nucleo.

## Pruebas ejecutadas

```bash
go test -count=1 ./modulos/orquesta-goal \
  -run 'TestGoalRequiredTestAttestationV0'

go test -count=1 ./modulos/orquesta-state-file \
  -run 'TestStoreV0GoalRequiredTest(Attestation|FinalSnapshot|Claim)'

go test -count=1 ./modulos/orquesta-app-director-service \
  -run 'TestObserveAppDirectorGoalV0BloqueaRunSiFaltanRequiredTestsV0'

go test -count=1 ./modulos/orquesta-app-codex-stack \
  -run 'Test(LocalGoalRequiredTestAttestorV0EsInyectadoYNoHaceFallback|PrepareAutoprogrammingRunV0GoalReady|CodexStackAutoprogrammingPrepareRunAPIV0GoalReady)'

go test -count=1 . \
  -run 'TestNeutralOrchestrationPackagesDoNotImportProductAdapters$'
```

## Pendiente de integracion

Un revisor debe rebasar 208H+208J sobre la rama principal actual, resolver
solo conflictos contractuales y repetir esta lista junto con los focales de
208H. D1/D2 siguen prohibidos hasta esa revision.
