<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño de app de escritorio sobre la API de Orquesta

## Objetivo

Definir una app de escritorio para Orquesta sin crear una segunda lógica de negocio ni una vía lateral a SQLite.

La app de escritorio debe ser un cliente rico para operadores, pero seguir siendo un cliente fino desde el punto de vista arquitectónico:

- usa la misma API HTTP/JSON que la web y la CLI cuando corresponda
- reutiliza los mismos casos de uso y contratos ya expuestos por Orquesta
- no introduce acceso directo a persistencia
- no crea un protocolo paralelo ad hoc para control de agentes

## Base reutilizada

Este diseño se apoya en:

- `docs/orquesta_v1_vision.md`
- `docs/uso_actual_app_orquesta.md`
- `docs/analisis_repos_control_agentes_2026-03-23.md`
- `docs/diseno_control_activo_agentes.md`
- `docs/op_087_autogestion_supervisada_agentes.md`

La conclusión común de esos documentos es estable:

- Orquesta debe tener un plano de control único
- web, CLI y desktop deben apoyarse en la misma API
- el control activo de agentes vive sobre `runtime_handles`, `runtime_orders`, `runtime_mailbox` y `runtime_checkpoints`

## Referencias de apps y repos ya analizados

Antes de fijar este diseño, Orquesta ya había estudiado referencias externas y repos comparables en `docs/analisis_repos_control_agentes_2026-03-23.md`.

Las conclusiones que sí afectan al diseño desktop son estas:

- el cliente debe quedarse en la capa de presentación y coordinación de flujos
- el control plane real debe vivir en Orquesta y no en la app
- la UI puede ser rica y densa, pero no debe acoplarse a PTYs, terminales ni persistencia directa
- `Tauri` o `Wails` son razonables para un spike, pero la decisión final debe quedar subordinada al contrato API y no al framework

## Qué problema resuelve la app desktop

La web cubre seguimiento y operación básica, pero una app de escritorio puede aportar mejor experiencia para:

- observación permanente del estado del workspace
- trabajo multi-panel y multi-ventana
- seguimiento de agentes, tareas, propuestas y runtimes a la vez
- notificaciones de escritorio
- vistas operativas densas sin dependencia del navegador
- futura integración con transportes locales controlados por el sistema operativo

No debe existir para saltarse la API ni para “hablar con la BD más rápido”.

## Principios no negociables

### 1. Cliente fino

La app desktop nunca accede a la base de datos de Orquesta de forma directa.

Todo pasa por:

- API HTTP/JSON
- contratos de entrada ya definidos
- endpoints nuevos solo si también sirven para web/CLI o automatización

### 2. Misma lógica, distinta presentación

La desktop no reimplementa reglas de negocio.

Su responsabilidad es:

- renderizar estado
- componer flujos de operador
- llamar a la API
- cachear estado efímero de UI

### 3. Offline limitado y explícito

No debe inventarse un modo offline con escrituras diferidas sobre tareas, propuestas o agentes.

Como máximo:

- caché local de lectura
- reintento de peticiones fallidas de bajo riesgo
- cola temporal de acciones de UI aún no enviadas, siempre visible al operador

### 4. Seguridad y trazabilidad

Toda acción de la app debe seguir dejando rastro en Orquesta:

- auditoría
- tarea
- runtime order
- propuesta
- sesión

## Arquitectura propuesta

## 1. Capas del cliente

### Presentación

- ventanas y paneles
- navegación
- tablas, timeline, formularios y vistas de detalle
- sistema de notificaciones

### Aplicación cliente

- adaptadores para consumir la API de Orquesta
- composición de flujos de operador
- sincronización de estado de interfaz
- traducción entre modelos de API y modelos de vista

### Infraestructura local mínima

- almacenamiento local solo para preferencias
- caché de lectura
- tokens o configuración del endpoint
- logs locales de cliente

## 2. Módulos de la app

Módulos mínimos recomendados:

- `dashboard`
  estado global, tareas en progreso, propuestas abiertas, agentes y proyectos
