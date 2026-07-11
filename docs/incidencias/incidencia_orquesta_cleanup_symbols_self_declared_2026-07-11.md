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

## Avance estructural 2026-07-11

Commits base: `b96e9b115` y corte de transporte pendiente de hash final.

- `GoalClosurePolicyV0` declara
  `required_acceptance_criteria_refs`: refs opacas de criterios verificables
  obligatorios. Cada ref debe estar mapeada a un `GoalRequiredTestV0`, cuya
  definicion congelada incluye `acceptance_criteria_refs`.
- La autoprogramacion acepta `acceptance_checks` tipados por tarea
  (`criterion_ref`, descripcion y comando). El compilador fusiona comandos
  iguales, congela la definicion y exige atestacion independiente para todos
  los checks. No deriva comandos ni refs desde lenguaje natural.
- V0 y V1 conservan checks y specs; las proyecciones MCP publican refs y conteo
  separados de los criterios cualitativos.
- El fallback idle ya no puede aceptar un `complete` autodeclarado cuando el
  spec exige atestacion independiente o refs de criterios obligatorios.

Cobertura demostrada: una ref obligatoria sin test mapeado invalida el spec;
una request con check incompleto/duplicado se rechaza; dos refs sobre el mismo
comando producen un solo test congelado; y el cierre requiere el receipt
independiente de ese test.

Residual para cerrar 208AG: migrar o clasificar las requests legacy que solo
usan `acceptance_criteria` textual. Esas frases no pueden asociarse
automaticamente a comandos sin inventar semantica. Hasta esa migracion deben
considerarse advisory y nunca citarse como evidencia de criterio verificado.
El cierre final exige que los creadores productivos (incluido Director humano
y backlog) emitan `acceptance_checks` para criterios verificables o los marquen
explicitamente cualitativos; despues se repetira el caso U1000 con el comando
de analizador como check independiente.
