# Informe: Ecosistema de Orquestación Multi-Agente 2026
## Análisis Comparativo e Ideas a Implementar en Orquesta

**Fecha:** 2026-03-29
**Autor:** Análisis automático + revisión humana
**Propósito:** Identificar proyectos similares en el ecosistema y extraer ideas concretas a implementar

---

## 1. Resumen Ejecutivo

Orquesta ocupa un nicho diferenciado: es la única plataforma OSS con **sistema de votación/gobernanza para decisiones arquitectónicas**, **identidades de agente persistentes entre proyectos**, e **inyección de skills desde BD al inicio de sesión**. Sin embargo, hay áreas donde proyectos externos llevan ventaja técnica significativa.

Este informe analiza 9 proyectos relevantes y extrae **5 mejoras concretas** priorizadas para implementar.

---

## 2. Proyectos Analizados

### 2.1 Mismo stack (Go)

#### GoClaw
- **URL:** https://github.com/nextlevelbuilder/goclaw
- **Stack:** Go 1.26+, PostgreSQL, pgvector, OpenTelemetry
- **Madurez:** 1.3k stars, 390 forks, producción validada
- **Descripción:** Plataforma multi-agente multi-tenant con tablero Kanban, presupuesto por proveedor LLM, delegación async/sync entre agentes, y 5 capas de permisos.
- **Solapamiento con Orquesta:** Tablero de tareas con dependencias (`blocked_by`), presupuesto por proveedor, delegación entre agentes, LLM-agnóstico.
- **Lo que tiene y nosotros no:**
  - Detección de prompt injection a nivel plataforma
  - Prevención de SSRF y denegación de comandos shell por patrones
  - Mailbox entre agentes (canal de mensajería separado de las tareas)
  - Imagen Docker < 50MB con arranque < 1s

#### Humanlayer Agent Control Plane (ACP)
- **URL:** https://github.com/humanlayer/agentcontrolplane
- **Stack:** Go, Kubernetes, CRDs
- **Madurez:** Alpha, 366 stars, Apache 2.0
- **Descripción:** Scheduler distribuido Kubernetes-nativo para agentes de larga duración. Agentes, Tareas y LLMs como recursos CRD declarativos.
- **Lo que tiene y nosotros no:**
  - Patrón de **tool call asíncrono**: el agente suspende mid-ejecución esperando aprobación humana o externa, luego reanuda. Más limpio que flags de sesión ad-hoc.
  - LLM config como recurso tipado separado (credenciales, parámetros de modelo)
  - MCP nativo para descubrimiento de herramientas

#### Eino (ByteDance/CloudWeGo)
- **URL:** https://github.com/cloudwego/eino
- **Stack:** Go 1.18+, graph-based
- **Madurez:** 10.3k stars, 162 releases, backing ByteDance
- **Descripción:** Framework Go de grafos para orquestación LLM. Interrupt/resume, multi-proveedor, callbacks por componente.
- **Lo que tiene y nosotros no:**
  - Patrón "callback aspects" (OnStart, OnEnd, OnError por componente) — observabilidad sin acoplamiento
  - Interfaz de componente Go idiomática y limpia — referencia para el modelo de conectores

#### AgenticGoKit
- **URL:** https://github.com/AgenticGoKit/AgenticGoKit
- **Stack:** Go 1.21+, OTLP/Jaeger, pgvector
- **Madurez:** Beta, 125 stars, desarrollo activo
- **Descripción:** Framework Go event-driven, LLM-agnóstico, con tipos formales de workflow.
- **Lo que tiene y nosotros no:**
  - Tipos de ejecución formales: `Sequential`, `Parallel`, `DAG`, `Loop`, `SubWorkflow`
  - OTEL/Jaeger zero-config integrado
  - Descubrimiento de herramientas MCP como patrón built-in

---

### 2.2 Frameworks Python/TS (ideas de arquitectura)

#### LangGraph
- **URL:** https://github.com/langchain-ai/langgraph
- **Stack:** Python, TypeScript, SQLite/PostgreSQL/Redis
- **Madurez:** Muy alta. ~400 empresas en producción (LinkedIn, Uber)
- **Lo que tiene y nosotros no:**
  - **Time-travel debugging maduro**: cada transición de estado es un checkpoint nombrado. Se puede hacer branch desde cualquier checkpoint pasado y re-ejecutar con inputs distintos. La implementación más completa del concepto que Orquesta tiene planificado en OP-092.
  - **Patrón SubGraph**: un equipo de agentes es en sí mismo un nodo del grafo mayor — orquestación jerárquica sin acoplamiento.
  - **Interrupt/Resume API formal**: suspensión mid-grafo, persistencia del motivo de interrupción, reanudación posterior.
  - **Thread fan-out**: una sesión padre lanza N sesiones hijo independientes con checkpoints propios, resultados se fusionan.

