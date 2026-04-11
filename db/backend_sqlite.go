package db

import (
	"database/sql"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"orquesta/storage"

	_ "modernc.org/sqlite"
)

type sqliteBackend struct{}

const (
	sqliteBootstrapRevisionKey = "_sqlite_bootstrap_revision"
	sqliteBootstrapRevision    = "2026-03-30-runtime-supervision-v1"
)

var sqliteAlterAddColumnPattern = regexp.MustCompile(`(?i)^\s*ALTER\s+TABLE\s+([^\s]+)\s+ADD\s+COLUMN\s+([^\s(]+)`)

func init() {
	RegisterBackend(sqliteBackend{})
}

func (sqliteBackend) Name() string { return "sqlite" }

func (sqliteBackend) Open(cfg storage.Config) (*sql.DB, error) {
	target := strings.TrimSpace(cfg.Path)
	if target == "" {
		target = strings.TrimSpace(cfg.DSN)
	}
	db, err := storage.Open(cfg)
	if err != nil {
		return nil, fmt.Errorf("abriendo DB sqlite en %s: %w", target, err)
	}
	if err := sqliteApplyConnectionPragmas(db, cfg); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configurando pragmas sqlite en %s: %w", target, err)
	}
	return db, nil
}

func (sqliteBackend) Prepare(db *sql.DB, cfg storage.Config) error {
	if !cfg.BootstrapSchema && cfg.SkipPostMigrations {
		return nil
	}
	revisionApplied, err := sqliteBootstrapRevisionApplied(db)
	if err != nil {
		return fmt.Errorf("verificando revision sqlite: %w", err)
	}
	needsBootstrap := false
	if cfg.BootstrapSchema {
		needsBootstrap, err = sqliteBootstrapRequired(db)
		if err != nil {
			return fmt.Errorf("verificando schema sqlite: %w", err)
		}
		if needsBootstrap {
			if err := aplicarSchemaPorDriver(db, "sqlite"); err != nil {
				return fmt.Errorf("aplicando schema sqlite: %w", err)
			}
		}
	}
	if cfg.SkipPostMigrations {
		return nil
	}
	if err := sqliteNormalizeOpenProposalState(db); err != nil {
		if !needsBootstrap && (sqliteBackend{}).IsBusy(err) {
			return nil
		}
		return fmt.Errorf("normalizando propuestas sqlite: %w", err)
	}
	if err := postMigracionesPorDriver(db, "sqlite"); err != nil {
		if !needsBootstrap && (sqliteBackend{}).IsBusy(err) {
			return nil
		}
		return fmt.Errorf("post-migraciones sqlite: %w", err)
	}
	if err := sqliteEnsureDefaultConfig(db); err != nil {
		if !needsBootstrap && (sqliteBackend{}).IsBusy(err) {
			return nil
		}
		return fmt.Errorf("repoblando config sqlite: %w", err)
	}
	if cfg.BootstrapSchema && !revisionApplied {
		if err := sqliteMarkBootstrapRevision(db); err != nil {
			if !needsBootstrap && (sqliteBackend{}).IsBusy(err) {
				return nil
			}
			return fmt.Errorf("registrando revision sqlite: %w", err)
		}
	}
	return nil
}

