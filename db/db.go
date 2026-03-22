package db

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"orquesta/storage"
)

var DB *Handle
var currentStorageConfig storage.Config

// Open abre la base de datos usando el conector configurado.
// Por defecto usa SQLite resuelto desde ORQUESTA_DB o desde el repo actual.
// Para otros drivers, ORQUESTA_DB_DRIVER y ORQUESTA_DB_DSN pasan a ser obligatorios.
func Open() error {
	cfg, err := storage.ResolveConfig(resolverRuta)
	if err != nil {
		return fmt.Errorf("resolviendo almacenamiento: %w", err)
	}
	if cfg.BootstrapSchema && !storage.DialectForDriver(cfg.Driver).SupportsSchemaBootstrap() {
		return fmt.Errorf("bootstrap de schema no soportado para driver %s", cfg.Driver)
	}
	db, err := storage.Open(cfg)
	if err != nil {
		return fmt.Errorf("abriendo DB (%s): %w", cfg.Driver, err)
	}
	currentStorageConfig = cfg
	if cfg.BootstrapSchema {
		if err := aplicarSchema(db); err != nil {
			db.Close()
			return fmt.Errorf("aplicando schema: %w", err)
		}
		if err := aplicarSemillasSchema(db); err != nil {
			db.Close()
			return fmt.Errorf("aplicando semillas de schema: %w", err)
		}
	}
	DB = newHandle(db, cfg.Driver)
	if err := DB.Ping(); err != nil {
		DB.Close()
		DB = nil
		return fmt.Errorf("verificando DB (%s): %w", cfg.Driver, err)
	}
	if cfg.BootstrapSchema {
		if err := postMigraciones(); err != nil {
			DB.Close()
			DB = nil
			return fmt.Errorf("aplicando post-migraciones: %w", err)
		}
	}
	return nil
}

// postMigraciones ejecuta ALTER TABLE idempotentes para columnas añadidas tras el schema inicial.
func postMigraciones() error {
	migraciones := []string{
		`ALTER TABLE agentes ADD COLUMN estado_sesion TEXT DEFAULT NULL`,
		`CREATE TABLE IF NOT EXISTS proyectos (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_proyectos_updated
			AFTER UPDATE ON proyectos
		BEGIN
			UPDATE proyectos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS asignaciones (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			agente      TEXT    NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
			estado      TEXT    NOT NULL DEFAULT 'activa'
			                   CHECK (estado IN ('planificada','activa','pausada','cerrada')),
			nota        TEXT    NOT NULL DEFAULT '',
			created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			cerrada_at  DATETIME
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_asignaciones_updated
			AFTER UPDATE ON asignaciones
		BEGIN
			UPDATE asignaciones SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS conectores (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_conectores_updated
			AFTER UPDATE ON conectores
		BEGIN
			UPDATE conectores SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS locks (
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
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_locks_scope_activo
			ON locks(scope_type, scope_key)
			WHERE estado = 'activa'`,
		`CREATE TRIGGER IF NOT EXISTS trig_locks_updated
			AFTER UPDATE ON locks
		BEGIN
			UPDATE locks SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS worktrees (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_worktrees_updated
			AFTER UPDATE ON worktrees
		BEGIN
			UPDATE worktrees SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS git_merges (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_git_merges_updated
			AFTER UPDATE ON git_merges
		BEGIN
			UPDATE git_merges SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS runtime_handles (
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
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_handles_agente_activo
			ON runtime_handles(agente)
			WHERE estado IN ('activo','pausado')`,
		`CREATE TRIGGER IF NOT EXISTS trig_runtime_handles_updated
			AFTER UPDATE ON runtime_handles
		BEGIN
			UPDATE runtime_handles SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS runtime_orders (
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
		)`,
		`CREATE TABLE IF NOT EXISTS pools_capacidad (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_pools_capacidad_updated
			AFTER UPDATE ON pools_capacidad
		BEGIN
			UPDATE pools_capacidad SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS pool_modelos (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_pool_modelos_updated
			AFTER UPDATE ON pool_modelos
		BEGIN
			UPDATE pool_modelos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS politicas_modelo (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_politicas_modelo_updated
			AFTER UPDATE ON politicas_modelo
		BEGIN
			UPDATE politicas_modelo SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS decisiones_proyecto (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto_id   INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
			categoria     TEXT NOT NULL DEFAULT 'general',
			titulo        TEXT NOT NULL,
			solucion      TEXT NOT NULL DEFAULT '',
			motivo        TEXT NOT NULL DEFAULT '',
			alternativas  TEXT NOT NULL DEFAULT '',
			impacto       TEXT NOT NULL DEFAULT '',
			estado        TEXT NOT NULL DEFAULT 'vigente'
			                    CHECK (estado IN ('vigente','experimental','reemplazada','descartada','archivada')),
			propuesta_id  INTEGER REFERENCES propuestas(id) ON DELETE SET NULL,
			tarea_id      INTEGER REFERENCES tareas(id) ON DELETE SET NULL,
			metadata_json TEXT NOT NULL DEFAULT '{}',
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(proyecto_id, titulo)
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_decisiones_proyecto_updated
			AFTER UPDATE ON decisiones_proyecto
		BEGIN
			UPDATE decisiones_proyecto SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`CREATE TABLE IF NOT EXISTS documentos_externos (
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
		)`,
		`CREATE TRIGGER IF NOT EXISTS trig_documentos_externos_updated
			AFTER UPDATE ON documentos_externos
		BEGIN
			UPDATE documentos_externos SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
		END`,
		`ALTER TABLE propuestas ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE sesiones ADD COLUMN conector_id INTEGER REFERENCES conectores(id)`,
		`ALTER TABLE sesiones ADD COLUMN proyecto_id INTEGER REFERENCES proyectos(id)`,
		`ALTER TABLE sesiones ADD COLUMN pool_id INTEGER REFERENCES pools_capacidad(id)`,
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
   "5. Leer el doc del módulo asignado en docs/modulos/MXX_*.md"]'
		 WHERE tipo_agente='programador' AND nombre='inicio-sesion'`,
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
	}
	for _, m := range migraciones {
		if _, err := DB.Exec(m); err != nil && !esErrorMigracionIgnorable(err) {
			return err
		}
	}
	for _, item := range []struct {
		clave string
		valor string
	}{
		{"pool_handoff_threshold_seconds", "1800"},
		{"pool_handoff_threshold_ratio", "0.10"},
		{"pool_default_budget_source", "manual"},
		{"model_policy_default_profile", "implementacion"},
		{"model_policy_default_reasoning", "high"},
	} {
		ensureDefaultConfig(item.clave, item.valor)
	}
	if err := BackfillVotosPendientes(); err != nil {
		return err
	}
	return nil
}

