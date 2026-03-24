<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Analisis de repositorios multiagente y patron recomendado para Orquesta

Fecha: 2026-03-23

## Resumen ejecutivo

La conclusion principal es simple:

- **si, PTY sirve**, pero solo como **adaptador de transporte local**
- **no, PTY no debe ser el modelo central de control**

Lo mejor que merece la pena copiar de los repos revisados no es "abrir terminales", sino esto:

1. **identidad persistente de agente separada de la sesion viva**
2. **cola de ordenes y plano de control unico**
3. **checkpoint, continuidad y handoff como primitives**
4. **observabilidad pasiva y arbol de runtimes**
5. **capability routing por adaptador**, no por terminal concreta
6. **supervision activa y recuperacion automatica**
7. **mezcla de autonomia + flujos deterministas**

La recomendacion para Orquesta es:

- mantener `pty/process` como primer transporte real
- envolverlo bajo un **RuntimeAdapter**
- gobernarlo desde un **daemon unico de Orquesta**
- añadir una **cola persistente de ordenes**
- persistir **checkpoints, handles, capacidades y mailbox entre agentes**

## Principio rector: autogestion supervisada

Si el objetivo es que no tengas que estar encima de los agentes, la arquitectura debe optimizar esto:

- que el agente **detecte problemas solo**
- que el agente **intente resolverlos solo**
- que el agente **pida ayuda a otro agente antes que al humano**
- que el agente **deje checkpoint antes de escalar**
- que el sistema **repare, reasigne o reanime** sin intervencion manual cuando sea posible

La palabra clave no es autonomia ciega, sino:

**autogestion supervisada**

Eso significa:

- maxima autonomia operativa
- guardrails duros
- escalacion tardia
- trazabilidad completa

## Que significa autogestion real en Orquesta

Un agente autogestionado no solo ejecuta tareas. Tambien sabe:

1. diagnosticar su propio bloqueo
2. reintentar con estrategia distinta
3. buscar contexto y decisiones previas
4. consultar a otro agente especializado
5. ceder la tarea si otro agente encaja mejor
6. pausar o handoff antes de quedarse sin presupuesto
7. escalar al humano solo con evidencia y contexto suficiente

## Flujo minimo de resolucion autonoma

Cuando un agente encuentra un problema, el orden correcto deberia ser:

1. **autodiagnostico**
   - clasificar si el problema es de codigo, test, entorno, permisos, dependencia, conflicto git o falta de contexto
2. **autorremediacion local**
   - reintento limitado
   - cambio de estrategia
   - lectura de logs
   - consulta de estado del proyecto
3. **consulta entre agentes**
   - mensaje a otro agente con mejor perfil
   - consulta al documentador o al agente dueño del modulo
4. **checkpoint**
   - guardar estado antes de relevo o pausa
5. **handoff o reasignacion**
   - pasar el trabajo a otro agente si ya no es racional insistir
6. **escalacion humana**
   - solo si sigue bloqueado tras los pasos anteriores
   - siempre con paquete de diagnostico

## Capacidades que necesita Orquesta para esa autogestion

### 1. Politica de autonomia por agente

Cada agente deberia tener una politica explicita en BD:

- numero maximo de reintentos
- tiempo maximo de bloqueo local
- cuando puede autoeditar
- cuando debe consultar a otro agente
- cuando debe crear propuesta
- cuando debe hacer handoff
- cuando debe escalar al humano

### 2. Motor de resolucion de bloqueos

No basta con marcar una tarea como `bloqueada`.

Hace falta un pequeño motor que permita:

- detectar patrones de bloqueo
- sugerir accion
- ejecutar remediaciones seguras
- registrar resultado

Estados utiles:

- `atascado`
- `reintentando`
- `consultando_par`
- `handoff_preparado`
- `escalado_humano`

### 3. Consulta entre agentes como primitive

Esto es clave para no depender de ti.

Si el agente A no sabe seguir, antes de pararse debe poder:

- consultar al agente B
- consultar al documentador
- consultar al "vigilante"
- pedir relevo

Sin mailbox entre agentes no hay autogestion real.

### 4. Checkpoint obligatorio antes de escalar

La escalacion humana debe ocurrir despues de:

- guardar resumen
- guardar rama y `cwd`
- guardar evidencia
- guardar ultimo intento
- proponer siguiente accion

