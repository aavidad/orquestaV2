package db

// Schema define el esquema completo de la base de datos de orquestación.
// Las migraciones se aplican en orden; nunca se modifican las existentes.
const Schema = `
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
    estado_sesion TEXT DEFAULT NULL,
    ultima_sesion DATETIME
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo
ON locks(scope_type, scope_key)
WHERE estado = 'activa';

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

-- ─── Control activo de agentes ────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS runtime_handles (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    agente           TEXT NOT NULL REFERENCES agentes(nombre),
    sesion_id        INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
    proyecto_id      INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    transporte       TEXT NOT NULL,
    handle_kind      TEXT NOT NULL,
    handle_ref       TEXT NOT NULL,
    estado           TEXT NOT NULL DEFAULT 'activo'
                         CHECK (estado IN ('activo','pausado','cerrado','fallido')),
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_agente_activo
ON runtime_handles(agente)
WHERE estado IN ('activo','pausado');

CREATE TABLE IF NOT EXISTS runtime_orders (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    agente           TEXT NOT NULL REFERENCES agentes(nombre),
    proyecto_id      INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
    tipo             TEXT NOT NULL
                         CHECK (tipo IN ('enviar_instruccion','pausar','continuar','handoff')),
    payload_json     TEXT NOT NULL DEFAULT '{}',
    estado           TEXT NOT NULL DEFAULT 'pendiente'
                         CHECK (estado IN ('pendiente','ejecutando','completada','fallida')),
    error_text       TEXT NOT NULL DEFAULT '',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    started_at       DATETIME,
    finished_at      DATETIME
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

CREATE TABLE IF NOT EXISTS politicas_modelo (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_tipo       TEXT NOT NULL
                        CHECK (scope_tipo IN ('global','perfil','proyecto','fase','tarea')),
    scope_ref        TEXT NOT NULL DEFAULT '',
    perfil_tarea     TEXT NOT NULL DEFAULT '*',
    pool_slug        TEXT NOT NULL DEFAULT '',
    model_slug       TEXT NOT NULL DEFAULT '',
    reasoning_effort TEXT NOT NULL DEFAULT '',
    prioridad        INTEGER NOT NULL DEFAULT 100,
    activa           INTEGER NOT NULL DEFAULT 1,
    metadata_json    TEXT NOT NULL DEFAULT '{}',
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

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

CREATE TRIGGER IF NOT EXISTS trig_conectores_updated
    AFTER UPDATE ON conectores
BEGIN
    UPDATE conectores SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
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

CREATE TRIGGER IF NOT EXISTS trig_runtime_handles_updated
    AFTER UPDATE ON runtime_handles
BEGIN
    UPDATE runtime_handles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
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

CREATE TRIGGER IF NOT EXISTS trig_git_merges_updated
    AFTER UPDATE ON git_merges
BEGIN
    UPDATE git_merges SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
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
