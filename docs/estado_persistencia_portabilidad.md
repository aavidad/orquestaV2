# Estado de portabilidad de persistencia

## Estado actual

- `storage` ya permite elegir driver con `ORQUESTA_DB_DRIVER` y `ORQUESTA_DB_DSN`.
- Hay rebinding de placeholders para `postgres`.
- `sqlite` ya no debe tratarse como backend operativo de referencia.
- El backend vivo debe entrar por driver/configuracion (`postgres`, `mysql`, etc.) y por SQL compatible con el motor real.

## Regla operativa vigente

- `SQLite` puede seguir existiendo como backend de compatibilidad, pruebas o fixtures locales.
- `SQLite` no debe volver a asumirse como fuente de verdad operativa del sistema.
- Ningun helper nuevo de schema, bootstrap o migracion puede meter DDL `sqlite-first` en rutas vivas del control plane.
- Toda compatibilidad nueva debe ser `driver-aware`: placeholders, tipos, constraints, indices y migraciones.

## Estado real por backend

- `Postgres`: backend operativo prioritario y el que debe usarse para endurecer la compatibilidad real del sistema.
- `MySQL/MariaDB`: no deben considerarse hoy backend operativo equivalente a Postgres; siguen existiendo huecos en `post-migrations`, `upsert`, `RETURNING id` y DDL auxiliar heredado.
- `SQLite`: legado local y compatibilidad, no referencia semantica del estado vivo.

## Lo que ya esta portable

- La apertura de conexion no esta fijada a SQLite.
- El acceso SQL pasa por un wrapper que rebindea queries segun driver.

## Lo que falta para un backend alternativo real

- DDL y migraciones por dialecto, no una unica schema SQLite-first.
- Sustituir dependencias SQLite-especificas: `sqlite_master`, `PRAGMA`, `AUTOINCREMENT`, `WAL` y bootstrap implicito.
- Validar en pruebas de integracion un backend externo real, no solo resolucion de config.
- Separar bien semantica de tipos, constraints e indices por motor.
- Cerrar el inventario de `ensure...Schema()` heredados, `ON CONFLICT ... excluded` crudos y helpers de `RETURNING id` que aun no tienen carril portable completo.

## Resumen corto

Hoy hay soporte de conexion multi-driver, pero no portabilidad completa de persistencia. Para decir que existe un backend alternativo real falta dialecto, migraciones y cobertura de pruebas por motor. La regla practica desde ahora es: no asumir SQLite en ningun camino operativo nuevo.
