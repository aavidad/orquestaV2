<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Orquesta v1 — Vision de producto y arquitectura objetivo

## 1. Proposito

Orquesta debe evolucionar desde un coordinador local de tareas a una plataforma de gobierno operativo de agentes sobre todo el workspace `~/Trabajo`.

Su objetivo no es solo arrancar agentes. Debe saber en todo momento:

- que proyectos existen
- que agentes estan asignados a cada proyecto
- en que fase esta cada proyecto
- que sesion externa corresponde a cada agente
- que reglas, skills y workflows debe recibir cada agente
- que ramas estan activas
- que cambios estan sin commit, sin push o pendientes de merge
- que decisiones estan pendientes de voto
- que porcentaje real de avance tiene cada proyecto

## 2. Modelo del workspace

El directorio `~/Trabajo` es la raiz operativa.

- Cada subcarpeta puede ser un proyecto independiente.
- Puede haber carpetas grupo que contienen varios proyectos relacionados.
- Ejemplo: `PlataformaMunicipal/` es un grupo; dentro puede convivir `ContaGrx/` y `orquestador/`.
- Orquesta no depende de un proyecto concreto; actua como servicio transversal sobre todo el workspace.

### Restriccion operativa actual

Aunque el objetivo final es que `orquestador` viva fisicamente en `~/Trabajo/orquestador`, a fecha `2026-03-22` sigue ubicado en `~/Trabajo/PlataformaMunicipal/orquestador`.

Ese traslado no debe ejecutarse en caliente mientras existan agentes activos sobre esa ruta.
Antes de moverlo hay que:

- cerrar o pausar sesiones activas
- guardar `external_session_id` y `resumen_continuidad`
- verificar worktrees y ramas
- hacer backup de la BD y del repo
- actualizar rutas persistidas y scripts de arranque

## 3. Principios no negociables

1. Nucleo hexagonal.
   El nucleo no depende de SQLite, GitHub, Codex, Claude, Gemini ni Terminator.

2. Base de datos como adaptador.
   La persistencia entra por puertos. No se puede seguir metiendo logica de negocio directamente en `db/`.

3. Conectores para proveedores y runtimes.
   El nucleo solo conoce contratos estables. Los detalles de cada modelo o forge viven en conectores.

4. Datos existentes intocables.
   No se pierde informacion actual. Las migraciones son aditivas e idempotentes.

5. Seguridad y filosofia del proyecto por encima de velocidad.
   Ninguna automatizacion puede imponerse a la seguridad, a la arquitectura o a la mantenibilidad.

6. Votacion para decisiones de arquitectura.
   Las decisiones importantes se registran como propuesta y requieren al menos dos opiniones no autoras.

7. Autonomia operativa segura.
   Los agentes pueden ejecutar acciones normales sin pedir permiso previo.
   Las acciones destructivas, irreversibles o de riesgo alto deben consultarse antes con Orquesta o con otro agente del mismo proyecto.

## 4. Capacidades objetivo

### 4.1. Multi-proyecto y multi-agente

- Proyectos explicitos con arbol `raiz -> grupo -> repo`.
- Identidad estable de agente: `Codex1`, `Codex2`, `Claude1`, `Gemini1`, etc.
- Asignaciones activas por proyecto.
- Reparto de slots por proyecto.
- Reasignacion dinamica sin perder continuidad.
- Pools de capacidad por licencia o runtime cuando exista mas de una licencia disponible.

### 4.1.b. Orquestador jerarquico

Orquesta debe poder actuar como un orquestador superior sobre varios pools de capacidad.

Ejemplo:

- `Codex` con 4 licencias
- `Claude` con 1 licencia
- `Android` con 1 licencia

El nivel superior reparte trabajo entre pools.
Cada pool puede abrir agentes hijos dentro de su capacidad si la politica lo permite.
Los pools deben poder ampliarse o reducirse y admitir nuevos proveedores y modelos futuros.
Ademas, el orquestador superior debe decidir por tarea el modelo y el nivel de razonamiento adecuados.

### 4.2. Sesiones reanudables

Para cada sesion debe guardarse al menos:

- `external_session_id`
- `cwd`
- `branch`
- `herramienta`
- `conector`
- `resumen_continuidad`
- `estado`
- `heartbeat_at`
- `host`
- `pid`
- presupuesto restante de sesion cuando el runtime lo exponga
- presupuesto restante semanal cuando el runtime lo exponga

