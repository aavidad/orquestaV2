package db

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var DB *Handle

type EventoNotificacion struct {
	Tipo       string // bloqueo | propuesta | fin_proyecto | mensaje
	ID         int64
	Codigo     string
	Agente     string
	Texto      string
	ProyectoID int64
}

var CanalNotificaciones = make(chan EventoNotificacion, 100)

// Open abre (o crea) la base de datos usando el backend configurado y aplica su preparación.
func Open() error {
	backend, target, err := resolveBackend()
	if err != nil {
		return err
	}
	db, err := backend.Open(target)
	if err != nil {
		return err
	}
	currentBackend = backend
	currentTarget = target
	if err := backend.Prepare(db); err != nil {
		_ = db.Close()
		currentBackend = nil
		currentTarget = ""
		return err
	}
	DB = newHandle(db, backend.Name())
	return nil
}

func Close() {
	if DB != nil {
		_ = DB.Close()
	}
	DB = nil
	currentBackend = nil
	currentTarget = ""
}

// Orden de resolución de la ruta SQLite:
//  1. Variable de entorno ORQUESTA_DB
//  2. Repositorio `orquesta` del workspace actual (../orquesta/orquesta.db)
//  3. Si el git-root ya es el repo `orquesta`, <git-root>/orquesta.db
//  4. ./orquesta.db
func resolverRuta() string {
	if v := os.Getenv("ORQUESTA_DB"); strings.TrimSpace(v) != "" {
		return v
	}
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return resolverRutaDesdeGitRoot(strings.TrimSpace(string(out)))
	}
	if wd, err := os.Getwd(); err == nil {
		if ruta := buscarRutaRepoOrquesta(wd); ruta != "" {
			return ruta
		}
	}
	return "orquesta.db"
}