Si no, el humano vuelve a hacer de memoria viva del sistema.

### 5. Supervisor automatico del sistema

Ademas del agente individual, Orquesta debe tener un supervisor que:

- detecte sesiones zombie
- detecte ausencia de heartbeat
- detecte presupuesto en riesgo
- detecte tareas estancadas
- fuerce `nudge`, `checkpoint`, `handoff` o `restart`

## De que repo copiaria la autogestion

### Gastown

Es la mejor referencia para:

- watchdogs
- deteccion de estancamiento
- nudges
- handoff
- continuidad entre sesiones

### CrewAI

Es la mejor referencia conceptual para:

- agentes autonomos con roles
- autonomia dentro de un marco controlado

### LangGraph

Es la mejor referencia para:

- checkpoint
- reanudacion exacta
- aprobaciones humanas solo cuando toca

### OpenClaw

Es la mejor referencia para:

- control plane central
- herramientas de sesion a sesion
- parcheo de politica en vivo

### Swarm

Es la mejor referencia para:

- handoff simple
- minimizar complejidad conceptual

## Decision de diseño derivada

Si la prioridad numero uno es la autogestion, entonces Orquesta no debe implementar solo:

- `arrancar`
- `enviar`
- `pausar`
- `continuar`

Debe implementar tambien:

- `consultar`
- `nudge`
- `checkpoint`
- `handoff`
- `restart`
- `escalar`
- `auto_resolver_bloqueo`

## Politica recomendada de escalacion

Propongo esta regla:

> un agente no puede molestar al humano en el primer bloqueo; primero debe intentar resolver, luego consultar a un par, luego dejar checkpoint y solo despues escalar.

Umbrales iniciales razonables:

- `2` reintentos locales
- `1` consulta a otro agente
- `1` checkpoint obligatorio antes de escalar
- escalacion inmediata solo si:
  - la accion es destructiva
  - hay riesgo de perdida de datos
  - el conflicto es de negocio/politica
  - faltan permisos externos imposibles de obtener solos

## Que construir primero si la autogestion es lo mas importante

El orden correcto cambia ligeramente.

### Prioridad 1

- `runtime_mailbox`
- `runtime_orders`
- `checkpoint`
- supervisor de salud
- `nudge` y `handoff`

### Prioridad 2

- politica de autonomia por agente
- motor de resolucion de bloqueos
- heuristicas de reintento y reasignacion

### Prioridad 3

- approval UI
- timeline de decisiones
- panel de bloqueos y recuperaciones

## Foto actual de Orquesta

### Reglas y workflows en BD

La base de datos ya almacena reglas, skills y workflows activos por tipo de agente.

Hoy existen dos perfiles principales:

- `programador`
- `documentador`

Las reglas activas revisadas muestran una linea clara:

- la **BD es la fuente de verdad**
- el arranque correcto pasa por `orquesta sesion inicio <agente>`
- hay disciplina fuerte de tareas, propuestas, auditoria y cierre de sesion
- ya hay politicas de autonomia segura, worktrees y continuidad

### Tareas actuales

Revision directa de la BD local `orquesta.db` el 2026-03-23:

- `213` tareas `completada`
- `35` tareas `asignada`
- `5` tareas `en_progreso`
- `1` tarea `bloqueada`
- `2` tareas `cancelada`

Las tareas activas mas relevantes para este analisis son:

- `224` Inventariar accesos directos a persistencia y mapear cobertura CLI/API pendiente
- `226` Desacoplar persistencia de Orquesta para soportar SQLite, MySQL u otros backends
- `227` Implementar bateria CLI/API de lectura, observabilidad y diagnostico
- `244` Alinear web y API al principio de cliente fino sobre Orquesta
- `253` Implementar modo daemon persistente en `orquesta serve` y API Single-Writer
- `254` Crear fichero de unidad systemd y script de instalacion del servicio Orquesta
- `255` Configurar agente `vigilante` como proceso auto-reanudable
- `256` Consolidar integracion de Terminator con sesiones y runtimes de Orquesta

Tambien siguen vivas varias tareas de control de agentes y observabilidad:

- `153` Diseñar control activo de agentes vivos
- `233` a `249` relacionadas con estrategia de control, modo servidor unico y observabilidad de runtimes

