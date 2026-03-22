package db

// schemaBootstrapDDL define configuración mínima global y catálogo de agentes.
// Sale de un schema spec estructurado para evitar sustituciones ciegas por driver.
var schemaBootstrapDDL = renderBootstrapSectionDDLForDriver("sqlite")

// schemaWorkflowDDL define el flujo operativo principal de Orquesta.
const schemaWorkflowDDL = `

-- ─── Tareas ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS tareas (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    titulo        TEXT    NOT NULL,
    descripcion   TEXT    NOT NULL DEFAULT '',
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
    tipo          TEXT    NOT NULL DEFAULT 'implementacion'
                          CHECK (tipo IN ('implementacion','arquitectura','seguridad','backlog','otro')),
    estado        TEXT    NOT NULL DEFAULT 'abierta'
                          CHECK (estado IN ('abierta','consenso','rechazada','backlog')),
    propuesto_por TEXT    NOT NULL,
    distribuidor  TEXT    NOT NULL DEFAULT 'claude',
    proyecto_id   INTEGER REFERENCES proyectos(id),
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
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    agente              TEXT    NOT NULL REFERENCES agentes(nombre),
    inicio              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fin                 DATETIME,
    activa              INTEGER NOT NULL DEFAULT 1,
    conector_id         INTEGER REFERENCES conectores(id),
    proyecto_id         INTEGER REFERENCES proyectos(id),
    pool_id             INTEGER REFERENCES pools_capacidad(id),
    estado              TEXT    NOT NULL DEFAULT 'activa',
    cwd                 TEXT    NOT NULL DEFAULT '',
    herramienta         TEXT    NOT NULL DEFAULT '',
    external_session_id TEXT    NOT NULL DEFAULT '',
    resume_payload_json TEXT    NOT NULL DEFAULT '',
    resumen_continuidad TEXT    NOT NULL DEFAULT '',
    branch              TEXT    NOT NULL DEFAULT '',
    heartbeat_at        DATETIME,
    host                TEXT    NOT NULL DEFAULT '',
    pid                 INTEGER
);

`

// schemaCoordinationDDL define coordinación multi-proyecto, conectores y worktrees.
const schemaCoordinationDDL = `

-- ─── Proyectos, asignaciones y coordinacion segura ─────────────────────────
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
    expires_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    liberada_at       DATETIME
);

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

-- ─── Gobierno Git y merges orquestados ────────────────────────────────────
CREATE TABLE IF NOT EXISTS git_merges (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id   INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    source_branch TEXT    NOT NULL,
    target_branch TEXT    NOT NULL,
    requested_by  TEXT    NOT NULL REFERENCES agentes(nombre),
    estado        TEXT    NOT NULL DEFAULT 'pendiente'
                    CHECK (estado IN ('pendiente','validando','aprobado','rechazado','ejecutando','fusionado','fallido','cancelado')),
    commit_origen TEXT    NOT NULL DEFAULT '',
    commit_merge  TEXT    NOT NULL DEFAULT '',
    notas         TEXT    NOT NULL DEFAULT '',
    metadata_json TEXT    NOT NULL DEFAULT '{}',
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

`

// schemaRuntimeDDL define control activo y órdenes runtime de agentes.
// Sale de un schema spec estructurado para evitar sustituciones ciegas por driver.
var schemaRuntimeDDL = renderRuntimeSectionDDLForDriver("sqlite")

// schemaCapacityDDL define pools, modelos y políticas de capacidad.
// Sale de un schema spec estructurado para evitar sustituciones ciegas por driver.
var schemaCapacityDDL = renderCapacitySectionDDLForDriver("sqlite")

// schemaKnowledgeDDL define memoria de proyecto, auditoría y gobernanza.
const schemaKnowledgeDDL = `

-- ─── Memoria y trazabilidad por proyecto ──────────────────────────────────
CREATE TABLE IF NOT EXISTS decisiones_proyecto (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id  INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    categoria    TEXT NOT NULL DEFAULT 'general',
    titulo       TEXT NOT NULL,
    solucion     TEXT NOT NULL DEFAULT '',
    motivo       TEXT NOT NULL DEFAULT '',
    alternativas TEXT NOT NULL DEFAULT '',
    impacto      TEXT NOT NULL DEFAULT '',
    estado       TEXT NOT NULL DEFAULT 'vigente'
                    CHECK (estado IN ('vigente','experimental','reemplazada','descartada','archivada')),
    propuesta_id INTEGER REFERENCES propuestas(id) ON DELETE SET NULL,
    tarea_id     INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(proyecto_id, titulo)
);

CREATE TABLE IF NOT EXISTS documentos_externos (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto_id    INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
    tipo_documento TEXT NOT NULL DEFAULT 'markdown',
    titulo         TEXT NOT NULL,
    ruta_ref       TEXT NOT NULL,
    resumen        TEXT NOT NULL DEFAULT '',
    estado         TEXT NOT NULL DEFAULT 'vigente'
                      CHECK (estado IN ('vigente','borrador','archivado')),
    fuente         TEXT NOT NULL DEFAULT 'manual'
                      CHECK (fuente IN ('manual','propuesta','tarea','externo')),
    propuesta_id   INTEGER REFERENCES propuestas(id) ON DELETE SET NULL,
    tarea_id       INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    metadata_json  TEXT NOT NULL DEFAULT '{}',
    created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(proyecto_id, ruta_ref)
);

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

`

// schemaBaseDDL define el DDL base comun de la base de datos de orquestación.
// El DDL auxiliar por dialecto se ensambla aparte en el renderer.
var schemaBaseDDL = joinDDLParts(
	schemaBootstrapDDL,
	schemaWorkflowDDL,
	schemaCoordinationDDL,
	schemaRuntimeDDL,
	schemaCapacityDDL,
	schemaKnowledgeDDL,
)

// Schema se mantiene como alias temporal del DDL base para compatibilidad.
var Schema = schemaBaseDDL
