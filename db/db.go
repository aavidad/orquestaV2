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

var DB *sql.DB

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
	if err := backend.Prepare(db); err != nil {
		_ = db.Close()
		currentBackend = nil
		return err
	}
	DB = db
	return nil
}

func Close() {
	if DB != nil {
		_ = DB.Close()
	}
	DB = nil
	currentBackend = nil
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
		`INSERT OR IGNORE INTO reglas (tipo_agente, categoria, titulo, descripcion)
		 VALUES ('documentador','general','Multilenguaje por defecto donde aplique',
		 'La documentación debe mantenerse al menos en castellano e inglés cuando el proyecto tenga sentido multilenguaje. Los idiomas deben ir en ficheros separados y el castellano actúa como idioma por defecto salvo indicación distinta.')`,
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
	return ejecutarConReintentos(func() error {
		_, err := db.Exec(Schema)
		return err
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

// Audit registra una acción en el log de auditoría.
func Audit(agente, accion, entidad string, entidadID int64, detalle string) {
	if DB == nil {
		return
	}
	_, _ = DB.Exec(
		`INSERT INTO audit_log (agente, accion, entidad, entidad_id, detalle) VALUES (?,?,?,?,?)`,
		agente, accion, entidad, entidadID, detalle,
	)
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