// postMigraciones ejecuta ALTER TABLE idempotentes para columnas añadidas tras el schema inicial.
func postMigraciones(conn *sql.DB) error {
	migraciones := []string{
		`ALTER TABLE sesiones ADD COLUMN conector_id INTEGER REFERENCES conectores(id)`,
		`ALTER TABLE agentes ADD COLUMN estado_sesion TEXT DEFAULT NULL`,
		`ALTER TABLE tareas ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE propuestas ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE sesiones ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE sesiones ADD COLUMN estado TEXT NOT NULL DEFAULT 'activa'`,
		`ALTER TABLE sesiones ADD COLUMN cwd TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN herramienta TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN external_session_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN resume_payload_json TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN resumen_continuidad TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN branch TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN heartbeat_at DATETIME`,
		`ALTER TABLE sesiones ADD COLUMN host TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE sesiones ADD COLUMN pid INTEGER`,
		`ALTER TABLE tareas ADD COLUMN contrato_definido INTEGER NOT NULL DEFAULT 0`,
		`CREATE TABLE IF NOT EXISTS proyectos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			nombre TEXT NOT NULL,
			ruta_abs TEXT NOT NULL UNIQUE,
			tipo TEXT NOT NULL DEFAULT 'repo' CHECK (tipo IN ('raiz','grupo','repo')),
			parent_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			activo INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS asignaciones (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
			estado TEXT NOT NULL DEFAULT 'activa' CHECK (estado IN ('planificada','activa','pausada','cerrada')),
			nota TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			cerrada_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS locks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			tarea_id INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			sesion_id INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			scope_type TEXT NOT NULL CHECK (scope_type IN ('project','task','module','path','branch','worktree','otro')),
			scope_key TEXT NOT NULL,
			ruta_abs TEXT NOT NULL DEFAULT '',
			branch TEXT NOT NULL DEFAULT '',
			motivo TEXT NOT NULL DEFAULT '',
			token_lease TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'activa' CHECK (estado IN ('activa','liberada','expirada','fallida')),
			heartbeat_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			liberada_at DATETIME
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo
			ON locks(scope_type, scope_key)
			WHERE estado = 'activa'`,
		`CREATE TABLE IF NOT EXISTS worktrees (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto_id INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
			tarea_id INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			lock_id INTEGER REFERENCES locks(id) ON DELETE SET NULL,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			nombre TEXT NOT NULL,
			ruta_abs TEXT NOT NULL UNIQUE,
			branch TEXT NOT NULL,
			base_ref TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'activa' CHECK (estado IN ('activa','cerrada','fallida')),
			motivo TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			cerrada_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS runtime_instances (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			sesion_id INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
			parent_runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			provider TEXT NOT NULL DEFAULT '',
			connector TEXT NOT NULL DEFAULT '',
			external_session_id TEXT NOT NULL DEFAULT '',
			logical_state TEXT NOT NULL DEFAULT 'arrancando',
			process_state TEXT NOT NULL DEFAULT 'desconocido',
			pid INTEGER,
			ppid INTEGER,
			child_count INTEGER NOT NULL DEFAULT 0,
			thread_count INTEGER NOT NULL DEFAULT 0,
			model TEXT NOT NULL DEFAULT '',
			reasoning TEXT NOT NULL DEFAULT '',
			task_profile TEXT NOT NULL DEFAULT '',
			cwd TEXT NOT NULL DEFAULT '',
			branch TEXT NOT NULL DEFAULT '',
			last_event_at DATETIME,
			last_heartbeat_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_instances_sesion
			ON runtime_instances(sesion_id)
			WHERE sesion_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS runtime_telemetry_samples (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			runtime_id INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
			cpu_pct REAL NOT NULL DEFAULT 0,
			mem_bytes INTEGER NOT NULL DEFAULT 0,
			rss_bytes INTEGER NOT NULL DEFAULT 0,
			open_fds INTEGER NOT NULL DEFAULT 0,
			child_count INTEGER NOT NULL DEFAULT 0,
			thread_count INTEGER NOT NULL DEFAULT 0,
			logical_state TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			sample_json TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runtime_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			runtime_id INTEGER NOT NULL REFERENCES runtime_instances(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			level TEXT NOT NULL DEFAULT 'info',
			message TEXT NOT NULL DEFAULT '',
			payload_json TEXT NOT NULL DEFAULT '{}',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS runtime_handles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			sesion_id INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			transporte TEXT NOT NULL DEFAULT 'cli',
			handle_kind TEXT NOT NULL DEFAULT 'session',
			handle_ref TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo','pausado','cerrado','fallido')),
			lease_token TEXT NOT NULL DEFAULT '',
			capabilities_json TEXT NOT NULL DEFAULT '{}',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			last_seen_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_sesion
			ON runtime_handles(sesion_id)
			WHERE sesion_id IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_handles_agente_estado
			ON runtime_handles(agente, estado, id DESC)`,
		`CREATE TABLE IF NOT EXISTS runtime_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			handle_id INTEGER REFERENCES runtime_handles(id) ON DELETE SET NULL,
			tipo TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			resultado_json TEXT NOT NULL DEFAULT '{}',
			error_text TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','tomada','ejecutando','completada','fallida','expirada','cancelada')),
			available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			finished_at DATETIME,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE runtime_orders ADD COLUMN available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_orders_estado_created
			ON runtime_orders(estado, available_at, id)`,
		`CREATE TABLE IF NOT EXISTS runtime_mailbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_agente TEXT NOT NULL REFERENCES agentes(nombre),
			to_agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			runtime_order_id INTEGER REFERENCES runtime_orders(id) ON DELETE SET NULL,
			kind TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			estado TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','entregado','consumido','expirado','cancelado')),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME,
			consumed_at DATETIME
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_mailbox_destino_estado
			ON runtime_mailbox(to_agente, estado, id DESC)`,
		`CREATE TABLE IF NOT EXISTS runtime_checkpoints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			sesion_id INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			checkpoint_kind TEXT NOT NULL DEFAULT 'manual',
			resumen TEXT NOT NULL DEFAULT '',
			branch TEXT NOT NULL DEFAULT '',
			cwd TEXT NOT NULL DEFAULT '',
			payload_json TEXT NOT NULL DEFAULT '{}',
			resume_strategy TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_runtime_checkpoints_agente_created
			ON runtime_checkpoints(agente, created_at DESC, id DESC)`,
		`CREATE TABLE IF NOT EXISTS conectores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			nombre TEXT NOT NULL,
			transporte TEXT NOT NULL DEFAULT 'cli' CHECK (transporte IN ('cli','mcp_stdio','mcp_http','api','otro')),
			comando TEXT NOT NULL DEFAULT '',
			args_json TEXT NOT NULL DEFAULT '[]',
			env_json TEXT NOT NULL DEFAULT '{}',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			activo INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_proyectos_updated
			AFTER UPDATE ON proyectos
		BEGIN
			UPDATE proyectos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_conectores_updated
			AFTER UPDATE ON conectores
		BEGIN
			UPDATE conectores SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_asignaciones_updated
			AFTER UPDATE ON asignaciones
		BEGIN
			UPDATE asignaciones SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_locks_updated
			AFTER UPDATE ON locks
		BEGIN
			UPDATE locks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_worktrees_updated
			AFTER UPDATE ON worktrees
		BEGIN
			UPDATE worktrees SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_runtime_instances_updated
			AFTER UPDATE ON runtime_instances
		BEGIN
			UPDATE runtime_instances SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_runtime_handles_updated
			AFTER UPDATE ON runtime_handles
		BEGIN
			UPDATE runtime_handles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TRIGGER IF NOT EXISTS trig_runtime_orders_updated
			AFTER UPDATE ON runtime_orders
		BEGIN
			UPDATE runtime_orders SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('workspace_root','')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('agent_loop_mode','sticky')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('agent_tick_seconds','30')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('lock_lease_seconds','90')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('worktree_root_name','.orquesta-worktrees')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('refactor_backup_required','true')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('refactor_min_reviewers','2')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('refactor_reviewer_must_be_non_author','true')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('refactor_preserve_security','true')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('refactor_preserve_project_philosophy','true')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('agent_autoexecute_safe','true')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('agent_confirm_dangerous_with_project_peer','true')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('propuesta_min_votes','2')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('propuesta_min_non_author_votes','2')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('pool_handoff_threshold_seconds','1800')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('pool_handoff_threshold_ratio','0.10')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('pool_default_budget_source','manual')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('model_policy_default_profile','implementacion')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('model_policy_default_reasoning','high')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('runtime_handle_stale_seconds','120')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('runtime_order_batch_size','10')`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('runtime_order_stale_seconds','120')`,
		`INSERT OR IGNORE INTO conectores (slug, nombre, transporte, comando, metadata_json) VALUES ('claude-code','Claude Code','cli','claude','{\"familia\":\"anthropic\",\"reanudable\":true}')`,
		`INSERT OR IGNORE INTO conectores (slug, nombre, transporte, comando, metadata_json) VALUES ('codex-cli','Codex CLI','cli','codex','{\"familia\":\"openai\",\"reanudable\":true}')`,
		`INSERT OR IGNORE INTO conectores (slug, nombre, transporte, comando, metadata_json) VALUES ('gemini-cli','Gemini CLI','cli','gemini','{\"familia\":\"google\",\"reanudable\":true}')`,
		`UPDATE reglas
		 SET descripcion='No escribir código sin propuesta OP-XXX aprobada en la app de orquestación.'
		 WHERE tipo_agente='programador' AND categoria='calidad' AND titulo='Propuesta antes de código'`,
		`UPDATE reglas
		 SET descripcion='Nueva propuesta: orquesta propuesta nueva "Título" --descripcion "Descripción" --agente <nombre>  |  Votar: orquesta votar <OP-XXX> <acuerdo|desacuerdo|abstencion> --agente <nombre> --comentario "razón"  |  Ver propuesta: orquesta propuesta ver <OP-XXX>  |  Ver pendientes de voto: incluidas en el briefing de sesion inicio.'
		 WHERE tipo_agente='programador' AND categoria='comandos' AND titulo='Propuestas y votación'`,
		`UPDATE workflows
		 SET pasos='["1. Leer la propuesta completa: orquesta propuesta ver <codigo>",
   "2. Analizar impacto técnico en módulos asignados",
   "3. Votar: orquesta votar <codigo> <acuerdo|desacuerdo|abstencion> --agente <mi-nombre> --comentario \"razón\"",
   "4. Si desacuerdo: añadir comentario técnico con alternativa concreta"]'
		 WHERE tipo_agente='programador' AND nombre='votar-propuesta'`,
		`UPDATE workflows
		 SET pasos='["1. Ejecutar: orquesta sesion inicio <mi-nombre>",
   "2. Ver tareas asignadas: orquesta tarea listar --agente <mi-nombre>",
   "3. Votar todas las propuestas con posicion pendiente para mi agente",
   "4. Iniciar la tarea en la app: orquesta tarea iniciar <id> <mi-nombre>",
   "5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md",
   "6. Operar con autonomía para acciones normales; si una acción es destructiva o peligrosa, consultar antes con Orquesta o con otro agente del mismo proyecto"]'
		 WHERE tipo_agente='programador' AND nombre='inicio-sesion'`,
		`INSERT OR IGNORE INTO reglas (tipo_agente, categoria, titulo, descripcion)
		 VALUES ('programador','sesion','Autonomía operativa segura',
		 'El agente puede ejecutar acciones normales sin pedir permiso previo. Si la acción es destructiva, de borrado, irreversible o de riesgo alto, debe consultar primero a Orquesta o a otro agente de su mismo proyecto antes de ejecutarla.')`,
		`INSERT OR IGNORE INTO reglas (tipo_agente, categoria, titulo, descripcion)
		 VALUES ('programador','arquitectura','Multilenguaje por defecto donde aplique',
		 'Salvo excepciones técnicas claras como drivers o componentes sin interfaz de usuario real, las aplicaciones deben nacer preparadas para al menos dos idiomas. El idioma por defecto será castellano y el idioma debe poder configurarse por usuario o despliegue.')`,
		`UPDATE workflows
		 SET pasos='["1. Asegurar que todo el trabajo está commiteado (git status limpio)",
   "2. Completar o bloquear mis tareas en la app de orquestación según corresponda",
   "3. Ejecutar: orquesta sesion fin <mi-nombre>"]'
		 WHERE tipo_agente='programador' AND nombre='fin-sesion'`,
		`UPDATE workflows
		 SET pasos='["1. Crear propuesta OP-XXX con orquesta propuesta nueva y esperar consenso en la app",
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
   "14. Gate final: go build ./... && go vet ./... && go test ./... → notificar a Antigravity"]'
		 WHERE tipo_agente='programador' AND nombre='crear-modulo'`,
		`UPDATE workflows
		 SET pasos='["1. Ejecutar: orquesta sesion inicio antigravity",
   "2. Ver tareas asignadas: orquesta tarea listar --agente antigravity",
   "3. Votar propuestas con posicion pendiente para antigravity",
   "4. Revisar docs/00_INDICE.md para detectar gaps"]'
		 WHERE tipo_agente='documentador' AND nombre='inicio-sesion'`,
		`INSERT OR IGNORE INTO reglas (tipo_agente, categoria, titulo, descripcion)
		 VALUES ('documentador','general','Referencia obligatoria de documentación externa',
		 'Si Antigravity crea o mantiene documentación fuera de orquestador, debe quedar siempre documentada también en Orquesta. La BD debe guardar una referencia clara con la ruta de esos ficheros y un resumen de su contenido o propósito.')`,
		`ALTER TABLE agentes ADD COLUMN consumo_dia_segundos INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE agentes ADD COLUMN consumo_semanal_segundos INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE agentes ADD COLUMN limite_dia_segundos INTEGER NOT NULL DEFAULT 18000`,
		`ALTER TABLE agentes ADD COLUMN limite_semanal_segundos INTEGER NOT NULL DEFAULT 126000`,
		`ALTER TABLE agentes ADD COLUMN last_usage_reset_at DATETIME`,
		`ALTER TABLE agentes ADD COLUMN estado_cuota TEXT NOT NULL DEFAULT 'activo' CHECK (estado_cuota IN ('activo','enfriamiento','agotado'))`,
		`INSERT OR IGNORE INTO config (clave, valor) VALUES ('quota_daily_seconds','18000')`,
		`ALTER TABLE agentes ADD COLUMN reanimar_at DATETIME`,
		`ALTER TABLE agentes ADD COLUMN motivo_pausa TEXT`,
	}
	for _, m := range migraciones {
		if err := ejecutarConReintentos(func() error {
			_, err := conn.Exec(m)
			return err
		}); err != nil && !esErrorMigracionIgnorable(err) {
			return fmt.Errorf("post-migraciones: %w", err)
		}
	}
	return nil
}

func resolverRutaDesdeGitRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return "orquesta.db"
	}
	if filepath.Base(root) == "orquesta" {
		return filepath.Join(root, "orquesta.db")
	}
	if ruta := rutaRepoOrquestaEnDirectorio(filepath.Dir(root)); ruta != "" {
		return ruta
	}
	return filepath.Join(root, "orquesta.db")
}

func buscarRutaRepoOrquesta(inicio string) string {
	actual := filepath.Clean(inicio)
	for {
		if ruta := rutaRepoOrquestaEnDirectorio(actual); ruta != "" {
			return ruta
		}
		siguiente := filepath.Dir(actual)
		if siguiente == actual {
			return ""
		}
		actual = siguiente
	}
}

func rutaRepoOrquestaEnDirectorio(base string) string {
	for _, nombre := range []string{"orquesta", "orquestador"} {
		candidato := filepath.Join(base, nombre)
		if existeFichero(filepath.Join(candidato, "go.mod")) {
			return filepath.Join(candidato, "orquesta.db")
		}
	}
	return ""
}

func existeFichero(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func aplicarSchema(db *sql.DB) error {
	if err := prepararSchemaLegacy(db); err != nil {
		return err
	}
	if err := ejecutarConReintentos(func() error {
		return aplicarDDL(db, schemaDDL())
	}); err != nil {
		return err
	}
	return ejecutarConReintentos(func() error {
		return aplicarSeedSQL(db, schemaSeedData())
	})
}

func ejecutarConReintentos(fn func() error) error {
	var err error
	for intento := 0; intento < 12; intento++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !esErrorPersistenciaBusy(err) {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
	return err
}

func esErrorPersistenciaBusy(err error) bool {
	if currentBackend != nil {
		return currentBackend.IsBusy(err)
	}
	return false
}

func esErrorMigracionIgnorable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "already exists")
}

func prepararSchemaLegacy(db *sql.DB) error {
	if err := reconstruirRuntimeHandlesLegacy(db); err != nil {
		return err
	}
	if err := reconstruirRuntimeOrdersLegacy(db); err != nil {
		return err
	}
	return nil
}

func tablaExiste(db *sql.DB, nombre string) (bool, error) {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`, nombre).Scan(&total)
	return total > 0, err
}

