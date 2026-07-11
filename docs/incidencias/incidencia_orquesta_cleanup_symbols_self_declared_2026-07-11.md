# BUG-ORQ-20260711-208AG: falso verde de cleanup por criterios autodeclarados

Estado: cerrado localmente.

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

Commits: `b96e9b115`, `2735b1dae`, `a245a90a8` y `7a91f8052`.

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

## Cierre local 2026-07-11

Los criterios textuales legacy quedan clasificados como advisory: nunca se
convierten en comandos ni se citan como atestacion. El Director humano acepta
`acceptance_checks` tipados y falla cerrado ante refs/comandos vacios o refs
duplicadas. Self-audit genera ref estable, descripcion y comando determinista;
backlog, request idle, ruta goal-first residente y MCP conservan el contrato.
El runtime recibe el `GoalRequiredTestSpecBinder` del stack, congela los tests
y exige atestacion independiente de todas las refs antes de aceptar cierre.

La revision previa al commit encontro y corrigio cuatro huecos: checks perdidos
en la ruta residente, binder no cableado, entrada humana no validada y schema
MCP no descubrible. El smoke focal reproduce ahora exactamente el residual
`U1000` de `codexAppServerWriteSetCheckpointDirRelV0` y demuestra que llega al
goal como check tipado con cierre independiente. La evidencia negativa original
se conserva en `/tmp/orquesta-cleanup-goal-20260711/staticcheck-after.txt`; en
el arbol vigente `/home/alberto/go/bin/staticcheck -checks U1000
./modulos/orquesta-runtime-codex-appserver` queda verde. Tambien quedan verdes
el smoke U1000, las baterias focales y `go test -count=1 ./...`.
