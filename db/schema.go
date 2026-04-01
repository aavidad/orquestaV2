package db

// Schema define el esquema completo de la base de datos de orquestación.
// Las migraciones se aplican en orden; nunca se modifican las existentes.
const Schema = `
-- ─── Configuración global ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS config (
    clave TEXT PRIMARY KEY,
    valor TEXT NOT NULL
);

-- ─── Conectores de agentes / LLM ───────────────────────────────────────────
CREATE TABLE IF NOT EXISTS conectores (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    slug          TEXT    NOT NULL UNIQUE,
    nombre        TEXT    NOT NULL,
    transporte    TEXT    NOT NULL DEFAULT 'cli'
                           CHECK (transporte IN ('cli','mcp_stdio','mcp_http','api','otro')),
    comando       TEXT    NOT NULL DEFAULT '',
    args_json     TEXT    NOT NULL DEFAULT '[]',
    env_json      TEXT    NOT NULL DEFAULT '{}',
    metadata_json TEXT    NOT NULL DEFAULT '{}',
    activo        INTEGER NOT NULL DEFAULT 1,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS conectores_operacion (
    conector_id          INTEGER PRIMARY KEY REFERENCES conectores(id) ON DELETE CASCADE,
    estado_operativo     TEXT    NOT NULL DEFAULT 'activo'
                                 CHECK (estado_operativo IN ('activo','circuito_abierto')),
    motivo               TEXT    NOT NULL DEFAULT '',
    fallos_consecutivos  INTEGER NOT NULL DEFAULT 0,
    cooldown_until       DATETIME,
    circuito_abierto_at  DATETIME,
    ultimo_error         TEXT    NOT NULL DEFAULT '',
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ─── Proyectos del workspace ───────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS proyectos (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    slug       TEXT    NOT NULL UNIQUE,
    nombre     TEXT    NOT NULL,
    ruta_abs   TEXT    NOT NULL UNIQUE,
    tipo       TEXT    NOT NULL DEFAULT 'repo'
                         CHECK (tipo IN ('raiz','grupo','repo')),
    parent_id  INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    activo     INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS proyectos_operacion (
    proyecto_id       INTEGER PRIMARY KEY REFERENCES proyectos(id) ON DELETE CASCADE,
    estado_operativo  TEXT    NOT NULL DEFAULT 'activo'
                             CHECK (estado_operativo IN ('activo','esperando_humano','bloqueado_externo','cerrado')),
    motivo            TEXT    NOT NULL DEFAULT '',
    objetivo_pct      INTEGER NOT NULL DEFAULT 100,
    min_agentes       INTEGER NOT NULL DEFAULT 0,
    max_agentes       INTEGER NOT NULL DEFAULT 0,
    prioridad         INTEGER NOT NULL DEFAULT 100,
    resume_automatico INTEGER NOT NULL DEFAULT 1,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS proyectos_autonomia (
    proyecto_id             INTEGER PRIMARY KEY REFERENCES proyectos(id) ON DELETE CASCADE,
    enabled                 INTEGER NOT NULL DEFAULT 0,
    objetivo_general        TEXT    NOT NULL DEFAULT '',
    definition_of_done_json TEXT    NOT NULL DEFAULT '{}',
    max_workers             INTEGER NOT NULL DEFAULT 0,
    supervisor_agente       TEXT    NOT NULL DEFAULT '',
    reviewer_agente         TEXT    NOT NULL DEFAULT '',
    reserve_reviewer        INTEGER NOT NULL DEFAULT 1,
    reserve_supervisor      INTEGER NOT NULL DEFAULT 1,
    review_required         INTEGER NOT NULL DEFAULT 1,
    auto_create_tasks       INTEGER NOT NULL DEFAULT 1,
    auto_close_project      INTEGER NOT NULL DEFAULT 1,
    estado_autonomia        TEXT    NOT NULL DEFAULT 'activo'
                                     CHECK (estado_autonomia IN ('activo','esperando_review','esperando_humano','cerrando','cerrado')),
    last_supervision_at     DATETIME,
    last_review_at          DATETIME,
    created_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS autonomia_ciclos (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id   INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    kind          TEXT    NOT NULL CHECK (kind IN ('supervision','review','review_feedback','closure')),
    agente        TEXT    NOT NULL DEFAULT '',
    sesion_id     INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    runtime_id    INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
    input_json    TEXT    NOT NULL DEFAULT '{}',
    decision_json TEXT    NOT NULL DEFAULT '{}',
    resultado     TEXT    NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_autonomia_ciclos_proyecto_kind
ON autonomia_ciclos(proyecto_id, kind, id DESC);

CREATE INDEX IF NOT EXISTS idx_proyectos_autonomia_enabled
ON proyectos_autonomia(enabled, estado_autonomia, proyecto_id);

-- ─── Agentes registrados ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS agentes (
    nombre       TEXT PRIMARY KEY,
    rol          TEXT NOT NULL CHECK (rol IN ('programador','documentador','admin')),
    activo       INTEGER NOT NULL DEFAULT 0,   -- 1 = en sesión ahora mismo
    habilitado   INTEGER NOT NULL DEFAULT 1,   -- 0 = retirado por Alberto (no vota, no trabaja)
    ultima_sesion DATETIME
);

-- ─── Tareas ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tareas (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo        TEXT    NOT NULL,
    descripcion   TEXT    NOT NULL DEFAULT '',
    proyecto_id   INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    modulo        TEXT    NOT NULL DEFAULT '',
    estado        TEXT    NOT NULL DEFAULT 'libre'
                          CHECK (estado IN ('libre','asignada','en_progreso','completada','bloqueada','cancelada','backlog')),
    agente        TEXT    REFERENCES agentes(nombre),
    propuesta_id  INTEGER REFERENCES propuestas(id),  -- propuesta que originó la tarea
    prioridad     TEXT    NOT NULL DEFAULT 'media'
                          CHECK (prioridad IN ('alta','media','baja')),
    dependencias  TEXT    NOT NULL DEFAULT '[]',  -- JSON array de IDs
    creado_por    TEXT    NOT NULL DEFAULT 'alberto',
    commit_cierre TEXT    NOT NULL DEFAULT '',
    notas         TEXT    NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completada_at DATETIME
);

-- ─── Propuestas de orquestación ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS propuestas (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    codigo        TEXT    NOT NULL UNIQUE,   -- OP-030, OP-031…
    titulo        TEXT    NOT NULL,
    descripcion   TEXT    NOT NULL DEFAULT '',
    proyecto_id   INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    tipo          TEXT    NOT NULL DEFAULT 'implementacion'
                          CHECK (tipo IN ('implementacion','arquitectura','seguridad','backlog','otro')),
    estado        TEXT    NOT NULL DEFAULT 'abierta'
                          CHECK (estado IN ('abierta','consenso','rechazada','backlog')),
    propuesto_por TEXT    NOT NULL,
    distribuidor  TEXT    NOT NULL DEFAULT 'claude',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cerrada_at    DATETIME
);

-- ─── Votos ─────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS votos (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    propuesta_id INTEGER NOT NULL REFERENCES propuestas(id) ON DELETE CASCADE,
    agente       TEXT    NOT NULL REFERENCES agentes(nombre),
    posicion     TEXT    NOT NULL DEFAULT 'pendiente'
                         CHECK (posicion IN ('pendiente','acuerdo','desacuerdo','abstencion')),
    comentario   TEXT    NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(propuesta_id, agente)
);

-- ─── Bloqueos ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS bloqueos (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    tarea_id      INTEGER REFERENCES tareas(id) ON DELETE CASCADE,
    propuesta_id  INTEGER REFERENCES propuestas(id) ON DELETE CASCADE,
    motivo        TEXT    NOT NULL,
    bloqueado_por TEXT    NOT NULL,
    resuelto      INTEGER NOT NULL DEFAULT 0,
    resolucion    TEXT    NOT NULL DEFAULT '',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resuelto_at   DATETIME
);

-- ─── Sesiones ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sesiones (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    agente                TEXT    NOT NULL REFERENCES agentes(nombre),
    conector_id           INTEGER REFERENCES conectores(id) ON DELETE SET NULL,
    proyecto_id           INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    pool_id               INTEGER REFERENCES pools_capacidad(id) ON DELETE SET NULL,
    inicio                DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fin                   DATETIME,
    activa                INTEGER NOT NULL DEFAULT 1,
    estado                TEXT    NOT NULL DEFAULT 'activa'
                               CHECK (estado IN ('activa','pausada','cerrada','fallida')),
    cwd                   TEXT    NOT NULL DEFAULT '',
    herramienta           TEXT    NOT NULL DEFAULT '',
    external_session_id   TEXT    NOT NULL DEFAULT '',
    resume_payload_json   TEXT    NOT NULL DEFAULT '',
    resumen_continuidad   TEXT    NOT NULL DEFAULT '',
    branch                TEXT    NOT NULL DEFAULT '',
    heartbeat_at          DATETIME,
    host                  TEXT    NOT NULL DEFAULT '',
    pid                   INTEGER
);

-- ─── Tracking ligero de threads del supervisor ────────────────────────────
CREATE TABLE IF NOT EXISTS supervisor_threads (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    supervisor    TEXT    NOT NULL,
    proyecto_slug TEXT    NOT NULL DEFAULT '',
    session_id    TEXT    NOT NULL DEFAULT '',
    thread_id     TEXT    NOT NULL,
    kind          TEXT    NOT NULL DEFAULT 'subagent'
                          CHECK (kind IN ('leader','subagent')),
    mode          TEXT    NOT NULL DEFAULT '',
    status        TEXT    NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active','idle','closed')),
    source        TEXT    NOT NULL DEFAULT '',
    last_turn_id  TEXT    NOT NULL DEFAULT '',
    turn_count    INTEGER NOT NULL DEFAULT 1,
    metadata_json TEXT    NOT NULL DEFAULT '{}',
    first_seen_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(supervisor, session_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_supervisor_threads_supervisor_session
ON supervisor_threads(supervisor, session_id, last_seen_at DESC);

CREATE INDEX IF NOT EXISTS idx_supervisor_threads_project
ON supervisor_threads(proyecto_slug, supervisor, last_seen_at DESC);

-- ─── Asignaciones agente → proyecto ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS asignaciones (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    agente      TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    estado      TEXT    NOT NULL DEFAULT 'activa'
                       CHECK (estado IN ('planificada','activa','pausada','cerrada')),
    nota        TEXT    NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cerrada_at  DATETIME
);

-- ─── Locks de coordinación ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS locks (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id       INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    tarea_id          INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    sesion_id         INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    agente            TEXT    NOT NULL REFERENCES agentes(nombre),
    scope_type        TEXT    NOT NULL
                              CHECK (scope_type IN ('project','task','module','path','branch','worktree','otro')),
    scope_key         TEXT    NOT NULL,
    ruta_abs          TEXT    NOT NULL DEFAULT '',
    branch            TEXT    NOT NULL DEFAULT '',
    motivo            TEXT    NOT NULL DEFAULT '',
    token_lease       TEXT    NOT NULL DEFAULT '',
    estado            TEXT    NOT NULL DEFAULT 'activa'
                              CHECK (estado IN ('activa','liberada','expirada','fallida')),
    heartbeat_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at        DATETIME NOT NULL,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    liberada_at       DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo
ON locks(scope_type, scope_key)
WHERE estado = 'activa';

-- ─── Worktrees de ejecución aislada ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS worktrees (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id       INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    tarea_id          INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    lock_id           INTEGER REFERENCES locks(id) ON DELETE SET NULL,
    agente            TEXT    NOT NULL REFERENCES agentes(nombre),
    nombre            TEXT    NOT NULL,
    ruta_abs          TEXT    NOT NULL UNIQUE,
    branch            TEXT    NOT NULL,
    base_ref          TEXT    NOT NULL DEFAULT '',
    estado            TEXT    NOT NULL DEFAULT 'activa'
                              CHECK (estado IN ('activa','cerrada','fallida')),
    motivo            TEXT    NOT NULL DEFAULT '',
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    cerrada_at        DATETIME
);

CREATE TABLE IF NOT EXISTS review_gates (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id     INTEGER REFERENCES proyectos(id) ON DELETE CASCADE,
    tarea_id        INTEGER REFERENCES tareas(id) ON DELETE CASCADE,
    worktree_id     INTEGER REFERENCES worktrees(id) ON DELETE SET NULL,
    requested_by    TEXT    NOT NULL DEFAULT '',
    reviewer_agente TEXT    NOT NULL DEFAULT '',
    estado          TEXT    NOT NULL DEFAULT 'pendiente'
                              CHECK (estado IN ('pendiente','en_revision','cambios_pedidos','aprobado','bloqueado')),
    severity_max    TEXT    NOT NULL DEFAULT '',
    findings_json   TEXT    NOT NULL DEFAULT '[]',
    resolved_at     DATETIME,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_review_gates_proyecto_estado
ON review_gates(proyecto_id, estado, id DESC);

-- ─── Runtimes observables ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS runtime_instances (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agente              TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    sesion_id           INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    parent_runtime_id   INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
    provider            TEXT    NOT NULL DEFAULT '',
    connector           TEXT    NOT NULL DEFAULT '',
    external_session_id TEXT    NOT NULL DEFAULT '',
    logical_state       TEXT    NOT NULL DEFAULT 'arrancando',
    process_state       TEXT    NOT NULL DEFAULT 'desconocido',
    pid                 INTEGER,
    ppid                INTEGER,
    child_count         INTEGER NOT NULL DEFAULT 0,
    thread_count        INTEGER NOT NULL DEFAULT 0,
    model               TEXT    NOT NULL DEFAULT '',
    reasoning           TEXT    NOT NULL DEFAULT '',
    task_profile        TEXT    NOT NULL DEFAULT '',
    cwd                 TEXT    NOT NULL DEFAULT '',
    branch              TEXT    NOT NULL DEFAULT '',
    last_event_at       DATETIME,
    last_heartbeat_at   DATETIME,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_instances_sesion
ON runtime_instances(sesion_id)
WHERE sesion_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS runtime_telemetry_samples (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    runtime_id          INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
    cpu_pct             REAL    NOT NULL DEFAULT 0,
    mem_bytes           INTEGER NOT NULL DEFAULT 0,
    rss_bytes           INTEGER NOT NULL DEFAULT 0,
    open_fds            INTEGER NOT NULL DEFAULT 0,
    child_count         INTEGER NOT NULL DEFAULT 0,
    thread_count        INTEGER NOT NULL DEFAULT 0,
    logical_state       TEXT    NOT NULL DEFAULT '',
    source              TEXT    NOT NULL DEFAULT '',
    sample_json         TEXT    NOT NULL DEFAULT '{}',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS runtime_events (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    runtime_id          INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
    kind                TEXT    NOT NULL,
    level               TEXT    NOT NULL DEFAULT 'info',
    message             TEXT    NOT NULL DEFAULT '',
    payload_json        TEXT    NOT NULL DEFAULT '{}',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS runtime_transcript (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    runtime_id          INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
    handle_id           INTEGER REFERENCES runtime_handles(id) ON DELETE SET NULL,
    agente              TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    stream              TEXT    NOT NULL DEFAULT 'pty_out'
                               CHECK (stream IN ('stdin','pty_out','system')),
    byte_offset         INTEGER,
    text                TEXT    NOT NULL DEFAULT '',
    normalized_text     TEXT    NOT NULL DEFAULT '',
    classification      TEXT    NOT NULL DEFAULT '',
    handling_note       TEXT    NOT NULL DEFAULT '',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    handled_at          DATETIME
);

CREATE INDEX IF NOT EXISTS idx_runtime_transcript_runtime
ON runtime_transcript(runtime_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_runtime_transcript_agente
ON runtime_transcript(agente, id DESC);

CREATE INDEX IF NOT EXISTS idx_runtime_transcript_pending_signal
ON runtime_transcript(classification, handled_at, id DESC);

CREATE TABLE IF NOT EXISTS runtime_handles (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agente              TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    sesion_id           INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    runtime_id          INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
    transporte          TEXT    NOT NULL DEFAULT 'cli',
    handle_kind         TEXT    NOT NULL DEFAULT 'session',
    handle_ref          TEXT    NOT NULL DEFAULT '',
    estado              TEXT    NOT NULL DEFAULT 'activo'
                               CHECK (estado IN ('activo','pausado','cerrado','fallido')),
    lease_token         TEXT    NOT NULL DEFAULT '',
    capabilities_json   TEXT    NOT NULL DEFAULT '{}',
    metadata_json       TEXT    NOT NULL DEFAULT '{}',
    last_seen_at        DATETIME,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_sesion
ON runtime_handles(sesion_id)
WHERE sesion_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_runtime_handles_agente_estado
ON runtime_handles(agente, estado, id DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_agente_activo
ON runtime_handles(agente, estado, id);

CREATE TABLE IF NOT EXISTS runtime_orders (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agente              TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    runtime_id          INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
    handle_id           INTEGER REFERENCES runtime_handles(id) ON DELETE SET NULL,
    tipo                TEXT    NOT NULL,
    payload_json        TEXT    NOT NULL DEFAULT '{}',
    resultado_json      TEXT    NOT NULL DEFAULT '{}',
    error_text          TEXT    NOT NULL DEFAULT '',
    estado              TEXT    NOT NULL DEFAULT 'pendiente'
                               CHECK (estado IN ('pendiente','tomada','ejecutando','completada','fallida','expirada','cancelada')),
    claimed_by          TEXT    NOT NULL DEFAULT '',
    lease_token         TEXT    NOT NULL DEFAULT '',
    attempt_count       INTEGER NOT NULL DEFAULT 0,
    lease_expires_at    DATETIME,
    available_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at          DATETIME,
    finished_at         DATETIME,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_orders_estado_created
ON runtime_orders(estado, available_at, id);

CREATE TABLE IF NOT EXISTS runtime_mailbox (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    from_agente         TEXT    NOT NULL,
    to_agente           TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    runtime_order_id    INTEGER REFERENCES runtime_orders(id) ON DELETE SET NULL,
    kind                TEXT    NOT NULL,
    payload_json        TEXT    NOT NULL DEFAULT '{}',
    estado              TEXT    NOT NULL DEFAULT 'pendiente'
                               CHECK (estado IN ('pendiente','entregado','consumido','expirado','cancelado')),
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    delivered_at        DATETIME,
    consumed_at         DATETIME
);

CREATE INDEX IF NOT EXISTS idx_runtime_mailbox_destino_estado
ON runtime_mailbox(to_agente, estado, id DESC);

CREATE TABLE IF NOT EXISTS runtime_checkpoints (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agente              TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    sesion_id           INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    runtime_id          INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
    checkpoint_kind     TEXT    NOT NULL DEFAULT 'manual',
    resumen             TEXT    NOT NULL DEFAULT '',
    branch              TEXT    NOT NULL DEFAULT '',
    cwd                 TEXT    NOT NULL DEFAULT '',
    payload_json        TEXT    NOT NULL DEFAULT '{}',
    resume_strategy     TEXT    NOT NULL DEFAULT '',
    source              TEXT    NOT NULL DEFAULT '',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_checkpoints_agente_created
ON runtime_checkpoints(agente, created_at DESC, id DESC);

-- ─── Pools de capacidad y políticas de modelo ─────────────────────────────
CREATE TABLE IF NOT EXISTS pools_capacidad (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    slug                  TEXT    NOT NULL UNIQUE,
    proveedor             TEXT    NOT NULL,
    runtime               TEXT    NOT NULL,
    plan                  TEXT    NOT NULL DEFAULT '',
    es_de_pago            INTEGER NOT NULL DEFAULT 0,
    capacidad_total       INTEGER NOT NULL DEFAULT 1,
    capacidad_reservada   INTEGER NOT NULL DEFAULT 0,
    permite_hijos         INTEGER NOT NULL DEFAULT 1,
    permite_modelos_multi INTEGER NOT NULL DEFAULT 1,
    permite_sobrecoste    INTEGER NOT NULL DEFAULT 0,
    politica_handoff      TEXT    NOT NULL DEFAULT 'preventivo',
    fuente_telemetria     TEXT    NOT NULL DEFAULT 'manual',
    metadata_json         TEXT    NOT NULL DEFAULT '{}',
    activo                INTEGER NOT NULL DEFAULT 1,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pool_modelos (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    pool_id             INTEGER REFERENCES pools_capacidad(id) ON DELETE CASCADE,
    model_slug           TEXT    NOT NULL,
    activo               INTEGER NOT NULL DEFAULT 1,
    prioridad            INTEGER NOT NULL DEFAULT 100,
    coste_relativo       REAL    NOT NULL DEFAULT 1.0,
    limite_conocido_json TEXT    NOT NULL DEFAULT '{}',
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pool_id, model_slug)
);

CREATE TABLE IF NOT EXISTS politicas_modelo (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_tipo        TEXT    NOT NULL
                             CHECK (scope_tipo IN ('global','perfil','proyecto','fase','tarea')),
    scope_ref         TEXT    NOT NULL DEFAULT '',
    perfil_tarea      TEXT    NOT NULL DEFAULT '*',
    pool_slug         TEXT    NOT NULL DEFAULT '',
    model_slug        TEXT    NOT NULL DEFAULT '',
    reasoning_effort TEXT NOT NULL DEFAULT '',
    prioridad         INTEGER NOT NULL DEFAULT 100,
    activa            INTEGER NOT NULL DEFAULT 1,
    metadata_json     TEXT    NOT NULL DEFAULT '{}',
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ─── Memoria de proyecto y gobernanza Git ─────────────────────────────────
CREATE TABLE IF NOT EXISTS decisiones_proyecto (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id    INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    categoria      TEXT    NOT NULL DEFAULT 'general',
    titulo         TEXT    NOT NULL,
    solucion       TEXT    NOT NULL,
    motivo         TEXT    NOT NULL DEFAULT '',
    alternativas   TEXT    NOT NULL DEFAULT '',
    impacto        TEXT    NOT NULL DEFAULT '',
    estado         TEXT    NOT NULL DEFAULT 'vigente',
    propuesta_id   INTEGER REFERENCES propuestas(id) ON DELETE SET NULL,
    tarea_id       INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    metadata_json  TEXT    NOT NULL DEFAULT '{}',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(proyecto_id, titulo)
);

CREATE TABLE IF NOT EXISTS documentos_externos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id      INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    tipo_documento   TEXT    NOT NULL DEFAULT 'markdown',
    titulo           TEXT    NOT NULL,
    ruta_ref         TEXT    NOT NULL,
    resumen          TEXT    NOT NULL,
    estado           TEXT    NOT NULL DEFAULT 'vigente',
    fuente           TEXT    NOT NULL DEFAULT 'manual',
    propuesta_id     INTEGER REFERENCES propuestas(id) ON DELETE SET NULL,
    tarea_id         INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    metadata_json    TEXT    NOT NULL DEFAULT '{}',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(proyecto_id, ruta_ref)
);

CREATE TABLE IF NOT EXISTS git_merges (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id     INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    source_branch   TEXT    NOT NULL,
    target_branch   TEXT    NOT NULL,
    requested_by    TEXT    NOT NULL,
    estado          TEXT    NOT NULL DEFAULT 'pendiente'
                             CHECK (estado IN ('pendiente','validando','aprobado','rechazado','ejecutando','fusionado','fallido','cancelado')),
    commit_origen   TEXT    NOT NULL DEFAULT '',
    commit_merge    TEXT    NOT NULL DEFAULT '',
    notas           TEXT    NOT NULL DEFAULT '',
    metadata_json   TEXT    NOT NULL DEFAULT '{}',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ─── Memoria histórica y seguimiento de progreso ──────────────────────────
CREATE TABLE IF NOT EXISTS memoria_proyectos (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto            TEXT    NOT NULL UNIQUE,
    resumen             TEXT    NOT NULL DEFAULT '',
    contexto            TEXT    NOT NULL DEFAULT '',
    preguntas_abiertas  TEXT    NOT NULL DEFAULT '',
    actualizado_por     TEXT    NOT NULL DEFAULT '',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memoria_fuentes (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto         TEXT    NOT NULL,
    tipo             TEXT    NOT NULL DEFAULT '',
    referencia       TEXT    NOT NULL,
    titulo           TEXT    NOT NULL DEFAULT '',
    url              TEXT    NOT NULL DEFAULT '',
    confianza        TEXT    NOT NULL DEFAULT '',
    detalle          TEXT    NOT NULL DEFAULT '',
    registrado_por   TEXT    NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memoria_hallazgos (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto         TEXT    NOT NULL,
    fuente_id        INTEGER REFERENCES memoria_fuentes(id) ON DELETE SET NULL,
    tipo             TEXT    NOT NULL DEFAULT '',
    titulo           TEXT    NOT NULL,
    descripcion      TEXT    NOT NULL DEFAULT '',
    impacto          TEXT    NOT NULL DEFAULT '',
    confianza        TEXT    NOT NULL DEFAULT '',
    registrado_por   TEXT    NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memoria_derivas (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto         TEXT    NOT NULL,
    hallazgo_id      INTEGER REFERENCES memoria_hallazgos(id) ON DELETE SET NULL,
    tipo             TEXT    NOT NULL DEFAULT '',
    severidad        TEXT    NOT NULL DEFAULT '',
    estado           TEXT    NOT NULL DEFAULT 'abierta',
    descripcion      TEXT    NOT NULL,
    evidencia        TEXT    NOT NULL DEFAULT '',
    detectada_por    TEXT    NOT NULL DEFAULT '',
    resolucion       TEXT    NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resuelta_at      DATETIME
);

CREATE TABLE IF NOT EXISTS fases_proyecto (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto         TEXT    NOT NULL,
    nombre           TEXT    NOT NULL,
    descripcion      TEXT    NOT NULL DEFAULT '',
    orden            INTEGER NOT NULL DEFAULT 100,
    peso             REAL    NOT NULL DEFAULT 1,
    estado           TEXT    NOT NULL DEFAULT 'pendiente',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(proyecto, nombre)
);

CREATE TABLE IF NOT EXISTS avance_tareas (
    tarea_id         INTEGER NOT NULL UNIQUE REFERENCES tareas(id) ON DELETE CASCADE,
    proyecto         TEXT    NOT NULL,
    fase_id          INTEGER REFERENCES fases_proyecto(id) ON DELETE SET NULL,
    progreso_pct     REAL    NOT NULL DEFAULT 0,
    actualizado_por  TEXT    NOT NULL DEFAULT '',
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS presupuestos_sesion (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    sesion_id           INTEGER NOT NULL REFERENCES sesiones(id) ON DELETE CASCADE,
    pool_id             INTEGER REFERENCES pools_capacidad(id) ON DELETE SET NULL,
    model_slug          TEXT    NOT NULL DEFAULT '',
    window_kind         TEXT    NOT NULL DEFAULT 'unknown',
    window_started_at   DATETIME,
    reset_at            DATETIME,
    remaining_seconds   INTEGER,
    remaining_messages  INTEGER,
    remaining_tokens    INTEGER,
    remaining_credits   REAL,
    budget_source       TEXT    NOT NULL DEFAULT 'manual',
    raw_snapshot_json   TEXT    NOT NULL DEFAULT '{}',
    checked_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS entidades_memoria (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    nombre              TEXT    NOT NULL,
    tipo                TEXT    NOT NULL
                               CHECK (tipo IN ('negocio','api','db','infra','regla')),
    valor_json          TEXT    NOT NULL DEFAULT '{}',
    metadata_json       TEXT    NOT NULL DEFAULT '{}',
    ultima_verificacion DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verificado_por      TEXT    NOT NULL DEFAULT '',
    proyecto_id         INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    UNIQUE(nombre, proyecto_id)
);

CREATE INDEX IF NOT EXISTS idx_entidades_memoria_proyecto_tipo
ON entidades_memoria(proyecto_id, tipo, nombre);

-- ─── Refinería: cola de validación antes de merge ───────────────────────────
CREATE TABLE IF NOT EXISTS refineria_solicitudes (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    tarea_id     INTEGER NOT NULL REFERENCES tareas(id) ON DELETE CASCADE,
    agente       TEXT    NOT NULL REFERENCES agentes(nombre),
    proyecto_id  INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    rama         TEXT    NOT NULL DEFAULT '',
    dir_trabajo  TEXT    NOT NULL DEFAULT '',
    cmd_test     TEXT    NOT NULL DEFAULT 'go test ./...',
    estado       TEXT    NOT NULL DEFAULT 'pendiente'
                         CHECK (estado IN ('pendiente','ejecutando','aprobada','rechazada','cancelada')),
    resultado    TEXT    NOT NULL DEFAULT '',
    error_texto  TEXT    NOT NULL DEFAULT '',
    commit_merge TEXT    NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at   DATETIME,
    finished_at  DATETIME
);

CREATE INDEX IF NOT EXISTS idx_refineria_estado
ON refineria_solicitudes(estado, created_at, id);

-- ─── Audit log ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS audit_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    agente     TEXT    NOT NULL,
    accion     TEXT    NOT NULL,
    entidad    TEXT    NOT NULL,
    entidad_id INTEGER,
    detalle    TEXT    NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ─── Triggers updated_at ───────────────────────────────────────────────────
CREATE TRIGGER IF NOT EXISTS trig_tareas_updated
    AFTER UPDATE ON tareas
BEGIN
    UPDATE tareas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_propuestas_updated
    AFTER UPDATE ON propuestas
BEGIN
    UPDATE propuestas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_votos_updated
    AFTER UPDATE ON votos
BEGIN
    UPDATE votos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_proyectos_updated
    AFTER UPDATE ON proyectos
BEGIN
    UPDATE proyectos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_asignaciones_updated
    AFTER UPDATE ON asignaciones
BEGIN
    UPDATE asignaciones SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_locks_updated
    AFTER UPDATE ON locks
BEGIN
    UPDATE locks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_worktrees_updated
    AFTER UPDATE ON worktrees
BEGIN
    UPDATE worktrees SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_runtime_instances_updated
    AFTER UPDATE ON runtime_instances
BEGIN
    UPDATE runtime_instances SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_runtime_handles_updated
    AFTER UPDATE ON runtime_handles
BEGIN
    UPDATE runtime_handles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_runtime_orders_updated
    AFTER UPDATE ON runtime_orders
BEGIN
    UPDATE runtime_orders SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_conectores_updated
    AFTER UPDATE ON conectores
BEGIN
    UPDATE conectores SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_git_merges_updated
    AFTER UPDATE ON git_merges
BEGIN
    UPDATE git_merges SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_pools_capacidad_updated
    AFTER UPDATE ON pools_capacidad
BEGIN
    UPDATE pools_capacidad SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_pool_modelos_updated
    AFTER UPDATE ON pool_modelos
BEGIN
    UPDATE pool_modelos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_politicas_modelo_updated
    AFTER UPDATE ON politicas_modelo
BEGIN
    UPDATE politicas_modelo SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_decisiones_proyecto_updated
    AFTER UPDATE ON decisiones_proyecto
BEGIN
    UPDATE decisiones_proyecto SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_documentos_externos_updated
    AFTER UPDATE ON documentos_externos
BEGIN
    UPDATE documentos_externos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_memoria_proyectos_updated
    AFTER UPDATE ON memoria_proyectos
BEGIN
    UPDATE memoria_proyectos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_memoria_derivas_updated
    AFTER UPDATE ON memoria_derivas
BEGIN
    UPDATE memoria_derivas SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_fases_proyecto_updated
    AFTER UPDATE ON fases_proyecto
BEGIN
    UPDATE fases_proyecto SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_avance_tareas_updated
    AFTER UPDATE ON avance_tareas
BEGIN
    UPDATE avance_tareas SET updated_at = CURRENT_TIMESTAMP WHERE tarea_id = NEW.tarea_id;
END;

-- ─── Reglas por tipo de agente ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS reglas (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    tipo_agente  TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    categoria    TEXT NOT NULL,
    titulo       TEXT NOT NULL,
    descripcion  TEXT NOT NULL DEFAULT '',
    activa       INTEGER NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tipo_agente, titulo)
);

-- ─── Skills por tipo de agente ─────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS skills (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    tipo_agente  TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    nombre       TEXT NOT NULL,
    descripcion  TEXT NOT NULL DEFAULT '',
    cuando_usar  TEXT NOT NULL DEFAULT '',
    escenario    TEXT NOT NULL DEFAULT '',
    prioridad    INTEGER NOT NULL DEFAULT 100,
    aliases_json TEXT NOT NULL DEFAULT '[]',
    herramientas_json TEXT NOT NULL DEFAULT '[]',
    origen       TEXT NOT NULL DEFAULT 'builtin',
    nivel_riesgo TEXT NOT NULL DEFAULT 'bajo',
    requiere_aprobacion INTEGER NOT NULL DEFAULT 0,
    activa       INTEGER NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tipo_agente, nombre)
);

-- ─── Workflows por tipo de agente ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS workflows (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    tipo_agente  TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    nombre       TEXT NOT NULL UNIQUE,
    descripcion  TEXT NOT NULL DEFAULT '',
    pasos        TEXT NOT NULL DEFAULT '[]',  -- JSON array de strings
    activo       INTEGER NOT NULL DEFAULT 1,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS governance_overrides (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tipo_agente TEXT    NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    scope_tipo  TEXT    NOT NULL CHECK (scope_tipo IN ('proyecto','agente')),
    scope_ref   TEXT    NOT NULL,
    entidad     TEXT    NOT NULL CHECK (entidad IN ('regla','skill','workflow')),
    entidad_id  INTEGER NOT NULL,
    accion      TEXT    NOT NULL CHECK (accion IN ('enable','disable')),
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scope_tipo, scope_ref, entidad, entidad_id)
);

-- ─── Versionado de catálogo ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS reglas_versiones (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    regla_id     INTEGER NOT NULL REFERENCES reglas(id) ON DELETE CASCADE,
    version_num  INTEGER NOT NULL,
    tipo_agente  TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    categoria    TEXT NOT NULL,
    titulo       TEXT NOT NULL,
    descripcion  TEXT NOT NULL DEFAULT '',
    activa       INTEGER NOT NULL DEFAULT 1,
    actor        TEXT NOT NULL DEFAULT '',
    accion       TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(regla_id, version_num)
);

CREATE TABLE IF NOT EXISTS skills_versiones (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_id     INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version_num  INTEGER NOT NULL,
    tipo_agente  TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    nombre       TEXT NOT NULL,
    descripcion  TEXT NOT NULL DEFAULT '',
    cuando_usar  TEXT NOT NULL DEFAULT '',
    escenario    TEXT NOT NULL DEFAULT '',
    prioridad    INTEGER NOT NULL DEFAULT 100,
    aliases_json TEXT NOT NULL DEFAULT '[]',
    herramientas_json TEXT NOT NULL DEFAULT '[]',
    origen       TEXT NOT NULL DEFAULT 'builtin',
    nivel_riesgo TEXT NOT NULL DEFAULT 'bajo',
    requiere_aprobacion INTEGER NOT NULL DEFAULT 0,
    activa       INTEGER NOT NULL DEFAULT 1,
    actor        TEXT NOT NULL DEFAULT '',
    accion       TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, version_num)
);

CREATE TABLE IF NOT EXISTS workflows_versiones (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id  INTEGER NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version_num  INTEGER NOT NULL,
    tipo_agente  TEXT NOT NULL CHECK (tipo_agente IN ('programador','documentador','admin')),
    nombre       TEXT NOT NULL,
    descripcion  TEXT NOT NULL DEFAULT '',
    pasos        TEXT NOT NULL DEFAULT '[]',
    activo       INTEGER NOT NULL DEFAULT 1,
    actor        TEXT NOT NULL DEFAULT '',
    accion       TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version_num)
);

-- ─── Permisos de edición del catálogo ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS catalogo_edicion_permisos (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    entidad          TEXT NOT NULL CHECK (entidad IN ('reglas','skills','workflows')),
    rol              TEXT NOT NULL CHECK (rol IN ('programador','documentador','admin')),
    alcance          TEXT NOT NULL CHECK (alcance IN ('mismo_rol','todos')),
    puede_crear      INTEGER NOT NULL DEFAULT 0,
    puede_editar     INTEGER NOT NULL DEFAULT 0,
    puede_activar    INTEGER NOT NULL DEFAULT 0,
    puede_versionar  INTEGER NOT NULL DEFAULT 0,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entidad, rol)
);

-- ─── Datos iniciales ───────────────────────────────────────────────────────
INSERT OR IGNORE INTO agentes (nombre, rol) VALUES
    ('alberto',    'admin'),
    ('claude',     'programador'),
    ('Codex1',     'programador'),
    ('Codex2',     'programador'),
    ('antigravity','documentador');

INSERT INTO config (clave, valor) VALUES
    ('distribuidor', 'claude'),
    ('version',      '1.0.0'),
    ('workspace_root',''),
    ('agent_loop_mode','sticky'),
    ('agent_tick_seconds','30'),
    ('lock_lease_seconds','90'),
    ('worktree_root_name','.orquesta-worktrees'),
    ('refactor_backup_required','true'),
    ('refactor_min_reviewers','2'),
    ('refactor_reviewer_must_be_non_author','true'),
    ('refactor_preserve_security','true'),
    ('refactor_preserve_project_philosophy','true'),
    ('agent_autoexecute_safe','true'),
    ('agent_confirm_dangerous_with_project_peer','true'),
    ('propuesta_min_votes','2'),
    ('propuesta_min_non_author_votes','2'),
    ('pool_handoff_threshold_seconds','1800'),
    ('pool_handoff_threshold_ratio','0.10'),
    ('pool_budget_snapshot_max_age_seconds','300'),
    ('pool_default_budget_source','manual'),
    ('connector_circuit_breaker_threshold','3'),
    ('connector_circuit_breaker_cooldown_seconds','300'),
	    ('model_policy_default_profile','implementacion'),
	    ('model_policy_default_reasoning','high'),
	    ('runtime_handle_stale_seconds','120'),
	    ('runtime_supervision_interval_seconds','30'),
	    ('runtime_supervision_batch_size','10'),
	    ('runtime_order_batch_size','10'),
	    ('runtime_order_lease_seconds','120'),
	    ('runtime_send_instruction_lease_seconds','35'),
	    ('runtime_order_stale_seconds','120')
	ON CONFLICT(clave) DO NOTHING;

INSERT INTO conectores (slug, nombre, transporte, comando, env_json, metadata_json) VALUES
    ('claude-code', 'Claude Code', 'cli', 'claude', '{}', '{"familia":"anthropic","reanudable":true}'),
    ('codex-cli',   'Codex CLI',   'cli', 'codex',  '{"ORQUESTA_BIN":"{{orquesta_executable}}","PATH":"{{orquesta_bin_dir}}:{{host_path}}"}', '{"familia":"openai","reanudable":true,"cwd_flag":"-C","launch_prompt_positional":true,"model_flag":"--model","reasoning_config_key":"model_reasoning_effort","can_send_input":false}'),
    ('gemini-cli',  'Gemini CLI',  'cli', 'gemini', '{}', '{"familia":"google","reanudable":true}')
ON CONFLICT(slug) DO NOTHING;

-- ─── Reglas: programador ────────────────────────────────────────────────────
INSERT INTO reglas (tipo_agente, categoria, titulo, descripcion) VALUES
-- Financiero
('programador','financiero','decimal.Decimal obligatorio',
 'Usar decimal.Decimal para cualquier cifra monetaria. Cero float32/float64 para importes.'),
('programador','financiero','Redondeo HALF_UP',
 'Redondear siempre a 2 decimales con HALF_UP.'),
-- Seguridad
('programador','seguridad','JWT RS256',
 'Autenticación con JWT RS256. Argon2id para contraseñas (memory=64MB, iterations=3, parallelism=2).'),
('programador','seguridad','Bloqueo tras 5 intentos',
 'Bloquear cuenta tras 5 intentos de autenticación fallidos.'),
('programador','seguridad','RBAC completo',
 'Control de acceso en cada endpoint: user → AD groups → roles internos → permisos → acción.'),
('programador','seguridad','Filtrado por tenant_id',
 'Filtrar siempre por tenant_id (BD por tenant). No usar sede_id como frontera de seguridad.'),
('programador','seguridad','Auditoría obligatoria',
 'Toda mutación requiere auditoría: SHA-256 encadenado, retención 5 años, AES-256-GCM at rest.'),
('programador','seguridad','Sin secretos hardcodeados',
 'Ningún secreto en código fuente. TLS obligatorio en todos los endpoints.'),
-- Arquitectura
('programador','arquitectura','Clean Architecture + DDD',
 'Capas: api → service → repository → domain. Interfaces primero.'),
('programador','arquitectura','main.go intocable',
 'main.go es intocable por agentes de módulo. Solo el agente de infraestructura lo integra.'),
('programador','arquitectura','Multi-tenant aislado',
 '1 ayuntamiento = 1 base de datos PostgreSQL aislada. Patrón H-72: tenantDB(ctx).'),
('programador','arquitectura','Propiedad exclusiva de ficheros',
 'Cada módulo tiene propiedad exclusiva de sus ficheros. No editar ficheros de otro módulo.'),
('programador','arquitectura','Módulos opcionales',
 'Todos los módulos deben ser opcionales vía variable MODULES. Disabled = sin instanciación.'),
-- Calidad
('programador','calidad','Commits frecuentes',
 'Commit tras cada bloque probado: entidades / repositorio / servicio / handler / migración. Formato: feat(MXX): descripción.'),
('programador','calidad','Gate antes de commit',
 'Antes de cada commit: go test ./internal/... debe pasar.'),
('programador','calidad','Gate de cierre de módulo',
 'Al cerrar módulo: go build ./..., go vet ./..., go test ./... deben pasar.'),
('programador','calidad','Tests DB orquestados',
 'Los tests que toquen BD deben usar los helpers comunes del proyecto y la plantilla compartida. No abrir Open() directo ni bootstraps manuales salvo cuando el propio test valide bootstrap, migraciones o compatibilidad legacy.'),
('programador','calidad','Tests coherentes, rápidos y mantenibles',
 'Los tests deben reutilizar fixtures y helpers comunes, evitar sleeps y aperturas redundantes de BD, conservar coherencia con el resto de la suite y paralelizarse solo cuando el estado compartido esté controlado.'),
('programador','calidad','Propuesta antes de código',
 'No escribir código sin propuesta OP-XXX aprobada en la app de orquestación.'),
('programador','calidad','Idioma castellano',
 'Todo en castellano. Inglés solo cuando lo exija framework, librería o protocolo.'),
('programador','arquitectura','Multilenguaje por defecto donde aplique',
 'Salvo excepciones técnicas claras como drivers o componentes sin interfaz de usuario real, las aplicaciones deben nacer preparadas para al menos dos idiomas. El idioma por defecto será castellano y el idioma debe poder configurarse por usuario o despliegue.'),
('programador','calidad','Cabecera GPLv3',
 'Incluir cabecera de licencia GPLv3 en todos los ficheros nuevos.'),
-- Sesión
('programador','sesion','Fuente de verdad: BD de orquesta',
 'Reglas, skills y workflows viven en la BD de orquesta — no en ficheros. Ejecutar siempre "orquesta sesion inicio <nombre>" al comenzar: muestra el briefing completo desde la BD.'),
('programador','sesion','Protocolo de inicio',
 'Paso 1: orquesta sesion inicio <nombre>  |  Paso 2: ver tareas asignadas (orquesta tarea listar --agente <nombre>)  |  Paso 3: votar propuestas pendientes  |  Paso 4: registrar tarea antes de tocar código.'),
('programador','sesion','Autonomía operativa segura',
 'El agente puede ejecutar acciones normales sin pedir permiso previo. Si la acción es destructiva, de borrado, irreversible o de riesgo alto, debe consultar primero a Orquesta o a otro agente de su mismo proyecto antes de ejecutarla.'),
('programador','sesion','Protocolo de fin',
 'Paso 1: git status limpio (todo commiteado)  |  Paso 2: orquesta sesion fin <nombre>.'),
('programador','sesion','Panel web de orquestación',
 'Alberto arranca el panel web con "orquesta serve" (http://localhost:16543). Muestra en tiempo real: agentes activos, progreso de tareas, propuestas abiertas con votos. Se auto-refresca cada 30 s. Secciones: Dashboard, Tareas (con filtros por estado), Propuestas (expandibles con votos). Los agentes NO necesitan arrancarlo; es para Alberto y para generar capturas de estado.'),
('programador','comandos','Gestión de tareas',
 'Iniciar: orquesta tarea iniciar <id> <agente>  |  Completar: orquesta tarea completar <id> <agente> --commit "feat(MXX): ..."  |  Bloquear: orquesta tarea bloquear <id> <agente> --motivo "razón"  |  Desbloquear: orquesta tarea desbloquear <id> <agente> --resolucion "cómo se resolvió"  |  Ver mis tareas: orquesta tarea listar --agente <nombre>'),
('programador','comandos','Propuestas y votación',
 'Nueva propuesta: orquesta propuesta nueva "Título" --descripcion "Descripción" --agente <nombre>  |  Votar: orquesta votar <OP-XXX> <acuerdo|desacuerdo|abstencion> --agente <nombre> --comentario "razón"  |  Ver propuesta: orquesta propuesta ver <OP-XXX>  |  Ver pendientes de voto: incluidas en el briefing de sesion inicio.')
ON CONFLICT(tipo_agente, titulo) DO NOTHING;

-- ─── Reglas: documentador ───────────────────────────────────────────────────
INSERT INTO reglas (tipo_agente, categoria, titulo, descripcion) VALUES
('documentador','general','Fuente de verdad: BD de orquesta',
 'Reglas, skills y workflows viven en la BD — no en ficheros. Ejecutar "orquesta sesion inicio antigravity" al comenzar para recibir el briefing completo.'),
('documentador','general','Solo documentación',
 'Antigravity no toca código fuente (.go, .sql, .ts, .yaml). Solo docs/.'),
('documentador','general','Ficheros asignados',
 'Propiedad exclusiva: docs/modulos/MXX_*.md e índice maestro docs/00_INDICE.md.'),
('documentador','general','Activación por notificación',
 'Documentar un módulo solo tras notificación explícita del agente programador que lo cierra.'),
('documentador','general','Referencia obligatoria de documentación externa',
 'Si Antigravity crea o mantiene documentación fuera de orquestador, debe quedar siempre documentada también en Orquesta. La BD debe guardar una referencia clara con la ruta de esos ficheros y un resumen de su contenido o propósito.'),
('documentador','general','Multilenguaje por defecto donde aplique',
 'La documentación debe mantenerse al menos en castellano e inglés cuando el proyecto tenga sentido multilenguaje. Los idiomas deben ir en ficheros separados y el castellano actúa como idioma por defecto salvo indicación distinta.'),
('documentador','general','Idioma castellano',
 'Toda la documentación en castellano. Términos técnicos en inglés solo si no tienen traducción.'),
('documentador','sesion','Protocolo de inicio',
 'Paso 1: orquesta sesion inicio antigravity  |  Paso 2: identificar módulos cerrados sin documentar  |  Paso 3: ver tareas asignadas.'),
('documentador','sesion','Protocolo de fin',
 'Paso 1: actualizar docs/00_INDICE.md si hay módulos nuevos  |  Paso 2: orquesta sesion fin antigravity.')
ON CONFLICT(tipo_agente, titulo) DO NOTHING;

-- ─── Skills: programador ────────────────────────────────────────────────────
INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar) VALUES
('programador','create-module',
 'Crear un módulo nuevo completo (14 pasos): dominio, repositorio, servicio, handler, migración, tests, integración.',
 'Cuando se asigna un módulo nuevo de la lista de Ola 2 o posterior.'),
('programador','develop-feature',
 'Desarrollar una feature dentro de un módulo existente sin romper otros módulos.',
 'Cuando hay una tarea de nueva funcionalidad dentro de un módulo ya existente.'),
('programador','fix-bug',
 'Corregir un bug con mínimo impacto, sin refactorizar código no relacionado.',
 'Cuando hay un bug confirmado con reproducción conocida.'),
('programador','security-review',
 'Auditar seguridad de un módulo: RBAC, cifrado, auditoría, inyección SQL, OWASP Top 10.',
 'Al cerrar un módulo o cuando se detecta deuda técnica de seguridad (ej: OP-022).'),
('programador','db-test-harness',
 'Revisar y normalizar tests que usan BD: helpers compartidos, fixtures coherentes, aperturas mínimas y tiempos razonables.',
 'Cuando se crean o refactorizan tests con persistencia, migraciones o control plane.'),
('programador','autofirma-integration',
 'Integrar firma digital AutoFirma según estándar @firma de la AEAT.',
 'Cuando se implementa firma electrónica en flujos administrativos. Ver OP-027 (📋 BACKLOG).'),
('programador','administracion-publica-segura',
 'Referencia normativa ENS/LOPDGDD para diseño de módulos de administración pública.',
 'Al diseñar módulos con datos personales o procesos sujetos a ENS.')
ON CONFLICT(tipo_agente, nombre) DO NOTHING;

-- ─── Skills: documentador ───────────────────────────────────────────────────
INSERT INTO skills (tipo_agente, nombre, descripcion, cuando_usar) VALUES
('documentador','document-module',
 'Generar documentación completa de un módulo cerrado: descripción, endpoints, entidades, flujos.',
 'Al recibir notificación de cierre de módulo de un agente programador.'),
('documentador','update-index',
 'Actualizar el índice maestro docs/00_INDICE.md con nuevos módulos documentados.',
 'Después de documentar cualquier módulo.'),
('documentador','review-docs',
 'Revisar documentación existente por coherencia, completitud y actualidad.',
 'Cuando se detectan discrepancias entre el código y la documentación.')
ON CONFLICT(tipo_agente, nombre) DO NOTHING;

-- ─── Workflows: programador (claude/codex) ──────────────────────────────────
INSERT INTO workflows (tipo_agente, nombre, descripcion, pasos) VALUES
('programador','inicio-sesion',
 'Protocolo obligatorio al comenzar cualquier sesión de trabajo.',
 '["1. Ejecutar: orquesta sesion inicio <mi-nombre>",
   "2. Ver tareas asignadas: orquesta tarea listar --agente <mi-nombre>",
   "3. Votar todas las propuestas con posicion pendiente para mi agente",
   "4. Iniciar la tarea en la app: orquesta tarea iniciar <id> <mi-nombre>",
   "5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md",
   "6. Operar con autonomía para acciones normales; si una acción es destructiva o peligrosa, consultar antes con Orquesta o con otro agente del mismo proyecto"]'),
('programador','fin-sesion',
 'Protocolo obligatorio al terminar cualquier sesión de trabajo.',
 '["1. Asegurar que todo el trabajo está commiteado (git status limpio)",
   "2. Completar o bloquear mis tareas en la app de orquestación según corresponda",
   "3. Ejecutar: orquesta sesion fin <mi-nombre>"]'),
('programador','crear-modulo',
 'Flujo completo para implementar un módulo nuevo (14 pasos).',
 '["1. Crear propuesta OP-XXX con orquesta propuesta nueva y esperar consenso en la app",
   "2. Crear fichero de dominio: internal/domain/<modulo>_entities.go",
   "3. Crear interfaces: internal/domain/<modulo>_interfaces.go",
   "4. Crear migración SQL: migrations/XXXXXX_<modulo>.up.sql",
   "5. Crear repositorio: internal/repository/postgres_<modulo>.go",
   "6. Commit: feat(MXX): dominio e interfaces",
   "7. Crear servicio: internal/service/<modulo>_service.go",
   "8. Commit: feat(MXX): servicio",
   "9. Crear tests: internal/service/<modulo>_service_test.go",
   "10. Ejecutar go test ./internal/service/... → debe pasar",
   "11. Commit: feat(MXX): tests servicio",
   "12. Crear handler REST: internal/api/<modulo>_handler.go",
   "13. Commit: feat(MXX): handler REST",
   "14. Gate final: go build ./... && go vet ./... && go test ./... → notificar a Antigravity"]'),
('programador','votar-propuesta',
 'Proceso para votar una propuesta OP-XXX.',
 '["1. Leer la propuesta completa: orquesta propuesta ver <codigo>",
   "2. Analizar impacto técnico en módulos asignados",
   "3. Votar: orquesta votar <codigo> <acuerdo|desacuerdo|abstencion> --agente <mi-nombre> --comentario \"razón\"",
   "4. Si desacuerdo: añadir comentario técnico con alternativa concreta"]')
ON CONFLICT(nombre) DO NOTHING;

-- ─── Workflows: documentador (antigravity) ──────────────────────────────────
INSERT INTO workflows (tipo_agente, nombre, descripcion, pasos) VALUES
('documentador','inicio-sesion',
 'Protocolo obligatorio al comenzar cualquier sesión de trabajo.',
 '["1. Ejecutar: orquesta sesion inicio antigravity",
   "2. Ver tareas asignadas: orquesta tarea listar --agente antigravity",
   "3. Votar propuestas con posicion pendiente para antigravity",
   "4. Revisar docs/00_INDICE.md para detectar gaps"]'),
('documentador','fin-sesion',
 'Protocolo obligatorio al terminar cualquier sesión de trabajo.',
 '["1. Actualizar docs/00_INDICE.md si se añadieron módulos",
   "2. Confirmar con el agente programador que la documentación es correcta",
   "3. Ejecutar: orquesta sesion fin antigravity"]'),
('documentador','documentar-modulo',
 'Flujo para documentar un módulo cerrado.',
 '["1. Recibir notificación del agente programador con el código del módulo",
   "2. Leer el código fuente del módulo (solo lectura, nunca editar)",
   "3. Crear docs/modulos/MXX_<nombre>.md con: descripción, entidades, endpoints, flujos, seguridad",
   "4. Actualizar docs/00_INDICE.md",
   "5. Notificar al programador que la documentación está lista"]')
ON CONFLICT(nombre) DO NOTHING;
`