func tablaTieneColumna(db *sql.DB, tabla, columna string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + tabla + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(strings.TrimSpace(name), columna) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func tablaSQL(db *sql.DB, nombre string) (string, error) {
	var sqlText sql.NullString
	err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='table' AND name = ?`, nombre).Scan(&sqlText)
	if err == sql.ErrNoRows || !sqlText.Valid {
		return "", nil
	}
	return sqlText.String, err
}

func tablaTieneFilas(db *sql.DB, nombre string) (bool, error) {
	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + nombre).Scan(&total); err != nil {
		return false, err
	}
	return total > 0, nil
}

func reconstruirRuntimeHandlesLegacy(db *sql.DB) error {
	const tableName = "runtime_handles"
	existe, err := tablaExiste(db, tableName)
	if err != nil || !existe {
		return err
	}
	cols := []string{
		"id", "agente", "proyecto_id", "sesion_id", "runtime_id", "transporte", "handle_kind",
		"handle_ref", "estado", "lease_token", "capabilities_json", "metadata_json",
		"last_seen_at", "created_at", "updated_at",
	}
	needsRebuild := false
	for _, col := range cols {
		ok, err := tablaTieneColumna(db, tableName, col)
		if err != nil {
			return err
		}
		if !ok {
			needsRebuild = true
			break
		}
	}
	if !needsRebuild {
		return nil
	}
	return rebuildSQLiteTable(
		db,
		tableName,
		[]string{"idx_runtime_handles_sesion", "idx_runtime_handles_agente_estado"},
		`CREATE TABLE runtime_handles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			sesion_id INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			transporte TEXT NOT NULL DEFAULT 'cli',
			handle_kind TEXT NOT NULL DEFAULT 'session',
			handle_ref TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo','pausado','cerrado','fallido')),
			lease_token TEXT NOT NULL DEFAULT '',
			capabilities_json TEXT NOT NULL DEFAULT '{}',
			metadata_json TEXT NOT NULL DEFAULT '{}',
			last_seen_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		[]string{
			`CREATE UNIQUE INDEX idx_runtime_handles_sesion ON runtime_handles(sesion_id) WHERE sesion_id IS NOT NULL`,
			`CREATE INDEX idx_runtime_handles_agente_estado ON runtime_handles(agente, estado, id DESC)`,
		},
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			insertCols := []string{
				"id", "agente", "proyecto_id", "sesion_id", "runtime_id", "transporte", "handle_kind",
				"handle_ref", "estado", "lease_token", "capabilities_json", "metadata_json",
				"last_seen_at", "created_at", "updated_at",
			}
			selectExprs := []string{
				runtimeLegacyExpr(legacyCols, "id", "NULL"),
				runtimeLegacyExpr(legacyCols, "agente", "''"),
				runtimeLegacyExpr(legacyCols, "proyecto_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "sesion_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "runtime_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "transporte", "'cli'"),
				runtimeLegacyExpr(legacyCols, "handle_kind", "'session'"),
				runtimeLegacyExpr(legacyCols, "handle_ref", "''"),
				runtimeLegacyEstadoHandleExpr(legacyCols),
				runtimeLegacyExpr(legacyCols, "lease_token", "''"),
				runtimeLegacyExpr(legacyCols, "capabilities_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "metadata_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "last_seen_at", "CURRENT_TIMESTAMP"),
				runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP"),
				runtimeLegacyExpr(legacyCols, "updated_at", runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP")),
			}
			return `INSERT INTO runtime_handles (` + strings.Join(insertCols, ", ") + `)
				SELECT ` + strings.Join(selectExprs, ", ") + ` FROM ` + legacy, nil
		},
	)
}

func reconstruirRuntimeOrdersLegacy(db *sql.DB) error {
	const tableName = "runtime_orders"
	existe, err := tablaExiste(db, tableName)
	if err != nil || !existe {
		return err
	}
	cols := []string{
		"id", "agente", "proyecto_id", "runtime_id", "handle_id", "tipo", "payload_json",
		"resultado_json", "error_text", "estado", "available_at", "created_at",
		"started_at", "finished_at", "updated_at",
	}
	needsRebuild := false
	for _, col := range cols {
		ok, err := tablaTieneColumna(db, tableName, col)
		if err != nil {
			return err
		}
		if !ok {
			needsRebuild = true
			break
		}
	}
	if !needsRebuild {
		sqlText, err := tablaSQL(db, tableName)
		if err != nil {
			return err
		}
		sqlNorm := strings.ToLower(sqlText)
		if strings.TrimSpace(sqlNorm) != "" &&
			(!strings.Contains(sqlNorm, "tomada") || !strings.Contains(sqlNorm, "ejecutando")) {
			needsRebuild = true
		}
	}
	if !needsRebuild {
		return nil
	}
	return rebuildSQLiteTable(
		db,
		tableName,
		[]string{"idx_runtime_orders_estado_created"},
		`CREATE TABLE runtime_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			handle_id INTEGER REFERENCES runtime_handles(id) ON DELETE SET NULL,
			tipo TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			resultado_json TEXT NOT NULL DEFAULT '{}',
			error_text TEXT NOT NULL DEFAULT '',
			estado TEXT NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente','tomada','ejecutando','completada','fallida','expirada','cancelada')),
			available_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			started_at DATETIME,
			finished_at DATETIME,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		[]string{
			`CREATE INDEX idx_runtime_orders_estado_created ON runtime_orders(estado, available_at, id)`,
		},
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			insertCols := []string{
				"id", "agente", "proyecto_id", "runtime_id", "handle_id", "tipo", "payload_json",
				"resultado_json", "error_text", "estado", "available_at", "created_at",
				"started_at", "finished_at", "updated_at",
			}
			selectExprs := []string{
				runtimeLegacyExpr(legacyCols, "id", "NULL"),
				runtimeLegacyExpr(legacyCols, "agente", "''"),
				runtimeLegacyExpr(legacyCols, "proyecto_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "runtime_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "handle_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "tipo", "'sync_status'"),
				runtimeLegacyExpr(legacyCols, "payload_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "resultado_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "error_text", "''"),
				runtimeLegacyEstadoOrderExpr(legacyCols),
				runtimeLegacyExpr(legacyCols, "available_at", runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP")),
				runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP"),
				runtimeLegacyExpr(legacyCols, "started_at", "NULL"),
				runtimeLegacyExpr(legacyCols, "finished_at", "NULL"),
				runtimeLegacyExpr(legacyCols, "updated_at", runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP")),
			}
			return `INSERT INTO runtime_orders (` + strings.Join(insertCols, ", ") + `)
				SELECT ` + strings.Join(selectExprs, ", ") + ` FROM ` + legacy, nil
		},
	)
}

func rebuildSQLiteTable(
	db *sql.DB,
	tableName string,
	indexes []string,
	createTableSQL string,
	createIndexes []string,
	copyBuilder func(legacy string, legacyCols map[string]bool) (string, []any),
) error {
	hasRows, err := tablaTieneFilas(db, tableName)
	if err != nil {
		return err
	}
	legacyCols, err := columnasTabla(db, tableName)
	if err != nil {
		return err
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		return err
	}
	defer func() {
		_, _ = db.Exec(`PRAGMA foreign_keys = ON`)
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, idx := range indexes {
		if _, err := tx.Exec(`DROP INDEX IF EXISTS ` + idx); err != nil {
			return err
		}
	}

	legacyName := tableName + `_legacy_rebuild`
	if _, err := tx.Exec(`DROP TABLE IF EXISTS ` + legacyName); err != nil {
		return err
	}
	if _, err := tx.Exec(`ALTER TABLE ` + tableName + ` RENAME TO ` + legacyName); err != nil {
		return err
	}
	if _, err := tx.Exec(createTableSQL); err != nil {
		return err
	}
	for _, stmt := range createIndexes {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	if hasRows && copyBuilder != nil {
		insertSQL, args := copyBuilder(legacyName, legacyCols)
		if strings.TrimSpace(insertSQL) != "" {
			if _, err := tx.Exec(insertSQL, args...); err != nil {
				return err
			}
		}
	}
	if _, err := tx.Exec(`DROP TABLE ` + legacyName); err != nil {
		return err
	}
	return tx.Commit()
}

func columnasTabla(db *sql.DB, tabla string) (map[string]bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + tabla + `)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var (
			cid       int
			name      string
			typ       string
			notNull   int
			dfltValue sql.NullString
			pk        int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		out[strings.ToLower(strings.TrimSpace(name))] = true
	}
	return out, rows.Err()
}

func runtimeLegacyExpr(cols map[string]bool, col, fallback string) string {
	if cols[strings.ToLower(strings.TrimSpace(col))] {
		return col
	}
	return fallback
}

func runtimeLegacyEstadoHandleExpr(cols map[string]bool) string {
	if !cols["estado"] {
		return `'activo'`
	}
	return `CASE
		WHEN estado IN ('activo','pausado','cerrado','fallido') THEN estado
		ELSE 'activo'
	END`
}

func runtimeLegacyEstadoOrderExpr(cols map[string]bool) string {
	if !cols["estado"] {
		return `'pendiente'`
	}
	return `CASE
		WHEN estado IN ('pendiente','tomada','ejecutando','completada','fallida','expirada','cancelada') THEN estado
		ELSE 'pendiente'
	END`
}

// DetectarActividadSospechosa analiza una cadena en busca de patrones de ataque o bypass.
func DetectarActividadSospechosa(input string) (bool, string) {
	dangerous := []string{"DROP TABLE", "DELETE FROM agents", "UPDATE agentes SET", "sqlite3 ", "os.Remove", "os.Exit", "eval(", "exec("}
	for _, p := range dangerous {
		if strings.Contains(strings.ToUpper(input), strings.ToUpper(p)) {
			return true, fmt.Sprintf("Patrón detectado: %s", p)
		}
	}
	return false, ""
}

type LogAuditoria struct {
	ID        int64     `json:"id"`
	Agente    string    `json:"agente"`
	Accion    string    `json:"accion"`
	Entidad   string    `json:"entidad"`
	EntidadID int64     `json:"entidad_id"`
	Detalle   string    `json:"detalle"`
	CreatedAt time.Time `json:"created_at"`
}

type FiltroAuditoria struct {
	Agente  *string
	Accion  *string
	Entidad *string
	Limite  int
}

// Audit registra una acción en el log de auditoría.
func Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	if DB == nil {
		return
	}
	// Monitorización defensiva de auditoría
	if sos, motivo := DetectarActividadSospechosa(detalle); sos {
		detalle = fmt.Sprintf("⚠️ ALERTA SEGURIDAD: %s | %s", motivo, detalle)
		// Auto-bloqueo preventivo
		reanimar := time.Now().Add(6 * time.Hour)
		_, _ = DB.Exec(`UPDATE agentes SET estado_cuota = 'enfriamiento', reanimar_at = ?, motivo_pausa = ? WHERE nombre = ?`,
			reanimar, "Intento de bypass de seguridad detectado (audit-warden)", agente)
	}

	_, _ = DB.Exec(
		`INSERT INTO audit_log (agente, accion, entidad, entidad_id, detalle) VALUES (?,?,?,?,?)`,
		agente, accion, entidad, entidadID, detalle,
	)
}

