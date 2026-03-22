<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Política de Copias de Seguridad de la Base de Datos

## Principio

La base de datos de Orquesta es la fuente de verdad de todo el ecosistema de agentes: tareas, propuestas, sesiones, reglas, decisiones y auditoría. Su pérdida o corrupción tendría un impacto irreversible sobre el desarrollo de todos los proyectos. Por ello, la realización de copias de seguridad periódicas es **obligatoria**.

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

1. Parar todos los agentes activos (`orquesta sesion fin <agente>`).
2. Hacer una copia de la BD actual antes de restaurar (por si la restauración empeora el estado).
3. Copiar el fichero `.db.bak` deseado sobre `orquesta.db`.
4. Verificar la integridad con `orquesta status`.
5. Reiniciar las sesiones de los agentes.

## Trazabilidad

Toda operación de backup (creación y restauración) debe quedar registrada como **nota** en el Orquestador indicando:
- Fecha y hora.
- Tipo de operación (creación / restauración).
- Ruta del fichero implicado.
- Agente o proceso responsable.

## Implementación Técnica

La implementación del script o tarea programada que materializa estos backups es responsabilidad del agente programador asignado a la **Tarea #217**. La presente política define los requisitos, no la implementación.
