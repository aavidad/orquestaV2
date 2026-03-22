<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Diseño mínimo — Observabilidad pasiva de runtimes

## Problema

El panel de Orquesta ya muestra estado global, tareas y propuestas, pero todavía no puede responder con precisión a preguntas como estas sin molestar al agente:

- qué agentes están realmente arrancados
- qué agentes hijos cuelgan de cada agente padre
- si un agente está trabajando, pensando, esperando I/O o parado
- cuántos procesos o hilos tiene abiertos
- qué modelo, perfil y rama está usando

Consultar activamente al agente con comandos como `/status` no debe ser la base de la observabilidad porque:

- roba foco a la sesión
- puede consumir tiempo o tokens
- introduce latencia artificial
- en algunos runtimes obliga al agente a responder o renderizar estado

## Objetivo

Que Orquesta mantenga un árbol de runtimes y un estado pasivo y continuo de cada agente sin interrumpir su trabajo.

La web y la futura app de escritorio deben limitarse a consultar y operar contra Orquesta. No deben hablar directamente con el runtime, el sistema operativo ni la base de datos.

## Decisión propuesta

`OP-082` propone un enfoque híbrido de observabilidad pasiva:

1. árbol de runtimes en Orquesta
2. telemetría pasiva de proceso y sistema operativo
3. adaptadores por proveedor cuando exista un canal no interactivo
4. fallback genérico si el runtime no expone mejor información

## Entidades mínimas

### `runtime_instances`

Instancia viva o reciente de un runtime asociado a un agente.

Campos sugeridos:

```sql
CREATE TABLE IF NOT EXISTS runtime_instances (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    agente                TEXT NOT NULL REFERENCES agentes(nombre),
    proyecto_id           INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    sesion_id             INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    parent_runtime_id     INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
    provider              TEXT NOT NULL,
    connector             TEXT NOT NULL,
    external_session_id   TEXT NOT NULL DEFAULT '',
    logical_state         TEXT NOT NULL DEFAULT 'arrancando',
    process_state         TEXT NOT NULL DEFAULT 'desconocido',
    pid                   INTEGER,
    ppid                  INTEGER,
    child_count           INTEGER NOT NULL DEFAULT 0,
    thread_count          INTEGER NOT NULL DEFAULT 0,
    model                 TEXT NOT NULL DEFAULT '',
    reasoning             TEXT NOT NULL DEFAULT '',
    task_profile          TEXT NOT NULL DEFAULT '',
    cwd                   TEXT NOT NULL DEFAULT '',
    branch                TEXT NOT NULL DEFAULT '',
    last_event_at         DATETIME,
    last_heartbeat_at     DATETIME,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### `runtime_telemetry_samples`

Muestras temporales para el panel y análisis.

Campos sugeridos:

```sql
CREATE TABLE IF NOT EXISTS runtime_telemetry_samples (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    runtime_id            INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
    cpu_pct               REAL NOT NULL DEFAULT 0,
    mem_bytes             INTEGER NOT NULL DEFAULT 0,
    rss_bytes             INTEGER NOT NULL DEFAULT 0,
    open_fds              INTEGER NOT NULL DEFAULT 0,
    child_count           INTEGER NOT NULL DEFAULT 0,
    thread_count          INTEGER NOT NULL DEFAULT 0,
    logical_state         TEXT NOT NULL DEFAULT '',
    source                TEXT NOT NULL,
    sample_json           TEXT NOT NULL DEFAULT '{}',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### `runtime_events`

Eventos relevantes para auditoría y timeline.

Campos sugeridos:

```sql
CREATE TABLE IF NOT EXISTS runtime_events (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    runtime_id            INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
    kind                  TEXT NOT NULL,
    level                 TEXT NOT NULL DEFAULT 'info',
    message               TEXT NOT NULL,
    payload_json          TEXT NOT NULL DEFAULT '{}',
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Estados lógicos recomendados

- `arrancando`
- `disponible`
- `pensando`
- `ejecutando_herramienta`
- `esperando_io`
- `bloqueado`
- `pausado`
- `handoff`
- `cerrado`

## Adaptadores previstos

### `generic_process`

Base obligatoria para todos los runtimes.

Lee de forma pasiva:

- `pid`, `ppid`
- hijos
- hilos
- CPU y memoria
- `cwd`
- tiempo desde última actividad observable

No requiere colaboración del agente.

### `claude_statusline`

Claude Code documenta una `status line` configurable que puede emitir JSON contextual sin interrumpir la sesión. Esta debe ser la primera integración específica.

### `gemini_streamjson`

Gemini CLI expone modo `stream-json` y telemetría. Es útil para sesiones lanzadas por Orquesta y para análisis en background.

### `codex_appserver`

Codex tiene indicios de `app-server` y `--remote`. Si el canal resulta estable y no intrusivo, puede usarse como adaptador específico. Si no, se mantiene `generic_process`.

## Endpoints mínimos

- `GET /api/runtimes`
- `GET /api/runtimes/tree`
- `GET /api/runtimes/:id`
- `GET /api/runtimes/:id/samples`
- `GET /api/runtimes/:id/events`

## Panel web mínimo

El panel debe poder mostrar:

- árbol padre/hijos
- agente, proyecto y conector
- estado lógico
- `pid`
- número de hijos
- número de hilos
- CPU y memoria
- modelo, razonamiento y perfil
- última actividad
- rama y worktree

La información del panel debe salir de la API de Orquesta. El panel no debe inspeccionar procesos ni runtimes por su cuenta.

## Regla importante

La observabilidad primaria debe ser pasiva. Si una fuente exige que el agente se detenga, renderice estado o responda activamente, esa fuente no puede ser el mecanismo principal del panel.

## Reparto sugerido

- `Codex1`
  modelo de datos y endpoints base

- `Codex2`
  panel web y árbol de runtimes

- `Codex3`
  adaptadores de telemetría por proveedor

- `antigravity`
  documentación ES/EN y guía de uso