### Lo que ya existe en codigo

Orquesta ya tiene piezas valiosas:

- `sesiones` con `external_session_id`, `resume_payload_json`, `resumen_continuidad`, `cwd`, `branch`, `pid`
- `runtime_instances`, `runtime_telemetry_samples` y `runtime_events`
- `worktrees` y `locks`
- `agentruntime/driver.go` con abstraccion por transporte: `cli`, `mcp_stdio`, `mcp_http`, `api`, `otro`
- wrappers manuales para Terminator y consola:
  - `scripts/agente_console.sh`
  - `scripts/terminator_agentes.sh`

### Limite actual

El limite actual no es de lanzado, sino de **gobierno vivo**:

- Orquesta puede preparar y registrar sesiones
- puede observar parte del runtime
- puede abrir consolas manuales
- pero **todavia no gobierna bien agentes vivos de forma unificada**

El propio diseño local ya apunta bien:

- `docs/diseno_control_activo_agentes.md`
- `docs/diseno_observabilidad_pasiva_runtimes_es.md`
- `docs/operacion_agentes_manuales.md`

La brecha exacta es esta:

- falta un **runtime handle** estable
- falta una **runtime order queue**
- falta un **canal de control unico**
- falta un **mailbox/session bus**
- falta un **checkpoint/handoff real**

## Criterio de evaluacion

He revisado los repos con foco en estas preguntas:

1. Como controlan agentes vivos
2. Como persisten identidad, contexto y continuidad
3. Como mezclan autonomia con control determinista
4. Como observan y recuperan agentes
5. Que piezas merece la pena copiar en Orquesta

## Analisis repo a repo

### 1. Gastown

Repo: <https://github.com/steveyegge/gastown>

#### Virtudes

- separa **identidad persistente** de **sesiones efimeras**
- persiste trabajo en **hooks git-backed** y worktrees
- tiene **mailboxes, handoffs e identidades**
- incluye **continuidad de sesiones** via logs de eventos
- tiene **watchdogs reales**:
  - `Witness` por rig
  - `Deacon` supervisor global
  - `Dogs` para mantenimiento
- incorpora **merge queue** con verificacion estilo Bors
- tiene dashboard y feed de monitorizacion

#### Lo mejor para copiar

- **identidad de agente persistente aunque la sesion muera**
- **session discovery + continuation**
- **watchdog multinivel**
- **nudge / handoff / recovery** como acciones de supervision
- **merge queue** para que los agentes no integren directamente a main
- **persistencia en worktree** como unidad operativa

#### Encaje en Orquesta

Encaja muy bien en:

- worktrees por agente
- handoff entre sesiones
- supervisor de salud
- merge orchestrado
- historial durable por agente

#### Lo que no copiaria tal cual

- la nomenclatura completa (`Mayor`, `Polecats`, `Dogs`, etc.)
- el acoplamiento a su ecosistema y ritual operativo
- la dependencia conceptual de `tmux` como señal de salud

#### Juicio

Es el repo con mejor material para copiar en **operacion real de agentes coding**.

### 2. CrewAI

Repo: <https://github.com/crewAIInc/crewAI>

#### Virtudes

- separa con claridad:
  - **Crews** = autonomia colaborativa
  - **Flows** = control determinista y event-driven
- insiste en **state management consistente**
- permite combinar autonomia y flujo productivo
- tiene buen discurso para entornos enterprise y automatizaciones complejas

#### Lo mejor para copiar

- la **separacion conceptual** entre:
  - bucle autonomo del agente
  - flujo orquestado por la plataforma
- el principio de **"autonomia dentro de guardrails"**
- modelar tareas largas como **flows** y no como simple chat continuo

#### Encaje en Orquesta

Muy alto a nivel conceptual.

Orquesta necesita exactamente eso:

- agentes con autonomia local
- operaciones criticas gobernadas por flujo

Ejemplos claros:

- `handoff`
- `pausa`
- `checkpoint`
- `reasignacion`
- `merge`
- `cierre de tarea`

#### Lo que no copiaria tal cual

- no montaria Orquesta sobre CrewAI como framework base
- no dependeria de piezas enterprise externas para el control core

#### Juicio

No es el mejor repo para copiar implementacion concreta del runtime, pero si uno de los mejores para copiar el **modelo mental correcto**.

