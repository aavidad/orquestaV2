# S13: destino runtime de recibos goal-first

Estado: parcialmente corregido localmente. Area: composicion goal-first/Codex.
Fecha: 2026-07-10.

## Hecho comprobado

El prompt Codex goal-first calculaba el fichero de resultado con
`codexGoalResultFilePathV0` dentro del primer scope de directorio del
write-set y ordenaba escribir alli `orquesta_goal_result_<goal_ref>.json`.

El observador de app-server y
`stackGoalMaterializedRefsSourceV0` buscan resultados/checkpoints en el
proyecto y en sus write-sets. Por tanto, retirar solo la instruccion del
prompt deja sin fallback observable a ejecuciones que no devuelven el marcador
final por thread. Una prueba local de ese cambio fallo en los contratos de
packet; se revirtio y no hay cambio funcional pendiente de commit.

## Corte local 2026-07-10

El primer adaptador corregido es `orquesta-runtime-codex-appserver`:

- checkpoint temprano y resultado Codex se dirigen a
  `.orquesta-runtime/goal-receipts/<goal_ref>/`, ruta ya ignorada por Git;
- el prompt excluye esos recibos de `artifact_paths` y del write-set de
  producto;
- el observador lee primero el recibo runtime, valida `goal_ref` y
  `external_goal_ref`, y solo despues conserva el escaneo del proyecto como
  fallback de compatibilidad;
- las pruebas focales demuestran checkpoint fuera del write-set y prioridad
  de recibo runtime frente a un resultado legacy con el mismo goal.

Verificacion local: `go test -count=1
./modulos/orquesta-runtime-codex-goal
./modulos/orquesta-runtime-codex-appserver`,
`go test -count=1 ./modulos/orquesta-app-codex-stack -run
'Test.*(GoalFirst|Materialized|Observe).*V0'` y
`bash scripts/test_orquesta_check_versioned_execution_artifacts.sh`.

## Causa estructural restante

El contrato neutral de goal declara producto, write-set y cierre, pero no hay
un puerto/adaptador que asigne y observe un destino de recibos de ejecucion
fuera del arbol versionable. El runtime conoce su directorio, mientras que el
builder de packet no lo recibe; por eso los recibos historicos terminaron bajo
`*/docs/` de modulos y comandos.

## Correccion requerida

1. Aplicar el mismo contrato a Claude, Gemini y cualquier backend goal-first
   que aun depende exclusivamente del fichero dentro del proyecto.
2. Proyectar la procedencia runtime en el observador/materializador comun para
   que la reconciliacion no tenga que volver a inferirla de cada proveedor.
3. Migrar con commits gobernados los 58 artefactos movibles, manteniendo los
   cinco historicos, cuatro fixtures y una referencia pendiente del inventario.
4. El guard Git S13 debe seguir rechazando cualquier recibo nuevo versionado.

## Limites

No eliminar ni mover los artefactos existentes hasta que el observador runtime
este activo y haya una migracion con referencias comprobadas. No tocar OPES ni
workloads historicos en este corte.
