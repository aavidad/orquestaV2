# Replan bloqueado al rotar la política de presupuesto

Fecha: 2026-07-29.

Estado: corregido en código; pendiente de promoción y verificación sobre el
Goal real.

## Síntoma

El Goal `goal:37f4bda4400b3a1fd70986800b3b78b1` podía leerse y conservaba
correctamente su ejecución fallida, pero
`orquesta.director.plan.propose` devolvía `500 internal` al crear el siguiente
rework.

El payload del sucesor, sus pruebas requeridas y su `write_set` eran válidos.
La misma transición pasaba sobre una copia de SQLite al reconstruir el
orquestador con la política histórica.

## Causa

El Goal nació con `runtime.max_output_bytes=1048576`. Al elevar el límite
residente a `67108864`, cambió también el hash de `BudgetPolicy`.

`historicalEffectPolicy` restauraba correctamente hash, revisión, TTL, demora
y límites históricos, pero no podía reconstruir el `DefaultDemand`. El
Director solo lo rellenaba desde la política residente cuando ambos hashes
coincidían. Tras la rotación quedaba un vector cero y
`compileWorkItemSpec` devolvía
`application.work_item_budget_demand_invalid`, que además se proyectaba de
forma opaca como `internal`.

## Corrección

- Todo replan usa como demanda por defecto la demanda durable del WorkItem
  causal. Una demanda explícita del sucesor sigue teniendo prioridad.
- Las ejecuciones nuevas del replan heredan `MaxOutputBytes` de la ejecución
  causal exacta. Otros WorkItems listos conservan el límite de la composición
  residente.
- El error `ErrWorkItemBudgetDemandInvalid` tiene identidad tipada y la
  superficie de comandos lo clasifica como `invalid_request`.
- No se liga por una regla nueva `DiskBytes` a `MaxOutputBytes`: ambos se
  validan y heredan de forma independiente para conservar compatibilidad con
  demandas explícitas y registros históricos.

## Evidencia

- La reproducción sobre backup falla con política rotada antes del arreglo y
  pasa al usar la política histórica.
- `TestDirectorReplanAfterBudgetPolicyRotationInheritsCausalDemandAndOutputLimit`
  cubre atestación fallida, rotación 1 MiB -> 64 MiB, herencia causal y replay.
- `TestSQLiteDirectorFailedAttestationReplanSurvivesPolicyRotationReadRestartAndRecovery`
  cubre la misma transición en SQLite, lectura inmediata, reinicio, recovery y
  replay.
- Los paquetes completos `internal/application`, `internal/commands` e
  `internal/adapters/state/sqlite` pasan con `-count=1`.

## Regla de continuidad

Cambiar configuración residente no puede borrar defaults históricos necesarios
para continuar un Goal ya admitido. El replan toma presupuesto y límites desde
su causa durable. La configuración actual gobierna Goals nuevos; una extensión
sin causa sobre un Goal de otra política debe declarar su demanda explícita y
falla de forma cerrada si intenta usar un default que no pertenece a ese Goal.