### 3. LangGraph

Repo: <https://github.com/langchain-ai/langgraph>

#### Virtudes

- **durable execution**
- **human-in-the-loop**
- **memoria de corto y largo plazo**
- soporte claro para **workflows stateful y long-running**
- buena historia de observabilidad y trazas

#### Lo mejor para copiar

- el concepto de **checkpoint duradero**
- el concepto de **interrupcion y reanudacion exacta**
- el modelado como **grafo de estados** en vez de secuencia plana
- la idea de que el estado del agente es **estructura persistida**, no solo transcript

#### Encaje en Orquesta

Encaja de forma excelente en:

- handoff
- pausas
- approval/human gate
- workflow de relevo por presupuesto
- diagnostico reproducible

#### Lo que no copiaria tal cual

- no introduciria toda la pila LangChain/LangGraph si Orquesta quiere seguir ligera y propia
- evitaria acoplar observabilidad central a LangSmith

#### Juicio

Es la mejor referencia para diseñar **checkpoint/handoff/resume** de verdad.

### 4. AutoGen

Repo: <https://github.com/microsoft/autogen>

#### Virtudes

- arquitectura **layered and extensible**
- separacion muy sana entre:
  - `Core API`
  - `AgentChat API`
  - `Extensions API`
- `Core API` con:
  - **message passing**
  - agentes **event-driven**
  - **runtime local y distribuido**
  - soporte cross-language `.NET` y Python
- tiene Studio para prototipado y GUI
- buen modelo de extensiones y tooling

#### Lo mejor para copiar

- la estratificacion de capas
- un **bus de mensajes entre agentes**
- un **runtime local/distribuido abstracto**
- la frontera limpia entre:
  - core de orquestacion
  - patrones de chat
  - extensiones de herramientas

#### Encaje en Orquesta

Muy alto si se quiere que Orquesta deje de pensar en "terminal" y pase a pensar en "runtime".

El mejor aprendizaje aqui es:

- PTY es una implementacion de bajo nivel
- lo correcto es un **Core Runtime API**
- sobre ese core ya puedes montar chat, web, CLI, desktop, mcp o remoto

#### Riesgo

El propio repo indica que usuarios nuevos revisen Microsoft Agent Framework primero. Eso sugiere que AutoGen sigue vivo, pero no debe tomarse como apuesta primaria de largo plazo.

#### Juicio

Muy buen repositorio para copiar la **arquitectura por capas**. Menos atractivo como dependencia central de producto.

### 5. OpenAgents

Interpretacion usada aqui: <https://github.com/xlang-ai/OpenAgents>

Nota: el enlace facilitado `openagents-ai/openagents` no resuelve de forma estable hoy; la referencia funcional y conocida del proyecto OpenAgents es `xlang-ai/OpenAgents`.

#### Virtudes

- enfoque **full-stack**
- backend + frontend + agentes reales
- muy buena separacion:
  - `backend`
  - `frontend`
  - `real_agents`
- patron claro de **one agent, one folder**
- `adapters` compartidos para cerrar el hueco entre backend y agente
- buena obsesion por llevar agentes a una **UI usable por no expertos**

#### Lo mejor para copiar

- el patron **one agent, one folder**
- una carpeta de **adapters comunes**
- que frontend, backend y agentes tengan fronteras explicitas
- la idea de que el producto debe ofrecer **superficie de usuario real**, no solo framework

#### Encaje en Orquesta

Encaja bien en:

- organizacion del codigo
- surface product de la web y futura desktop
- adaptadores por tipo de agente/conector

#### Lo que no copiaria tal cual

- su stack original es mas orientado a agentes de producto generalista que a coordinacion fina de coding agents
- el control de runtime no es su principal virtud frente a Gastown, LangGraph u OpenClaw

#### Juicio

Buena referencia de **producto full-stack y empaquetado de agentes**, menos fuerte en control operativo fino.

### 6. Swarm

Repo: <https://github.com/openai/swarm>

#### Virtudes

- modelo muy pequeño y claro
- dos primitives potentes:
  - `Agent`
  - `handoff`
- `client.run()` expresa un loop muy facil de entender
- excelente como referencia para no sobrediseñar

#### Lo mejor para copiar

- **handoff como primitive de primer nivel**
- loop explicito de ejecucion
- simplicidad en la transferencia entre agentes
- contexto separado de la identidad del agente

