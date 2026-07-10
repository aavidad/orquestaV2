# Auditoria de variables del nucleo - 2026-07-10

## Alcance y evidencia

Esta auditoria cubre solo `cmd/orquesta-server`, la composicion local del
nucleo. No clasifica todavia conectores OPES, herramientas documentales ni
workloads historicos.

La prueba AST
`TestServerEnvRegistryASTV0LecturasORQUESTARegistradas` contabilizaba 105
lecturas `ORQUESTA_*` sin metadata efectiva. La primera ola de registro de
runtime, Director Operativo, supervisor, arranque y review las reduce a 59.
El nuevo ratchet queda en 59: una lectura nueva sin clasificar vuelve rojo.

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

## Residual clasificado para la fase de limpieza

| Familia | Tratamiento posterior |
| --- | --- |
| Gemini y otros proveedores opt-in | Secciones tipadas de configuración y retiro de lecturas directas por proveedor. |
| OPES y domain-work | Fuera del núcleo; revisar después de cerrar autonomía local. |
| Runner independiente de tests requeridos | Registrar como sección de atestación y probar aislamiento. |
| Promoción/guardian | Separar inputs del servidor de variables exclusivas de proceso-hijo, con registros distintos. |
| MCP smoke/harness | Declarar como harness o retirar si no tiene consumidor. |

No se ha eliminado ninguna variable ni fichero. La limpieza posterior debe
trabajar por familia, con búsqueda de referencias, prueba focal y commit
recuperable por cada retirada.