#### CrewAI
- **URL:** https://github.com/crewAIInc/crewAI
- **Stack:** Python
- **Madurez:** 46k+ stars, 60% Fortune 500 US, 1.1B acciones de agente en Q3 2025
- **Lo que tiene y nosotros no:**
  - Identidad estructurada del agente: campos `role`, `goal`, `backstory` — más que un nombre, una persona persistente
  - Memoria compartida a nivel equipo (long-term memory entre agentes del mismo crew)
  - Tipos de proceso formales: `sequential`, `hierarchical`, `consensual`

#### Mission Control
- **URL:** https://mc.builderz.dev
- **Stack:** Next.js 16, TypeScript, SQLite WAL
- **Madurez:** Activo, MIT
- **Lo que tiene y nosotros no:**
  - **Patrón de adaptadores de framework**: 6 adaptadores built-in (CrewAI, LangGraph, AutoGen, AutoGen, Claude SDK, OpenClaw) como plugins — formalización limpia de conectores externos
  - WebSocket + SSE push en lugar de polling
  - Grafo de memoria: mapa visual de relaciones entre agentes y contexto
  - Planificación de tareas en lenguaje natural ("cada lunes a las 9am")

---

### 2.3 Herramientas de Observabilidad y Gobernanza

#### Langfuse
- **URL:** https://github.com/langfuse/langfuse
- **Stack:** TypeScript/Next.js, PostgreSQL. Self-hostable.
- **Madurez:** 10k+ stars, YC W23
- **Lo que tiene y nosotros no:**
  - Versionado de prompts ligado a cada ejecución — saber qué versión de cada skill estaba activa en cada sesión
  - Session replay visual: replay completo de una sesión con cada llamada LLM, tool call y token count
  - OpenTelemetry nativo — trazas como OTEL spans
  - Dataset + pipeline de evaluación: captura trazas reales, etiquétalas, corre evals automáticos

#### Microsoft Agent Governance Toolkit
- **URL:** https://github.com/microsoft/agent-governance-toolkit
- **Stack:** Python, TypeScript, .NET. Preview, Apache 2.0.
- **Madurez:** 342 stars, respaldo Microsoft
- **Lo que tiene y nosotros no:**
  - **Modelo de anillos de privilegio** (4 niveles, como OS ring model): ring 0 = plena confianza, ring 3 = sandbox aislado
  - **Identidad criptográfica Ed25519** con puntuación de confianza (escala 0-1000) — Orquesta usa identidades nombradas pero sin attestation
  - **Kill switch** por agente individual sin reiniciar la plataforma
  - **Error budgets y SLOs por agente** (estilo SRE)
  - Evaluación de política en < 1ms antes de cada acción de agente
  - Chaos engineering: degradar agentes intencionalmente para probar respuesta de gobernanza

#### AgentOps
- **URL:** https://github.com/AgentOps-AI/agentops
- **Stack:** Python SDK, MIT
- **Madurez:** Ampliamente integrado (CrewAI, AutoGen, LangChain, OpenAI SDK)
- **Lo que tiene y nosotros no:**
  - Tabla de precios mantenida para 400+ LLMs — referencia para el módulo de presupuesto de Orquesta
  - Session replay visual
  - Detección de fallos con alertas automáticas de anomalías

---

## 3. Análisis de Gaps

### Lo que Orquesta tiene y nadie más

| Característica | Orquesta | Otros |
|---|---|---|
| Votación/gobernanza para decisiones arquitectónicas | ✅ | No encontrado en ningún OSS |
| Identidades de agente estables entre proyectos (no por sesión) | ✅ | Parcial en CrewAI (por crew) |
| Inyección de skills desde BD al inicio de sesión | ✅ | Más cercano: CrewAI backstory, LangGraph system prompt |
| REST API + dashboard como plano de control único | ✅ | Mission Control (proyecto separado) |
| Escaneo de workspace Git para descubrimiento de proyectos | ✅ | No encontrado |

### Gaps identificados en Orquesta