#### Limite

- es **stateless between calls**
- no resuelve por si mismo:
  - sesiones vivas
  - PTY
  - observabilidad pasiva
  - daemon unico
  - colas persistentes

#### Encaje en Orquesta

Muy bueno para el **modelo logico de handoff** y poco mas.

#### Juicio

Hay que copiar su sencillez, no usarlo como arquitectura operativa completa.

### 7. OpenClaw

Repo: <https://github.com/openclaw/openclaw>

#### Virtudes

- tiene una idea muy potente de **Gateway** como plano de control
- separa donde corre `exec` y donde viven las capacidades del dispositivo
- usa `node.invoke` para enrutar acciones locales/remotas
- maneja **session model** con politicas y parches por sesion
- expone herramientas de **agente a agente**:
  - `sessions_list`
  - `sessions_history`
  - `sessions_send`
- incorpora control fino de permisos y capacidades

#### Lo mejor para copiar

- **gateway central** en vez de hablar directamente con cada terminal
- **capability discovery**
- **routing por nodo/capacidad**, no por UI concreta
- **session patching** para cambiar politicas en vivo
- **session-to-session messaging**

#### Encaje en Orquesta

Encaje altisimo.

Esto es probablemente lo que mejor responde a tu duda sobre PTY:

- PTY es util
- pero la mejor idea es un **gateway/control plane**
- PTY seria solo un adaptador bajo ese gateway

#### Lo que no copiaria tal cual

- toda la parte de dispositivos, Tailscale, voz, media o mobile
- el alcance de asistente personal generalista

#### Juicio

Junto con Gastown, es de lo mas valioso para copiar en **control real de agentes vivos**.

### 8. LangCrew

Repo: <https://github.com/01-ai/langcrew>

#### Virtudes

- construido sobre LangGraph
- intenta combinar:
  - primitivas robustas
  - conceptos faciles tipo CrewAI
- trae:
  - HITL
  - dynamic workflow orchestration
  - event-driven processes
  - full-stack UI
  - observabilidad
  - guardrails

#### Lo mejor para copiar

- el enfoque de **frontend protocol + React components**
- la visualizacion clara de:
  - plan
  - scheduling
  - ejecucion
  - tool calls
- approval system y UX de human-in-the-loop

#### Riesgo

- el repo esta muy temprano
- sirve mas como fuente de ideas que como base estable

#### Juicio

Interesante para copiar **UX, approval flow y visualizacion**, no para convertirlo en dependencia base de Orquesta.

## Que piezas copiaria de cada uno

| Repo | Lo mejor a copiar |
| --- | --- |
| Gastown | identidad persistente, hooks/worktrees, watchdogs, merge queue, handoff/nudge |
| CrewAI | separacion crews vs flows, autonomia bajo control |
| LangGraph | durable execution, checkpoints, HITL, memoria stateful |
| AutoGen | core runtime por mensajes, capas, extensibilidad, runtime distribuido |
| OpenAgents | estructura full-stack, adapters, one-agent-one-folder |
| Swarm | primitive de handoff y loop minimo y claro |
| OpenClaw | gateway/control plane, session tools, capability routing |
| LangCrew | UI de agente, approvals, visualizacion, observabilidad de producto |

## Patron recomendado para Orquesta

### Principio base

**No construir Orquesta alrededor de terminales.**

Construirla alrededor de:

- `AgentIdentity`
- `RuntimeHandle`
- `RuntimeOrder`
- `RuntimeCheckpoint`
- `RuntimeMailbox`
- `RuntimeCapability`

Y luego dejar `PTY` como una implementacion concreta de `RuntimeHandle`.

### Arquitectura objetivo

#### 1. Plano de control unico

`orquesta serve` debe convertirse en el unico proceso con autoridad de escritura y control.

Clientes:

- web
- CLI
- futura app de escritorio
- wrappers manuales

Todos deben hablar con la API de Orquesta. No con SQLite, no con el OS, no con Terminator directamente.

#### 2. Adaptadores de runtime

Crear una interfaz comun:

```text
RuntimeAdapter
  Start(handle, request)
  Send(handle, message)
  Pause(handle)
  Resume(handle)
  Checkpoint(handle)
  Stop(handle)
  Inspect(handle)
  Capabilities(handle)
```

