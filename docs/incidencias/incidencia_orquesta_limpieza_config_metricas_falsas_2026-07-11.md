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

## Evidencia de control

En `ed5c0bff94f2` la auditoria reproducible informa 1.177 candidatos `deadcode`,
pero cero privados sin referencias textuales. Los ocho modulos sin importador
son adaptadores opt-in ya clasificados para conservar. Por tanto no existe otra
retirada destructiva automatica autorizada en este corte.
