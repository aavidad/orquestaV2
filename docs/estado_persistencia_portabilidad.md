# Estado de portabilidad de persistencia

## Estado actual

- `storage` ya permite elegir driver con `ORQUESTA_DB_DRIVER` y `ORQUESTA_DB_DSN`.
- Hay rebinding de placeholders para `postgres`.
- `sqlite` sigue siendo el camino por defecto y el unico con bootstrap automatico completo.

## Lo que ya esta portable

- La apertura de conexion no esta fijada a SQLite.
- El acceso SQL pasa por un wrapper que rebindea queries segun driver.

## Lo que falta para un backend alternativo real

- DDL y migraciones por dialecto, no una unica schema SQLite-first.
- Sustituir dependencias SQLite-especificas: `sqlite_master`, `PRAGMA`, `AUTOINCREMENT`, `WAL` y bootstrap implicito.
- Validar en pruebas de integracion un backend externo real, no solo resolucion de config.
- Separar bien semantica de tipos, constraints e indices por motor.

## Resumen corto

Hoy hay soporte de conexion multi-driver, pero no portabilidad completa de persistencia. Para decir que existe un backend alternativo real falta dialecto, migraciones y cobertura de pruebas por motor.
