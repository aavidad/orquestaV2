# Contratos: orquesta-estado-vivo

## Frontera

`orquesta-estado-vivo` es nucleo neutral. No observa procesos, no lee stores,
no consulta HTTP, no ejecuta comandos y no importa otros modulos de Orquesta.
Las composiciones y adaptadores convierten sus fuentes a `EvidenciaEstadoV0`.

## `EvidenciaEstadoV0`

Una evidencia representa una observacion de una sola fuente. La fuente conserva
su estado bruto en `Estado` y aporta flags estructurales cuando los conoce:
`Scope`, identidad o generacion runtime, `RuntimeObservationAttempted`,
`RuntimeObservado`, `RuntimeIdentityMismatch`, `ProcesoVivo`,
`Terminal` y `Aceptado`. `RuntimeObservado`
distingue una identidad runtime inspeccionada y muerta de la ausencia de una
observacion runtime. `Terminal` representa un resultado terminal durable, no
una inferencia desde el string del state. Para ejercer autoridad de cierre debe
traer `Fuente` no vacia y al menos una `EvidenceRefs` durable; sin ambas queda
`indeterminate` con `durable_terminal_evidence_missing` y reparacion requerida.

`ScopeGoalExecutionV0` (`goal_execution`) exige `RuntimeIdentityRef` o
`RuntimeGenerationRef` coincidente para que una
observacion runtime pueda afectar la vida del goal. `ScopeBackendServiceV0`
describe el servicio compartido: aunque este vivo, nunca confirma que un goal
concreto siga ejecutandose. `ProcesoVivo` tampoco implica por si solo que el
runtime haya sido observado. `RuntimeObservationAttempted=true` con
`RuntimeObservado=false` representa una observacion incompleta (por ejemplo,
timeout): no equivale a proceso muerto y no autoriza terminal ni `running`.

La evidencia de registry aporta la identidad esperada sin marcar intento de
observacion; el snapshot aporta la identidad efectivamente observada. Puede
haber varias identidades de proceso para un mismo run. Solo divergen cuando
una identidad observada no pertenece al conjunto esperado, no por ser dos
procesos causales distintos del mismo run.

La evidencia no decide fase. La frase operativa local es:

Las fuentes aportan evidencia; SOLO ConstruirProyeccionCicloVidaV0 decide fase

## `DerivarVeredictoCausalV0`

Es la unica autoridad pura que reconcilia state persistido, resultado durable
y liveness runtime. Devuelve `VeredictoCausalV0` con una clase tipada:

- `running_confirmed`: la identidad runtime coincidente de `goal_execution`
  fue observada viva; es la unica
  clase con `PublicarRunning=true`;
- `terminal_by_artifact`: hay resultado terminal durable y no hay runtime del
  goal confirmado vivo;
- `process_dead_state_stale`: state declara `running` y runtime fue observado
  sin proceso vivo;
- `divergent_needs_repair`: terminal y vivo coexisten, la identidad runtime
  falta/no coincide, o las refs causales se contradicen;
- `indeterminate`: hubo timeout/observacion incompleta, falta observar
  liveness para un state `running`, la evidencia aun es insuficiente o un
  terminal no trae referencia durable. No publica `running`; el terminal no
  demostrado exige reparacion y el timeout exige reobservacion.

La funcion conserva refs opacas, no lee stores/artefactos/procesos y no decide
por nombres de fuente. Los adaptadores solo traducen sus observaciones a
`EvidenciaEstadoV0`. `ConstruirProyeccionCicloVidaV0` consume este veredicto;
no mantiene otra reconciliacion de terminal/runtime en el borde de proyeccion.
`NodoCicloVidaV0` conserva el `VeredictoCausalV0` completo junto a `Fase`; la
fase queda por compatibilidad aditiva para consumidores legacy. MCP, server y
serializacion JSON consumen ese mismo veredicto y no recalculan identidad,
terminalidad o liveness.

## `ConstruirProyeccionCicloVidaV0`

La funcion es pura y determinista:

- recibe todas las evidencias como datos;
- recibe `ahora` y `umbralHuerfano`;
- agrupa por `RunRef`; si falta, por `GoalRef`;
- ordena nodos y evidencias antes de devolver;
- devuelve `SchemaVersion` igual a `orquesta_proyeccion_ciclo_vida.v0`;
- no inventa verde cuando no hay evidencia: devuelve fase `desconocido`.

## Precedencia

1. Proceso vivo sin terminal: `proceso_vivo`.
2. Terminal aceptado sin proceso vivo: `terminal_aceptado`.
3. Terminal no aceptado sin proceso vivo: `terminal_rework`.
4. Proceso vivo y terminal simultaneos: `conflicto` con
   `proceso_vivo_tras_terminal`.
5. Estado persistido `blocked` o `bloqueado`: `bloqueado`.
6. Solo `run_marker` sin estado ni proceso: `huerfano` si supera
   `umbralHuerfano`, si no `lanzado`.
7. `receipt` no terminal con indicio de artefacto: `entregado_parcial`.
8. Sin evidencias: `desconocido`.
9. Un nodo por trabajo, agrupado por `RunRef` o `GoalRef`.

Como normalizacion neutral adicional, `outbox`/`queue` pendientes proyectan
`solicitado`, `wait_external` proyecta `lanzado`, y una fuente de proceso con
`ProcesoVivo=true` proyecta `proceso_vivo`.
