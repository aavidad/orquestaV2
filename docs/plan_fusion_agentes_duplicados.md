<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Plan de fusión de agentes duplicados por mayúsculas y minúsculas

## Objetivo

Resolver duplicados del tipo:

- `Codex1` / `codex1`
- `Codex2` / `codex2`
- `codex` frente al esquema canónico `CodexN`

Sin perder trazabilidad ni romper referencias.

## Decisión de nombre canónico

La convención que ya usa el equipo y la documentación es:

- `Codex1`
- `Codex2`
- `Codex3`
- `Codex4`

Por tanto:

- el formato canónico para agentes Codex numerados debe ser `CodexN`
- no deben generarse nuevos agentes automáticos en minúsculas

## Estado actual

La parte preventiva ya puede endurecerse en código:

- el alta automática debe producir `CodexN`
- no debe seguir creando `codexN`

La fusión histórica de registros existentes ya dispone de operación explícita en la app:

- `orquesta agente fusionar <origen> <destino>`
- `POST /api/agentes/{origen}/fusionar`

Sigue siendo una operación delicada, pero deja de depender de SQL manual.

## Riesgo principal

El nombre del agente aparece como clave referenciada por varias tablas.

Superficies mínimas conocidas:

- `agentes`
- `sesiones`
- `tareas`
- `audit_log`
- `votos`
- `asignaciones`
- `worktrees`
- tablas runtime con columnas `agente`, `from_agente` y `to_agente`

Por eso no conviene resolverlo con borrados sueltos ni con SQL manual ad hoc.

## Precondiciones

Antes de ejecutar la fusión real:

1. backup completo de la BD
2. lista cerrada de parejas a fusionar
3. nombre canónico decidido por pareja
4. agentes implicados sin sesiones activas
5. plan transaccional único
6. validación posterior de integridad

## Estrategia recomendada

### Fase 1. Prevención

- corregir el alta automática para que use `CodexN`
- impedir que la app siga generando nuevos duplicados

### Fase 2. Inventario

Para cada agente no canónico:

- contar sesiones
- contar tareas
- contar votos
- contar asignaciones
- contar worktrees
- contar referencias runtime

Resultado esperado:

- mapa exacto `origen -> destino`

### Fase 3. Fusión transaccional

Para cada pareja `origen -> destino`:

1. verificar que `destino` existe
2. mover referencias de `origen` a `destino`
3. resolver colisiones si una misma tabla permite conflicto lógico
4. auditar la operación
5. eliminar o retirar `origen` solo al final

Todo dentro de transacción, o al menos por bloque atómico bien delimitado.

### Fase 4. Validación

Comprobar:

- que no quedan referencias al nombre viejo
- que las sesiones siguen siendo legibles
- que las tareas siguen apuntando al agente correcto
- que el historial y auditoría no se han roto
- que los worktrees siguen resolviendo

## Política de colisiones

Hay que decidir explícitamente cómo actuar si:

- ambos nombres tienen tarea activa
- ambos tienen sesión activa o reciente
- ambos tienen votos o asignaciones que colisionan en la lógica de negocio

Regla conservadora recomendada:

- no fusionar automáticamente parejas con actividad viva o ambigua
- dejar esas parejas en estado revisable y resolverlas con criterio manual

## Implementación mínima ya disponible en la app

La app ya ofrece una operación explícita y no improvisada:

- backup previo obligatorio
- fusión `origen -> destino`
- validación y auditoría

El alcance actual es conservador:

- fusiona referencias históricas
- descarta el voto del origen si el destino ya había votado la misma propuesta
- aborta si el origen conserva actividad viva o ambigua

La parte que sigue pendiente como mejora futura es una previsualización más rica del impacto antes de ejecutar.

## Relación con la tarea actual

La tarea `#160` queda cubierta por dos frentes ya implementados:

- prevención de nuevos duplicados
- fusión histórica segura con backup y transacción