func sqliteEnsureDefaultConfig(db *sql.DB) error {
	entries := defaultConfigEntries()
	for _, item := range entries {
		clave := strings.TrimSpace(item.Clave)
		if clave == "" {
			continue
		}
		valor := item.Valor
		if err := ejecutarConReintentos(func() error {
			_, err := db.Exec(
				`INSERT INTO config (clave, valor) VALUES (?, ?) ON CONFLICT(clave) DO NOTHING`,
				clave,
				valor,
			)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

func (sqliteBackend) Backup(db *sql.DB, path string) error {
	if _, err := db.Exec(fmt.Sprintf("VACUUM INTO %s", literalSQLite(path))); err != nil {
		return fmt.Errorf("crear respaldo sqlite: %w", err)
	}
	return nil
}

func (sqliteBackend) Verify(raw *sql.DB, cfg storage.Config) *InformePersistencia {
	informe := nuevoInformePersistencia(cfg)
	if !verificarConexionPersistencia(informe, raw) {
		return informe
	}
	quickCheck, err := escalarCadena(raw, "sqlite", `PRAGMA quick_check`)
	if err != nil {
		registrarComprobacionPersistencia(informe, "quick_check", "error", err.Error())
	} else if strings.EqualFold(strings.TrimSpace(quickCheck), "ok") {
		registrarComprobacionPersistencia(informe, "quick_check", "ok", "ok")
	} else {
		registrarComprobacionPersistencia(informe, "quick_check", "error", quickCheck)
	}
	journalMode, err := escalarCadena(raw, "sqlite", `PRAGMA journal_mode`)
	if err != nil {
		registrarComprobacionPersistencia(informe, "config_sqlite", "warn", "journal_mode no disponible: "+err.Error())
	} else {
		journalMode = strings.ToLower(strings.TrimSpace(journalMode))
		if sqliteDSNMode(cfg.DSN) == "ro" {
			registrarComprobacionPersistencia(informe, "config_sqlite", "ok", "journal_mode="+journalMode+" (read-only)")
		} else if journalMode != "wal" {
			registrarComprobacionPersistencia(informe, "config_sqlite", "error", "journal_mode="+journalMode+" (se esperaba wal)")
		} else {
			registrarComprobacionPersistencia(informe, "config_sqlite", "ok", "journal_mode="+journalMode)
		}
	}
	verificarTablasCorePersistencia(informe, raw, "sqlite")
	return informe
}

func (sqliteBackend) IsBusy(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "sqlite_busy")
}

func literalSQLite(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "''") + "'"
}

func sqliteApplyConnectionPragmas(db *sql.DB, cfg storage.Config) error {
	if db == nil {
		return nil
	}
	if sqliteDSNMode(cfg.DSN) == "ro" {
		return nil
	}
	if _, err := db.Exec(`PRAGMA journal_mode = WAL`); err != nil {
		if (sqliteBackend{}).IsBusy(err) {
			return nil
		}
		return err
	}
	return nil
}

func sqliteDSNMode(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return ""
	}
	if idx := strings.IndexByte(dsn, '?'); idx >= 0 && idx+1 < len(dsn) {
		if values, err := url.ParseQuery(dsn[idx+1:]); err == nil {
			return strings.TrimSpace(strings.ToLower(values.Get("mode")))
		}
	}
	return ""
}

func sqliteBootstrapRequired(db *sql.DB) (bool, error) {
	requiredTables := []string{
		"config",
		"agentes",
		"sesiones",
		"tareas",
		"especificaciones_funcion",
		"propuestas",
		"runtime_handles",
		"runtime_orders",
		"autonomia_ciclos",
	}
	for _, table := range requiredTables {
		exists, err := tablaExiste(db, table)
		if err != nil {
			return false, err
		}
		if !exists {
			return true, nil
		}
	}

	requiredColumns, err := sqliteRequiredPostMigrationColumns()
	if err != nil {
		return false, err
	}
	for _, check := range requiredColumns {
		ok, err := tablaTieneColumna(db, check.table, check.column)
		if err != nil {
			return false, err
		}
		if !ok {
			return true, nil
		}
	}

	needsRebuild, err := sqliteRuntimeHandlesNeedsRebuild(db)
	if err != nil {
		return false, err
	}
	if needsRebuild {
		return true, nil
	}

	needsRebuild, err = sqliteRuntimeOrdersNeedsRebuild(db)
	if err != nil {
		return false, err
	}
	if needsRebuild {
		return true, nil
	}

	needsRebuild, err = sqliteAutonomiaCiclosNeedsRebuild(db)
	if err != nil {
		return false, err
	}
	if needsRebuild {
		return true, nil
	}

	needsRebuild, err = sqliteRuntimeMailboxNeedsRebuild(db)
	if err != nil {
		return false, err
	}
	return needsRebuild, nil
}

func sqliteBootstrapRevisionApplied(db *sql.DB) (bool, error) {
	exists, err := tablaExiste(db, "config")
	if err != nil || !exists {
		return false, err
	}
	var value string
	err = db.QueryRow(`SELECT valor FROM config WHERE clave = ?`, sqliteBootstrapRevisionKey).Scan(&value)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(value) == sqliteBootstrapRevision, nil
}

func sqliteMarkBootstrapRevision(db *sql.DB) error {
	return ejecutarConReintentos(func() error {
		_, err := db.Exec(
			`INSERT INTO config (clave, valor) VALUES (?, ?) ON CONFLICT(clave) DO UPDATE SET valor=excluded.valor`,
			sqliteBootstrapRevisionKey,
			sqliteBootstrapRevision,
		)
		return err
	})
}

type sqliteSchemaCheck struct {
	table  string
	column string
}

func sqliteRequiredPostMigrationColumns() ([]sqliteSchemaCheck, error) {
	stmts := postMigrationStatementsForDriver("sqlite")
	out := make([]sqliteSchemaCheck, 0, len(stmts))
	seen := make(map[string]struct{}, len(stmts))
	for _, stmt := range stmts {
		matches := sqliteAlterAddColumnPattern.FindStringSubmatch(strings.TrimSpace(stmt))
		if len(matches) != 3 {
			continue
		}
		table := sqliteCleanIdentifier(matches[1])
		column := sqliteCleanIdentifier(matches[2])
		if table == "" || column == "" {
			continue
		}
		key := table + "." + column
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, sqliteSchemaCheck{table: table, column: column})
	}
	return out, nil
}