Primeros adaptadores:

- `pty`
- `process`
- `mcp_stdio`
- `mcp_http`
- `remote_api`

#### 2.b. Desacoplamiento de Autenticación
Siguiendo la decisión de soportar tanto terminales interactivos como agentes *headless*, el `RuntimeAdapter` debe conocer y aislar la estrategia de identidad:
- **Terminales Interactivos (OAuth):** Agentes que requieren un PTY porque el flujo de login exige abrir un navegador web o introducir un código de dispositivo.
- **Procesos Headless (Key Tokens):** Agentes gobernados remotamente sin terminal interactivo (ej. MCP), para los que Orquesta inyectará silenciosamente el *Token* en su entorno sin intervención humana.

#### 3. Cola persistente de órdenes

Copiar y materializar ya la idea de `runtime_orders`.

Estados:

- `pendiente`
- `tomada`
- `ejecutando`
- `completada`
- `fallida`
- `expirada`

Tipos minimos:

- `start`
- `send_instruction`
- `pause`
- `resume`
- `checkpoint`
- `handoff_prepare`
- `handoff_commit`
- `stop`
- `sync_status`

#### 4. Handle vivo separado de la sesion

Añadir `runtime_handles`.

Campos minimos:

- `agente`
- `sesion_id`
- `transporte`
- `handle_kind`
- `handle_ref`
- `capabilities_json`
- `estado`
- `lease_token`
- `last_seen_at`
- `metadata_json`

#### 5. Checkpoint y continuidad

No basta con guardar:

- `external_session_id`
- `resumen_continuidad`

Hace falta un objeto de continuidad mejor:

- `checkpoint_id`
- `sesion_origen_id`
- `runtime_id`
- `branch`
- `cwd`
- `objective`
- `last_user_visible_state`
- `last_tool_state`
- `open_files`
- `pending_orders`
- `handoff_reason`
- `resume_strategy`

Esto es lo que mas se parece a LangGraph + Swarm + Gastown.

#### 6. Mailbox entre agentes

Copiar la idea de:

- `sessions_send`
- mailboxes
- `nudge`
- `handoff`

Tabla sugerida:

```sql
runtime_mailbox (
  id,
  from_agente,
  to_agente,
  kind,
  payload_json,
  estado,
  created_at,
  delivered_at,
  consumed_at
)
```

Tipos:

- `mensaje`
- `nudge`
- `handoff`
- `consulta`
- `respuesta`
- `escalacion`

#### 7. Observabilidad pasiva primero

Mantener vuestra linea actual. Es correcta.

Copiar:

- Gastown: watchdogs y feed
- LangGraph: trazas y estado
- OpenClaw: session metadata y capability view

No basar el panel en preguntarle al agente por chat.

#### 8. Supervisor de salud

Copiar literalmente el patron de Gastown:

- supervisor local por runtime
- supervisor global por sistema
- detección de cuelgues (Watchdog): falta de `heartbeat` durante >T segundos.
- detección de bucles (Circuit Breaker): detección de 3 intentos idénticos de la misma herramienta con el mismo input/output o repetición de patrón ruidoso en el terminal.
- acciones de recuperacion automatizadas.

Estados sugeridos:

- `arrancando`
- `disponible`
- `pensando`
- `ejecutando_herramienta`
- `esperando_io`
- `bloqueado`
- `pausado`
- `handoff`
- `zombie`
- `cerrado`

Acciones automáticas:

- `nudge`
- `checkpoint`
- `restart`
- `handoff`
- `escalate`

#### 9. Aprobaciones y control humano

Copiar de LangGraph y LangCrew:

- approval gates
- inspection/edit de estado antes de continuar
- UI clara para aprobar:
  - ordenes destructivas
  - handoffs
  - merges
  - acceso a capacidades elevadas

#### 10. Integracion Git segura

Copiar de Gastown el principio, no la implementacion exacta:

- worktree por agente
- merge queue central
- gates previos a merge
- ningun agente integra directo a main

## Respuesta directa a la duda PTY vs "algo mejor"

### PTY si

Usad PTY para:

- lanzar procesos interactivos locales
- inyectar input cuando el runtime lo soporte
- leer output crudo
- ligar pid, cwd, branch y shell

### PTY no como centro del sistema