Esto es obligatorio porque `resume --last` no sirve en escenarios multiagente ni cuando varios agentes trabajan sobre el mismo proyecto.

### 4.3. Reglas, skills y workflows servidos por Orquesta

Orquesta debe suministrar a cada agente:

- reglas globales
- reglas por rol
- reglas por proyecto
- skills
- workflow de fase
- contexto vivo del proyecto

El agente no debe depender de ficheros manuales dispersos para arrancar con contexto correcto.

### 4.4. API para web y app de escritorio

Toda funcion importante debe estar disponible:

- desde CLI
- desde API HTTP/JSON
- desde la web
- desde una futura app de escritorio

La web no sera un adorno. Debe poder:

- asignar agentes
- mover agentes entre proyectos
- crear y gestionar tareas
- abrir y seguir propuestas
- votar
- ver sesiones
- ver locks y worktrees
- ver ramas y estado Git
- lanzar merges
- arrancar o pausar agentes

## 5. Arquitectura hexagonal propuesta

### 5.1. Nucleo

El nucleo contiene:

- casos de uso
- servicios de aplicacion
- reglas de negocio
- politicas de reparto
- control de fases
- calculo de progreso

### 5.2. Puertos

Puertos de salida a definir o consolidar:

- persistencia de proyectos, tareas, propuestas, sesiones, votos y configuracion
- runtime de agentes
- forge y VCS
- MCP
- memoria y hallazgos
- investigacion externa
- notificaciones
- backup y snapshots

### 5.3. Adaptadores de entrada

- CLI
- API HTTP/JSON
- web
- app desktop
- servidor MCP

### 5.4. Adaptadores de salida

- SQLite inicial
- Git
- Terminator
- conectores LLM
- conectores forge
- buscadores y fuentes externas

## 6. Capa de conectores

### 6.1. Conectores LLM

El nucleo debe ser agnostico al modelo. La integracion se hace mediante conectores por proveedor o runtime.

Transportes previstos:

- `cli`
- `mcp_stdio`
- `mcp_http`
- `api`

Modelos o herramientas previstas:

- Codex
- Claude
- Gemini
- Grok
- futuros modelos no existentes hoy

El orquestador debe poder fijar por tarea o fase:

- modelo objetivo
- nivel de razonamiento o esfuerzo
- perfil de tarea

Ejemplo:

- arquitectura del orquestador: modelo frontier con razonamiento `xhigh`
- script simple: razonamiento `medium` o `low`

### 6.2. Conectores forge

El mismo patron se aplica al mundo Git remoto.

Conectores previstos:

- GitHub
- GitLab
- Gitea

La creacion de un repositorio remoto es una capacidad opcional de bootstrap, pero no es el objetivo principal del producto.
Lo principal es conocer el estado real del proyecto y gobernar su ciclo de vida.

## 7. MCP

MCP encaja como protocolo de integracion, no como cerebro del sistema.

Orquesta debe poder actuar como servidor MCP y exponer:

- `resources` con estado del proyecto
- `prompts` con reglas, skills y workflows
- `tools` con acciones controladas por el orquestador

Ejemplos:

- `orquesta.tareas.listar`
- `orquesta.propuestas.votar`
- `orquesta.sesion.guardar`
- `orquesta.heartbeat`
- `orquesta.fuentes.registrar`

## 8. Fases del ciclo de vida

Las fases deben existir como entidades de primer nivel para proyecto y tarea.

### 8.1. Fase 0 — Descubrimiento del workspace

- escaneo de `~/Trabajo`
- deteccion de proyectos, grupos y repos
- tecnologias detectadas
- ramas principales
- estado Git base

### 8.2. Fase 1 — Investigacion de referencias

- busqueda de proyectos GitHub u otros forges
- librerias reutilizables
- componentes existentes
- riesgos de licencia
- patrones aplicables

### 8.3. Fase 2 — Investigacion dirigida

Investigacion guiada por la necesidad concreta del proyecto:

- leyes
- reglamentos
- estandares
- APIs oficiales
- documentacion primaria
- restricciones sectoriales

### 8.4. Fase 3 — Sintesis y memoria

- consolidacion de fuentes
- separacion de hechos e inferencias
- memoria de proyecto
- preguntas abiertas

