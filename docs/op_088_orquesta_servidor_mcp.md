<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# OP-088 — Orquesta como Servidor MCP (Patrón "Control Plane Organizacional")

## Estado de vigencia 2026-05-26

Documento en cuarentena T124: vision historica de integracion con gestores
externos. No es contrato vivo del nucleo, no obliga a instalar OpenClaw y no
declara `/api/mcp` ni rutas legacy como superficie vigente. La superficie MCP
actual es el adaptador `modulos/orquesta-mcp`; el transporte real del servidor,
cuando la composicion lo habilita, es opt-in en `cmd/orquesta-server` y usa
`/mcp` con tools/resources registrados por puerto.
Las menciones historicas a `--mcp-stdio` no autorizan backend Goal stdio ni
operacion normal por stdio; Goal-first vigente usa `app_server_tmux`.

OpenClaw, Claude Desktop, LangGraph u otros gestores externos solo pueden entrar
como clientes/adaptadores de composicion con refs opacas, write-set, identidad,
auditoria y pruebas propias. El indice federado debe excluir este documento de
planificacion automatica salvo una tarea que declare explicitamente
`legacy_external_orchestrator_doc_quarantine` o composicion externa.

## Objetivo

Formalizar la arquitectura para que gestores externos de agentes (como OpenClaw, Claude Desktop o frameworks futuros) puedan interactuar y gobernar los proyectos locales a través del Model Context Protocol (MCP), manteniendo a Orquesta como la única fuente de verdad y autoridad.

Esta decisión afecta a:
- La superficie de la API de Orquesta.
- La gestión de los Worktrees y sesiones.
- La distribución de las Reglas, Skills y Workflows a agentes externos.
- La auditoría de seguridad y control de cambios en el proyecto.

## Problema real

A medida que aparecen nuevos frameworks y gestores de agentes autónomos cada vez más potentes (como OpenClaw), es una tentación dejar que operen directamente sobre el repositorio `~/Trabajo`. 

Sin embargo, si un agente externo accede directamente:
- No respeta las reglas LOPD o de seguridad (ej. OP-069, puertas de aprobación).
- No genera registros en el `audit_log` de la base de datos de Orquesta.
- Ignora el sistema de "Locks" y Worktrees, provocando colisiones.
- No es consciente de los votos, propuestas y 14 fases del ciclo de vida de Orquesta.

## Principio rector propuesto

**Invertir el flujo: Orquesta como Servidor, la IA como Cliente.**

Orquesta debe dejar de ser visto (internamente) solo como un "script que lanza IAs", y debe consolidarse como un **Servidor MCP** (`orquesta serve --mcp-stdio/http`). 

Bajo este modelo:
1.  **OpenClaw (o similar)** aporta el motor de razonamiento continuo y delegación multi-agente.
2.  **Orquesta** aporta el "Control Plane Organizacional": custodia las bases de datos, expone el estado, protege el `main` de integraciones precipitadas, e impone los "Guardrails".

## Interfaz MCP Propuesta para Orquesta

Para que un gestor como OpenClaw opere correctamente, Orquesta expondrá:

### 1. Resources (Estado pasivo y observabilidad)
- `orquesta://proyectos/listar`: Árbol de proyectos y estado de Git.
- `orquesta://proyecto/{id}/estado`: Fases superadas, ramas activas, locks vivos.
- `orquesta://tareas/bloqueos`: Lista de tareas con estado `bloqueada` por requerimientos humanos o técnicos.

### 2. Prompts (Contexto y Briefings)
- `orquesta://prompts/briefing/{agente}`: Inyección automática del catálogo activo de `reglas`, `skills` y `workflows` servido por Orquesta al contexto de OpenClaw.

### 3. Tools (Acciones Controladas)
Acciones que OpenClaw podrá ejecutar, todas filtradas por los controles de Orquesta:
- `orquesta.tarea.listar(filtros)`
- `orquesta.tarea.tomar(tarea_id, agente)`
- `orquesta.tarea.completar(tarea_id, commit_hash)`
- `orquesta.propuesta.votar(codigo, voto, justificacion)`
- `orquesta.runtime.checkpoint(sesion_id, payload)`
- `orquesta.repo.request_merge(rama_origen)`

