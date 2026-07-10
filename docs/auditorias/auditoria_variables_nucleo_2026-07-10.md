# Auditoria de variables del nucleo - 2026-07-10

## Alcance y evidencia

Esta auditoria cubre solo `cmd/orquesta-server`, la composicion local del
nucleo. No clasifica todavia conectores OPES, herramientas documentales ni
workloads historicos.

La prueba AST
`TestServerEnvRegistryASTV0LecturasORQUESTARegistradas` contabilizaba 105
lecturas `ORQUESTA_*` sin metadata efectiva. La primera ola de registro de
runtime, Director Operativo, supervisor, arranque y review las reduce a 59.
La segunda ola consolida Gemini en `gemini_runtime` y el runner de tests
requeridos en `required_test_runner`. Elimina construcciones duplicadas y
reduce el ratchet a 36. Una lectura nueva sin clasificar vuelve rojo.

## Decisiones

- `orquesta.config.json` es la superficie canónica ya existente. El esquema
  es tipado y falla ante claves desconocidas; no se introduce un mapa genérico
  de variables que oculte errores de escritura.
- La precedencia vigente se conserva: variable explícita, fichero canónico y
  finalmente default. La retirada de aliases solo será posible cuando cada
  sección tenga prueba de precedencia y proyección a `effective_config`.
- Las variables de runtime/Director/Supervisor registradas en
  `server_env_registry_runtime_v0.go` pasan a ser inventariables y visibles
  como configuración efectiva. No cambia su valor ni su default en este
  corte.
- `gemini_runtime` ya concentra `enabled`, comando, directorios,
  `HOME`, `PATH`, modelo, aprobacion, formato y argumentos. El backend
  goal-first reutiliza la misma resolucion y conserva
  `env > fichero > default`, incluido el override explicito
  `ORQUESTA_GEMINI_ENABLED=false`.
- `required_test_runner` ya concentra su opt-in, allowlist, directorio de
  evidencia, entorno proyectado y limites. La seccion solo construye el
  ejecutor aislado; no ejecuta tests por configurarse.
- `autoprogramming.promotion` ya concentra su opt-in, archivo, refs opacas y
  mensaje de commit. `self_programming_only` valida el archivo resuelto por
  la misma configuracion antes de permitirlo.

## Residual clasificado para la fase de limpieza

| Familia | Tratamiento posterior |
| --- | --- |
| Gemini | Primera familia cerrada localmente: seccion tipada, perfil unico, registro de metadatos y ratchet 59 -> 49. |
| Runner de tests requeridos | Segunda familia cerrada localmente: seccion tipada, allowlist/entorno deterministas y ratchet 49 -> 42. |
| Promocion de autoprogramacion | Tercera familia cerrada localmente: seccion tipada, guard de archivo compartido y ratchet 42 -> 36. |
| Otros proveedores opt-in | Secciones tipadas de configuración y retiro de lecturas directas por proveedor. |
| OPES y domain-work | Fuera del núcleo; revisar después de cerrar autonomía local. |
| Runner independiente de tests requeridos | Registrar como sección de atestación y probar aislamiento. |
| Promoción/guardian | Cuarta familia cerrada localmente: sección tipada, precedencia comprobada y metadatos de las 29 entradas; el runner hijo conserva su allowlist. |
| MCP smoke/harness | Declarar como harness o retirar si no tiene consumidor. |

No se ha eliminado ninguna variable ni fichero. La limpieza posterior debe
trabajar por familia, con búsqueda de referencias, prueba focal y commit
recuperable por cada retirada.

## Actualizacion Codex 2026-07-10: guardian de promocion tipado

Se consolido `autoprogramming.promotion.guardian` dentro del mismo fichero
canonico. La seccion tipada cubre opt-in, comando, allowlist del runner, rutas
de estado y binarios, build/tests/health, limites, reparacion Codex y las
referencias de evidencia. El entorno conserva prioridad explicita para
compatibilidad; una variable `enabled` explicita desactiva el guardian aunque
el fichero lo tenga habilitado.

Las 29 variables de la familia quedan registradas con alcance
`autoprogramming_promotion_guardian`. El registro no expone valores: documenta
la configuracion efectiva y el runner hijo sigue proyectando unicamente su
allowlist. No se ejecuto promotion, guardian, servidor, agente ni proveedor.

Los timeout canonicos de la familia usan ahora `*_TIMEOUT_MS`; los nombres
historicos sin unidad se mantienen solo como aliases deprecados que aceptan
duracion Go. La precedencia comprobada es canonico, alias legacy, fichero y
default. La extraccion de las estructuras de configuracion y de las
proyecciones efectivas tambien mantiene ambos ficheros bajo el limite T90.

Pruebas locales:

```bash
go test -count=1 ./cmd/orquesta-server \
  -run 'TestAutoprogrammingPromotionGuardian(ConfigFileCanonicoV0|DistingueConfigInvalidaV0|DistinguePathPolicyInvalidaV0|EnvV0ConservaRefsOpacas|RepairCodexConfigFromEnvV0|RunnerEnvV0NoHeredarCredencialesPorDefecto|RunnerEnvV0PermiteOptInExplicito|RunnerEnvV0ConservaSoloAllowlistYVarsGuardian|RunnerEnvV0NoHeredaVarsGuardianDelPadre|EnvV0TodasClavesRegistradasComoChildProcess|ChildEnvRegistryV0NoPublicaValoresCrudos)'
```

Resultado: verde. La proyeccion efectiva de Gemini, runner de tests requeridos
y promotion tambien queda cubierta; el ratchet AST baja de 36 a 5. Las cinco
lecturas residuales pertenecen a OPES, conector fuera de este corte. El
harness opt-in `ORQUESTA_MCP_REAL_SMOKE_CONFIRM` queda clasificado
explícitamente y no forma parte de configuración de runtime.

## Actualizacion Codex 2026-07-11: cierre de configuracion OPES y ratchet cero

La familia OPES que quedaba fuera de la primera ola se consolido en el snapshot
tipado `serverOPESConfigSnapshotV0`: workdir, timeout HTTP e intentos REST usan
la misma precedencia `env > orquesta.config.json > default`, con metadata
efectiva. El guard de workdir, el conector `domain_work` y la deteccion de
contexto OPES consumen ese snapshot, sin cambiar el opt-in ni ejecutar OPES.

La secuencia del bridge OPES se migro a la utilidad generica de listas del
fichero de configuracion. Por tanto
`TestServerEnvRegistryASTV0LecturasORQUESTARegistradas` baja de cinco lecturas
residuales a cero y su baseline queda en `0`: una nueva lectura de
`ORQUESTA_*` no registrada falla la prueba.

En la misma pasada se retiraron cinco wrappers privados sin consumidores y se
movio el helper comun de listas fuera de `domain_work`. Se verificaron los
focales de configuracion/bridge y los paquetes runtime Codex/state-file. No se
arranco servidor, bridge, OPES, agente ni proveedor.