- `tareas`
  listado, detalle y acciones
- `propuestas`
  listado, detalle, votos y cierre
- `agentes`
  estado visible, asignaciones, sesiones, pausas y acciones de control permitidas
- `runtimes`
  árbol de runtimes, órdenes, mailbox y checkpoints
- `proyectos`
  vista del workspace, asignaciones y progreso
- `configuracion`
  endpoint, identidad del operador, preferencias y notificaciones

## 3. Cliente API compartido

La app desktop debe tener un cliente API explícito y testeable.

Contratos mínimos que debe consumir desde el inicio:

- `GET /api/status`
- `GET /api/agentes`
- `GET /api/tareas`
- `GET /api/propuestas`
- `GET /api/asignaciones`
- `GET /api/worktrees`
- `GET /api/locks`
- `POST /api/tareas/...`
- `POST /api/propuestas/...`
- `POST /api/agente/...`
- endpoints de runtimes y time travel cuando el servidor los estabilice

La regla es simple:

- si una capacidad no existe por API, no pertenece todavía a la desktop como capacidad real

## Flujos operativos iniciales

## 1. Seguimiento global

La pantalla principal debe responder rápido a estas preguntas:

- qué agentes están activos o degradados
- qué tareas están en progreso, bloqueadas o libres
- qué propuestas esperan voto o cierre
- qué proyectos concentran más carga o riesgo

## 2. Operación de tareas y propuestas

Primera fase útil:

- listar
- filtrar
- ver detalle
- votar
- tomar/iniciar/completar/bloquear tareas

Esto ya tiene valor sin tocar aún control activo avanzado.

## 3. Control de agentes

Segunda fase útil:

- ver sesión, heartbeat, presupuesto y última actividad
- pausar
- continuar
- preparar handoff
- inspeccionar órdenes y mailbox

La desktop no debe inyectar texto a consolas ni controlar PTYs de forma directa.
Debe crear órdenes en Orquesta para que el plano de control haga el resto.

## 4. Time travel y runtime timeline

Cuando la API lo cubra con estabilidad, la desktop podrá aportar mucho valor en:

- timeline de `runtime_orders`
- detalle de `runtime_mailbox`
- checkpoints y continuidad
- comparación entre estados

Pero siempre como lectura y operación sobre API, no como acceso lateral a tablas.

## Tecnología recomendada

No fijar todavía framework final como decisión cerrada, pero sí criterios:

- escritorio moderno con buen soporte multiplataforma
- consumo sencillo de HTTP/JSON
- empaquetado razonable para Linux, Windows y macOS
- buen soporte de tablas densas, notificaciones y estado reactivo

Dos opciones razonables para spike:

- `Tauri`
  si se prioriza ligereza, empaquetado moderno y reutilización de UI web
- `Wails`
  si se prioriza cercanía al ecosistema Go y una integración más natural con el stack actual

En ambos casos la regla sigue siendo la misma:

- el backend real vive en Orquesta
- la app desktop no incrusta la lógica de negocio de Orquesta

## Riesgos a evitar

- duplicar lógica de negocio del servidor en el cliente
- leer `orquesta.db` desde la app
- acoplar la UI a SQLite, a scripts sueltos o a comandos no soportados por API
- introducir control activo por atajos locales fuera de `runtime_orders`
- convertir la app desktop en un “segundo servidor”

## Plan de implementación sugerido

### Fase 1

- cliente API
- autenticación/configuración básica del endpoint
- dashboard
- tareas y propuestas

### Fase 2

- agentes, asignaciones y sesiones
- notificaciones
- estado degradado y errores de conectividad

### Fase 3

- runtimes, mailbox, checkpoints y handoff
- time travel
- vistas de diagnóstico y exportación

## Criterio de aceptación

La app de escritorio estará bien diseñada si cumple esto:

- no necesita SQLite directo
- no rompe el principio de cliente fino
- reutiliza la API de Orquesta como fuente de verdad
- sirve para operar mejor, no para inventar otro plano de control
