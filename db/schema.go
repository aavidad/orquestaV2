package db

// Schema define el esquema completo de la base de datos de orquestación.
// Las migraciones se aplican en orden; nunca se modifican las existentes.
const Schema = `
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

-- ─── Configuración global ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS config (
    clave TEXT PRIMARY KEY,
    valor TEXT NOT NULL
);

-- ─── Agentes registrados ───────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS agentes (
    nombre       TEXT PRIMARY KEY,
    rol          TEXT NOT NULL CHECK (rol IN ('programador','documentador','admin')),
    activo       INTEGER NOT NULL DEFAULT 0,   -- 1 = en sesión ahora mismo
    habilitado   INTEGER NOT NULL DEFAULT 1,   -- 0 = retirado por Alberto (no vota, no trabaja)
    ultima_sesion DATETIME
);

-- ─── Proyectos registrados ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS proyectos (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    slug       TEXT    NOT NULL UNIQUE,
    nombre     TEXT    NOT NULL,
    ruta_abs   TEXT    NOT NULL DEFAULT '',
    tipo       TEXT    NOT NULL DEFAULT 'repo',
    parent_id  INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    activo     INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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
    proyecto_id   INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
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
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    agente  TEXT    NOT NULL REFERENCES agentes(nombre),
    inicio  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fin     DATETIME,
    activa  INTEGER NOT NULL DEFAULT 1
);

-- ─── Pools de capacidad y modelos ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS pools_capacidad (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    slug                  TEXT NOT NULL UNIQUE,
    proveedor             TEXT NOT NULL,
    runtime               TEXT NOT NULL,
    plan                  TEXT NOT NULL DEFAULT '',
    es_de_pago            INTEGER NOT NULL DEFAULT 0,
    capacidad_total       INTEGER NOT NULL DEFAULT 1,
    capacidad_reservada   INTEGER NOT NULL DEFAULT 0,
    permite_hijos         INTEGER NOT NULL DEFAULT 1,
    permite_modelos_multi INTEGER NOT NULL DEFAULT 1,
    permite_sobrecoste    INTEGER NOT NULL DEFAULT 0,
    politica_handoff      TEXT NOT NULL DEFAULT 'preventivo',
    fuente_telemetria     TEXT NOT NULL DEFAULT 'manual',
    metadata_json         TEXT NOT NULL DEFAULT '{}',
    activo                INTEGER NOT NULL DEFAULT 1,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pool_modelos (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    pool_id              INTEGER NOT NULL REFERENCES pools_capacidad(id) ON DELETE CASCADE,
    model_slug           TEXT NOT NULL,
    activo               INTEGER NOT NULL DEFAULT 1,
    prioridad            INTEGER NOT NULL DEFAULT 100,
    coste_relativo       REAL NOT NULL DEFAULT 1.0,
    limite_conocido_json TEXT NOT NULL DEFAULT '{}',
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pool_id, model_slug)
);

-- ─── Presupuestos de sesión ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS presupuestos_sesion (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    sesion_id              INTEGER NOT NULL REFERENCES sesiones(id) ON DELETE CASCADE,
    pool_id                INTEGER,
    model_slug             TEXT NOT NULL DEFAULT '',
    window_kind            TEXT NOT NULL DEFAULT 'unknown',
    window_started_at      DATETIME,
    reset_at               DATETIME,
    remaining_seconds      INTEGER,
    remaining_messages     INTEGER,
    remaining_tokens       INTEGER,
    remaining_credits      REAL,
    budget_source          TEXT NOT NULL DEFAULT 'manual',
    raw_snapshot_json      TEXT NOT NULL DEFAULT '{}',
    checked_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ─── Control activo de agentes vivos ───────────────────────────────────────
CREATE TABLE IF NOT EXISTS runtime_handles (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    agente           TEXT NOT NULL REFERENCES agentes(nombre),
    sesion_id        INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    transporte       TEXT NOT NULL,
    handle_kind      TEXT NOT NULL,
    handle_ref       TEXT NOT NULL,
    estado           TEXT NOT NULL DEFAULT 'activo'
                      CHECK (estado IN ('activo','pausado','cerrado','reemplazado','error')),
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS runtime_orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    agente           TEXT NOT NULL REFERENCES agentes(nombre),
    sesion_id        INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    tipo             TEXT NOT NULL
                      CHECK (tipo IN ('enviar_instruccion','pausar','continuar','handoff')),
    payload_json     TEXT NOT NULL DEFAULT '{}',
    estado           TEXT NOT NULL DEFAULT 'pendiente'
                      CHECK (estado IN ('pendiente','en_progreso','completada','fallida','cancelada')),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at       DATETIME,
    finished_at      DATETIME
);

-- ─── Observabilidad pasiva de runtimes ────────────────────────────────────
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
    source              TEXT    NOT NULL,
    sample_json         TEXT    NOT NULL DEFAULT '{}',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_telemetry_samples_runtime_created
ON runtime_telemetry_samples(runtime_id, created_at DESC);

CREATE TABLE IF NOT EXISTS runtime_events (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    runtime_id          INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
    kind                TEXT    NOT NULL,
    level               TEXT    NOT NULL DEFAULT 'info',
    message             TEXT    NOT NULL DEFAULT '',
    payload_json        TEXT    NOT NULL DEFAULT '{}',
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runtime_events_runtime_created
ON runtime_events(runtime_id, created_at DESC);

-- ─── Permisos y versionado del catálogo ───────────────────────────────────
CREATE TABLE IF NOT EXISTS catalogo_edicion_permisos (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    entidad          TEXT    NOT NULL
                             CHECK (entidad IN ('reglas','skills','workflows')),
    rol              TEXT    NOT NULL
                             CHECK (rol IN ('programador','documentador','admin')),
    alcance          TEXT    NOT NULL DEFAULT 'mismo_rol'
                             CHECK (alcance IN ('mismo_rol','todos')),
    puede_crear      INTEGER NOT NULL DEFAULT 0,
    puede_editar     INTEGER NOT NULL DEFAULT 0,
    puede_activar    INTEGER NOT NULL DEFAULT 0,
    puede_versionar  INTEGER NOT NULL DEFAULT 0,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entidad, rol)
);

CREATE TABLE IF NOT EXISTS reglas_versiones (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    regla_id      INTEGER NOT NULL REFERENCES reglas(id) ON DELETE CASCADE,
    version_num   INTEGER NOT NULL,
    tipo_agente   TEXT    NOT NULL,
    categoria     TEXT    NOT NULL,
    titulo        TEXT    NOT NULL,
    descripcion   TEXT    NOT NULL DEFAULT '',
    activa        INTEGER NOT NULL DEFAULT 1,
    actor         TEXT    NOT NULL,
    accion        TEXT    NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(regla_id, version_num)
);

CREATE TABLE IF NOT EXISTS skills_versiones (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_id      INTEGER NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version_num   INTEGER NOT NULL,
    tipo_agente   TEXT    NOT NULL,
    nombre        TEXT    NOT NULL,
    descripcion   TEXT    NOT NULL DEFAULT '',
    cuando_usar   TEXT    NOT NULL DEFAULT '',
    activa        INTEGER NOT NULL DEFAULT 1,
    actor         TEXT    NOT NULL,
    accion        TEXT    NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(skill_id, version_num)
);

CREATE TABLE IF NOT EXISTS workflows_versiones (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    workflow_id   INTEGER NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version_num   INTEGER NOT NULL,
    tipo_agente   TEXT    NOT NULL,
    nombre        TEXT    NOT NULL,
    descripcion   TEXT    NOT NULL DEFAULT '',
    pasos         TEXT    NOT NULL DEFAULT '[]',
    activo        INTEGER NOT NULL DEFAULT 1,
    actor         TEXT    NOT NULL,
    accion        TEXT    NOT NULL,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version_num)
);

-- ─── Conectores forge y subproyectos remotos ──────────────────────────────
CREATE TABLE IF NOT EXISTS conectores_forge (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    slug          TEXT    NOT NULL UNIQUE,
    tipo          TEXT    NOT NULL
                         CHECK (tipo IN ('github','gitlab','gitea','otro')),
    owner         TEXT    NOT NULL,
    owner_kind    TEXT    NOT NULL DEFAULT 'org'
                         CHECK (owner_kind IN ('org','user')),
    api_base_url  TEXT    NOT NULL DEFAULT '',
    token_env     TEXT    NOT NULL DEFAULT '',
    metadata_json TEXT    NOT NULL DEFAULT '{}',
    activo        INTEGER NOT NULL DEFAULT 1,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS subproyectos_remotos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto        TEXT    NOT NULL,
    conector_slug   TEXT    NOT NULL REFERENCES conectores_forge(slug),
    forge_tipo      TEXT    NOT NULL,
    owner           TEXT    NOT NULL,
    repo_name       TEXT    NOT NULL,
    repo_full_name  TEXT    NOT NULL,
    visibility      TEXT    NOT NULL DEFAULT 'private'
                             CHECK (visibility IN ('public','private')),
    descripcion     TEXT    NOT NULL DEFAULT '',
    html_url        TEXT    NOT NULL DEFAULT '',
    clone_url       TEXT    NOT NULL DEFAULT '',
    ssh_url         TEXT    NOT NULL DEFAULT '',
    default_branch  TEXT    NOT NULL DEFAULT '',
    estado          TEXT    NOT NULL DEFAULT 'registrado'
                             CHECK (estado IN ('registrado','creado','error','dry_run')),
    metadata_json   TEXT    NOT NULL DEFAULT '{}',
    registrado_por  TEXT    NOT NULL DEFAULT 'orquesta',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(conector_slug, repo_full_name)
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

CREATE TRIGGER IF NOT EXISTS trig_conectores_forge_updated
    AFTER UPDATE ON conectores_forge
BEGIN
    UPDATE conectores_forge SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_subproyectos_remotos_updated
    AFTER UPDATE ON subproyectos_remotos
BEGIN
    UPDATE subproyectos_remotos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
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

-- ─── Memoria de proyecto ───────────────────────────────────────────────────
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
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto        TEXT    NOT NULL,
    tipo            TEXT    NOT NULL DEFAULT 'documentacion'
                            CHECK (tipo IN ('documentacion','codigo','norma','decision','incidencia','externa','otra')),
    referencia      TEXT    NOT NULL,
    titulo          TEXT    NOT NULL DEFAULT '',
    url             TEXT    NOT NULL DEFAULT '',
    confianza       TEXT    NOT NULL DEFAULT 'media'
                            CHECK (confianza IN ('alta','media','baja')),
    detalle         TEXT    NOT NULL DEFAULT '',
    registrado_por  TEXT    NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memoria_hallazgos (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto        TEXT    NOT NULL,
    fuente_id       INTEGER REFERENCES memoria_fuentes(id) ON DELETE SET NULL,
    tipo            TEXT    NOT NULL DEFAULT 'hecho'
                            CHECK (tipo IN ('hecho','inferencia','riesgo','decision','pregunta')),
    titulo          TEXT    NOT NULL,
    descripcion     TEXT    NOT NULL DEFAULT '',
    impacto         TEXT    NOT NULL DEFAULT 'medio'
                            CHECK (impacto IN ('alto','medio','bajo')),
    confianza       TEXT    NOT NULL DEFAULT 'media'
                            CHECK (confianza IN ('alta','media','baja')),
    registrado_por  TEXT    NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS memoria_derivas (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto        TEXT    NOT NULL,
    hallazgo_id     INTEGER REFERENCES memoria_hallazgos(id) ON DELETE SET NULL,
    tipo            TEXT    NOT NULL DEFAULT 'documental'
                            CHECK (tipo IN ('documental','arquitectonica','operativa','fuente','contexto','otra')),
    severidad       TEXT    NOT NULL DEFAULT 'media'
                            CHECK (severidad IN ('alta','media','baja')),
    estado          TEXT    NOT NULL DEFAULT 'abierta'
                            CHECK (estado IN ('abierta','en_revision','resuelta','descartada')),
    descripcion     TEXT    NOT NULL,
    evidencia       TEXT    NOT NULL DEFAULT '',
    detectada_por   TEXT    NOT NULL DEFAULT '',
    resolucion      TEXT    NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resuelta_at     DATETIME
);

CREATE TABLE IF NOT EXISTS decisiones_proyecto (
    id                       INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto                 TEXT    NOT NULL,
    titulo                   TEXT    NOT NULL,
    solucion_elegida         TEXT    NOT NULL DEFAULT '',
    motivo                   TEXT    NOT NULL DEFAULT '',
    alternativas_descartadas TEXT    NOT NULL DEFAULT '',
    impacto                  TEXT    NOT NULL DEFAULT 'medio'
                                      CHECK (impacto IN ('alto','medio','bajo')),
    propuesta_codigo         TEXT    NOT NULL DEFAULT '',
    tarea_id                 INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
    registrado_por           TEXT    NOT NULL DEFAULT '',
    created_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS documentacion_externa (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto        TEXT    NOT NULL,
    tipo_documento  TEXT    NOT NULL DEFAULT 'referencia'
                            CHECK (tipo_documento IN ('referencia','manual','guia','norma','inventario','otra')),
    ruta            TEXT    NOT NULL,
    resumen         TEXT    NOT NULL DEFAULT '',
    estado          TEXT    NOT NULL DEFAULT 'vigente'
                            CHECK (estado IN ('vigente','pendiente','archivado','obsoleto')),
    registrado_por  TEXT    NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS fases_proyecto (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    proyecto        TEXT    NOT NULL,
    nombre          TEXT    NOT NULL,
    descripcion     TEXT    NOT NULL DEFAULT '',
    orden           INTEGER NOT NULL DEFAULT 100,
    peso            REAL    NOT NULL DEFAULT 1.0,
    estado          TEXT    NOT NULL DEFAULT 'pendiente'
                            CHECK (estado IN ('pendiente','activa','bloqueada','completada')),
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(proyecto, nombre)
);

CREATE TABLE IF NOT EXISTS avance_tareas (
    tarea_id        INTEGER PRIMARY KEY REFERENCES tareas(id) ON DELETE CASCADE,
    proyecto        TEXT    NOT NULL,
    fase_id         INTEGER REFERENCES fases_proyecto(id) ON DELETE SET NULL,
    progreso_pct    REAL    NOT NULL DEFAULT 0,
    actualizado_por TEXT    NOT NULL DEFAULT '',
    updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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

CREATE TRIGGER IF NOT EXISTS trig_decisiones_proyecto_updated
    AFTER UPDATE ON decisiones_proyecto
BEGIN
    UPDATE decisiones_proyecto SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_documentacion_externa_updated
    AFTER UPDATE ON documentacion_externa
BEGIN
    UPDATE documentacion_externa SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
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

CREATE TRIGGER IF NOT EXISTS trig_runtime_handles_updated
    AFTER UPDATE ON runtime_handles
BEGIN
    UPDATE runtime_handles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trig_runtime_instances_updated
    AFTER UPDATE ON runtime_instances
BEGIN
    UPDATE runtime_instances SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
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

-- ─── Datos iniciales ───────────────────────────────────────────────────────
INSERT OR IGNORE INTO agentes (nombre, rol) VALUES
    ('alberto',    'admin'),
    ('claude',     'programador'),
    ('codex1',     'programador'),
    ('codex2',     'programador'),
    ('antigravity','documentador');

INSERT OR IGNORE INTO config (clave, valor) VALUES
    ('distribuidor', 'claude'),
    ('version',      '1.0.0'),
    ('pool_handoff_threshold_seconds', '1800'),
    ('pool_handoff_threshold_ratio', '0.10'),
    ('pool_default_budget_source', 'manual'),
    ('model_policy_default_profile', 'implementacion'),
    ('model_policy_default_reasoning', 'high');

-- ─── Reglas: programador ────────────────────────────────────────────────────
INSERT OR IGNORE INTO reglas (tipo_agente, categoria, titulo, descripcion) VALUES
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
('programador','calidad','Propuesta antes de código',
 'No escribir código sin propuesta OP-XXX aprobada en la app de orquestación.'),
('programador','calidad','Idioma castellano',
 'Todo en castellano. Inglés solo cuando lo exija framework, librería o protocolo.'),
('programador','calidad','Cabecera GPLv3',
 'Incluir cabecera de licencia GPLv3 en todos los ficheros nuevos.'),
-- Sesión
('programador','sesion','Fuente de verdad: BD de orquesta',
 'Reglas, skills y workflows viven en la BD de orquesta — no en ficheros. Ejecutar siempre "orquesta sesion inicio <nombre>" al comenzar: muestra el briefing completo desde la BD.'),
('programador','sesion','Protocolo de inicio',
 'Paso 1: orquesta sesion inicio <nombre>  |  Paso 2: ver tareas asignadas (orquesta tarea listar --agente <nombre>)  |  Paso 3: votar propuestas pendientes  |  Paso 4: registrar tarea antes de tocar código.'),
('programador','sesion','Protocolo de fin',
 'Paso 1: git status limpio (todo commiteado)  |  Paso 2: orquesta sesion fin <nombre>.'),
('programador','sesion','Panel web de orquestación',
 'Alberto arranca el panel web con "orquesta serve" (http://localhost:8080). Muestra en tiempo real: agentes activos, progreso de tareas, propuestas abiertas con votos. Se auto-refresca cada 30 s. Secciones: Dashboard, Tareas (con filtros por estado), Propuestas (expandibles con votos). Los agentes NO necesitan arrancarlo; es para Alberto y para generar capturas de estado.'),
('programador','comandos','Gestión de tareas',
 'Iniciar: orquesta tarea iniciar <id> <agente>  |  Completar: orquesta tarea completar <id> <agente> --commit "feat(MXX): ..."  |  Bloquear: orquesta tarea bloquear <id> <agente> --motivo "razón"  |  Desbloquear: orquesta tarea desbloquear <id> <agente> --resolucion "cómo se resolvió"  |  Ver mis tareas: orquesta tarea listar --agente <nombre>'),
('programador','comandos','Propuestas y votación',
 'Nueva propuesta: orquesta propuesta nueva "Título" --descripcion "Descripción" --agente <nombre>  |  Votar: orquesta votar <OP-XXX> <acuerdo|desacuerdo|abstencion> --agente <nombre> --comentario "razón"  |  Ver propuesta: orquesta propuesta ver <OP-XXX>  |  Ver pendientes de voto: incluidas en el briefing de sesion inicio.');

-- ─── Reglas: documentador ───────────────────────────────────────────────────
INSERT OR IGNORE INTO reglas (tipo_agente, categoria, titulo, descripcion) VALUES
('documentador','general','Fuente de verdad: BD de orquesta',
 'Reglas, skills y workflows viven en la BD — no en ficheros. Ejecutar "orquesta sesion inicio antigravity" al comenzar para recibir el briefing completo.'),
('documentador','general','Solo documentación',
 'Antigravity no toca código fuente (.go, .sql, .ts, .yaml). Solo docs/.'),
('documentador','general','Ficheros asignados',
 'Propiedad exclusiva: docs/modulos/MXX_*.md e índice maestro docs/00_INDICE.md.'),
('documentador','general','Activación por notificación',
 'Documentar un módulo solo tras notificación explícita del agente programador que lo cierra.'),
('documentador','general','Idioma castellano',
 'Toda la documentación en castellano. Términos técnicos en inglés solo si no tienen traducción.'),
('documentador','sesion','Protocolo de inicio',
 'Paso 1: orquesta sesion inicio antigravity  |  Paso 2: identificar módulos cerrados sin documentar  |  Paso 3: ver tareas asignadas.'),
('documentador','sesion','Protocolo de fin',
 'Paso 1: actualizar docs/00_INDICE.md si hay módulos nuevos  |  Paso 2: orquesta sesion fin antigravity.');

-- ─── Skills: programador ────────────────────────────────────────────────────
INSERT OR IGNORE INTO skills (tipo_agente, nombre, descripcion, cuando_usar) VALUES
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
('programador','autofirma-integration',
 'Integrar firma digital AutoFirma según estándar @firma de la AEAT.',
 'Cuando se implementa firma electrónica en flujos administrativos. Ver OP-027 (📋 BACKLOG).'),
('programador','administracion-publica-segura',
 'Referencia normativa ENS/LOPDGDD para diseño de módulos de administración pública.',
 'Al diseñar módulos con datos personales o procesos sujetos a ENS.');

-- ─── Skills: documentador ───────────────────────────────────────────────────
INSERT OR IGNORE INTO skills (tipo_agente, nombre, descripcion, cuando_usar) VALUES
('documentador','document-module',
 'Generar documentación completa de un módulo cerrado: descripción, endpoints, entidades, flujos.',
 'Al recibir notificación de cierre de módulo de un agente programador.'),
('documentador','update-index',
 'Actualizar el índice maestro docs/00_INDICE.md con nuevos módulos documentados.',
 'Después de documentar cualquier módulo.'),
('documentador','review-docs',
 'Revisar documentación existente por coherencia, completitud y actualidad.',
 'Cuando se detectan discrepancias entre el código y la documentación.');

-- ─── Workflows: programador (claude/codex) ──────────────────────────────────
INSERT OR IGNORE INTO workflows (tipo_agente, nombre, descripcion, pasos) VALUES
('programador','inicio-sesion',
 'Protocolo obligatorio al comenzar cualquier sesión de trabajo.',
 '["1. Ejecutar: orquesta sesion inicio <mi-nombre>",
   "2. Ver tareas asignadas: orquesta tarea listar --agente <mi-nombre>",
   "3. Votar todas las propuestas con posicion pendiente para mi agente",
   "4. Iniciar la tarea en la app: orquesta tarea iniciar <id> <mi-nombre>",
   "5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md"]'),
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
   "4. Si desacuerdo: añadir comentario técnico con alternativa concreta"]');

-- ─── Permisos del catálogo editable ───────────────────────────────────────
INSERT OR IGNORE INTO catalogo_edicion_permisos (entidad, rol, alcance, puede_crear, puede_editar, puede_activar, puede_versionar) VALUES
('reglas','programador','mismo_rol',1,1,1,1),
('reglas','documentador','mismo_rol',1,1,1,1),
('reglas','admin','todos',1,1,1,1),
('skills','programador','mismo_rol',1,1,1,1),
('skills','documentador','mismo_rol',1,1,1,1),
('skills','admin','todos',1,1,1,1),
('workflows','programador','mismo_rol',1,1,1,1),
('workflows','documentador','mismo_rol',1,1,1,1),
('workflows','admin','todos',1,1,1,1);

-- ─── Versiones iniciales del catálogo ─────────────────────────────────────
INSERT OR IGNORE INTO reglas_versiones (regla_id, version_num, tipo_agente, categoria, titulo, descripcion, activa, actor, accion)
SELECT id, 1, tipo_agente, categoria, titulo, descripcion, activa, 'orquesta', 'seed'
FROM reglas;

INSERT OR IGNORE INTO skills_versiones (skill_id, version_num, tipo_agente, nombre, descripcion, cuando_usar, activa, actor, accion)
SELECT id, 1, tipo_agente, nombre, descripcion, cuando_usar, activa, 'orquesta', 'seed'
FROM skills;

INSERT OR IGNORE INTO workflows_versiones (workflow_id, version_num, tipo_agente, nombre, descripcion, pasos, activo, actor, accion)
SELECT id, 1, tipo_agente, nombre, descripcion, pasos, activo, 'orquesta', 'seed'
FROM workflows;

-- ─── Workflows: documentador (antigravity) ──────────────────────────────────
INSERT OR IGNORE INTO workflows (tipo_agente, nombre, descripcion, pasos) VALUES
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
   "5. Notificar al programador que la documentación está lista"]');
`