### 4. ¿Son redundantes OpenClaw y Orquesta?
Es la duda más natural: si OpenClaw ya es un orquestador de agentes, ¿para qué queremos a Orquesta?
La respuesta es que cumplen roles completamente distintos y **ortogonales**:

*   **OpenClaw es el "Orquestador de Inteligencia" (Agent Orchestrator):** Se encarga de saber qué LLM es mejor para cada tarea rutinaria, enrutar la memoria, gestionar la bolsa de contexto para no fallar (Token Limits), aislar los procesos IA en contenedores ligeros, y comunicarse con el usuario por Telegram/Discord.
*   **Orquesta es el "Orquestador Organizacional" (Domain Orchestrator):** No le importa qué inteligencia se use (si es Claude, un becario humano o OpenClaw). Su trabajo es asegurar que **el código y la Ley de la Diputación se respetan**. Controla el acceso a Git, exige auditoría a través del backend activo de Orquesta, impone las 14 fases (hasta que no hay documento OP-XXX, no se programa), gestiona el multi-tenancy y provee los contenedores (`git worktrees`) aislados.

Usar OpenClaw sobre tu repositorio a lo bruto sería el caos. **Orquesta como Servidor MCP es el muro que subordina la Inteligencia Creativa (OpenClaw) a las Reglas de Negocio (Orquesta)**.

## 5. Aclaración vital: Esto es 100% Opcional
Es fundamental aclarar que **Orquesta no va a depender de OpenClaw ni de ningún framework externo para funcionar.** 

Siguiendo la Arquitectura Hexagonal de Orquesta:
- El uso nativo (Terminal, CLI local, `orquesta serve`, `pty`, y tu futura interfaz Web/Desktop) sigue siendo el **motor por defecto y completamente autosuficiente**.
- El "Servidor MCP" es simplemente un **Adaptador de Entrada extra**. Igual que Orquesta puede recibir órdenes por la Web o la CLI, simplemente se abre una puerta nueva para que, *si el día de mañana alguien quiere conectar un OpenClaw o un LangGraph*, pueda hacerlo sin romper nada. De momento, no necesitas instalar nada extra y la app sigue operando de forma autónoma con los conectores locales (como has estado haciendo hasta hoy).

## Preguntas de Voto

### A. ¿Quién mantiene el monopolio de escritura en el repositorio local?
Opciones:
- `A1`: El agente externo (OpenClaw) escribe directamente en el FS, Orquesta solo mira pasivamente.
- `A2`: Orquesta provee Worktrees aislados; el agente escribe en el Worktree, pero `main` solo se toca vía tools de Orquesta.

Recomendación: `A2`.

### B. Mapeo de Identidades MCP
Opciones:
- `B1`: Todas las peticiones del gestor externo salen bajo una única identidad "mcp-client" genérica.
- `B2`: El servidor MCP exige que en cada petición `tool call` el gestor externo firme con la identidad real del agente (ej. `OpenClaw-Worker1`) para validarla contra la tabla `agentes`.

Recomendación: `B2`.

## Impacto esperado si se aprueba

- **Agnosticismo total del LLM**: Podremos usar cualquier framework potente de la industria que soporte MCP, sin que nuestras reglas de consistencia de software (ENS, LOPD, testing gates) se devalúen.
- **Preparación para el futuro**: Orquesta se posiciona como el "Jefe de Producto/Scrum Master automatizado", mientras comoditiza el rol de los "Agentes Programadores" delegándolos a la mejor tecnología del momento.
- Encaje perfecto con las implementaciones preparadas en **OP-087** (Runtime Orders & Mailbox) que serán expuestas directamente como tools MCP.

## Recomendación inicial de Antigravity
- Posición recomendada: **ACUERDO**.
- Fundamento: Evita reconstruir la lógica de "Swarms" (OpenClaw/LangGraph) dentro de Orquesta en Go, y permite apalancar esos gestores externos sin perder la soberanía de los datos municipales y el compliance regulatorio.
