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