| Gap | Prioridad | Referencia |
|---|---|---|
| Time-travel: modelo de datos de checkpoints por paso | Alta | LangGraph |
| Versionado de skills ligado a cada sesión | Alta | Langfuse |
| Patrón formal de adaptadores para conectores externos | Media | Mission Control |
| Kill switch por agente sin reiniciar el servidor | Media | MS Governance Toolkit |
| Push WebSocket/SSE en lugar de polling en el dashboard | Media | Mission Control |
| Mailbox inter-agente (mensajería separada de tareas) | Media | GoClaw |
| Tipos de ejecución formales (Sequential/Parallel/DAG/Loop) | Baja | AgenticGoKit |
| Detección de prompt injection a nivel plataforma | Baja | GoClaw |
| Identidad criptográfica de agente (Ed25519) | Baja | MS Governance Toolkit |

---

## 4. Mejoras Propuestas (Priorizadas)

### MEJORA-1: Modelo de Checkpoints por Paso (Time-Travel)
**Referencia:** LangGraph
**OP relacionada:** OP-092
**Descripción:**
Actualmente Orquesta tiene planificado el time-travel pero no implementado. LangGraph tiene el modelo más maduro del ecosistema.

**Qué adoptar del modelo LangGraph:**
- Cada paso de ejecución del agente genera un checkpoint con: `session_id`, `step_number`, `timestamp`, `state_snapshot` (payload serializado), `parent_step`
- Los checkpoints son inmutables una vez escritos
- Para "viajar en el tiempo": crear una nueva sesión con `parent_session_id` + `from_step` — la sesión hereda el estado hasta ese paso y continúa desde ahí
- El fan-out (sesión padre → N sesiones hija) se modela como checkpoints con `branch_id`

**Esquema de tabla sugerido:**
```sql
CREATE TABLE session_checkpoints (
    id          INTEGER PRIMARY KEY,
    session_id  INTEGER NOT NULL REFERENCES sesiones(id),
    step        INTEGER NOT NULL,
    branch_id   TEXT,
    parent_step INTEGER,
    payload     BLOB NOT NULL,  -- estado serializado
    created_at  DATETIME NOT NULL,
    UNIQUE(session_id, step, branch_id)
);
```

**Impacto:** Debugging de sesiones fallidas, auditoría de decisiones, recuperación de estado.

---

### MEJORA-2: Versionado de Skills por Sesión
**Referencia:** Langfuse
**OP relacionada:** OP-121
**Descripción:**
Orquesta inyecta skills al inicio de sesión pero no registra qué versión de cada skill estaba activa. Esto impide correlacionar el comportamiento del agente con la versión del skill que lo provocó.

**Qué adoptar:**
- En el momento de inicio de sesión, registrar un snapshot de los skills inyectados: `skill_id`, `version`, `content_hash`
- Este snapshot queda ligado a la sesión de forma permanente (no mutable)
- En la UI de sesión, mostrar los skills activos en ese momento — permite comparar sesiones con distintas versiones de skills

**Esquema de tabla sugerido:**
```sql
CREATE TABLE session_skills_snapshot (
    id          INTEGER PRIMARY KEY,
    session_id  INTEGER NOT NULL REFERENCES sesiones(id),
    skill_id    INTEGER NOT NULL REFERENCES skills(id),
    version     INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    injected_at DATETIME NOT NULL
);
```

**Impacto:** Trazabilidad completa. Cuando un agente se comporta mal, sabemos exactamente qué skills tenía en ese momento.

---

### MEJORA-3: Kill Switch por Agente
**Referencia:** Microsoft Agent Governance Toolkit
**Descripción:**
Actualmente detener un agente requiere intervención en el proceso externo o reinicio del conector. Se necesita un mecanismo de parada de emergencia por agente individual desde la API de Orquesta.

**Qué implementar:**
- Nuevo campo en la tabla de agentes: `kill_switch_active BOOLEAN DEFAULT FALSE`
- Endpoint: `POST /api/agentes/{id}/kill` — activa el kill switch
- El conector del agente comprueba este flag en cada heartbeat (o Orquesta lo notifica via el canal de control existente)
- Al activarse: la sesión activa pasa a estado `killed`, se registra en auditoría con timestamp y actor
- Endpoint: `POST /api/agentes/{id}/revive` — desactiva el kill switch y permite nueva sesión

**Impacto:** Control de emergencia sin reiniciar el servidor. Esencial para producción.