func sqliteCleanIdentifier(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), "`\"[]")
}

func sqliteRuntimeHandlesNeedsRebuild(db *sql.DB) (bool, error) {
	cols := []string{
		"id", "agente", "proyecto_id", "sesion_id", "runtime_id", "transporte", "handle_kind",
		"handle_ref", "estado", "lease_token", "capabilities_json", "metadata_json",
		"last_seen_at", "created_at", "updated_at",
	}
	for _, col := range cols {
		ok, err := tablaTieneColumna(db, "runtime_handles", col)
		if err != nil {
			return false, err
		}
		if !ok {
			return true, nil
		}
	}
	return false, nil
}

func sqliteRuntimeOrdersNeedsRebuild(db *sql.DB) (bool, error) {
	cols := []string{
		"id", "agente", "proyecto_id", "runtime_id", "handle_id", "tipo", "payload_json",
		"resultado_json", "error_text", "estado", "available_at", "claimed_by",
		"lease_token", "attempt_count", "lease_expires_at", "created_at",
		"started_at", "finished_at", "updated_at",
	}
	for _, col := range cols {
		ok, err := tablaTieneColumna(db, "runtime_orders", col)
		if err != nil {
			return false, err
		}
		if !ok {
			return true, nil
		}
	}
	sqlText, err := tablaSQL(db, "runtime_orders")
	if err != nil {
		return false, err
	}
	sqlNorm := strings.ToLower(strings.TrimSpace(sqlText))
	if sqlNorm != "" && (!strings.Contains(sqlNorm, "tomada") || !strings.Contains(sqlNorm, "ejecutando")) {
		return true, nil
	}
	return false, nil
}

func sqliteAutonomiaCiclosNeedsRebuild(db *sql.DB) (bool, error) {
	sqlText, err := tablaSQL(db, "autonomia_ciclos")
	if err != nil {
		return false, err
	}
	sqlNorm := strings.ToLower(strings.TrimSpace(sqlText))
	return sqlNorm == "" || !strings.Contains(sqlNorm, "review_feedback"), nil
}

func sqliteRuntimeMailboxNeedsRebuild(db *sql.DB) (bool, error) {
	sqlText, err := tablaSQL(db, "runtime_mailbox")
	if err != nil {
		return false, err
	}
	sqlNorm := strings.ToLower(strings.TrimSpace(sqlText))
	if sqlNorm == "" {
		return false, nil
	}
	return strings.Contains(sqlNorm, "from_agente text not null references agentes(nombre)") ||
		strings.Contains(sqlNorm, "from_agente         text    not null references agentes(nombre)"), nil
}

func sqliteNormalizeOpenProposalState(db *sql.DB) error {
	exists, err := tablaExiste(db, "propuestas")
	if err != nil || !exists {
		return err
	}
	for _, col := range []string{"estado", "cerrada_at"} {
		ok, err := tablaTieneColumna(db, "propuestas", col)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	var inconsistent int
	if err := db.QueryRow(`SELECT COUNT(*) FROM propuestas WHERE estado = 'abierta' AND cerrada_at IS NOT NULL`).Scan(&inconsistent); err != nil {
		return err
	}
	if inconsistent == 0 {
		return nil
	}
	return ejecutarConReintentos(func() error {
		_, err := db.Exec(`UPDATE propuestas SET cerrada_at = NULL WHERE estado = 'abierta' AND cerrada_at IS NOT NULL`)
		return err
	})
}
