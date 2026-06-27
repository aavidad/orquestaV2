# Contratos

## GoalWorkSpecV0

`GoalWorkSpecV0` es la unidad que Orquesta entrega a un runtime con soporte de
goal. El runtime ejecuta el bucle operativo; Orquesta conserva autoridad sobre:

- objetivo;
- refs de contexto;
- reglas aplicables;
- write-set;
- tests requeridos;
- criterios de aceptacion;
- contratos de artefactos;
- politica de cierre;
- politica de rework.

El contrato es neutral: no contiene comando Codex, HOME, modelo, proveedor, DB,
HTTP, OPES interno ni rutas absolutas.

## GoalWorkResultV0

`GoalWorkResultV0` resume el estado observable del goal:

- `running`: el goal sigue vivo;
- `complete`: el runtime declara cierre;
- `blocked`: el runtime no puede progresar sin input o cambio externo;
- `invalid`: la observacion no puede aceptarse.

`complete` no basta para cerrar el trabajo en Orquesta. El resultado debe traer
el mismo `goal_ref` del `GoalWorkSpecV0` y la composicion debe validar
evidencias, tests, artefactos y receipts de dominio antes de cerrar.
Las refs observadas (`goal_ref`, `external_goal_ref`, artefactos, evidencias,
tests y receipts) pasan por validacion estructural antes de aceptar cierre.

## GoalObservationRequestV0

`GoalObservationRequestV0` identifica el goal que una composicion quiere
observar. `goal_ref` es obligatorio y `external_goal_ref` es opcional; ambos se
validan como refs opacas, no como rutas locales absolutas.

## Lifecycle neutral

`StartGoalWorkV0` y `ObserveGoalWorkV0` son el caso de uso neutral del contrato
goal-first:

- `StartGoalWorkV0` recibe un `GoalWorkSpecV0`, lanza por
  `GoalWorkLauncherPortV0`, normaliza `GoalLaunchReceiptV0`, construye
  `GoalWorkStateV0` y lo guarda por `GoalWorkStateStorePortV0`.
- `ObserveGoalWorkV0` carga `GoalWorkStateV0` por `run_ref`, construye la
  `GoalObservationRequestV0`, observa por `GoalWorkObservationPortV0`, actualiza
  `LastResult`/estado/evidencias y, si el resultado es terminal, valida cierre
  con `GoalWorkClosureValidatorPortV0`.

El lifecycle no abre fases, no emite eventos de run, no decide colas y no cierra
producto externo. Devuelve si el goal esta terminal, si la closure fue aceptada
o si necesita rework; cada composicion proyecta eso a su propio `RunStore`,
cola, API o dominio.

## Puertos

- `GoalWorkLauncherPortV0`: lanza un goal desde un spec.
- `GoalWorkObservationPortV0`: observa estado/evidencias de un goal.
- `GoalWorkClosureValidatorPortV0`: valida si el resultado cierra el contrato.
- `GoalWorkStateStorePortV0`: guarda y carga el estado durable del goal por
  `run_ref`.

Los adaptadores concretos implementan esos puertos fuera del nucleo.
