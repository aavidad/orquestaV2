package storage

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"

<<<<<<< HEAD
=======
	_ "github.com/jackc/pgx/v5/stdlib"
>>>>>>> origin/orq-orquestador-codex2
	_ "modernc.org/sqlite"
)

type Config struct {
	Driver          string
	DSN             string
	Path            string
	MaxOpenConns    int
	BootstrapSchema bool
}

func ResolveConfig(pathResolver func() string) (Config, error) {
<<<<<<< HEAD
	driver := normalizeDriver(envFirst("ORQUESTA_DB_DRIVER", "ORQUESTA_DB_BACKEND"))
=======
	driver := normalizeDriver(os.Getenv("ORQUESTA_DB_DRIVER"))
>>>>>>> origin/orq-orquestador-codex2
	if driver == "" {
		driver = "sqlite"
	}

	cfg := Config{
		Driver:          driver,
		MaxOpenConns:    defaultMaxOpenConns(driver),
		BootstrapSchema: defaultBootstrapSchema(driver),
	}
<<<<<<< HEAD
	if v := strings.TrimSpace(envFirst("ORQUESTA_DB_MAX_OPEN_CONNS")); v != "" {
=======
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB_MAX_OPEN_CONNS")); v != "" {
>>>>>>> origin/orq-orquestador-codex2
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("ORQUESTA_DB_MAX_OPEN_CONNS invalido: %q", v)
		}
		cfg.MaxOpenConns = n
	}
<<<<<<< HEAD
	if v := strings.TrimSpace(envFirst("ORQUESTA_DB_BOOTSTRAP")); v != "" {
=======
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB_BOOTSTRAP")); v != "" {
>>>>>>> origin/orq-orquestador-codex2
		enabled, err := parseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("ORQUESTA_DB_BOOTSTRAP invalido: %q", v)
		}
		cfg.BootstrapSchema = enabled
	}

<<<<<<< HEAD
	dsn := strings.TrimSpace(envFirst("ORQUESTA_DB_DSN"))
	switch driver {
	case "sqlite":
		cfg.Path = strings.TrimSpace(envFirst("ORQUESTA_DB"))
=======
	dsn := strings.TrimSpace(os.Getenv("ORQUESTA_DB_DSN"))
	switch driver {
	case "sqlite":
		cfg.Path = strings.TrimSpace(os.Getenv("ORQUESTA_DB"))
>>>>>>> origin/orq-orquestador-codex2
		if cfg.Path == "" && pathResolver != nil {
			cfg.Path = strings.TrimSpace(pathResolver())
		}
		if cfg.Path == "" {
			cfg.Path = "orquesta.db"
		}
		if dsn == "" {
			dsn = SQLiteDSN(cfg.Path)
		}
<<<<<<< HEAD
=======
	case "mysql", "postgres":
		if dsn == "" {
			return Config{}, fmt.Errorf("ORQUESTA_DB_DSN es obligatorio para driver %s", driver)
		}
>>>>>>> origin/orq-orquestador-codex2
	default:
		if dsn == "" {
			return Config{}, fmt.Errorf("ORQUESTA_DB_DSN es obligatorio para driver %s", driver)
		}
	}
	cfg.DSN = dsn
	return cfg, nil
}

func Open(cfg Config) (*sql.DB, error) {
	driver := normalizeDriver(cfg.Driver)
<<<<<<< HEAD
=======
	sqlDriver := sqlDriverName(driver)
>>>>>>> origin/orq-orquestador-codex2
	if driver == "" {
		return nil, fmt.Errorf("driver de almacenamiento obligatorio")
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("dsn de almacenamiento obligatorio para driver %s", driver)
	}
<<<<<<< HEAD
	sqlDriver := sqlDriverName(driver)
=======
>>>>>>> origin/orq-orquestador-codex2
	if !driverRegistered(sqlDriver) {
		return nil, fmt.Errorf("driver de almacenamiento %q no está enlazado en el binario", driver)
	}
	db, err := sql.Open(sqlDriver, cfg.DSN)
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	return db, nil
}

func SQLiteDSN(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "orquesta.db"
	}
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + "_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
}

func normalizeDriver(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "sqlite", "sqlite3":
		return "sqlite"
	case "postgresql":
		return "postgres"
	default:
		return strings.TrimSpace(strings.ToLower(v))
	}
}

func defaultMaxOpenConns(driver string) int {
	if normalizeDriver(driver) == "sqlite" || normalizeDriver(driver) == "" {
		return 1
	}
	return 10
}

func defaultBootstrapSchema(driver string) bool {
	switch normalizeDriver(driver) {
<<<<<<< HEAD
	case "", "sqlite":
=======
	case "", "sqlite", "postgres":
>>>>>>> origin/orq-orquestador-codex2
		return true
	default:
		return false
	}
}

func sqlDriverName(driver string) string {
	switch normalizeDriver(driver) {
	case "postgres":
		return "pgx"
	default:
		return normalizeDriver(driver)
	}
}

func parseBool(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "si", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("valor booleano invalido")
	}
}

func driverRegistered(name string) bool {
	for _, registered := range sql.Drivers() {
		if registered == name {
			return true
		}
	}
	return false
}
<<<<<<< HEAD

func envFirst(keys ...string) string {
	for _, key := range keys {
		if value, ok := lookupEnv(key); ok {
			return value
		}
	}
	return ""
}

func lookupEnv(key string) (string, bool) {
	value, ok := os.LookupEnv(key)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}
=======
>>>>>>> origin/orq-orquestador-codex2
