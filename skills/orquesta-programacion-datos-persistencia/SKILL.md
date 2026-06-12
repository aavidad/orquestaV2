---
name: orquesta-programacion-datos-persistencia
description: Trabajar con datos y persistencia en apps consumidoras de Orquesta sin contaminar el core: schemas, migraciones, stores, backup, replay, idempotencia y conectores opt-in.
---

# Orquesta Programacion Datos Persistencia

Usa esta skill cuando una tarea toque stores, ficheros durables, SQL, JSONL,
backups, migraciones, replay o deduplicacion.

## Frontera

- El core puro no conoce DB concreta.
- La persistencia entra por puerto/adaptador de composicion.
- Las apps externas no comparten DB ni filesystem interno con Orquesta.

## Reglas

- Definir owner del dato y formato versionado.
- Mantener idempotencia para writes repetidos.
- Usar refs opacas en eventos publicos.
- Redactar rutas, DSN, tokens y payloads sensibles en errores.
- Crear backups antes de migraciones o cambios destructivos.
- No borrar datos legacy sin inventario y evidencia.

## Pruebas

- Store en memoria/fake.
- Store file/SQL si existe.
- Replay.
- Conflicto de ref con payload distinto.
- Recuperacion tras recrear instancia.

## Entrega

Devuelve schema/formato, migracion si aplica, backup, pruebas y plan de rollback.