---

### MEJORA-4: Push WebSocket/SSE en el Dashboard
**Referencia:** Mission Control
**Descripción:**
El dashboard actual actualiza estado por polling. Cambiar a SSE (Server-Sent Events) es sencillo en Go y elimina la carga de polling continuo.

**Qué implementar:**
- Endpoint SSE: `GET /api/events` — stream de eventos del servidor
- Tipos de eventos iniciales: `agente.estado_cambio`, `sesion.inicio`, `sesion.fin`, `tarea.estado_cambio`, `presupuesto.alerta`
- El cliente (dashboard web) se suscribe al stream y actualiza la UI reactivamente
- Fallback: si SSE no está disponible, el cliente vuelve a polling

**Ventaja sobre WebSocket:** SSE es unidireccional (servidor → cliente), más simple, funciona sobre HTTP/1.1, reconexión automática nativa en navegadores.

**Impacto:** Dashboard en tiempo real, menor carga de servidor, mejor experiencia de usuario.

---

### MEJORA-5: Mailbox Inter-Agente
**Referencia:** GoClaw
**Descripción:**
Actualmente los agentes se coordinan a través de tareas. Un mailbox es un canal de mensajería ligero y directo entre agentes, separado del sistema de tareas — útil para notificaciones, consultas rápidas, y coordinación informal.

**Qué implementar:**
- Tabla `mensajes_agente`: `from_agente_id`, `to_agente_id`, `subject`, `body`, `thread_id`, `read_at`, `created_at`
- Endpoints: `POST /api/agentes/{id}/mensajes` (enviar), `GET /api/agentes/{id}/mensajes` (leer bandeja)
- Los mensajes se entregan al agente en el próximo ciclo de heartbeat o via SSE (ver MEJORA-4)
- Diferencia con tareas: los mensajes no tienen estado ni dependencias — son efímeros y conversacionales

**Impacto:** Coordinación más natural entre agentes. Reduce el uso de tareas para comunicación informal.

---

## 5. Lo que NO adoptar (y por qué)

| Idea | Motivo para no adoptar |
|---|---|
| Kubernetes CRDs (Humanlayer ACP) | Añade complejidad de infraestructura que contradice el principio de simplicidad del proyecto. SQLite + daemon es suficiente para el caso de uso municipal. |
| Identidad criptográfica Ed25519 (MS Toolkit) | Overhead de gestión de claves sin beneficio claro en el contexto de un workspace local confiable. Revisar si escala a despliegues multi-nodo. |
| Pipeline de evaluación LLM (Langfuse eval) | Fuera de alcance para la versión actual. El sistema de votación de Orquesta cubre la gobernanza de calidad a nivel arquitectónico. |
| Tipos de proceso formales DAG/Loop (AgenticGoKit) | El modelo de tareas con dependencias de Orquesta ya cubre estos casos. Añadir tipos formales sería sobre-ingeniería en esta fase. |

---

## 6. Roadmap de Implementación Sugerido

| Fase | Mejoras | Justificación |
|---|---|---|
| **Fase A** (inmediata) | MEJORA-2 (versionado skills) + MEJORA-3 (kill switch) | Bajo coste de implementación, alto valor operativo inmediato |
| **Fase B** (corto plazo) | MEJORA-4 (SSE push dashboard) | Mejora la experiencia del usuario sin cambios de esquema |
| **Fase C** (medio plazo) | MEJORA-1 (checkpoints time-travel) | Requiere diseño cuidadoso del esquema y coordinación con OP-092 |
| **Fase D** (cuando haya demanda) | MEJORA-5 (mailbox inter-agente) | Útil pero no bloqueante; implementar cuando la coordinación informal sea un pain point real |

---

## 7. Referencias

- GoClaw: https://github.com/nextlevelbuilder/goclaw
- Humanlayer ACP: https://github.com/humanlayer/agentcontrolplane
- Eino (ByteDance): https://github.com/cloudwego/eino
- AgenticGoKit: https://github.com/AgenticGoKit/AgenticGoKit
- LangGraph: https://github.com/langchain-ai/langgraph
- CrewAI: https://github.com/crewAIInc/crewAI
- Mission Control: https://mc.builderz.dev
- Langfuse: https://github.com/langfuse/langfuse
- Microsoft Agent Governance Toolkit: https://github.com/microsoft/agent-governance-toolkit
- AgentOps: https://github.com/AgentOps-AI/agentops
