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
    ('version',      '1.0.0');

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
