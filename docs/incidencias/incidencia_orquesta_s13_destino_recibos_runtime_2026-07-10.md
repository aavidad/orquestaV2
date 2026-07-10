# S13: destino runtime de recibos goal-first

Estado: abierto. Area: composicion goal-first/Codex. Fecha: 2026-07-10.

## Hecho comprobado

El prompt Codex goal-first calcula el fichero de resultado con
`codexGoalResultFilePathV0` en
`modulos/orquesta-runtime-codex-goal/packet_v0.go`. Elige el primer scope de
directorio del write-set y ordena escribir alli
`orquesta_goal_result_<goal_ref>.json`.

El observador de app-server y
`stackGoalMaterializedRefsSourceV0` buscan resultados/checkpoints en el
proyecto y en sus write-sets. Por tanto, retirar solo la instruccion del
prompt deja sin fallback observable a ejecuciones que no devuelven el marcador
final por thread. Una prueba local de ese cambio fallo en los contratos de
packet; se revirtio y no hay cambio funcional pendiente de commit.

## Causa estructural

El contrato neutral de goal declara producto, write-set y cierre, pero no hay
un puerto/adaptador que asigne y observe un destino de recibos de ejecucion
fuera del arbol versionable. El runtime conoce su directorio, mientras que el
builder de packet no lo recibe; por eso los recibos historicos terminaron bajo
`*/docs/` de modulos y comandos.

## Correccion requerida

1. La composicion/adaptador runtime asigna un directorio por `goal_ref` bajo
   su runtime aislado y lo pasa al lanzamiento sin introducir paths locales en
   `orquesta-goal` ni en el core puro.
2. El observador lee primero ese recibo runtime, valida `goal_ref` e identidad
   de lanzamiento y publica su procedencia como evidencia durable.
3. Solo entonces el prompt deja de ordenar
   `checkpoint_started_*`/`orquesta_goal_result_*` dentro del write-set.
   La lectura desde el proyecto queda temporalmente como compatibilidad para
   migrar los 58 artefactos movibles ya inventariados.
4. Un test debe demostrar que un resultado runtime cierra el goal y que un
   recibo de proyecto no nuevo no se confunde con otro goal. El guard Git S13
   debe seguir rechazando cualquier recibo nuevo versionado.

## Limites

No eliminar ni mover los artefactos existentes hasta que el observador runtime
este activo y haya una migracion con referencias comprobadas. No tocar OPES ni
workloads historicos en este corte.
