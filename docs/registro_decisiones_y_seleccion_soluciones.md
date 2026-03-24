<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Registro de decisiones y criterio de selección de soluciones

## Objetivo

Definir cómo debe usarse el registro de decisiones por proyecto en Orquesta y qué criterio seguir para seleccionar y dejar trazada una solución.

El registro no es un acta literaria ni un simple historial de ideas.
Es la memoria operativa de las decisiones que afectan al proyecto y que deben poder reconstruirse después.

## Base funcional

La funcionalidad ya existe en Orquesta y se apoya en:

- la entidad `DecisionProyecto` en `db/project_memory.go`
- la tabla `decisiones_proyecto`
- el servicio de memoria de proyecto en `memoriaproyecto/service.go`
- la vista web de proyecto en `cmd/proyectos_web.go`
- la API `GET/POST /api/proyectos/{slug}/decisiones`

## Qué guarda una decisión

Cada decisión de proyecto puede guardar:

- categoría
- título
- solución elegida
- motivo
- alternativas consideradas
- impacto
- estado
- propuesta relacionada opcional
- tarea relacionada opcional
- metadatos JSON opcionales

Campos obligatorios reales:

- `proyecto_id`
- `titulo`
- `solucion`

Valores por defecto si no se indican:

- `categoria = general`
- `estado = vigente`
- `metadata_json = {}`

## Estados válidos

Estados aceptados por la implementación actual:

- `vigente`
- `experimental`
- `reemplazada`
- `descartada`
- `archivada`

Interpretación recomendada:

- `vigente`
  solución activa que rige hoy
- `experimental`
  solución en evaluación o adopción parcial
- `reemplazada`
  decisión que fue válida pero ya tuvo sucesora
- `descartada`
  opción estudiada y rechazada explícitamente
- `archivada`
  decisión conservada por trazabilidad, sin vigencia operativa

## Regla de unicidad

La implementación actual resuelve unicidad por:

- `proyecto_id`
- `titulo`

Consecuencia:

- dentro del mismo proyecto, una decisión con el mismo título se actualiza
- el título debe ser estable y representar una unidad real de decisión

No conviene usar títulos ambiguos o cambiantes como:

- “tema varios”
- “pendientes”
- “cambios de hoy”

## Cuándo registrar una decisión

Conviene registrar una decisión cuando:

- se elige una solución entre varias opciones plausibles
- la elección condiciona arquitectura, operación o mantenimiento
- la decisión debe sobrevivir a un relevo de agente
- existe relación con una propuesta o una tarea concreta
- una elección deja de ser obvia y merece trazabilidad

No conviene registrar una decisión cuando:

- es una acción trivial o reversible sin impacto real
- solo existe una observación técnica sin elección asociada
- lo correcto es registrar una propuesta, no una solución ya tomada

## Criterio de selección de soluciones

La solución elegida no debe ser “la que alguien prefirió” sin más.

El criterio mínimo debería poder explicarse así:

1. qué problema concreto había que resolver
2. qué alternativas reales se consideraron
3. por qué la solución elegida fue mejor en este contexto
4. qué impacto operativo, técnico o de producto tiene

Por eso el registro incluye:

- `solucion`
- `motivo`
- `alternativas`
- `impacto`

## Regla práctica

Si una solución no puede resumirse de forma clara y no se puede comparar con al menos una alternativa plausible, probablemente todavía no está madura para registrarse como decisión final.

## Relación con propuestas y tareas

El registro de decisiones no sustituye ni a las propuestas ni a las tareas.

Encaje correcto:

- la propuesta decide o debate una dirección
- la tarea ejecuta trabajo concreto
- la decisión fija la solución elegida para el proyecto

Cuando exista vínculo real, conviene rellenar:

- `propuesta_id`
- `tarea_id`

Eso evita perder el puente entre:

- debate
- ejecución
- memoria final del proyecto

## Uso web

En la vista de detalle de proyecto existe un bloque de “Decisiones”.

Desde ahí se puede:

- listar decisiones ya registradas
- crear una nueva decisión
- asociarla opcionalmente a propuesta o tarea

Campos visibles en la UI:

- título
- categoría
- solución
- motivo
- alternativas
- impacto
- estado
- propuesta relacionada
- tarea relacionada

## Uso por API

Endpoints disponibles:

- `GET /api/proyectos/{slug}/decisiones`
- `POST /api/proyectos/{slug}/decisiones`

Payload mínimo útil:

```json
{
  "titulo": "Persistencia mediada",
  "solucion": "Usar servicio y repositorio en vez de SQL directo"
}
```

Payload completo posible:

```json
{
  "categoria": "arquitectura",
  "titulo": "Persistencia mediada",
  "solucion": "Usar servicio y repositorio en vez de SQL directo",
  "motivo": "Permite soportar SQLite y MySQL sin acoplar handlers",
  "alternativas": "SQL directo en handlers y utilidades",
  "impacto": "Reduce contención y facilita evolución de backend",
  "estado": "vigente",
  "propuesta_id": 56,
  "tarea_id": 226,
  "metadata_json": "{\"alcance\":\"control_plane\"}"
}
```

## Buenas prácticas

- usar un título estable y específico
- escribir la solución en positivo, no como queja
- explicar el motivo con contexto del proyecto, no con frases genéricas
- enumerar alternativas reales, aunque sea en una línea
- dejar el impacto en términos operativos o arquitectónicos
- marcar como `reemplazada` una decisión antigua en vez de borrarla mentalmente

## Qué no debe hacerse

- no usar el registro como backlog encubierto
- no registrar solo el problema sin solución
- no rellenar `alternativas` con texto vacío si sí hubo alternativas reales
- no cambiar el título constantemente y perder continuidad de la misma decisión
- no sustituir una propuesta abierta por una decisión cerrada prematuramente

## Criterio de aceptación

El registro está bien usado si:

- la solución elegida queda explícita
- se entiende por qué se eligió y frente a qué alternativas
- la decisión puede relacionarse con propuesta o tarea cuando aplica
- un agente nuevo puede reanudar trabajo sin reabrir el debate desde cero
