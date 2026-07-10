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
reduce el ratchet a 42. Una lectura nueva sin clasificar vuelve rojo.

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

## Residual clasificado para la fase de limpieza

| Familia | Tratamiento posterior |
| --- | --- |
| Gemini | Primera familia cerrada localmente: seccion tipada, perfil unico, registro de metadatos y ratchet 59 -> 49. |
| Runner de tests requeridos | Segunda familia cerrada localmente: seccion tipada, allowlist/entorno deterministas y ratchet 49 -> 42. |
| Otros proveedores opt-in | Secciones tipadas de configuración y retiro de lecturas directas por proveedor. |
| OPES y domain-work | Fuera del núcleo; revisar después de cerrar autonomía local. |
| Runner independiente de tests requeridos | Registrar como sección de atestación y probar aislamiento. |
| Promoción/guardian | Separar inputs del servidor de variables exclusivas de proceso-hijo, con registros distintos. |
| MCP smoke/harness | Declarar como harness o retirar si no tiene consumidor. |

No se ha eliminado ninguna variable ni fichero. La limpieza posterior debe
trabajar por familia, con búsqueda de referencias, prueba focal y commit
recuperable por cada retirada.