// ListarAuditoria recupera registros del log de auditoría.
func ListarAuditoria(f FiltroAuditoria) ([]*LogAuditoria, error) {
	q := `SELECT id, agente, accion, entidad, entidad_id, detalle, created_at FROM audit_log WHERE 1=1`
	args := []any{}
	if f.Agente != nil {
		q += " AND agente = ?"
		args = append(args, *f.Agente)
	}
	if f.Accion != nil {
		q += " AND accion = ?"
		args = append(args, *f.Accion)
	}
	if f.Entidad != nil {
		q += " AND entidad = ?"
		args = append(args, *f.Entidad)
	}
	q += " ORDER BY id DESC"
	if f.Limite > 0 {
		q += fmt.Sprintf(" LIMIT %d", f.Limite)
	} else {
		q += " LIMIT 50"
	}
	rows, err := DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*LogAuditoria
	for rows.Next() {
		l := &LogAuditoria{}
		if err := rows.Scan(&l.ID, &l.Agente, &l.Accion, &l.Entidad, &l.EntidadID, &l.Detalle, &l.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// ConfigGet devuelve el valor de una clave de configuración.
func ConfigGet(clave string) (string, error) {
	var v string
	err := DB.QueryRow(`SELECT valor FROM config WHERE clave = ?`, clave).Scan(&v)
	return v, err
}

// ConfigSet actualiza o inserta una clave de configuración.
func ConfigSet(clave, valor string) error {
	_, err := DB.Exec(
		`INSERT INTO config (clave, valor) VALUES (?,?) ON CONFLICT(clave) DO UPDATE SET valor=excluded.valor`,
		clave, valor,
	)
	return err
}

// ConfigAll devuelve toda la configuración.
func ConfigAll() (map[string]string, error) {
	rows, err := DB.Query(`SELECT clave, valor FROM config ORDER BY clave`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}
