# BUG-ORQ-20260711-218: lease de perfil aislado escapaba a `/srv`

Fecha: 2026-07-11. Estado: cerrado localmente.

## Evidencia

Al ejecutar la auditoria de codigo con
`orquesta_use_isolated_test_env /tmp/orquesta-code-audit-env-20260711`, el
preflight de disco preparo correctamente caches bajo `/tmp`, pero el lease de
puertos intento crear
`/srv/orquesta-self/runtime/test-cache/port-leases` y fallo con
`PermissionError`. La raiz explicita gobernaba caches, pero no el default del
lease.

Clasificacion: fuga de default entre recursos de una misma sesion. Es la misma
familia preventiva que 208AH: una raiz declarada no es efectiva si un recurso
auxiliar conserva una ruta global/remota independiente.

## Cierre

`orquesta_acquire_test_port_lease` conserva
`ORQUESTA_TEST_PORT_LEASE_ROOT` como override explicito. Sin override usa
`ORQUESTA_TEST_CACHE_ROOT/port-leases`; si tampoco existe cache global, deriva
`port-leases` del directorio padre de la raiz aislada. Asi varias sesiones del
mismo padre siguen compartiendo locks sin salir de su filesystem declarado.

`scripts/test_orquesta_test_batches.sh` cubre una raiz explicita sin ninguna
variable global y exige que `ORQUESTA_TEST_PORT_LOCK_DIR` quede junto a esa
raiz. Verificacion: test de batches, guards F3, `bash -n` y `git diff --check`
verdes. La auditoria real pudo continuar y genero
`/tmp/orquesta-code-audit-20260711/auditoria_codigo_20260711T044218Z.json`.

Relacionado: [208AH](incidencia_orquesta_home_lleno_reapertura_subagente_2026-07-11.md).
