package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"orquesta/storage"
)

const (
	persistenciaBusyMaxIntentos = 30
	persistenciaBusyBackoff     = 200 * time.Millisecond
)

var DB *Handle

type OpenOptions struct {
	BootstrapSchema    *bool
	SkipPostMigrations *bool
	ReadOnly           bool
}

type EventoNotificacion struct {
	Tipo       string // bloqueo | propuesta | fin_proyecto | mensaje
	ID         int64
	Codigo     string
	Agente     string
	Texto      string
	ProyectoID int64
	Payload    map[string]any
}

var CanalNotificaciones = make(chan EventoNotificacion, 100)

func EmitirNotificacion(ev EventoNotificacion) bool {
	select {
	case CanalNotificaciones <- ev:
		return true
	default:
		log.Printf("orquesta/db: notificacion descartada por canal saturado tipo=%s proyecto=%d agente=%s", strings.TrimSpace(ev.Tipo), ev.ProyectoID, strings.TrimSpace(ev.Agente))
		return false
	}
}

// Open inicializa el sistema de persistencia de Orquesta.
// Resuelve el backend (SQLite/otros), aplica migraciones idempotentes y
// prepara los canales de notificación en tiempo real.
func Open() error {
	return OpenWithOptions(OpenOptions{})
}

func OpenWithOptions(opts OpenOptions) error {
	runtimeHandleHotReset()
	resetRuntimeOrdersHotIndex()
	backend, cfg, err := resolveOpenConfig()
	if err != nil {
		return err
	}
	cfg = applyOpenOptions(cfg, opts)
	db, err := backend.Open(cfg)
	if err != nil {
		return err
	}
	currentBackend = backend
	currentConfig = cfg
	currentTarget = configTarget(cfg)
	if err := backend.Prepare(db, cfg); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("orquesta/db: close tras prepare fallida: %v", closeErr)
		}
		persistenceMu.Lock()
		currentBackend = nil
		currentConfig = storage.Config{}
		currentTarget = ""
		persistenceMu.Unlock()
		return err
	}
	handle := newHandle(db, backend.Name())
	persistenceMu.Lock()
	DB = handle
	currentBackend = backend
	currentConfig = cfg
	currentTarget = configTarget(cfg)
	persistenceMu.Unlock()
	return nil
}

func OpenRecoveryReadOnly() error {
	disabled := false
	skipPost := true
	return OpenWithOptions(OpenOptions{
		BootstrapSchema:    &disabled,
		SkipPostMigrations: &skipPost,
		ReadOnly:           true,
	})
}

func resolveOpenConfig() (Backend, storage.Config, error) {
	backend, cfg, err := resolveBackend()
	if err != nil {
		return nil, storage.Config{}, err
	}
	return backend, adjustConfigForOpen(cfg), nil
}

func adjustConfigForOpen(cfg storage.Config) storage.Config {
	if shouldUseSQLiteRecoveryOpen(cfg) {
		cfg.BootstrapSchema = false
		cfg.SkipPostMigrations = true
	}
	return cfg
}

func applyOpenOptions(cfg storage.Config, opts OpenOptions) storage.Config {
	if opts.BootstrapSchema != nil {
		cfg.BootstrapSchema = *opts.BootstrapSchema
	}
	if opts.SkipPostMigrations != nil {
		cfg.SkipPostMigrations = *opts.SkipPostMigrations
	}
	if opts.ReadOnly && normalizedDriverName(cfg.Driver) == "sqlite" {
		cfg.DSN = storage.SQLiteDSNWithMode(cfg.Path, "ro")
	}
	return cfg
}

