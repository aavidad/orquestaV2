<!--
Software libre bajo licencia GNU GPL v3
Proyecto: PlataformaMunicipal — Orquesta
Autor: Alberto Avidad Fernandez
Oficina de Software Libre (OSL) - Diputacion de Granada
-->

# Minimal Design — Passive Runtime Observability

## Problem

The Orquesta panel already shows global status, tasks, and proposals, but it still cannot answer these questions accurately without disturbing the agent:

- which agents are actually running
- which child agents belong to each parent agent
- whether an agent is working, thinking, waiting for I/O, or paused
- how many processes or threads it currently owns
- which model, profile, and branch it is using

Actively querying the agent with commands such as `/status` must not be the basis of observability because it:

- steals focus from the session
- may consume time or tokens
- adds artificial latency
- can force the runtime to render or answer state interactively

## Goal

Orquesta must maintain a runtime tree and a passive, continuous state view for each agent without interrupting its work.

The web UI and the future desktop app must remain thin clients. They should only query and command Orquesta, never inspect the runtime, the operating system, or the database directly.

## Proposed Decision

`OP-082` proposes a hybrid passive observability approach:

1. a runtime tree inside Orquesta
2. passive process and operating-system telemetry
3. provider-specific adapters when a non-interactive channel exists
4. a generic fallback when the runtime exposes nothing better

## Minimum Entities

### `runtime_instances`

A live or recent runtime instance attached to an agent.

Suggested fields:

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
    process_state         TEXT NOT NULL DEFAULT 'unknown',
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

Time-series samples for panel rendering and analysis.

Suggested fields:

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

Relevant events for audit and timeline views.

Suggested fields:

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

## Recommended Logical States

- `arrancando`
- `disponible`
- `pensando`
- `ejecutando_herramienta`
- `esperando_io`
- `bloqueado`
- `pausado`
- `handoff`
- `cerrado`

## Planned Adapters

### `generic_process`

Mandatory baseline for every runtime.

It reads passively:

- `pid`, `ppid`
- child process count
- thread count
- CPU and memory
- `cwd`
- time since last visible activity

No cooperation from the agent is required.

### `claude_statusline`

Claude Code documents a configurable status line that can emit contextual JSON without interrupting the session. This should be the first provider-specific integration.

### `gemini_streamjson`

Gemini CLI exposes `stream-json` and telemetry. It is useful for sessions launched by Orquesta and for background analysis.

### `codex_appserver`

Codex appears to expose `app-server` and `--remote`. If that channel proves stable and non-intrusive, it can be used as a provider adapter. Otherwise Orquesta should keep `generic_process`.

## Minimum Endpoints

- `GET /api/runtimes`
- `GET /api/runtimes/tree`
- `GET /api/runtimes/:id`
- `GET /api/runtimes/:id/samples`
- `GET /api/runtimes/:id/events`

## Minimum Web Panel

The panel should show:

- parent/child runtime tree
- agent, project, and connector
- logical state
- `pid`
- child count
- thread count
- CPU and memory
- model, reasoning, and profile
- last activity
- branch and worktree

Panel data must come from the Orquesta API. The UI must not inspect processes or runtimes on its own.

## Important Rule

Primary observability must remain passive. If a data source requires the agent to stop, render status, or actively answer, it cannot be the panel’s main mechanism.

## Suggested Split

- `Codex1`
  data model and base endpoints

- `Codex2`
  web panel and runtime tree

- `Codex3`
  provider-specific telemetry adapters

- `antigravity`
  ES/EN documentation and usage guide