No lo useis como unidad de abstraccion principal porque:

1. es fragil
2. es poco estructurado
3. no expresa capacidades
4. no modela bien handoff
5. no modela bien pause/resume semantico
6. no sirve bien para remoto, mobile, MCP o API
7. hace que la web y la desktop acaben acopladas a detalles de terminal

### Mejor enfoque

El enfoque correcto para Orquesta es:

- **control plane primero**
- **adapter por transporte despues**

En una frase:

> PTY debe ser un driver, no la arquitectura.

## Roadmap recomendado para copiar "lo mejor de cada uno"

### Fase 1. Cerrar el plano de control minimo

Copiar sobre todo de OpenClaw, Gastown y vuestro propio diseño local.

Entregables:

- `runtime_handles`
- `runtime_orders`
- `runtime_mailbox`
- daemon unico en `orquesta serve`
- endpoints API para enviar, pausar, continuar, checkpoint y handoff

### Fase 2. Handoff y continuidad real

Copiar sobre todo de LangGraph, Swarm y Gastown.

Entregables:

- checkpoints persistentes
- handoff `prepare/commit`
- continuidad reproducible
- reasignacion entre agentes

### Fase 3. Observabilidad y supervision

Copiar sobre todo de Gastown y LangGraph.

Entregables:

- watchdog local/global
- estados `zombie/stalled`
- nudge automatico
- panel de arbol de runtimes
- timeline de ordenes y eventos

### Fase 4. UX y producto

Copiar sobre todo de LangCrew y OpenAgents.

Entregables:

- vista de plan, herramientas, ordenes y bloqueos
- approval UI
- timeline por agente
- visualizacion de dependencias y handoffs

## Mi recomendacion concreta

Si hay que decidir hoy que copiar primero, yo haria esto:

1. **Gastown**
   para identidad persistente, watchdogs, merge queue y continuidad operativa
2. **OpenClaw**
   para gateway/control plane, session tools y capability routing
3. **LangGraph**
   para checkpoint/handoff/HITL stateful
4. **CrewAI**
   para separar autonomia de flujos deterministas
5. **Swarm**
   para simplificar el modelo de handoff
6. **AutoGen**
   para la estratificacion del runtime core
7. **LangCrew**
   para UX de approval/visualizacion
8. **OpenAgents**
   para empaquetado full-stack y adapters

## Decision propuesta para Orquesta

### Copiar

- identidad persistente del agente
- worktree por agente
- mailbox y mensajeria entre sesiones
- watchdog y supervisor global
- merge queue central
- gateway/control plane unico
- runtime adapters por transporte
- checkpoint/handoff stateful
- approval UI
- timeline y observabilidad pasiva

### No copiar

- acoplar el sistema al terminal emulator
- hacer que la web hable con el runtime directamente
- depender de un framework externo como corazon total de Orquesta
- confundir transcript con estado operativo

## Siguiente movimiento recomendado en Orquesta

La mejor siguiente tarea tecnica no es "mejorar Terminator".

La mejor siguiente tarea es:

**implementar `runtime_handles + runtime_orders + RuntimeAdapter` dentro del daemon unico de Orquesta.**

Eso os permite:

- mantener lo manual
- gobernar lo automatico
- soportar PTY hoy
- soportar MCP/API/WS mañana

## Fuentes

### Repositorios revisados

- Gastown: <https://github.com/steveyegge/gastown>
- CrewAI: <https://github.com/crewAIInc/crewAI>
- LangGraph: <https://github.com/langchain-ai/langgraph>
- AutoGen: <https://github.com/microsoft/autogen>
- Swarm: <https://github.com/openai/swarm>
- OpenClaw: <https://github.com/openclaw/openclaw>
- LangCrew: <https://github.com/01-ai/langcrew>
- OpenAgents usado para este analisis: <https://github.com/xlang-ai/OpenAgents>

### Documentacion local usada

- `docs/diseno_control_activo_agentes.md`
- `docs/diseno_observabilidad_pasiva_runtimes_es.md`
- `docs/operacion_agentes_manuales.md`
- `db/reglas.go`
- `db/tareas.go`
- `db/schema.go`
- `db/runtimes.go`
- `agentruntime/driver.go`
- `scripts/agente_console.sh`
- `scripts/terminator_agentes.sh`