func shouldUseSQLiteRecoveryOpen(cfg storage.Config) bool {
	if normalizedDriverName(cfg.Driver) != "sqlite" {
		return false
	}
	if !envBoolEnabled("ORQUESTA_FORCE_LOCAL_DB") && !envBoolEnabled("ORQUESTA_FORCE_LOCAL") {
		return false
	}
	target := strings.TrimSpace(cfg.Path)
	if target == "" {
		return false
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Size() > 0
}

func envBoolEnabled(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return value == "1" || value == "true" || value == "yes" || value == "si" || value == "on"
}

func Close() {
	persistenceMu.Lock()
	handle := DB
	DB = nil
	currentBackend = nil
	currentConfig = storage.Config{}
	currentTarget = ""
	persistenceMu.Unlock()
	if handle != nil {
		_ = handle.Close()
	}
	runtimeHandleHotReset()
	resetRuntimeOrdersHotIndex()
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
	return postMigracionesPorDriver(conn, DriverName())
}

func postMigracionesPorDriver(conn *sql.DB, driver string) error {
	migraciones := postMigrationStatementsForDriver(driver)
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
	return aplicarSchemaPorDriver(db, DriverName())
}

func aplicarSchemaPorDriver(db *sql.DB, driver string) error {
	if normalizedDriverName(driver) == "sqlite" {
		if err := prepararSchemaLegacy(db); err != nil {
			return err
		}
	}
	if err := ejecutarConReintentos(func() error {
		return aplicarDDL(db, schemaDDLForDriver(driver))
	}); err != nil {
		return err
	}
	return ejecutarConReintentos(func() error {
		return aplicarSeedSQL(db, schemaSeedDataForDriver(driver))
	})
}

func ejecutarConReintentos(fn func() error) error {
	var err error
	for intento := 0; intento < persistenciaBusyMaxIntentos; intento++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !esErrorPersistenciaBusy(err) {
			return err
		}
		time.Sleep(persistenciaBusyBackoff)
	}
	return err
}

func consultarConReintentos[T any](fn func() (T, error)) (T, error) {
	var (
		out T
		err error
	)
	for intento := 0; intento < persistenciaBusyMaxIntentos; intento++ {
		out, err = fn()
		if err == nil {
			return out, nil
		}
		if !esErrorPersistenciaBusy(err) {
			return out, err
		}
		time.Sleep(persistenciaBusyBackoff)
	}
	return out, err
}

func esErrorPersistenciaBusy(err error) bool {
	backend, _, _ := currentPersistenceState()
	if backend != nil {
		return backend.IsBusy(err)
	}
	return false
}

func esErrorMigracionIgnorable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column name") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "duplicate key name")
}

func prepararSchemaLegacy(db *sql.DB) error {
	if err := reconstruirPoliticasModeloLegacy(db); err != nil {
		return err
	}
	if err := reconstruirRuntimeHandlesLegacy(db); err != nil {
		return err
	}
	if err := reconstruirRuntimeOrdersLegacy(db); err != nil {
		return err
	}
	if err := reconstruirRuntimeMailboxLegacy(db); err != nil {
		return err
	}
	if err := reconstruirAutonomiaCiclosLegacy(db); err != nil {
		return err
	}
	return nil
}

