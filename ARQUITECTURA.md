<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Arquitectura objetivo de Orquesta

Documento canonico consolidado:

- `docs/BIBLIA_APP_ORQUESTA.md`

Este fichero sigue siendo la referencia especifica de arquitectura, pero la doctrina estatica del proyecto se consolida ya en la biblia para evitar divergencias entre agentes y documentos dispersos.

Este fichero resume la direccion arquitectonica.
La especificacion ampliada de trabajo y la matriz de voto viven en:

- `docs/orquesta_v1_vision.md`
- `docs/op_049_matriz_voto.md`
- `docs/orquesta_v1_roadmap.md`
- `docs/diseno_microprogramacion_dirigida_agentes.md`
- `docs/operacion_agentes_manuales.md`

### Manuales Oficiales v1.0
- [Manual de Usuario](docs/manual_usuario.md): Guía funcional y operativa.
- [Manual de Programador](docs/manual_programador.md): Guía de integración y APIs (MCP, A2UI).
- [Manual de Técnico de Sistemas](docs/manual_tecnico_sistemas.md): Guía de despliegue y mantenimiento (Docker, DB).

## Objetivo

Orquesta pasa de ser un coordinador local de tareas a ser el nucleo de orquestacion de todo el workspace `~/Trabajo`.
El nucleo no debe depender de un LLM concreto. Debe depender de un contrato estable de `conector`.

Nota operativa actual:

- el repositorio operativo actual vive en `~/Trabajo/orquesta`
- las referencias antiguas a la ruta legacy previa del repositorio son historicas y no deben usarse como verdad operativa
- cualquier traslado fisico futuro debe tratarse como migracion explicita, no como suposicion documental

## Principios

1. La identidad del agente es estable.
   `codex1`, `codex2`, `claude1`, `gemini1` son identidades de trabajo. No dependen del proyecto.

2. El proyecto es una entidad explicita.
   Un agente se asigna a un proyecto. La sesion vive dentro de esa asignacion.

3. El conector encapsula el runtime.
   El nucleo sabe que existe un conector con transporte, comando y metadata.
   El nucleo no necesita saber si por debajo corre Codex CLI, Claude Code, Gemini CLI o un runtime futuro.

4. La sesion es reanudable.
   Cada sesion puede guardar `external_session_id`, `resume_payload_json`, `cwd`, `branch` y `resumen_continuidad`.

5. Las reglas, skills y workflows salen de la BD.
   El agente recibe su briefing desde Orquesta, no desde ficheros dispersos.

6. La base de datos es un adaptador, no el núcleo.
   La lógica de negocio no debe depender de SQLite ni de SQL directo.
   El núcleo habla con puertos; la persistencia entra por adaptadores.

7. La web y la app desktop son clientes finos.
   No deben hablar directamente con SQLite, Git, runtimes, proveedores LLM ni conectores externos.
   Solo hablan con Orquesta y Orquesta actúa como único plano de control.

8. La delegacion a agentes es cerrada y verificable.
   El orquestador define el contrato exacto de trabajo: archivo, firma, write_set, tests obligatorios y restricciones.
   Los agentes ejecutan microtareas de implementacion. No rediseñan arquitectura ni abren frentes amplios por su cuenta.

## Desarrollo hexagonal

- `core` o `application`
  Casos de uso, reglas y servicios.
- `ports`
  Interfaces de persistencia, runtime, VCS, memoria, investigación y notificación.
- `adapters/inbound`
  CLI, HTTP, MCP, web, app desktop.
- `adapters/outbound`
  SQLite, Git, conectores LLM, web search, filesystem.

Regla práctica:

- lo nuevo debe entrar por servicio + puertos
- `db/` no debe volver a actuar como núcleo de aplicación
- la migración del legado se hará por fases
- la unidad canónica de delegación a agentes es la especificación de función, no el frente amplio
- una base nueva debe nacer neutra en agentes: la persistencia puede sembrar catálogo y conectores base, pero no debe registrar flotas legacy por defecto
- el alta de agentes y la composición de flota se hacen de forma explícita desde la app/API; no desde seeds ocultas del esquema
- `server_autobootstrap` es opt-in explícito y no puede activarse por defecto en una instalación nueva
- **Módulos e Idioma:** Todos los adaptadores y aplicaciones de dominio deben utilizar nomenclatura exclusiva en **castellano**, habitualmente sufijados con `app` (P.ej. usar `tareasapp`, `sesionesapp` en vez de `taskapp` o `sessionapp`). El objetivo es erradicar completamente el inglés de los nombres de carpeta para evitar confusiones de los modelos LLM (agentes).
- **i18n por defecto:** toda superficie nueva visible, clave pública o documento nuevo debe nacer preparada para i18n; no se admite crear funcionalidad nueva que obligue a rehacer la internacionalización después.

## Modelo actual

### Proyectos

- Tabla `proyectos`
- Tipos: `raiz`, `grupo`, `repo`
- Soporta arbol de proyectos con `parent_id`
- Registra `ruta_abs` y `slug`

### Asignaciones

- Tabla `asignaciones`
- Une `agente -> proyecto`
- Solo una asignacion activa por agente
- Sirve para repartir capacidad: por ejemplo `AutofirmaV2 = 2 agentes`, `ProyectoX = 4 agentes`

### Sesiones

- Tabla `sesiones`
- Guarda:
  - `proyecto_id`
  - `conector_id`
  - `cwd`
  - `herramienta`
  - `external_session_id`
  - `resume_payload_json`
  - `resumen_continuidad`
  - `branch`
  - `heartbeat_at`
  - `host`
  - `pid`

### Conectores

- Tabla `conectores`
- Contrato mínimo:
  - `slug`
  - `nombre`
  - `transporte`
  - `comando`
  - `args_json`
  - `env_json`
  - `metadata_json`

Ejemplos iniciales:

- `codex-cli`
- `claude-code`
- `gemini-cli`

## Comandos ya soportados

- `orquesta proyecto descubrir <ruta>`
- `orquesta proyecto listar`
- `orquesta asignacion activar --agente <agente> --proyecto <slug|id>`
- `orquesta conector listar`
- `orquesta conector ver <slug|id>`
- `orquesta conector registrar ...`
- `orquesta sesion inicio <agente> --proyecto <slug> --conector <slug>`
- `orquesta sesion guardar <agente> --proyecto <slug> ...`
- `orquesta sesion continuar <agente> --proyecto <slug>`

## Patrones tomados de los repos revisados

### `Z-M-Huang/claude-codex-gemini`

- Gestion explicita de sesiones por runtime
- Persistencia de contexto reanudable
- Registro estructurado del estado de cada agente

Aplicacion en Orquesta:

- `conectores`
- `external_session_id`
- `resume_payload_json`
- `resumen_continuidad`

### `nxtg-ai/forge-orchestrator`

- Nucleo desacoplado del runtime
- Capa de adaptadores/proveedores
- Coordinacion centrada en tareas y ejecucion

Aplicacion en Orquesta:

- el nucleo depende del contrato `conector`
- el runtime se mueve a datos configurables y no a condicionales por modelo

### `smtg-ai/claude-squad`

- Identidad estable de agentes
- Trabajo paralelo con contexto separado
- Afinidad entre agente y contexto de trabajo

Aplicacion en Orquesta:

- identidad estable de agentes
- asignaciones activas por proyecto
- sesiones reanudables por proyecto

### `dnnyngyen/gemini-cli-orchestrator`

- Operativa CLI clara para lanzar y seguir agentes
- Flujo sencillo de sesion y continuidad

Aplicacion en Orquesta:

- comandos `sesion inicio`, `sesion guardar`, `sesion continuar`
- modelo de conector CLI como primer transporte