### 8.5. Fase 4 — Planificacion

- propuestas
- backlog
- dependencias
- slots de agentes
- fases de ejecucion

### 8.6. Fase 5 — Arquitectura y gobernanza

- contratos
- reglas
- conectores
- criterios de aceptacion
- gates de seguridad

### 8.7. Fase 6 — Preparacion de ejecucion

- backups
- worktrees
- locks
- sesiones
- ramas de trabajo

### 8.8. Fase 7 — Desarrollo

- implementacion
- seguimiento continuo
- actualizacion de memoria

### 8.9. Fase 8 — Documentacion

- documentacion funcional
- documentacion tecnica
- runbooks
- decisiones relevantes

### 8.10. Fase 9 — Pruebas y validacion

- tests unitarios
- integracion
- benchmarks
- validacion funcional
- seguridad

### 8.11. Fase 10 — Refactorizacion competitiva

Ver seccion especifica de refactorizacion.

### 8.12. Fase 11 — Revision final

- revision por pares
- deteccion de deriva
- decision de integracion

### 8.13. Fase 12 — Commit

- commit atomico
- mensaje trazable
- referencia a tarea y propuesta

### 8.14. Fase 13 — Push

- publicacion remota
- sincronizacion de estado

### 8.15. Fase 14 — Seguimiento posterior

- memoria aprendida
- deuda tecnica
- continuidad

## 9. Investigacion como fase fuerte

La investigacion inicial es una capacidad central del producto.

No debe limitarse a leer el repo local. Debe permitir a los agentes:

- buscar por internet
- estudiar software existente
- revisar normativa y estandares
- localizar APIs y documentacion oficial
- registrar fuentes
- sintetizar hallazgos

La salida no es texto libre sin estructura. Deben quedar artefactos persistentes:

- fuentes
- hallazgos
- decisiones preliminares
- preguntas abiertas

## 10. Operacion continua de agentes

Los agentes deben trabajar en modo continuo hasta:

- terminar la tarea
- necesitar una aclaracion real
- recibir una orden de pausa o reasignacion
- acercarse al limite de sesion o presupuesto operativo

Para ello Orquesta debe imponer:

- `heartbeat`
- `tick` periodico
- lectura constante de propuestas y votos pendientes
- deteccion de cambios de asignacion
- lectura del estado de la fase actual
- vigilancia de presupuesto de sesion y handoff preventivo

## 11. Terminator y consola de agentes

Se necesita soporte practico para operar varios agentes manuales a la vez.

### Requisitos

- arranque de varias ventanas o pestañas de Terminator
- cabecera visible con `agente + proyecto`
- un `cwd` aislado por agente
- uso automatico de worktree si varios agentes comparten proyecto

### Persistencia al cerrar

Al cerrar una sesion manual del runtime deben guardarse:

- `external_session_id`
- `resumen_continuidad`
- `cwd`
- `branch`
- `herramienta`

Este guardado debe ser determinista. Para Codex la estrategia valida es:

- worktree o `cwd` unico por agente
- preferencia por `external_session_id` ya conocido en Orquesta
- si la sesion es nueva, arranque con token de contexto unico

## 12. Locks y worktrees

### Locks

Tipos previstos:

- `file`
- `dir`
- `module`
- `task`
- `project`

Con lease, heartbeat y expiracion.

### Worktrees

- un worktree por agente cuando haga falta aislamiento
- worktrees opcionales por tarea
- worktrees obligatorios para refactor competitivo y para varios agentes sobre el mismo proyecto

## 13. Memoria, hallazgos y deteccion de deriva

Capacidades a igualar o superar frente a herramientas externas:

- knowledge capture
- memoria de proyecto
- memoria de tarea
- fuentes registradas
- hallazgos
- decisiones
- deteccion de deriva entre plan, propuesta, fase y cambios reales

La deriva debe compararse contra:

- tarea
- propuesta
- fase
- rama
- diff actual

## 14. Plan generation y modo headless

Patrones a incorporar de otras herramientas:

- generacion de plan desde contexto e investigacion
- modo headless supervisado
- ejecucion continua de agentes
- captura de conocimiento

Orquesta debe igualar:

- locking
- knowledge capture
- drift detection
- plan generation
- headless mode
- paralelismo practico
- worktrees
- pipeline multi-modelo
- analisis pesado
- uso de MCP como capa de contexto

