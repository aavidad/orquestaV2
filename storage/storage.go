package storage

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

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
	driver := normalizeDriver(envFirst("ORQUESTA_DB_DRIVER", "ORQUESTA_DB_BACKEND"))
	if driver == "" {
		driver = "sqlite"
	}

	cfg := Config{
		Driver:          driver,
		MaxOpenConns:    defaultMaxOpenConns(driver),
		BootstrapSchema: defaultBootstrapSchema(driver),
	}
	if v := strings.TrimSpace(envFirst("ORQUESTA_DB_MAX_OPEN_CONNS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("ORQUESTA_DB_MAX_OPEN_CONNS invalido: %q", v)
		}
		cfg.MaxOpenConns = n
	}
	if v := strings.TrimSpace(envFirst("ORQUESTA_DB_BOOTSTRAP")); v != "" {
		enabled, err := parseBool(v)
		if err != nil {
			return Config{}, fmt.Errorf("ORQUESTA_DB_BOOTSTRAP invalido: %q", v)
		}
		cfg.BootstrapSchema = enabled
	}

	dsn := strings.TrimSpace(envFirst("ORQUESTA_DB_DSN"))
	switch driver {
	case "sqlite":
		cfg.Path = strings.TrimSpace(envFirst("ORQUESTA_DB"))
		if cfg.Path == "" && pathResolver != nil {
			cfg.Path = strings.TrimSpace(pathResolver())
		}
		if cfg.Path == "" {
			cfg.Path = "orquesta.db"
		}
		if dsn == "" {
			dsn = SQLiteDSN(cfg.Path)
		}
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
	if driver == "" {
		return nil, fmt.Errorf("driver de almacenamiento obligatorio")
	}
	if strings.TrimSpace(cfg.DSN) == "" {
		return nil, fmt.Errorf("dsn de almacenamiento obligatorio para driver %s", driver)
	}
	sqlDriver := sqlDriverName(driver)
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

func DisplayTarget(cfg Config) string {
	driver := normalizeDriver(cfg.Driver)
	if driver == "sqlite" || driver == "" {
		if strings.TrimSpace(cfg.Path) != "" {
			return strings.TrimSpace(cfg.Path)
		}
		return strings.TrimSpace(cfg.DSN)
	}
	return redactDSN(strings.TrimSpace(cfg.DSN))
}

func redactDSN(dsn string) string {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return ""
	}
	if strings.Contains(dsn, "://") {
		if parsed, err := url.Parse(dsn); err == nil {
			if parsed.User != nil {
				username := parsed.User.Username()
				if _, hasPassword := parsed.User.Password(); hasPassword {
					parsed.User = url.UserPassword(username, "***")
					return parsed.String()
				}
			}
			return parsed.String()
		}
	}
	at := strings.LastIndex(dsn, "@")
	if at <= 0 {
		return dsn
	}
	auth := dsn[:at]
	rest := dsn[at:]
	colon := strings.Index(auth, ":")
	if colon < 0 {
		return dsn
	}
	return auth[:colon+1] + "***" + rest
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
	case "", "sqlite":
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
