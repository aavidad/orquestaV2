<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Orquesta v1 — Roadmap de implementacion

## Objetivo del roadmap

Convertir la vision de `OP-049` en un orden de trabajo que reduzca riesgo, preserve datos y permita avanzar con varios agentes en paralelo.

## Principios de ejecucion

1. Ninguna migracion destructiva.
2. Cada bloque debe dejar el sistema mejor que antes.
3. La API y la CLI deben crecer sobre los mismos casos de uso.
4. La concurrencia contra SQLite debe reducirse, no aumentar.
5. Las capacidades de alto riesgo se activan despues de tener trazabilidad y control.
6. La hexagonalizacion es gate: no se debe seguir metiendo logica de negocio nueva en `db/` para acelerar la web, la API o la futura app de escritorio.

## Bloque 1 — Consolidacion del modelo ya abierto

### Objetivo

Cerrar correctamente la base ya implementada de:

- proyectos
- asignaciones
- sesiones reanudables
- conectores
- locks
- worktrees
- API JSON base

### Entregables

- limpieza de puertos y adaptadores
- repositorios SQLite coherentes
- comandos CLI consistentes
- tests de humo de sesiones, locks y worktrees
- cierre de voto de `OP-050`, `OP-051` y `OP-052` sin duplicidades activas
- congelacion del traslado fisico del repo mientras existan agentes activos en la ruta actual

## Bloque 2 — Hexagonalizacion progresiva

### Objetivo

Sacar del paquete `db` la logica que aun actua como nucleo.

### Prioridad real

Este bloque actua como prerequisito de arquitectura.
Se puede seguir corrigiendo o completando superficie visible, pero no debe crecer la deuda estructural.
Antes de ampliar en serio:

- panel web completo
- control activo de agentes vivos
- app de escritorio

hay que mover casos de uso y reglas de negocio fuera de `db/`.

### Entregables

- servicios de aplicacion para propuestas, tareas, sesiones y proyectos
- puertos de persistencia claros
- adaptadores SQLite reducidos a persistencia

## Bloque 3 — Memoria y conocimiento

### Objetivo

Igualar la parte fuerte de knowledge capture y prepararse para la fase de investigacion.

### Entregables

- `memoria_proyecto`
- `memoria_tarea`
- `fuentes`
- `hallazgos`
- `decisiones`
- API para registrar y consultar conocimiento

## Bloque 4 — Deteccion de deriva

### Objetivo

Detectar cuando una sesion o una rama se desvian de:

- la tarea
- la propuesta
- la fase
- la rama esperada

### Entregables

- modelo de deriva
- verificacion en `tick`
- alertas o bloqueos de integracion

## Bloque 5 — Fases y progreso

### Objetivo

Hacer que proyecto y tarea tengan fase explicita y progreso calculable.

### Entregables

- catalogo de fases
- estado de fase por proyecto
- estado de fase por tarea
- formula de progreso basada en tareas, gates y ramas

## Bloque 6 — Gobierno Git

### Objetivo

Dar visibilidad y control real sobre ramas, commits, push y merge.

### Entregables

- inventario de repositorios por proyecto
- ramas por agente
- snapshots Git
- estado `dirty`, `ahead`, `behind`
- merges orquestados y gates previos

## Bloque 7 — MCP y conectores avanzados

### Objetivo

Servir contexto, reglas y operaciones de Orquesta mediante MCP y reforzar la abstraccion de runtimes.

### Entregables

- servidor MCP local
- `resources`
- `prompts`
- `tools`
- conectores `mcp_stdio` y `mcp_http`

## Bloque 8 — Investigacion como producto

### Objetivo

Convertir la fase de investigacion en una capacidad de primer nivel.

### Entregables

- workflows de investigacion
- registro de fuentes
- sintesis estructurada
- integracion con memoria y planificacion

## Bloque 9 — Operacion manual y runtime

### Objetivo

Cerrar bien el ciclo practico de agentes manuales y semiautomaticos.

### Entregables

- wrappers de consola estables
- persistencia determinista de `external_session_id`
- soporte Terminator
- arranque, pausa y reanudacion desde Orquesta
- control de presupuesto restante por sesion
- checkpoint y relevo preventivo segun politica aprobada

## Bloque 10 — Refactorizacion competitiva

### Objetivo

Permitir experimentos controlados con varias variantes y seleccion rigurosa.

### Entregables

- experimentos de refactor
- candidatos aislados
- backups previos
- scorecards
- revision por al menos dos no autores

## Bloque 11 — Web y app de escritorio

### Objetivo

Hacer usable toda la orquestacion sin depender de la terminal.

### Entregables

- panel de agentes
- panel de proyectos
- panel de propuestas y votos
- panel de Git y merges
- panel de fases y progreso

## Reparto recomendado entre agentes

### Codex1

- documentacion
- arquitectura
- orden de implementacion
- integracion transversal

### Codex2

- MCP
- conectores de runtime
- capa de prompts/resources/tools

### Codex3

- memoria
- fuentes
- hallazgos
- deriva

## Gates por bloque

Cada bloque debe cerrar con:

- build correcto
- tests correctos
- migraciones aditivas verificadas
- documentacion actualizada
- propuesta o voto si hay decision estructural
