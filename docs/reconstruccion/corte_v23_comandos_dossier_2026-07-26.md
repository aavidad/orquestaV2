# V23: comandos públicos del dossier

Fecha: 2026-07-26.

Estado: **partial_green_unsealed**. `public_dossier_commands` queda completado
y se retira de `deferred_scopes`. Este corte no completa V23: permanecen
pendientes la generación editorial y la única transición atómica de
confirmación, freeze y creación del Goal.

## Cadena y gates

| Gate exacto | Evidencia |
| --- | --- |
| `orchestrator_auth` | `ffd29df9`; `TestOrchestratorIntakeUsesAuthenticatedScopeInsteadOfSpoofedRequestFields` |
| `plan_invalid` | `2e149be4`; `TestBuildIntakeDossierRejectsEveryCompilerInvalidPlanMetadata` valida `compileWorkItemSpec` y `goal.NewPlan` antes de admitir dossier |
| `commands_spoof` | `65b6b934`; `TestIntakeDossierCommandsBindAuthorityRejectSpoofAndProjectCompletePlan` rechaza identidad, request y auth aportadas por payload |
| `bootstrap_restart_exact` | `0fb8a50d`; `TestV23DossierCommandsPersistReplayAndCanonicalReadAcrossRestart` conserva bytes, receipt canónico e identidad tras SQLite restart |

Los comandos públicos son `orquesta.intakes.dossier.prepare` y
`orquesta.intakes.dossier.get`: salen del registry e i18n canónicos y atan
actor, proyecto y request a la autoridad de la composición, no a campos del
payload.

## Falso verde cerrado

`BUG-ORQ-20260726-526` queda cerrado por `2e149be4`. Antes, el dossier podía
aceptar un plan con forma serializable pero no ejecutable. El builder compila
cada `WorkItemSpec` y exige que `goal.NewPlan` acepte el plan; el gate
`plan_invalid` mantiene negativos de metadata de governance, tests y scope.

## Residual no bloqueante

`BudgetDemand` no pertenece al `planInput` público existente. Por tanto, un
dossier creado por el adaptador Go con demand no hace round-trip lossless en el
GET público. No se fuerza un campo API nuevo en este corte: requiere contrato
de compatibilidad explícito y no bloquea los comandos acreditados.

No hay E2E Firecracker verde declarado por este documento.
