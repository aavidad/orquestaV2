<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Política de Copias de Seguridad del Backend SQLite

> Nota doctrinal: esta política cubre el backend SQLite.
> La vía operativa oficial para snapshots manuales es `orquesta respaldo bd`.
> La verificación operativa normal vive en `orquesta persistencia verificar`.
> Si el backend activo no es SQLite, debe existir una política equivalente específica del motor.

## Principio

Cuando el backend activo es SQLite, la base de datos de Orquesta sigue siendo la fuente de verdad de todo el ecosistema de agentes: tareas, propuestas, sesiones, reglas, decisiones y auditoría. Su pérdida o corrupción tendría un impacto irreversible sobre el desarrollo de todos los proyectos. Por ello, la realización de copias de seguridad periódicas es **obligatoria**.

## Frecuencia y Retención

| Tipo          | Frecuencia   | Retención mínima |
|---------------|-------------|-----------------|
| Diaria        | Cada 24 h   | 7 días          |
| Semanal       | Cada lunes  | 4 semanas       |
| Mensual       | Día 1 de cada mes | 3 meses   |

## Ruta de Almacenamiento

Las copias de seguridad deben alojarse **fuera del directorio de trabajo** del orquestador para que un borrado accidental del repositorio no se lleve también el backup:

```
~/Trabajo/backups/orquestador/
  └── YYYY-MM-DD_HH-MM-SS_orquesta.db.bak
```

## Procedimiento de Restauración

1. Parar el daemon y los agentes activos de forma ordenada.
2. Hacer una copia del fichero SQLite actual antes de restaurar.
3. Restaurar el `.db.bak` deseado sobre el fichero SQLite activo.
4. Verificar con `orquesta persistencia info` y `orquesta persistencia verificar`.
5. Volver a levantar el daemon y reanudar la operativa.

## Trazabilidad

Toda operación de backup (creación y restauración) debe quedar registrada como **nota** en el Orquestador indicando:
- Fecha y hora.
- Tipo de operación (creación / restauración).
- Ruta del fichero implicado.
- Agente o proceso responsable.

## Implementación Técnica

La implementación técnica de referencia puede apoyarse en:
- `orquesta respaldo bd` para snapshots manuales por el camino oficial
- `scripts/backup_bd.sh` y `scripts/verificar_bd.sh` únicamente como automatización/rescate específica de SQLite

La presente política define los requisitos, no la implementación.