func reconstruirPoliticasModeloLegacy(db *sql.DB) error {
	const tableName = "politicas_modelo"
	existe, err := tablaExiste(db, tableName)
	if err != nil || !existe {
		return err
	}
	sqlText, err := tablaSQL(db, tableName)
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(sqlText), "'agente'") {
		return nil
	}
	return rebuildSQLiteTable(
		db,
		tableName,
		nil,
		`CREATE TABLE politicas_modelo (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			scope_tipo        TEXT    NOT NULL
			                         CHECK (scope_tipo IN ('global','perfil','proyecto','fase','agente','tarea')),
			scope_ref         TEXT    NOT NULL DEFAULT '',
			perfil_tarea      TEXT    NOT NULL DEFAULT '*',
			pool_slug         TEXT    NOT NULL DEFAULT '',
			model_slug        TEXT    NOT NULL DEFAULT '',
			reasoning_effort  TEXT    NOT NULL DEFAULT '',
			prioridad         INTEGER NOT NULL DEFAULT 100,
			activa            INTEGER NOT NULL DEFAULT 1,
			metadata_json     TEXT    NOT NULL DEFAULT '{}',
			created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		nil,
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			insertCols := []string{
				"id", "scope_tipo", "scope_ref", "perfil_tarea", "pool_slug",
				"model_slug", "reasoning_effort", "prioridad", "activa",
				"metadata_json", "created_at", "updated_at",
			}
			selectExprs := []string{
				runtimeLegacyExpr(legacyCols, "id", "NULL"),
				runtimeLegacyExpr(legacyCols, "scope_tipo", "'global'"),
				runtimeLegacyExpr(legacyCols, "scope_ref", "''"),
				runtimeLegacyExpr(legacyCols, "perfil_tarea", "'*'"),
				runtimeLegacyExpr(legacyCols, "pool_slug", "''"),
				runtimeLegacyExpr(legacyCols, "model_slug", "''"),
				runtimeLegacyExpr(legacyCols, "reasoning_effort", "''"),
				runtimeLegacyExpr(legacyCols, "prioridad", "100"),
				runtimeLegacyExpr(legacyCols, "activa", "1"),
				runtimeLegacyExpr(legacyCols, "metadata_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP"),
				runtimeLegacyExpr(legacyCols, "updated_at", runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP")),
			}
			return `INSERT INTO politicas_modelo (` + strings.Join(insertCols, ", ") + `)
				SELECT ` + strings.Join(selectExprs, ", ") + ` FROM ` + legacy, nil
		},
	)
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
			claimed_by TEXT NOT NULL DEFAULT '',
			lease_token TEXT NOT NULL DEFAULT '',
			attempt_count INTEGER NOT NULL DEFAULT 0,
			lease_expires_at DATETIME,
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
				"resultado_json", "error_text", "estado", "available_at", "claimed_by",
				"lease_token", "attempt_count", "lease_expires_at", "created_at",
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
				runtimeLegacyExpr(legacyCols, "claimed_by", "''"),
				runtimeLegacyExpr(legacyCols, "lease_token", "''"),
				runtimeLegacyExpr(legacyCols, "attempt_count", "0"),
				runtimeLegacyExpr(legacyCols, "lease_expires_at", "NULL"),
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

func reconstruirAutonomiaCiclosLegacy(db *sql.DB) error {
	const tableName = "autonomia_ciclos"
	existe, err := tablaExiste(db, tableName)
	if err != nil || !existe {
		return err
	}
	sqlText, err := tablaSQL(db, tableName)
	if err != nil {
		return err
	}
	sqlNorm := strings.ToLower(sqlText)
	if strings.TrimSpace(sqlNorm) != "" && strings.Contains(sqlNorm, "review_feedback") {
		return nil
	}
	return rebuildSQLiteTable(
		db,
		tableName,
		[]string{"idx_autonomia_ciclos_proyecto_kind"},
		`CREATE TABLE autonomia_ciclos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			proyecto_id INTEGER NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE,
			kind TEXT NOT NULL CHECK (kind IN ('supervision','review','review_feedback','closure')),
			agente TEXT NOT NULL DEFAULT '',
			sesion_id INTEGER REFERENCES sesiones(id) ON DELETE SET NULL,
			runtime_id INTEGER REFERENCES runtime_instances(id) ON DELETE SET NULL,
			input_json TEXT NOT NULL DEFAULT '{}',
			decision_json TEXT NOT NULL DEFAULT '{}',
			resultado TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		[]string{
			`CREATE INDEX idx_autonomia_ciclos_proyecto_kind ON autonomia_ciclos(proyecto_id, kind, id DESC)`,
		},
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			insertCols := []string{
				"id", "proyecto_id", "kind", "agente", "sesion_id", "runtime_id",
				"input_json", "decision_json", "resultado", "created_at",
			}
			selectExprs := []string{
				runtimeLegacyExpr(legacyCols, "id", "NULL"),
				runtimeLegacyExpr(legacyCols, "proyecto_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "kind", "'supervision'"),
				runtimeLegacyExpr(legacyCols, "agente", "''"),
				runtimeLegacyExpr(legacyCols, "sesion_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "runtime_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "input_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "decision_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "resultado", "''"),
				runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP"),
			}
			return `INSERT INTO autonomia_ciclos (` + strings.Join(insertCols, ", ") + `)
				SELECT ` + strings.Join(selectExprs, ", ") + ` FROM ` + legacy, nil
		},
	)
}

