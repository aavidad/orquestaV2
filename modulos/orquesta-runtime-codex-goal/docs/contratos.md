# Contratos

## CodexGoalStartPacketV0

Paquete publico para iniciar un Codex Goal desde un `GoalWorkSpecV0`.

Contiene:

- `goal_ref`;
- refs de request/proyecto y tipo de trabajo cuando existan;
- `objective`;
- prompt compacto;
- refs de contexto;
- reglas;
- write-set;
- tests requeridos;
- criterios de aceptacion;
- contratos de artefactos;
- evidencias de entrada;
- presupuesto operativo;
- politica de cierre;
- politica de rework.

No contiene HOME, token, OAuth, proveedor, modelo, comando, ruta absoluta ni
transcript.

## CodexGoalStarterPortV0

Puerto de composicion que crea el goal real. El adaptador no implementa llamadas
directas a herramientas internas; solo define la frontera.

La composicion `cmd/orquesta-server` aporta una implementacion opt-in con
`ORQUESTA_CODEX_GOAL_BACKEND=app_server_proxy`: usa `codex app-server proxy`
contra un daemon local ya disponible, no `codex exec`, y mantiene el transporte
fuera de este modulo.

## CodexGoalObservationRequestV0

Paquete publico para observar un Codex Goal ya lanzado. Contiene `goal_ref` y,
si existe, `external_goal_ref`. Ambas refs se validan como opacas antes de
llamar al puerto real.

## CodexGoalObserverPortV0

Puerto de composicion que observa el goal real y devuelve estado, resumen,
artefactos, tests, receipts y evidencias. El adaptador convierte esa respuesta a
`GoalWorkResultV0`, valida refs/status y rechaza observaciones cuyo `goal_ref`
no coincida con el pedido.

En el backend `app_server_proxy`, la observacion usa la `external_goal_ref`
persistida como `threadId` y consulta `thread/goal/get`.

## CodexGoalLauncherV0

Implementa `GoalWorkLauncherPortV0` usando un `CodexGoalStarterPortV0`.

## CodexGoalObserverV0

Implementa `GoalWorkObservationPortV0` usando un `CodexGoalObserverPortV0`.
`complete` observado no equivale a cierre aceptado; el cierre sigue pasando por
`GoalWorkClosureValidatorPortV0` o un validador de composicion equivalente.