func esErrorMigracionIgnorable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	for _, fragmento := range []string{
		"already exists",
		"duplicate column name",
		"duplicate key name",
		"duplicate column",
	} {
		if strings.Contains(msg, fragmento) {
			return true
		}
	}
	return false
}

func Close() {
	if DB != nil {
		DB.Close()
	}
	DB = nil
	currentStorageConfig = storage.Config{}
}

func DriverName() string {
	return currentStorageConfig.Driver
}

func BootstrapSchemaEnabled() bool {
	return currentStorageConfig.BootstrapSchema
}

func PlaceholderStyle() string {
	return storage.DialectForDriver(currentStorageConfig.Driver).PlaceholderStyle()
}

func QueryRebindingEnabled() bool {
	return storage.DialectForDriver(currentStorageConfig.Driver).RebindParameters()
}

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

func resolverRutaDesdeGitRoot(root string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		return "orquesta.db"
	}
	if esRepoOrquesta(filepath.Base(root)) {
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
	for _, nombreRepo := range nombresRepoOrquesta() {
		candidato := filepath.Join(base, nombreRepo)
		if existeFichero(filepath.Join(candidato, "go.mod")) {
			return filepath.Join(candidato, "orquesta.db")
		}
	}
	return ""
}

func nombresRepoOrquesta() []string {
	return []string{"orquestador", "orquesta"}
}

func esRepoOrquesta(nombre string) bool {
	for _, candidato := range nombresRepoOrquesta() {
		if nombre == candidato {
			return true
		}
	}
	return false
}

func existeFichero(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
		upsertValuesSQL(
			"config",
			[]string{"clave", "valor"},
			[]string{"clave"},
			[]upsertAssignment{
				{Column: "valor"},
			},
		),
		clave, valor,
	)
	return err
}

func ensureDefaultConfig(clave, valor string) {
	if DB == nil {
		return
	}
	_, _ = DB.Exec(
		insertIgnoreValuesSQL("config", []string{"clave", "valor"}, []string{"clave"}),
		clave, valor,
	)
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
