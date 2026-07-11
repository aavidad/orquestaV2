# BUG-ORQ-20260711-208AG: falso verde de cleanup por criterios autodeclarados

Estado: abierto.

El goal `019f4f10-0c7f-7421-ae18-143dc6482642` publicó `result=complete`,
afirmando que había eliminado exactamente diez grupos y que los tests habían
pasado. La revisión independiente encontró dos símbolos requeridos todavía
marcados como U1000 por `/home/alberto/go/bin/staticcheck`:

- `writeTmuxOwnerMarkerAtomicV0`
- `cleanupStartedTmuxGenerationV0`

La misma ejecución detectó además el residual emergente
`codexAppServerWriteSetCheckpointDirRelV0`. Evidencia retenida en
`/tmp/orquesta-cleanup-goal-20260711/staticcheck-after.txt`.

Clasificación: falso verde de criterios/checklist self-declared, relacionado
con `BUG-ORQ-20260710-208H` y su atestación independiente. Los required tests
verifican tests y receipts, pero no prueban la invariante de aceptación de que
no queden símbolos residuales. Por tanto, Orquesta habría aceptado el receipt
autodeclarado si `BUG-ORQ-20260711-208AF` no hubiera bloqueado previamente el
flujo.

Criterio de cierre: un verificador independiente debe ejecutar, además de los
tests requeridos, las invariantes de aceptación del write-set; debe detectar
los símbolos residuales y rechazar el checklist autodeclarado. No se declara
cierre por el `result=complete` del goal ni por tests que no cubran ese criterio.

Referencias: [inventario vivo](../inventario_bugs_estado_vivo.md),
[inventario historico](../inventario_bugs_orquesta_2026-06-30.md),
[208AF](incidencia_orquesta_goal_observer_high_consumption_terminal_reconcile_2026-07-11.md),
y [pruebas de atestación de 208H](../pruebas_revisor_208h_2026-07-10.md).