### `oh-my-codex`

- estado canónico de equipo y workers fuera del pane
- `dispatch request` durable con transiciones `pending -> notified -> delivered/failed`
- `inbox` y `mailbox` como verdad de trabajo, no `send-keys` ad hoc
- guard de readiness antes de inyectar en `tmux`
- worker protocol con `ACK` inicial, claim-safe de tarea y marcado explícito de entrega

Aplicacion en Orquesta:

- `tmux` se usa como transporte persistente, no como fuente de verdad
- la entrega a agentes debe pasar por estado duradero y confirmación de consumo
- la unidad de trabajo delegada debe ser microprogramación dirigida por especificación de función
- el agente debe recibir una instrucción acotada y verificable, no un frente arquitectónico abierto

Referencia documental asociada:

- `docs/diseno_microprogramacion_dirigida_agentes.md`

## Siguiente fase recomendada

1. Crear un paquete de dominio para conectores.
   Por ejemplo `internal/runtime` o `runtime/` con interfaz:

   ```go
   type Driver interface {
       Lanzar(ctx context.Context, req Lanzamiento) (SesionExterna, error)
       Reanudar(ctx context.Context, req Reanudacion) (SesionExterna, error)
       Heartbeat(ctx context.Context, sesion SesionExterna) error
       Detener(ctx context.Context, sesion SesionExterna) error
   }
   ```

2. Resolver el driver por `conector.transporte`.
   El nucleo decide por `cli`, `mcp_stdio`, `mcp_http`, `api`, sin codificar proveedores concretos.

3. Hacer de `orquesta serve` el escritor principal.
   La CLI y la web deberían hablar con el servidor local para reducir contencion sobre SQLite.

4. Añadir politicas de reparto de agentes.
   Ejemplo: `slots_por_proyecto`, prioridad, exclusiones y colas.

5. Inyectar reglas, skills y workflows por sesion/proyecto.
   No solo por rol del agente, tambien por proyecto y por tipo de conector.

## Fase de refactor competitivo

La refactorizacion competitiva solo debe existir si mantiene la filosofia del proyecto y mejora o preserva la seguridad.

### Reglas duras

1. Backup previo obligatorio.
   Antes de modificar nada, Orquesta debe generar una copia de seguridad o snapshot verificable del estado de trabajo.

2. Contrato congelado.
   Antes del experimento se fijan tests, benchmark, firmas publicas y restricciones funcionales.

3. Aislamiento por candidato.
   Cada candidato corre en su propia rama o worktree. Nunca sobre la misma copia de trabajo.

4. Seguridad no regresiva.
   Ningun candidato puede ganar si reduce controles de seguridad, endurecimiento, auditoria o validaciones.

5. Filosofia del proyecto no regresiva.
   Ningun candidato puede ganar si rompe convenciones arquitectonicas, reglas de dominio o estilo de mantenimiento acordado.

6. Revision minima por pares no autores.
   El ganador solo puede seleccionarse si al menos dos agentes revisores independientes lo aprueban.
   Ningun revisor puede ser autor del candidato revisado.

7. Correctitud antes que rendimiento.
   El orden de evaluacion es: correccion, seguridad, filosofia del proyecto, rendimiento, memoria, mantenibilidad.

### Modelo recomendado

- `experimento_refactor`
- `candidato_refactor`
- `revision_candidato`
- `metrica_candidato`
- `backup_experimento`

### Politica inicial en configuracion

- `refactor_backup_required=true`
- `refactor_min_reviewers=2`
- `refactor_reviewer_must_be_non_author=true`
- `refactor_preserve_security=true`
- `refactor_preserve_project_philosophy=true`

## Restricciones

- No perder datos actuales.
- Todas las migraciones deben ser aditivas e idempotentes.
- SQLite sigue valiendo como almacenamiento inicial, pero el acceso concurrente debe centralizarse.
