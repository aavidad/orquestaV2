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

Cierre local: el lifecycle de rework transporta ahora
`GoalRequiredTestSpecBinder` desde los puertos del stack. La regresion conserva
un required test congelado, exige atestacion independiente y comprueba que el
binder se invoca antes del launcher. El focal y el paquete completo
`./modulos/orquesta-app-codex-stack` quedan verdes.

## Evidencia de control

En `ed5c0bff94f2` la auditoria reproducible informa 1.177 candidatos `deadcode`,
pero cero privados sin referencias textuales. Los ocho modulos sin importador
son adaptadores opt-in ya clasificados para conservar. Por tanto no existe otra
retirada destructiva automatica autorizada en este corte.
