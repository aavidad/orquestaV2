# V23: comandos públicos del dossier

Fecha: 2026-07-26.

Estado: **partial_green_unsealed**. `public_dossier_commands` y la
confirmación causal quedan implementados y cubiertos por E2E de bootstrap
SQLite, todavía sin sello V23. Este corte no completa V23:
permanece pendiente la generación editorial y la acreditación del catálogo y
los flujos restantes del Wizard.

## Cadena y gates

| Gate exacto | Evidencia |
| --- | --- |
| `orchestrator_auth` | `ffd29df9`; `TestOrchestratorIntakeUsesAuthenticatedScopeInsteadOfSpoofedRequestFields` |
| `plan_invalid` | `2e149be4`; `TestBuildIntakeDossierRejectsEveryCompilerInvalidPlanMetadata` valida `compileWorkItemSpec` y `goal.NewPlan` antes de admitir dossier |
| `commands_spoof` | `65b6b934`; `TestIntakeDossierCommandsBindAuthorityRejectSpoofAndProjectCompletePlan` rechaza identidad, request y auth aportadas por payload |
| `bootstrap_restart_exact` | `0fb8a50d`; `TestV23DossierCommandsPersistReplayAndCanonicalReadAcrossRestart` conserva bytes, receipt canónico e identidad tras SQLite restart |
| `confirmation_atomic_replay_recovery` | `c4a1ae9a`, `2f21ddc8` y `3540a7b0`; la confirmación crea dossier congelado, Goal, AppSpec, plan y outbox en una transacción, conserva replay tras progreso y recovery acepta generación viva posterior |
| `bootstrap_confirmation_exact` | `3540a7b0`; `TestV23DossierCommandsPersistReplayAndCanonicalReadAcrossRestart` ejecuta prepare, restart, confirm y replay exacto por la superficie pública |

Los comandos públicos son `orquesta.intakes.dossier.prepare` y
`orquesta.intakes.dossier.get`: salen del registry e i18n canónicos y atan
actor, proyecto y request a la autoridad de la composición, no a campos del
payload.

El comando mutante `orquesta.intakes.dossier.confirm` aplica la confirmación
explícita del `dossier_ref` exacto y devuelve el receipt causal. Su contrato no
acepta replays con dossier, Goal o autorización sustituidos, ni records vivos
incompletos. El replay conserva sus bindings inmutables aunque el Goal haya
progresado; recovery exige que el AppSpec inicial sea generación 1, pero admite
`plan_generation >= 1` en el Goal vivo.

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

Los cuatro P1 de confirmación quedan cerrados solo en candidato por
`c4a1ae9a`/`2f21ddc8`, con exposición pública en `3540a7b0`:
`TestConfirmIntakeDossierExactReplayAcceptsLiveGoalProgress`,
`TestV23ConfirmIntakeDossierRejectsSubstitutedPersistedBindings`,
`TestV23ConfirmIntakeDossierReplayRejectsIncompleteLiveRecord` y
`TestV23DossierConfirmationRecoveryAcceptsLivePlanGeneration`. No constituyen
por sí solos un sello V23.

Firecracker no es dependencia ni gate de V23. Su activación opt-in pertenece a
un corte posterior.
