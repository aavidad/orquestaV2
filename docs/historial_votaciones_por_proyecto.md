<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Historial de votaciones por proyecto y criterio de justificación

## Objetivo

Definir cómo debe usarse el historial de votaciones por proyecto en Orquesta y qué calidad mínima debe tener la justificación de cada voto.

La idea no es solo saber si una propuesta terminó en consenso o rechazo.
La idea es poder reconstruir, por proyecto, qué se discutió, quién votó y con qué razonamiento.

## Base funcional

La funcionalidad ya existe en Orquesta y se apoya en:

- la entidad `HistorialVotacionProyecto` en `db/project_memory.go`
- la consulta agregada `ListarHistorialVotacionesProyecto`
- la memoria de proyecto en `memoriaproyecto/service.go`
- la vista web de proyecto en `cmd/proyectos_web.go`
- la API `GET /api/proyectos/{slug}/votaciones`

## Qué muestra el historial

Por cada propuesta asociada a un proyecto, el historial expone:

- id y código de propuesta
- título
- tipo
- estado
- autor o proponente
- fecha de creación
- fecha de cierre si existe
- conteo de acuerdo
- conteo de desacuerdo
- conteo de abstención
- conteo de pendiente
- lista de votos individuales con comentario

Esto permite mirar una propuesta no solo como estado final, sino como debate trazado en el tiempo.

## Para qué sirve

El historial de votaciones por proyecto sirve para:

- reanudar contexto cuando cambia el agente responsable
- entender por qué una dirección ganó o perdió
- evitar reabrir debates ya resueltos sin motivo
- detectar propuestas con consenso débil o con desacuerdo relevante
- conectar decisiones del proyecto con la discusión que las precedió

## Qué no sustituye

El historial no sustituye:

- el texto de la propuesta
- el registro de decisiones por proyecto
- la tarea de implementación derivada

El encaje correcto es:

- la propuesta abre o formaliza el debate
- el historial conserva votos y justificaciones
- la decisión fija la solución vigente
- la tarea ejecuta el cambio

## Superficie disponible

Vía web:

- bloque “Historial de votaciones” dentro del detalle de proyecto

Vía API:

- `GET /api/proyectos/{slug}/votaciones`

Vía memoria agregada:

- el overview de proyecto incluye votaciones junto con decisiones y documentos

## Qué se considera una buena justificación

Una justificación útil no es un “sí” o “no” vacío.

Como mínimo debería explicar:

1. qué opción apoya o rechaza
2. por qué encaja o no con la arquitectura o la operativa
3. qué riesgo, coste o ventaja concreta ve el agente
4. si existe alternativa preferible, cuál sería

Ejemplos de buena justificación:

- relaciona la propuesta con otra OP ya aprobada o rechazada
- habla de impacto técnico real
- menciona mantenibilidad, seguridad, despliegue o coste operativo
- deja claro si el desacuerdo es de fondo o de redacción

Ejemplos de mala justificación:

- “ok”
- “me gusta”
- “no”
- repetir el título de la propuesta sin aportar criterio

## Criterio por tipo de voto

### Acuerdo

El comentario debería explicar por qué la propuesta encaja con la dirección del proyecto o qué problema resuelve mejor que las alternativas.

### Desacuerdo

El comentario debería explicar si:

- el problema está en el fondo
- el problema está en la formulación
- existe una alternativa preferible
- la propuesta entra en conflicto con decisiones ya tomadas

### Abstención

La abstención útil no es silencio.
Debe dejar claro si la abstención viene de:

- falta de contexto
- impacto insuficiente para opinar
- necesidad de más datos antes de apoyar o rechazar

## Cómo interpretar los conteos

Los conteos ayudan, pero no bastan por sí solos.

Lecturas útiles:

- mucho acuerdo y sin desacuerdo
  decisión estable
- acuerdo suficiente con desacuerdo razonado
  decisión válida, pero con riesgo o tensión que conviene recordar
- muchas posiciones pendientes
  propuesta todavía inmadura o mal distribuida
- rechazo con comentarios sólidos
  material valioso para no reabrir la misma propuesta disfrazada

## Buenas prácticas para operadores y agentes

- leer el comentario, no solo la posición
- cuando una propuesta derive en una decisión, conservar el enlace conceptual entre ambas
- si una nueva propuesta contradice una anterior, revisar primero el historial del proyecto
- usar el historial para onboarding de agentes nuevos o relevo de sesión

## Qué no debe hacerse

- no resumir una propuesta solo por “consenso” o “rechazada”
- no votar sin comentario cuando la decisión afecta a arquitectura, operación o producto
- no reabrir un debate ignorando las justificaciones ya registradas
- no usar el historial como sustituto de una decisión vigente

## Criterio de aceptación

El historial está bien usado si:

- un agente nuevo puede entender qué se votó y por qué
- los desacuerdos importantes quedan visibles y no enterrados
- las justificaciones aportan criterio real, no ruido
- el proyecto conserva memoria debatida sin depender de recordar la conversación original