func reconstruirRuntimeMailboxLegacy(db *sql.DB) error {
	const tableName = "runtime_mailbox"
	existe, err := tablaExiste(db, tableName)
	if err != nil || !existe {
		return err
	}
	sqlText, err := tablaSQL(db, tableName)
	if err != nil {
		return err
	}
	sqlNorm := strings.ToLower(strings.TrimSpace(sqlText))
	if sqlNorm != "" && !strings.Contains(sqlNorm, "from_agente         text    not null references agentes(nombre)") &&
		!strings.Contains(sqlNorm, "from_agente text not null references agentes(nombre)") {
		return nil
	}
	return rebuildSQLiteTable(
		db,
		tableName,
		[]string{"idx_runtime_mailbox_destino_estado"},
		`CREATE TABLE runtime_mailbox (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_agente TEXT NOT NULL,
			to_agente TEXT NOT NULL REFERENCES agentes(nombre),
			proyecto_id INTEGER REFERENCES proyectos(id) ON DELETE SET NULL,
			runtime_order_id INTEGER REFERENCES runtime_orders(id) ON DELETE SET NULL,
			kind TEXT NOT NULL,
			payload_json TEXT NOT NULL DEFAULT '{}',
			estado TEXT NOT NULL DEFAULT 'pendiente'
				CHECK (estado IN ('pendiente','entregado','consumido','expirado','cancelado')),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			delivered_at DATETIME,
			consumed_at DATETIME
		)`,
		[]string{
			`CREATE INDEX idx_runtime_mailbox_destino_estado ON runtime_mailbox(to_agente, estado, id DESC)`,
		},
		func(legacy string, legacyCols map[string]bool) (string, []any) {
			insertCols := []string{
				"id", "from_agente", "to_agente", "proyecto_id", "runtime_order_id",
				"kind", "payload_json", "estado", "created_at", "delivered_at", "consumed_at",
			}
			selectExprs := []string{
				runtimeLegacyExpr(legacyCols, "id", "NULL"),
				runtimeLegacyExpr(legacyCols, "from_agente", "'server'"),
				runtimeLegacyExpr(legacyCols, "to_agente", "''"),
				runtimeLegacyExpr(legacyCols, "proyecto_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "runtime_order_id", "NULL"),
				runtimeLegacyExpr(legacyCols, "kind", "'instruction'"),
				runtimeLegacyExpr(legacyCols, "payload_json", "'{}'"),
				runtimeLegacyExpr(legacyCols, "estado", "'pendiente'"),
				runtimeLegacyExpr(legacyCols, "created_at", "CURRENT_TIMESTAMP"),
				runtimeLegacyExpr(legacyCols, "delivered_at", "NULL"),
				runtimeLegacyExpr(legacyCols, "consumed_at", "NULL"),
			}
			return `INSERT INTO runtime_mailbox (` + strings.Join(insertCols, ", ") + `)
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
		if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
			log.Printf("orquesta/db: no se pudo reactivar foreign_keys tras rebuild de %s: %v", tableName, err)
		}
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
		if _, err := DB.Exec(`UPDATE agentes SET estado_cuota = 'enfriamiento', reanimar_at = ?, motivo_pausa = ? WHERE nombre = ?`,
			reanimar, "Intento de bypass de seguridad detectado (audit-warden)", agente); err != nil {
			log.Printf("orquesta/db: audit bloqueo preventivo fallido agente=%s accion=%s entidad=%s id=%d err=%v", strings.TrimSpace(agente), strings.TrimSpace(accion), strings.TrimSpace(entidad), entidadID, err)
		}
	}

	if _, err := DB.Exec(
		`INSERT INTO audit_log (agente, accion, entidad, entidad_id, detalle) VALUES (?,?,?,?,?)`,
		agente, accion, entidad, entidadID, detalle,
	); err != nil {
		log.Printf("orquesta/db: audit insert fallido agente=%s accion=%s entidad=%s id=%d err=%v detalle=%q", strings.TrimSpace(agente), strings.TrimSpace(accion), strings.TrimSpace(entidad), entidadID, err, strings.TrimSpace(detalle))
	}
}

// ListarAuditoria recupera registros del log de auditoría.
func ListarAuditoria(f FiltroAuditoria) ([]*LogAuditoria, error) {
	return consultarConReintentos(func() ([]*LogAuditoria, error) {
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
	})
}

// ConfigGet devuelve el valor de una clave de configuración.
func ConfigGet(clave string) (string, error) {
	return consultarConReintentos(func() (string, error) {
		var v string
		err := DB.QueryRow(`SELECT valor FROM config WHERE clave = ?`, clave).Scan(&v)
		return v, err
	})
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
