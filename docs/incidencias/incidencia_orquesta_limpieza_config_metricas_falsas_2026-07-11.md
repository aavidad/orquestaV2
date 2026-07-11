# Incidencia de limpieza: env daemon y metrica de duplicados

Fecha: 2026-07-11.

## BUG-ORQ-20260711-237: allowlist por prefijo no registrada

Estado: abierto.

El ratchet AST del servidor informa cero lecturas `ORQUESTA_*` fuera del
registro, pero `serverDaemonStartEnvironmentV0` acepta variables del proceso
padre mediante prefijos amplios (`ORQUESTA_SERVER_*`, `ORQUESTA_CODEX_*` y
otros). Una clave inventada como `ORQUESTA_SERVER_UNREGISTERED=1` puede cruzar
al daemon sin existir en el registro ni aparecer en `effective_config`.

Impacto: la configuracion efectiva no describe toda la superficie heredada y
el ratchet puede dar un falso verde. El arreglo debe permitir solo claves
registradas para el proceso servidor o una allowlist explicita de proceso hijo;
los secretos y claves desconocidas deben quedar fuera.

Criterio de cierre:

- prueba negativa con una clave de prefijo valido pero no registrada;
- conservacion de las claves registradas y derivadas requeridas;
- focales de daemon, registro y configuracion verdes;
- ejecucion por Orquesta con atestacion independiente antes de declarar cierre.

## BUG-ORQ-20260711-238: duplicacion nominal presentada como codigo duplicado

Estado: abierto, diagnostico confirmado.

La metrica `helper_duplicate_definitions=307` no compara firmas, cuerpos ni
semantica. Agrupa cualquier funcion productiva cuyo nombre empiece por
`compact`, `contains` o `firstNonEmpty`; la medicion fresca se reparte en 236,
28 y 46 definiciones respectivamente. Cruza paquetes y mezcla funciones no
equivalentes, por ejemplo compactacion de errores, structs, refs y textos.

Impacto: la auditoria y el nightly presentan coincidencias nominales como el
hallazgo mas grave de duplicacion y pueden inducir refactors masivos falsos.

Criterio de cierre:

- renombrar la senal como solape nominal o sustituirla por una medicion que
  compare implementacion compatible;
- no usar el valor historico 307 como autorizacion de consolidacion;
- actualizar tests, nightly y auditorias sin subir un baseline artificial;
- conservar la regla de revision por paquete y prueba focal.

## BUG-ORQ-20260711-239: rework residente sin binder de tests

Estado: cerrado localmente, pendiente de replay integrado.

El primer goal de BUG-237 alcanzo el umbral `material_progress_replan_required`
sin diff. El supervisor residente detecto el cierre reparable, pero
`maybePrepareGoalFirstResidentReworkV0` invoco `StartGoalWorkV0` sin transportar
`GoalRequiredTestSpecBinder`. Como el spec original tenia tests obligatorios, el
rework fallo con `goal_work_lifecycle_invalid:
ports.goal_required_test_spec_binder` y la cola original ya estaba `stopped`.

Impacto: Orquesta recomienda `replan`, dispone de launcher y binder en el stack,
pero no puede materializar por si misma la reparacion de un goal con tests. La
observacion manual no debe convertirse en un flujo operativo alternativo.

Criterio de cierre:

- el lifecycle de rework recibe el binder ya inyectado en el stack;
- regresion con spec de rework que conserve tests obligatorios;
- focal y paquete del stack verdes;
- repetir BUG-237 por Orquesta y comprobar que el trabajo llega a cierre o que
  un nuevo replan se lanza causalmente sin intervencion manual.

Cierre empirico: el lifecycle de rework transporta ahora
`GoalRequiredTestSpecBinder` desde los puertos del stack. La regresion conserva
un required test congelado, exige atestacion independiente y comprueba que el
binder se invoca antes del launcher. El focal y el paquete completo
`./modulos/orquesta-app-codex-stack` quedan verdes. Tras desplegar el binario
local, el estado bloqueado materializo el rework
`request-ref-bug237-daemon-env-20260711-rework-38a04c4c4f93b543` y lanzo un
nuevo goal real; BUG-239 queda cerrado.

## BUG-ORQ-20260711-240: rework sin OrchestrationRun causal

Estado: cerrado localmente, pendiente de replay integrado.

El rework de BUG-239 persistio y ejecuto su `GoalWorkState`, pero el launcher
residente no creo la `OrchestrationRunV0` con el nuevo `run_ref`. Cuando el
governor lo paro terminal, `ObserveAppDirectorGoalV0` intento reflejar el cierre
en `RunStore`, no encontro la run hija y el HTTP devolvio 500 generico. El goal
hijo quedo bloqueado y sin procesos vivos, pero no era observable por API.

Criterio de cierre:

- crear la run hija valida antes de lanzar el goal residente;
- reentrada idempotente que repare reworks ya persistidos sin run;
- la observacion terminal devuelve estado publico, nunca 500 por run ausente;
- focal, paquete y replay del estado vivo verdes.

Cierre local: el launcher asegura una run goal-first minima y valida antes de
arrancar. Si la run ya existe, comprueba identidad y forma; si el goal hijo ya
estaba persistido pero faltaba la run, la reentrada la materializa sin relanzar
el proveedor. La regresion ejecuta el supervisor dos veces, conserva una sola
llamada al launcher/binder y carga la misma run activa. Focal y paquete completo
del stack verdes.

## Evidencia de control

En `ed5c0bff94f2` la auditoria reproducible informa 1.177 candidatos `deadcode`,
pero cero privados sin referencias textuales. Los ocho modulos sin importador
son adaptadores opt-in ya clasificados para conservar. Por tanto no existe otra
retirada destructiva automatica autorizada en este corte.