## 15. Pipeline secuencial entre modelos

Debe poder declararse por workflow, no por codigo rigido.

Ejemplo:

- un agente investiga
- otro planifica
- otro implementa
- otro revisa
- otro documenta o valida

Las sesiones deben poder separarse por tipo:

- investigacion
- planificacion
- desarrollo
- revision
- refactorizacion

## 16. Gobierno Git

Orquesta debe conocer el estado Git de cada proyecto y de cada rama activa.

### Telemetria minima

- `HEAD` actual
- rama
- upstream
- commits locales sin push
- commits remotos sin traer
- estado `ahead/behind`
- cambios sin commit
- ficheros sin seguimiento
- ultimo commit asociado a tarea

### Orquestacion de ramas

Se debe poder ver y gestionar:

- que rama usa cada agente
- que worktree la contiene
- si esta lista para revision
- si esta lista para merge
- si esta bloqueada
- si ya fue mergeada

### Merge y control de integracion

La app debe poder:

- lanzar revisiones
- aprobar o rechazar integracion
- ejecutar merge si pasa gates
- registrar la decision

`push` no equivale a terminado.
`commit` no equivale a integrado.

## 17. Medicion de progreso

El porcentaje de avance del proyecto no debe salir de lineas de codigo ni del numero de commits.

Debe ser una combinacion de:

- porcentaje de tareas completadas
- porcentaje de fases superadas
- propuestas cerradas
- gates en verde
- ramas integradas

## 18. Refactorizacion competitiva

### Objetivo

Poder pedir a varios agentes que refactoricen una misma unidad de trabajo y elegir la mejor variante.

### Reglas duras

1. Backup obligatorio antes de modificar nada.
2. Contrato congelado antes de competir.
3. Un candidato por worktree o rama aislada.
4. Seguridad no regresiva.
5. Filosofia del proyecto no regresiva.
6. Al menos dos revisores no autores.
7. La correccion pesa mas que el rendimiento.

### Evaluacion

Se compara por:

- correccion
- seguridad
- filosofia del proyecto
- rendimiento
- memoria
- claridad y mantenibilidad

## 19. Politica de backup

Antes de:

- refactorizacion competitiva
- merge automatico sensible
- cambios destructivos
- experimentos de alto riesgo

Orquesta debe exigir un backup o snapshot verificable.

## 20. API objetivo

La API debe cubrir como minimo:

- proyectos
- agentes
- asignaciones
- tareas
- propuestas
- votos
- sesiones
- locks
- worktrees
- ramas
- merges
- progreso
- memoria y fuentes
- fases
- conectores

## 21. Fuentes de inspiracion externas

### 21.1. `forge-orchestrator`

A igualar:

- locking
- knowledge capture
- drift detection
- plan generation
- headless mode

### 21.2. `claude-squad`

A igualar:

- paralelismo practico
- aislamiento por worktree

### 21.3. `claude-codex-gemini`

A igualar:

- pipeline secuencial entre modelos
- sesiones separadas por tipo

### 21.4. `gemini-cli-orchestrator`

A igualar:

- analisis pesado
- MCP como capa de contexto

## 22. Decisiones ya tomadas en esta linea

- mantener Orquesta como nucleo en lugar de sustituirlo por otro proyecto
- copiar patrones utiles y no la arquitectura completa de terceros
- separar politica de orquestacion de runtime del agente
- imponer `cwd` aislado para sesiones multiagente
- persistir `external_session_id` y `resumen_continuidad`
- usar conectores para LLM y forge
- centralizar reglas, skills y workflows
- documentar y votar la arquitectura objetivo mediante `OP-049`

## 23. Temas abiertos

- servidor central como unico escritor frente a CLI con escritura directa
- nivel exacto de automatizacion del arranque y pausa de runtimes
- formula exacta de progreso ponderado
- politica de merge automatizado segun gates y riesgo

## 24. Criterio de exito

Orquesta estara en su direccion correcta cuando pueda:

- gestionar varios proyectos del workspace sin mezclar estados
- arrancar varios agentes con continuidad real
- cambiar agentes de proyecto sin perder sesion
- entregar a cada agente sus reglas, skills y workflows
- seguir Git, fases, tareas y propuestas desde web o API
- soportar distintos LLM mediante conectores
- documentar, votar y ejecutar decisiones con trazabilidad
