<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Plan de traslado físico de Orquesta a `~/Trabajo/orquestador`

## Objetivo

Mover el repositorio de Orquesta desde su ubicación actual en:

- `~/Trabajo/PlataformaMunicipal/orquestador`

a su ubicación objetivo:

- `~/Trabajo/orquestador`

sin perder continuidad operativa ni dejar sesiones, rutas o automatizaciones rotas.

## Restricción principal

El traslado no debe ejecutarse mientras haya agentes activos o sesiones vivas sobre la ruta actual.

Antes de mover nada, Orquesta debe poder demostrar:

- cero sesiones activas ligadas a la ruta actual
- cero handoffs pendientes sobre esa ruta
- worktrees y ramas identificados
- backup reciente de base de datos y repositorio

## Riesgos a controlar

- sesiones externas que conservan `cwd` o rutas antiguas
- scripts de arranque apuntando al path viejo
- documentación operativa desactualizada
- worktrees asociados a la ruta antigua
- rutas persistidas en sesiones, proyectos o configuración local

## Precondiciones

1. `orquesta status` sin agentes activos sobre la ruta actual.
2. `orquesta sesion listar` sin sesiones vivas que apunten al directorio antiguo.
3. Backup de:
   - `orquesta.db`
   - repositorio Git
   - configuración local relevante
4. Worktrees, ramas y cambios sin commit identificados.
5. Registro de continuidad guardado para cualquier sesión que deba reanudarse después.

## Flujo recomendado

## Fase 1. Congelación operativa

- no arrancar agentes nuevos sobre la ruta antigua
- cerrar o pausar agentes existentes
- guardar `external_session_id` y `resumen_continuidad`
- verificar que no queden órdenes de runtime pendientes de depender de la ruta antigua

## Fase 2. Verificación previa

- confirmar `git status` comprensible y conocido
- revisar `worktrees`
- revisar configuración y scripts que mencionen la ruta antigua
- revisar documentación que instruya a arrancar Orquesta desde el path anterior

## Fase 3. Movimiento

- mover el repositorio a `~/Trabajo/orquestador`
- validar que `.git`, worktrees y ramas conservan integridad
- ajustar scripts, rutas de servicio y documentación

## Fase 4. Normalización posterior

- actualizar `cwd` por defecto en wrappers y manuales
- revisar referencias persistidas que deban apuntar a la ruta nueva
- arrancar Orquesta desde la ubicación nueva
- validar CLI, API y panel web

## Fase 5. Reanudación

- reabrir sesiones necesarias desde la nueva ubicación
- comprobar que los agentes reciben continuidad correcta
- confirmar que no quedan referencias activas a la ruta anterior

## Checklist mínimo

- `status` limpio o al menos conocido
- `sesion listar` sin activas en la ruta antigua
- backup de BD hecho
- backup o referencia Git hecha
- wrappers actualizados
- documentación actualizada
- panel y CLI verificados tras el movimiento

## Qué debe actualizarse

- documentación de uso actual
- operativa de agentes manuales
- scripts de arranque
- posibles rutas en configuración local
- cualquier automatización externa que invoque Orquesta por path absoluto

## Qué no debe hacerse

- mover el repo “en caliente”
- confiar en que `resume --last` arreglará sesiones ligadas a la ruta antigua
- dejar scripts o docs mezclando ruta vieja y ruta nueva
- hacer el traslado sin backup previo

## Criterio de cierre

El traslado estará bien planificado si el equipo puede ejecutarlo con una secuencia explícita, verificable y reversible, sin depender de memoria informal del operador.
