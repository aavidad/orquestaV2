package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"orquesta/storage"
)

type Backend interface {
	Name() string
	Open(target string) (*sql.DB, error)
	Prepare(*sql.DB) error
	Backup(*sql.DB, string) error
	IsBusy(error) bool
}

var (
	backendRegistry = map[string]Backend{}
	currentBackend  Backend
	currentTarget   string
)

func IsOpen() bool {
	return DB != nil
}

func CurrentDBPath() string {
	if strings.TrimSpace(currentTarget) != "" {
		return currentTarget
	}
	return backendTarget(backendName())
}

func RegisterBackend(backend Backend) {
	if backend == nil {
		return
	}
	backendRegistry[strings.TrimSpace(strings.ToLower(backend.Name()))] = backend
}

func backendName() string {
	if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB_DRIVER")); v != "" {
		if name := storage.DialectForDriver(v).Name; name != "" {
			return name
		}
	}
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("ORQUESTA_DB_BACKEND"))); v != "" {
		if name := storage.DialectForDriver(v).Name; name != "" {
			return name
		}
	}
	return "sqlite"
}

func backendTarget(name string) string {
	switch strings.TrimSpace(strings.ToLower(name)) {
	case "sqlite":
		return resolverRuta()
	default:
		if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB_DSN")); v != "" {
			return v
		}
		if v := strings.TrimSpace(os.Getenv("ORQUESTA_DB")); v != "" {
			return v
		}
		return ""
	}
}

func resolveBackend() (Backend, string, error) {
	name := backendName()
	backend, ok := backendRegistry[name]
	if !ok {
		return nil, "", fmt.Errorf("backend de persistencia no soportado: %s", name)
	}
	target := backendTarget(name)
	if strings.TrimSpace(target) == "" {
		return nil, "", fmt.Errorf("target/DSN vacío para backend %s", name)
	}
	return backend, target, nil
}

func CurrentBackendName() string {
	if currentBackend == nil {
		return ""
	}
	return currentBackend.Name()
}

func DriverName() string {
	if currentBackend != nil {
		return normalizedDriverName(currentBackend.Name())
	}
	return normalizedDriverName(backendName())
}

func normalizedDriverName(driver string) string {
	return storage.DialectForDriver(driver).Name
}

func PlaceholderStyle() string {
	return storage.DialectForDriver(DriverName()).PlaceholderStyle()
}

func QueryRebindingEnabled() bool {
	return storage.DialectForDriver(DriverName()).RebindParameters()
}

func BackupTo(path string) error {
	if DB == nil {
		return fmt.Errorf("la base de datos no está inicializada")
	}
	if currentBackend == nil {
		return fmt.Errorf("backend de persistencia no inicializado")
	}
	return currentBackend.Backup(DB.DB, path)
}
